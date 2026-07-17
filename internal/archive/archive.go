package archive

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Active coding agents may briefly create mode-000 lock or temporary files
// before replacing or deleting them. Retry only permission failures; stable
// unreadable files still fail after this bounded window.
var stagePermissionRetries = 40
var stagePermissionRetryDelay = 50 * time.Millisecond
var openStageSource = os.Open

// Create bundles the given relative files (rooted at srcRoot) into a gzip'd tar
// at bundlePath and returns a completed manifest. Files are stored using their
// forward-slash relative paths so archives are portable across OSes.
func Create(srcRoot string, rels []string, bundlePath string, m Manifest) (Manifest, error) {
	bundleDir := filepath.Dir(bundlePath)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return m, err
	}
	// Build beside the destination and publish only after the tarball closes
	// successfully. A failed collection therefore leaves the last good bundle
	// untouched.
	f, err := os.CreateTemp(bundleDir, ".bundle-*.tmp")
	if err != nil {
		return m, err
	}
	tmpBundle := f.Name()
	defer os.Remove(tmpBundle)

	gz, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		return m, err
	}
	tw := tar.NewWriter(gz)

	sort.Strings(rels)
	m.Files = m.Files[:0]

	for _, rel := range rels {
		abs := filepath.Join(srcRoot, filepath.FromSlash(rel))
		info, err := os.Lstat(abs)
		if err != nil {
			// Active applications continuously create and remove locks and temp
			// files. A discovered path that is already gone has no bytes left to
			// sync; every path that still exists remains included.
			if os.IsNotExist(err) {
				continue
			}
			return m, fmt.Errorf("inspect %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(abs)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return m, fmt.Errorf("read symlink %s: %w", rel, err)
			}
			if err := writeSymlink(tw, rel, target, info); err != nil {
				return m, fmt.Errorf("archive symlink %s: %w", rel, err)
			}
			m.Files = append(m.Files, FileEntry{
				Path:       rel,
				Mode:       uint32(info.Mode().Perm()),
				SHA256:     textSHA256(target),
				Type:       "symlink",
				LinkTarget: target,
			})
			continue
		}
		if !info.Mode().IsRegular() {
			return m, fmt.Errorf("unsupported non-regular state path: %s", rel)
		}
		entry, staged, err := stageFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return m, fmt.Errorf("stage %s: %w", rel, err)
		}
		if staged == nil {
			continue
		}
		sum, err := writeFile(tw, staged, rel, entry)
		staged.Close()
		if err != nil {
			return m, fmt.Errorf("archive %s: %w", rel, err)
		}
		m.Files = append(m.Files, FileEntry{
			Path:   rel,
			Size:   entry.Size(),
			Mode:   uint32(entry.Mode().Perm()),
			SHA256: sum,
		})
	}

	if err := tw.Close(); err != nil {
		return m, err
	}
	if err := gz.Close(); err != nil {
		return m, err
	}
	if err := f.Sync(); err != nil {
		return m, err
	}
	if err := f.Close(); err != nil {
		return m, err
	}
	m.BundleParts, err = publishBundle(tmpBundle, bundlePath)
	if err != nil {
		return m, err
	}
	return m, nil
}

func writeSymlink(tw *tar.Writer, rel, target string, info os.FileInfo) error {
	return tw.WriteHeader(&tar.Header{
		Name:     rel,
		Mode:     int64(info.Mode().Perm()),
		ModTime:  info.ModTime(),
		Typeflag: tar.TypeSymlink,
		Linkname: target,
	})
}

func textSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// bundlePartSize is a variable so tests can exercise chunking without creating
// huge fixtures. Production archives use 5 MiB chunks.
var bundlePartSize int64 = 5 * 1024 * 1024

func publishBundle(tmpBundle, bundlePath string) ([]BundlePart, error) {
	info, err := os.Stat(tmpBundle)
	if err != nil {
		return nil, err
	}
	if info.Size() <= bundlePartSize {
		if err := removeBundleOutputs(bundlePath); err != nil {
			return nil, err
		}
		if err := os.Rename(tmpBundle, bundlePath); err != nil {
			return nil, err
		}
		return nil, nil
	}

	src, err := os.Open(tmpBundle)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	type pendingPart struct {
		tmp   string
		final string
		meta  BundlePart
	}
	var pending []pendingPart
	defer func() {
		for _, p := range pending {
			if p.tmp != "" {
				_ = os.Remove(p.tmp)
			}
		}
	}()
	for index, remaining := 0, info.Size(); remaining > 0; index++ {
		size := min(remaining, bundlePartSize)
		part, err := os.CreateTemp(filepath.Dir(bundlePath), ".bundle-part-*.tmp")
		if err != nil {
			return nil, err
		}
		h := sha256.New()
		if _, err := io.CopyN(io.MultiWriter(part, h), src, size); err != nil {
			part.Close()
			return nil, err
		}
		if err := part.Sync(); err != nil {
			part.Close()
			return nil, err
		}
		if err := part.Close(); err != nil {
			return nil, err
		}
		name := fmt.Sprintf("%s.part-%03d", filepath.Base(bundlePath), index)
		pending = append(pending, pendingPart{
			tmp:   part.Name(),
			final: filepath.Join(filepath.Dir(bundlePath), name),
			meta:  BundlePart{Name: name, Size: size, SHA256: hex.EncodeToString(h.Sum(nil))},
		})
		remaining -= size
	}
	if err := removeBundleOutputs(bundlePath); err != nil {
		return nil, err
	}
	parts := make([]BundlePart, 0, len(pending))
	for i := range pending {
		if err := os.Rename(pending[i].tmp, pending[i].final); err != nil {
			return nil, err
		}
		parts = append(parts, pending[i].meta)
		pending[i].tmp = ""
	}
	return parts, nil
}

func removeBundleOutputs(bundlePath string) error {
	if err := os.Remove(bundlePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	parts, err := filepath.Glob(bundlePath + ".part-*")
	if err != nil {
		return err
	}
	for _, part := range parts {
		if err := os.Remove(part); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// stageFile takes a bounded point-in-time copy before writing a tar header. If
// the source disappears or shrinks while active, no partial entry is emitted.
func stageFile(abs string) (os.FileInfo, *os.File, error) {
	src, err := openStageFile(abs)
	if err != nil {
		return nil, nil, err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() {
		return info, nil, nil
	}
	staged, err := os.CreateTemp("", "contix-stage-*")
	if err != nil {
		return nil, nil, err
	}
	if _, err := io.CopyN(staged, src, info.Size()); err != nil {
		name := staged.Name()
		staged.Close()
		os.Remove(name)
		return nil, nil, err
	}
	if _, err := staged.Seek(0, io.SeekStart); err != nil {
		name := staged.Name()
		staged.Close()
		os.Remove(name)
		return nil, nil, err
	}
	return info, staged, nil
}

func openStageFile(abs string) (*os.File, error) {
	var lastErr error
	for attempt := 0; attempt <= stagePermissionRetries; attempt++ {
		src, err := openStageSource(abs)
		if err == nil {
			return src, nil
		}
		lastErr = err
		if !os.IsPermission(err) || attempt == stagePermissionRetries {
			return nil, err
		}
		time.Sleep(stagePermissionRetryDelay)
	}
	return nil, lastErr
}

// writeFile streams one staged file into the tar writer and returns its SHA-256.
func writeFile(tw *tar.Writer, src *os.File, rel string, info os.FileInfo) (string, error) {
	defer os.Remove(src.Name())
	hdr := &tar.Header{
		Name:    rel,
		Mode:    int64(info.Mode().Perm()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return "", err
	}
	h := sha256.New()
	mw := io.MultiWriter(tw, h)
	if _, err := io.CopyN(mw, src, info.Size()); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Extract unpacks a bundle into destRoot. It refuses entries that would escape
// destRoot (zip-slip guard). Returns the list of extracted relative paths.
func Extract(bundlePath, destRoot string) ([]string, error) {
	f, err := openBundle(bundlePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	cleanDest := filepath.Clean(destRoot)
	var extracted []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return extracted, err
		}
		rel := filepath.FromSlash(hdr.Name)
		target := filepath.Join(cleanDest, rel)
		if !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) && target != cleanDest {
			return extracted, fmt.Errorf("refusing unsafe path in archive: %s", hdr.Name)
		}
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return extracted, err
			}
			continue
		}
		if err := ensureNoSymlinkParent(cleanDest, target); err != nil {
			return extracted, err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return extracted, err
		}
		if hdr.Typeflag == tar.TypeSymlink {
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				return extracted, err
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return extracted, err
			}
			extracted = append(extracted, filepath.ToSlash(hdr.Name))
			continue
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			return extracted, fmt.Errorf("unsupported archive entry type for %s", hdr.Name)
		}
		if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(target); err != nil {
				return extracted, err
			}
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode)&0o777)
		if err != nil {
			return extracted, err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return extracted, err
		}
		if err := out.Chmod(os.FileMode(hdr.Mode) & 0o777); err != nil {
			out.Close()
			return extracted, err
		}
		if err := out.Close(); err != nil {
			return extracted, err
		}
		extracted = append(extracted, filepath.ToSlash(hdr.Name))
	}
	return extracted, nil
}

func ensureNoSymlinkParent(root, target string) error {
	rel, err := filepath.Rel(root, filepath.Dir(target))
	if err != nil {
		return err
	}
	current := root
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to restore through symlink parent: %s", current)
		}
	}
	return nil
}

// Exists reports whether a single-file or chunked bundle exists.
func Exists(bundlePath string) bool {
	if _, err := os.Stat(bundlePath); err == nil {
		return true
	}
	parts, _ := filepath.Glob(bundlePath + ".part-*")
	return len(parts) > 0
}

type bundleReader struct {
	files  []*os.File
	reader io.Reader
}

func (r *bundleReader) Read(p []byte) (int, error) { return r.reader.Read(p) }
func (r *bundleReader) Close() error {
	var first error
	for _, f := range r.files {
		if err := f.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func openBundle(bundlePath string) (io.ReadCloser, error) {
	if f, err := os.Open(bundlePath); err == nil {
		return f, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	parts, err := filepath.Glob(bundlePath + ".part-*")
	if err != nil {
		return nil, err
	}
	sort.Strings(parts)
	if len(parts) == 0 {
		return nil, os.ErrNotExist
	}
	r := &bundleReader{}
	var readers []io.Reader
	for _, part := range parts {
		f, err := os.Open(part)
		if err != nil {
			r.Close()
			return nil, err
		}
		r.files = append(r.files, f)
		readers = append(readers, f)
	}
	r.reader = io.MultiReader(readers...)
	return r, nil
}

// Verify checks that every file listed in the manifest exists under root and
// matches its recorded SHA-256. Returns a list of human-readable mismatches
// (empty means perfect fidelity).
func Verify(root string, m Manifest) ([]string, error) {
	var problems []string
	for _, fe := range m.Files {
		abs, err := safeTarget(root, fe.Path)
		if err != nil {
			return nil, err
		}
		if fe.Type == "symlink" {
			info, err := os.Lstat(abs)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", fe.Path, err))
				continue
			}
			if info.Mode()&os.ModeSymlink == 0 {
				problems = append(problems, fmt.Sprintf("%s: expected symlink", fe.Path))
				continue
			}
			target, err := os.Readlink(abs)
			if err != nil || target != fe.LinkTarget || textSHA256(target) != fe.SHA256 {
				problems = append(problems, fmt.Sprintf("%s: symlink target mismatch", fe.Path))
			}
			continue
		}
		sum, err := fileSHA256(abs)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", fe.Path, err))
			continue
		}
		if sum != fe.SHA256 {
			problems = append(problems, fmt.Sprintf("%s: checksum mismatch", fe.Path))
		}
	}
	return problems, nil
}

// Conflicts returns manifest paths that already exist locally with different
// content or a different file type. Missing local paths are not conflicts.
func Conflicts(root string, m Manifest) ([]string, error) {
	var conflicts []string
	for _, fe := range m.Files {
		abs, err := safeTarget(root, fe.Path)
		if err != nil {
			return nil, err
		}
		info, err := os.Lstat(abs)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("compare %s: %w", fe.Path, err)
		}
		if fe.Type == "symlink" {
			if info.Mode()&os.ModeSymlink == 0 {
				conflicts = append(conflicts, fe.Path)
				continue
			}
			target, err := os.Readlink(abs)
			if err != nil {
				return nil, fmt.Errorf("compare symlink %s: %w", fe.Path, err)
			}
			if target != fe.LinkTarget {
				conflicts = append(conflicts, fe.Path)
			}
			continue
		}
		if !info.Mode().IsRegular() {
			conflicts = append(conflicts, fe.Path)
			continue
		}
		sum, err := fileSHA256(abs)
		if err != nil {
			return nil, fmt.Errorf("compare %s: %w", fe.Path, err)
		}
		if sum != fe.SHA256 {
			conflicts = append(conflicts, fe.Path)
		}
	}
	return conflicts, nil
}

func safeTarget(root, rel string) (string, error) {
	cleanRoot := filepath.Clean(root)
	target := filepath.Join(cleanRoot, filepath.FromSlash(rel))
	if target != cleanRoot && !strings.HasPrefix(target, cleanRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("refusing unsafe manifest path: %s", rel)
	}
	return target, nil
}

func fileSHA256(abs string) (string, error) {
	f, err := os.Open(abs)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

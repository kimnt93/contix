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
)

// Create bundles the given relative files (rooted at srcRoot) into a gzip'd tar
// at bundlePath and returns a completed manifest. Files are stored using their
// forward-slash relative paths so archives are portable across OSes.
func Create(srcRoot string, rels []string, bundlePath string, m Manifest) (Manifest, error) {
	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o755); err != nil {
		return m, err
	}
	f, err := os.Create(bundlePath)
	if err != nil {
		return m, err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	sort.Strings(rels)
	m.Files = m.Files[:0]

	for _, rel := range rels {
		abs := filepath.Join(srcRoot, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil {
			return m, fmt.Errorf("stat %s: %w", rel, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		sum, err := writeFile(tw, abs, rel, info)
		if err != nil {
			return m, fmt.Errorf("archive %s: %w", rel, err)
		}
		m.Files = append(m.Files, FileEntry{
			Path:   rel,
			Size:   info.Size(),
			Mode:   uint32(info.Mode().Perm()),
			SHA256: sum,
		})
	}

	if err := tw.Close(); err != nil {
		return m, err
	}
	if err := gz.Close(); err != nil {
		return m, err
	}
	return m, f.Sync()
}

// writeFile streams one file into the tar writer and returns its SHA-256.
func writeFile(tw *tar.Writer, abs, rel string, info os.FileInfo) (string, error) {
	hdr := &tar.Header{
		Name:    rel,
		Mode:    int64(info.Mode().Perm()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return "", err
	}
	src, err := os.Open(abs)
	if err != nil {
		return "", err
	}
	defer src.Close()

	h := sha256.New()
	mw := io.MultiWriter(tw, h)
	if _, err := io.Copy(mw, src); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Extract unpacks a bundle into destRoot. It refuses entries that would escape
// destRoot (zip-slip guard). Returns the list of extracted relative paths.
func Extract(bundlePath, destRoot string) ([]string, error) {
	f, err := os.Open(bundlePath)
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
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return extracted, err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode)&0o777)
		if err != nil {
			return extracted, err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return extracted, err
		}
		out.Close()
		extracted = append(extracted, filepath.ToSlash(hdr.Name))
	}
	return extracted, nil
}

// Verify checks that every file listed in the manifest exists under root and
// matches its recorded SHA-256. Returns a list of human-readable mismatches
// (empty means perfect fidelity).
func Verify(root string, m Manifest) ([]string, error) {
	var problems []string
	for _, fe := range m.Files {
		abs := filepath.Join(root, filepath.FromSlash(fe.Path))
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

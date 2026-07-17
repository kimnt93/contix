// Package syncer orchestrates collecting agent and machine state into the sync
// repo and restoring it on another machine with verification and path rewriting.
package syncer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"contix/internal/archive"
	"contix/internal/config"
	"contix/internal/pathrewrite"
	"contix/internal/tool"
)

// PushResult summarises a single tool push.
type PushResult struct {
	Tool    string
	Files   int
	Bytes   int64
	Version string
	Parts   int
	Skipped string // reason, if skipped
}

// PullResult summarises a single tool pull.
type PullResult struct {
	Tool          string
	Files         int
	SourceVersion string
	LocalVersion  string
	VersionOK     bool
	Mismatches    []string // fidelity check problems (empty == perfect)
	DirsRenamed   int
	FilesRewrite  int
	Skipped       string
	Conflicts     []string // local paths that differ and were not overwritten
	DeferredPath  string   // user-writable copy when the destination needs elevation
	Destination   string
}

// toolDir returns the repo subdirectory for a tool.
func toolDir(cfg config.Config, name string) string {
	return filepath.Join(cfg.RepoPath, name)
}

// Push collects a tool's portable state into the sync repository.
func Push(cfg config.Config, t tool.Tool) (PushResult, error) {
	res := PushResult{Tool: t.Name}
	home := t.Home()
	bundlePath := filepath.Join(toolDir(cfg, t.Name), archive.BundleName)
	if fi, err := os.Stat(home); err != nil || !fi.IsDir() {
		res.Skipped = fmt.Sprintf("no state dir at %s", home)
		keepPreviousSnapshot(bundlePath, &res)
		return res, nil
	}

	rels, err := t.IncludedFiles()
	if err != nil {
		return res, err
	}
	if len(rels) == 0 {
		res.Skipped = "no matching files"
		keepPreviousSnapshot(bundlePath, &res)
		return res, nil
	}

	version := detectVersion(t, home)
	res.Version = version

	m := archive.NewManifest(t.Name, version, home)
	m, err = archive.Create(home, rels, bundlePath, m)
	if err != nil {
		return res, err
	}
	if err := archive.WriteManifest(filepath.Join(toolDir(cfg, t.Name), archive.ManifestName), m); err != nil {
		return res, err
	}

	var total int64
	for _, fe := range m.Files {
		total += fe.Size
	}
	res.Files = len(m.Files)
	res.Bytes = total
	res.Parts = len(m.BundleParts)
	if res.Parts == 0 {
		res.Parts = 1
	}
	return res, nil
}

func keepPreviousSnapshot(bundlePath string, res *PushResult) {
	if archive.Exists(bundlePath) {
		res.Skipped += "; previous synced state kept"
	}
}

// Pull restores a tool's state from the repo onto this machine.
func Pull(cfg config.Config, t tool.Tool, userMaps []pathrewrite.Mapping, rewrite, ignoreConflicts bool) (PullResult, error) {
	res := PullResult{Tool: t.Name}
	manifestPath := filepath.Join(toolDir(cfg, t.Name), archive.ManifestName)
	bundlePath := filepath.Join(toolDir(cfg, t.Name), archive.BundleName)

	if !archive.Exists(bundlePath) {
		res.Skipped = "nothing synced for this tool yet; local state kept"
		return res, nil
	}
	m, err := archive.ReadManifest(manifestPath)
	if err != nil {
		return res, fmt.Errorf("read manifest: %w", err)
	}

	home := t.Home()
	restoreRoot := home
	if t.RestoreFallback != nil && t.WriteProbe != "" {
		res.Destination = filepath.Join(home, filepath.FromSlash(t.WriteProbe))
		if !writableFile(res.Destination) {
			restoreRoot = t.RestoreFallback()
			res.DeferredPath = filepath.Join(restoreRoot, filepath.FromSlash(t.WriteProbe))
		}
	}
	if !ignoreConflicts && res.DeferredPath == "" {
		conflicts, err := archive.Conflicts(home, m)
		if err != nil {
			return res, err
		}
		if len(conflicts) > 0 {
			res.Conflicts = conflicts
			return res, nil
		}
	}
	if err := os.MkdirAll(restoreRoot, 0o755); err != nil {
		return res, err
	}

	extracted, err := archive.Extract(bundlePath, restoreRoot)
	if err != nil {
		return res, err
	}
	res.Files = len(extracted)

	// Fidelity check against the canonical (pre-rewrite) bytes.
	problems, err := archive.Verify(restoreRoot, m)
	if err != nil {
		return res, err
	}
	res.Mismatches = problems

	// Version comparison ("must have same version").
	res.SourceVersion = m.ToolVersion
	res.LocalVersion = detectVersion(t, home)
	res.VersionOK = res.SourceVersion == "" || res.LocalVersion == "" ||
		res.SourceVersion == res.LocalVersion

	// Path rewriting for cross-machine resume.
	if rewrite && res.DeferredPath == "" {
		rw := pathrewrite.New(m.SourceHome, userMaps)
		if rw.Active() {
			files, dirs, err := rw.Apply(restoreRoot)
			if err != nil {
				return res, err
			}
			res.FilesRewrite = files
			res.DirsRenamed = dirs
		}
	}
	return res, nil
}

func writableFile(path string) bool {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	return f.Close() == nil
}

var versionRe = regexp.MustCompile(`\d+\.\d+\.\d+[\w.\-]*`)

// detectVersion resolves a tool's version, preferring its state file and
// falling back to probing the CLI binary.
func detectVersion(t tool.Tool, home string) string {
	if t.Version != nil {
		if v := t.Version(home); v != "" {
			return v
		}
	}
	bin := t.Binary
	if bin == "" {
		return ""
	}
	if _, err := exec.LookPath(bin); err != nil {
		return ""
	}
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	if m := versionRe.FindString(strings.TrimSpace(string(out))); m != "" {
		return m
	}
	return ""
}

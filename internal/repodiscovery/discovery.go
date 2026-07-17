// Package repodiscovery finds user-owned git working trees beneath configured
// roots without descending into build caches, tool state or repositories that
// have already been found.
package repodiscovery

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

var ignoredDirNames = map[string]bool{
	".cache":       true,
	".claude":      true,
	".codex":       true,
	".config":      true,
	".hermes":      true,
	".local":       true,
	".npm":         true,
	".rustup":      true,
	".cargo":       true,
	"node_modules": true,
	"vendor":       true,
	"venv":         true,
	".venv":        true,
	"__pycache__":  true,
	"target":       true,
}

// Discover returns git repository roots found beneath roots. excluded paths
// (notably contix's own sync repo) are pruned entirely.
func Discover(roots, excluded []string) ([]string, error) {
	exclude := make(map[string]bool, len(excluded))
	for _, p := range excluded {
		if p != "" {
			exclude[cleanAbs(p)] = true
		}
	}
	found := make(map[string]bool)
	for _, root := range roots {
		root = cleanAbs(root)
		if root == "" {
			continue
		}
		if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			p = filepath.Clean(p)
			if d.IsDir() && exclude[p] {
				return filepath.SkipDir
			}
			if p != root && d.IsDir() && ignoredDirNames[d.Name()] {
				return filepath.SkipDir
			}
			if !d.IsDir() {
				return nil
			}
			if info, err := os.Lstat(filepath.Join(p, ".git")); err == nil && (info.IsDir() || info.Mode().IsRegular()) {
				found[p] = true
				return filepath.SkipDir
			}
			return nil
		}); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	out := make([]string, 0, len(found))
	for p := range found {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

func cleanAbs(p string) string {
	if p == "" {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(abs)
}

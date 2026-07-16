// Package tool defines the syncable tools (Codex, Claude Code): where their
// state lives, which paths are worth syncing, and how to detect their version.
package tool

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"contix/internal/platform"
)

// Tool describes a syncable application.
type Tool struct {
	// Name is the stable identifier used on the CLI and in the repo layout.
	Name string
	// Home returns the absolute state directory for this tool on this machine.
	Home func() string
	// Exclude lists path patterns that must never be synced. Everything under
	// Home that does not match an Exclude pattern is synced.
	Exclude []string
	// Version detects the installed tool version from its state dir. Returns
	// "" when unknown.
	Version func(home string) string
}

// Registry returns all known tools keyed by name.
func Registry() map[string]Tool {
	return map[string]Tool{
		"codex":  codex(),
		"claude": claude(),
	}
}

// Names returns the sorted list of known tool names.
func Names() []string {
	return []string{"claude", "codex"}
}

// Lookup returns a tool by name.
func Lookup(name string) (Tool, bool) {
	t, ok := Registry()[name]
	return t, ok
}

func codex() Tool {
	return Tool{
		Name: "codex",
		Home: platform.CodexHome,
		// Sync everything under the Codex home except the items below.
		Exclude: []string{
			// Machine-locked credentials — never sync (security).
			"auth.json",
			".credentials.json",
			// 300MB+ telemetry log that regenerates on its own.
			"logs_*.sqlite",
			// SQLite shared-memory sidecar; rebuilt on open, unsafe to copy.
			"*.sqlite-shm",
			// Nested git repos would corrupt the sync repo if embedded.
			".git",
		},
		Version: func(home string) string {
			// version.json: {"version":"x.y.z", ...}
			b, err := os.ReadFile(filepath.Join(home, "version.json"))
			if err != nil {
				return ""
			}
			var v struct {
				Version string `json:"version"`
			}
			if json.Unmarshal(b, &v) == nil {
				return v.Version
			}
			return ""
		},
	}
}

func claude() Tool {
	return Tool{
		Name: "claude",
		Home: platform.ClaudeHome,
		// Sync everything under the Claude home except the items below.
		Exclude: []string{
			// Machine-locked credentials — never sync (security).
			".credentials.json",
			// SQLite shared-memory sidecar; rebuilt on open, unsafe to copy.
			"*.sqlite-shm",
			// Nested git repos (e.g. plugin marketplaces) would corrupt the
			// sync repo if embedded.
			".git",
		},
		Version: func(home string) string {
			// Claude stores no reliable version file; leave to CLI probing.
			return ""
		},
	}
}

// IncludedFiles walks the tool's home directory and returns the relative
// (forward-slash) paths of all regular files that are not excluded. Everything
// under Home is synced except paths matching an Exclude pattern. Symlinks are
// skipped for safety.
func (t Tool) IncludedFiles() ([]string, error) {
	home := t.Home()
	info, err := os.Stat(home)
	if err != nil || !info.IsDir() {
		return nil, err
	}
	var out []string
	err = filepath.WalkDir(home, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		rel, rerr := filepath.Rel(home, p)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			// Prune excluded directories entirely.
			if matchAny(rel, t.Exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if matchAny(rel, t.Exclude) {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	return out, err
}

// matchAny reports whether rel matches any of the patterns.
func matchAny(rel string, patterns []string) bool {
	for _, p := range patterns {
		if match(rel, p) {
			return true
		}
	}
	return false
}

// match implements contix's pattern semantics:
//   - "dir/"        directory prefix: matches the dir and everything under it
//   - "*.ext"       glob on the basename (no slash)
//   - "name"        segment name: matches if any path segment equals name,
//     or the path is exactly name / under name/
//   - "a/b*.c"      full-path glob (contains slash)
//   - "a/b/c"       exact path or prefix directory
func match(rel, pat string) bool {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.HasSuffix(pat, "/"):
		d := strings.TrimSuffix(pat, "/")
		return rel == d || strings.HasPrefix(rel, d+"/")
	case strings.Contains(pat, "*") && !strings.Contains(pat, "/"):
		ok, _ := path.Match(pat, path.Base(rel))
		return ok
	case !strings.Contains(pat, "/"):
		if rel == pat || strings.HasPrefix(rel, pat+"/") {
			return true
		}
		for _, seg := range strings.Split(rel, "/") {
			if seg == pat {
				return true
			}
		}
		return false
	case strings.Contains(pat, "*"):
		ok, _ := path.Match(pat, rel)
		return ok
	default:
		return rel == pat || strings.HasPrefix(rel, pat+"/")
	}
}

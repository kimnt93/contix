// Package tool defines syncable agent and machine state roots and how to detect
// a related tool version.
package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"contix/internal/platform"
)

// Tool describes a syncable application.
type Tool struct {
	// Name is the stable identifier used on the CLI and in the repo layout.
	Name string
	// Home returns the absolute state directory for this tool on this machine.
	Home func() string
	// Include is an optional allowlist rooted at Home. Agent and SSH targets leave
	// it empty to sync everything; hosts uses it to select /etc/hosts, not /etc.
	Include []string
	// Binary is the executable probed with --version. Empty disables probing.
	Binary string
	// Processes lists executable names stopped by collect --force-close. It is
	// separate from Binary because some products have several state roots or
	// helper processes.
	Processes []string
	// Version detects the installed tool version from its state dir. Returns
	// "" when unknown.
	Version func(home string) string
	// RestoreFallback is used when WriteProbe cannot be opened for writing. It
	// lets privileged files be safely staged without touching the local copy.
	RestoreFallback func() string
	WriteProbe      string
}

// Registry returns all known tools keyed by name.
func Registry() map[string]Tool {
	return map[string]Tool{
		"antigravity": antigravity(),
		"claude":      claude(),
		"codex":       codex(),
		"hermes":      hermes(),
		"hosts":       hosts(),
		"kiro":        kiro(),
		"openclaw":    openclaw(),
		"ssh":         sshConfig(),
	}
}

// Names returns the sorted list of known tool names.
func Names() []string {
	registry := Registry()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RetiredNames are old IDE targets removed from automatic collection. Their
// bundles are deleted from the sync repo on the next collect, never from the
// application's local state directory.
func RetiredNames() []string {
	return []string{
		"antigravity-editor", "antigravity-extensions",
		"cursor", "cursor-home", "kiro-editor",
		"vscode", "vscode-home", "vscodium", "vscodium-home",
		"void", "void-home", "windsurf", "windsurf-agent", "windsurf-home",
	}
}

// Lookup returns a tool by name.
func Lookup(name string) (Tool, bool) {
	t, ok := Registry()[name]
	return t, ok
}

func codex() Tool {
	return Tool{
		Name:      "codex",
		Home:      platform.CodexHome,
		Binary:    "codex",
		Processes: []string{"codex", "codex-code-mode"},
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
		Name:      "claude",
		Home:      platform.ClaudeHome,
		Binary:    "claude",
		Processes: []string{"claude"},
		Version: func(home string) string {
			// Claude stores no reliable version file; leave to CLI probing.
			return ""
		},
	}
}

func hermes() Tool {
	return Tool{
		Name:      "hermes",
		Home:      platform.HermesHome,
		Binary:    "hermes",
		Processes: []string{"hermes", "hermes-agent"},
		Version:   func(home string) string { return "" },
	}
}

func kiro() Tool {
	return Tool{
		Name:      "kiro",
		Home:      platform.KiroHome,
		Binary:    "kiro-cli",
		Processes: []string{"kiro-cli"},
	}
}

func antigravity() Tool {
	return Tool{
		Name:      "antigravity",
		Home:      platform.AntigravityHome,
		Binary:    "antigravity",
		Processes: []string{"antigravity"},
	}
}

func openclaw() Tool {
	return Tool{
		Name:      "openclaw",
		Home:      platform.OpenClawHome,
		Binary:    "openclaw",
		Processes: []string{"openclaw", "openclaw-gatewa", "openclaw-gateway"},
	}
}

func sshConfig() Tool {
	return Tool{
		Name:   "ssh",
		Home:   platform.SSHHome,
		Binary: "ssh",
	}
}

func hosts() Tool {
	return Tool{
		Name:            "hosts",
		Home:            platform.HostsDir,
		Include:         []string{"hosts"},
		RestoreFallback: platform.HostsStagingDir,
		WriteProbe:      "hosts",
	}
}

// IncludedFiles walks the target root and returns every regular file and
// symlink. Hosts is the sole allowlisted target because its root is /etc.
func (t Tool) IncludedFiles() ([]string, error) {
	home := t.Home()
	info, err := os.Stat(home)
	if err != nil || !info.IsDir() {
		return nil, err
	}
	var out []string
	err = filepath.WalkDir(home, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("read state path %s: %w", p, err)
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
			if len(t.Include) > 0 && !couldContainIncluded(rel, t.Include) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			out = append(out, rel)
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported non-regular state path: %s", p)
		}
		if len(t.Include) > 0 && !includeMatchAny(rel, t.Include) {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	return out, err
}

func includeMatchAny(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		if includeMatch(rel, pattern) {
			return true
		}
	}
	return false
}

// includeMatch uses root-relative semantics: an allowlisted "hosts" means only
// <root>/hosts, never a nested path with the same basename.
func includeMatch(rel, pattern string) bool {
	rel = filepath.ToSlash(rel)
	pattern = filepath.ToSlash(pattern)
	if strings.HasSuffix(pattern, "/") {
		dir := strings.TrimSuffix(pattern, "/")
		return rel == dir || strings.HasPrefix(rel, dir+"/")
	}
	if strings.Contains(pattern, "*") {
		ok, _ := path.Match(pattern, rel)
		return ok
	}
	return rel == pattern || strings.HasPrefix(rel, pattern+"/")
}

func couldContainIncluded(rel string, patterns []string) bool {
	if includeMatchAny(rel, patterns) {
		return true
	}
	rel = strings.TrimSuffix(filepath.ToSlash(rel), "/")
	for _, pattern := range patterns {
		pattern = strings.TrimSuffix(filepath.ToSlash(pattern), "/")
		if strings.Contains(pattern, "*") || strings.HasPrefix(pattern, rel+"/") {
			return true
		}
	}
	return false
}

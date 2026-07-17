// Package tool defines syncable coding-agent state roots and how to detect a
// related tool version.
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
	// Include is an optional allowlist rooted at Home. An empty list syncs the
	// complete coding-agent state root.
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
}

// Registry returns all known tools keyed by name.
func Registry() map[string]Tool {
	return map[string]Tool{
		"aider":           aider(),
		"amp":             amp(),
		"antigravity":     antigravity(),
		"auggie":          auggie(),
		"claude":          claude(),
		"cline":           cline(),
		"codex":           codex(),
		"continue":        continueCLI(),
		"copilot":         copilot(),
		"cursor":          cursor(),
		"droid":           droid(),
		"goose":           goose(),
		"goose-config":    gooseConfig(),
		"kiro":            kiro(),
		"opencode":        openCode(),
		"opencode-config": openCodeConfig(),
		"qwen":            qwen(),
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
		"cursor-home", "kiro-editor",
		"vscode", "vscode-home", "vscodium", "vscodium-home",
		"void", "void-home", "windsurf", "windsurf-agent", "windsurf-home",
		"hermes", "openclaw", "ssh", "hosts",
	}
}

// Group expands coding agents that use more than one official state root.
func Group(name string) ([]string, bool) {
	groups := map[string][]string{
		"goose":    {"goose", "goose-config"},
		"opencode": {"opencode", "opencode-config"},
	}
	items, ok := groups[name]
	return items, ok
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

func cursor() Tool {
	return Tool{
		Name:      "cursor",
		Home:      platform.CursorHome,
		Include:   []string{"mcp.json", "cli-config.json", "rules", "commands", "skills", "hooks", "hooks.json"},
		Binary:    "cursor-agent",
		Processes: []string{"cursor-agent"},
	}
}

func openCode() Tool {
	return Tool{
		Name:      "opencode",
		Home:      platform.OpenCodeDataHome,
		Binary:    "opencode",
		Processes: []string{"opencode"},
	}
}

func openCodeConfig() Tool {
	return Tool{
		Name:      "opencode-config",
		Home:      platform.OpenCodeConfigHome,
		Binary:    "opencode",
		Processes: []string{"opencode"},
	}
}

func copilot() Tool {
	return Tool{Name: "copilot", Home: platform.CopilotHome, Binary: "copilot", Processes: []string{"copilot"}}
}

func cline() Tool {
	return Tool{
		Name: "cline", Home: platform.ClineHome,
		Include: []string{"data/settings", "data/teams", "data/sessions", "plugins", "hooks"},
		Binary:  "cline", Processes: []string{"cline"},
	}
}

func continueCLI() Tool {
	return Tool{
		Name: "continue", Home: platform.ContinueHome,
		Include: []string{
			"config.yaml", "config.json", "config.ts", ".env", "permissions.yaml",
			"rules", "models", "mcpServers", "prompts", "agents", "sessions",
		},
		Binary: "cn", Processes: []string{"cn"},
	}
}

func aider() Tool {
	return Tool{
		Name: "aider", Home: platform.Home,
		Include: []string{
			".aider.conf.yml", ".aider.model.settings.yml", ".aider.model.metadata.json",
			".aider.input.history", ".aider.chat.history.md", ".aider.llm.history",
		},
		Binary: "aider", Processes: []string{"aider"},
	}
}

func qwen() Tool {
	return Tool{Name: "qwen", Home: platform.QwenHome, Binary: "qwen", Processes: []string{"qwen"}}
}

func droid() Tool {
	return Tool{Name: "droid", Home: platform.DroidHome, Binary: "droid", Processes: []string{"droid"}}
}

func amp() Tool {
	return Tool{Name: "amp", Home: platform.AmpHome, Binary: "amp", Processes: []string{"amp"}}
}

func auggie() Tool {
	return Tool{Name: "auggie", Home: platform.AuggieHome, Binary: "auggie", Processes: []string{"auggie"}}
}

func goose() Tool {
	return Tool{Name: "goose", Home: platform.GooseDataHome, Binary: "goose", Processes: []string{"goose"}}
}

func gooseConfig() Tool {
	return Tool{Name: "goose-config", Home: platform.GooseConfigHome, Binary: "goose", Processes: []string{"goose"}}
}

// IncludedFiles walks the target root and returns its selected regular files
// and symlinks.
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

// includeMatch uses root-relative semantics: an allowlisted file means only
// that exact root file, never a nested path with the same basename.
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

// Package platform resolves cross-platform home directories and tool state
// locations for Linux, macOS and Windows.
package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Home returns the current user's home directory.
func Home() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	// Fallbacks for unusual environments.
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	if h := os.Getenv("USERPROFILE"); h != "" {
		return h
	}
	return "."
}

// ConfigDir returns the base directory where contix stores its own
// configuration and local sync repository.
//
//	Linux:   $XDG_CONFIG_HOME/contix or ~/.config/contix
//	macOS:   ~/Library/Application Support/contix
//	Windows: %AppData%\contix
func ConfigDir() string {
	if v := os.Getenv("CONTIX_CONFIG_DIR"); v != "" {
		return v
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "contix")
	}
	if app := os.Getenv("AppData"); app != "" { // Windows
		return filepath.Join(app, "contix")
	}
	return filepath.Join(Home(), ".config", "contix")
}

// CodexHome resolves the Codex CLI state directory, honouring CODEX_HOME.
func CodexHome() string {
	if v := os.Getenv("CODEX_HOME"); v != "" {
		return v
	}
	return filepath.Join(Home(), ".codex")
}

// ClaudeHome resolves the Claude Code state directory, honouring
// CLAUDE_CONFIG_DIR.
func ClaudeHome() string {
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
		return v
	}
	return filepath.Join(Home(), ".claude")
}

// KiroHome resolves Kiro's global state directory, honouring the official
// KIRO_HOME override used by Kiro CLI.
func KiroHome() string {
	if v := os.Getenv("KIRO_HOME"); v != "" {
		return v
	}
	return filepath.Join(Home(), ".kiro")
}

// AntigravityHome resolves the parent of Antigravity's agent data. Google
// stores its portable artifacts, knowledge and conversations below
// ~/.gemini/antigravity, with global rules at ~/.gemini/GEMINI.md.
func AntigravityHome() string {
	if v := os.Getenv("ANTIGRAVITY_HOME"); v != "" {
		return expandHomePath(v)
	}
	if v := os.Getenv("GEMINI_CLI_HOME"); v != "" {
		return filepath.Join(expandHomePath(v), ".gemini")
	}
	return filepath.Join(Home(), ".gemini")
}

func expandHomePath(value string) string {
	if value == "~" {
		return Home()
	}
	prefix := "~" + string(filepath.Separator)
	if strings.HasPrefix(value, prefix) {
		return filepath.Join(Home(), strings.TrimPrefix(value, prefix))
	}
	return value
}

func CursorHome() string {
	if v := os.Getenv("CONTIX_CURSOR_HOME"); v != "" {
		return expandHomePath(v)
	}
	return filepath.Join(Home(), ".cursor")
}

func OpenCodeConfigHome() string {
	if v := os.Getenv("OPENCODE_CONFIG_DIR"); v != "" {
		return expandHomePath(v)
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(Home(), ".config")
	}
	return filepath.Join(base, "opencode")
}

func OpenCodeDataHome() string {
	if v := os.Getenv("CONTIX_OPENCODE_DATA_HOME"); v != "" {
		return expandHomePath(v)
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		base = filepath.Join(Home(), ".local", "share")
	}
	return filepath.Join(base, "opencode")
}

func CopilotHome() string {
	if v := os.Getenv("COPILOT_HOME"); v != "" {
		return expandHomePath(v)
	}
	return filepath.Join(Home(), ".copilot")
}

func ClineHome() string {
	if v := os.Getenv("CONTIX_CLINE_HOME"); v != "" {
		return expandHomePath(v)
	}
	return filepath.Join(Home(), ".cline")
}

func ContinueHome() string {
	if v := os.Getenv("CONTIX_CONTINUE_HOME"); v != "" {
		return expandHomePath(v)
	}
	return filepath.Join(Home(), ".continue")
}

func QwenHome() string {
	if v := os.Getenv("CONTIX_QWEN_HOME"); v != "" {
		return expandHomePath(v)
	}
	return filepath.Join(Home(), ".qwen")
}

func DroidHome() string {
	if v := os.Getenv("CONTIX_DROID_HOME"); v != "" {
		return expandHomePath(v)
	}
	return filepath.Join(Home(), ".factory")
}

func AmpHome() string {
	if v := os.Getenv("CONTIX_AMP_HOME"); v != "" {
		return expandHomePath(v)
	}
	return filepath.Join(Home(), ".config", "amp")
}

func AuggieHome() string {
	if v := os.Getenv("CONTIX_AUGGIE_HOME"); v != "" {
		return expandHomePath(v)
	}
	return filepath.Join(Home(), ".augment")
}

func GooseDataHome() string {
	if v := os.Getenv("GOOSE_PATH_ROOT"); v != "" {
		return expandHomePath(v)
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(Home(), "Library", "Application Support", "Block", "goose")
	case "windows":
		base := os.Getenv("APPDATA")
		if base == "" {
			base = os.Getenv("AppData")
		}
		if base == "" {
			base = filepath.Join(Home(), "AppData", "Roaming")
		}
		return filepath.Join(base, "Block", "goose")
	default:
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(Home(), ".local", "share")
		}
		return filepath.Join(base, "goose")
	}
}

func GooseConfigHome() string {
	if v := os.Getenv("GOOSE_PATH_ROOT"); v != "" {
		return filepath.Join(expandHomePath(v), "config")
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(GooseDataHome(), "config")
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(Home(), ".config")
	}
	return filepath.Join(base, "goose")
}

// Package platform resolves cross-platform home directories and tool state
// locations for Linux, macOS and Windows.
package platform

import (
	"os"
	"path/filepath"
	"runtime"
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

// HermesHome resolves the Hermes Agent state directory, honouring HERMES_HOME.
func HermesHome() string {
	if v := os.Getenv("HERMES_HOME"); v != "" {
		return v
	}
	return filepath.Join(Home(), ".hermes")
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
		return v
	}
	return filepath.Join(Home(), ".gemini")
}

// SSHHome resolves the user's SSH configuration directory. CONTIX_SSH_HOME is
// primarily useful for non-standard setups and isolated testing.
func SSHHome() string {
	if v := os.Getenv("CONTIX_SSH_HOME"); v != "" {
		return v
	}
	return filepath.Join(Home(), ".ssh")
}

// HostsDir returns the directory containing the system hosts file.
func HostsDir() string {
	if v := os.Getenv("CONTIX_HOSTS_DIR"); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		return filepath.Join(root, "System32", "drivers", "etc")
	}
	return "/etc"
}

// HostsStagingDir is a user-writable holding area used when the system hosts
// file cannot be replaced without administrator privileges.
func HostsStagingDir() string {
	return filepath.Join(ConfigDir(), "pending", "hosts")
}

// editorDataDir resolves the Electron/VS Code-style application data directory
// used for settings, workspace state, histories, caches and authentication.
func editorDataDir(envName, product string) string {
	if v := os.Getenv(envName); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(Home(), "Library", "Application Support", product)
	case "windows":
		base := os.Getenv("APPDATA")
		if base == "" {
			base = os.Getenv("AppData")
		}
		if base == "" {
			base = filepath.Join(Home(), "AppData", "Roaming")
		}
		return filepath.Join(base, product)
	default:
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			base = filepath.Join(Home(), ".config")
		}
		return filepath.Join(base, product)
	}
}

func editorHome(envName, dir string) string {
	if v := os.Getenv(envName); v != "" {
		return v
	}
	return filepath.Join(Home(), dir)
}

func CursorDataHome() string   { return editorDataDir("CONTIX_CURSOR_DATA_HOME", "Cursor") }
func CursorHome() string       { return editorHome("CONTIX_CURSOR_HOME", ".cursor") }
func WindsurfDataHome() string { return editorDataDir("CONTIX_WINDSURF_DATA_HOME", "Windsurf") }
func WindsurfHome() string     { return editorHome("CONTIX_WINDSURF_HOME", ".windsurf") }
func WindsurfAgentHome() string {
	return editorHome("CONTIX_WINDSURF_AGENT_HOME", filepath.Join(".codeium", "windsurf"))
}
func VSCodeDataHome() string   { return editorDataDir("CONTIX_VSCODE_DATA_HOME", "Code") }
func VSCodeHome() string       { return editorHome("CONTIX_VSCODE_HOME", ".vscode") }
func VSCodiumDataHome() string { return editorDataDir("CONTIX_VSCODIUM_DATA_HOME", "VSCodium") }
func VSCodiumHome() string     { return editorHome("CONTIX_VSCODIUM_HOME", ".vscode-oss") }
func VoidDataHome() string     { return editorDataDir("CONTIX_VOID_DATA_HOME", "Void") }
func VoidHome() string         { return editorHome("CONTIX_VOID_HOME", ".void") }
func KiroIDEHome() string      { return editorDataDir("CONTIX_KIRO_IDE_HOME", "Kiro") }
func AntigravityIDEHome() string {
	return editorDataDir("CONTIX_ANTIGRAVITY_IDE_HOME", "Antigravity")
}
func AntigravityExtensionsHome() string {
	return editorHome("CONTIX_ANTIGRAVITY_EXTENSIONS_HOME", ".antigravity")
}

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

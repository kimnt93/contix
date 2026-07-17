// Package platform resolves cross-platform home directories and tool state
// locations for Linux, macOS and Windows.
package platform

import (
	"os"
	"path/filepath"
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

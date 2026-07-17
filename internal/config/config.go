// Package config loads and saves contix's configuration: where the local sync
// repository lives, its git remote, and the machine identity used for path
// rewriting.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"contix/internal/platform"
)

// Config is contix's persisted configuration.
type Config struct {
	// RepoPath is the local directory that holds the git-backed sync repo.
	RepoPath string `json:"repo_path"`
	// Remote is the git remote URL (optional; enables auto push/pull).
	Remote string `json:"remote,omitempty"`
	// Branch is the git branch used for syncing.
	Branch string `json:"branch"`
	// Home is this machine's home directory recorded at init time. Used as a
	// hint; the live home is always re-resolved at runtime.
	Home string `json:"home"`
}

// Path returns the config file location.
func Path() string {
	return filepath.Join(platform.ConfigDir(), "config.json")
}

// Default returns a config populated with sensible defaults.
func Default() Config {
	return Config{
		RepoPath: filepath.Join(platform.ConfigDir(), "repo"),
		Branch:   "main",
		Home:     platform.Home(),
	}
}

// ErrNotConfigured is returned when no config file exists yet.
var ErrNotConfigured = errors.New("contix is not configured; run 'contix init' first")

// Load reads the config from disk.
func Load() (Config, error) {
	b, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, ErrNotConfigured
		}
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	if c.Branch == "" {
		c.Branch = "main"
	}
	return c, nil
}

// Save writes the config to disk, creating parent directories as needed.
func (c Config) Save() error {
	if err := os.MkdirAll(filepath.Dir(Path()), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), b, 0o600)
}

// Exists reports whether a config file is present.
func Exists() bool {
	_, err := os.Stat(Path())
	return err == nil
}

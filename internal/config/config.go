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
	// AutoPush pushes to the remote automatically after each push command.
	AutoPush bool `json:"auto_push"`
	// Home is this machine's home directory recorded at init time. Used as a
	// hint; the live home is always re-resolved at runtime.
	Home string `json:"home"`
	// Repos are absolute paths to git working repositories whose branches and
	// uncommitted work should be synced.
	Repos []string `json:"repos,omitempty"`
	// AutoDiscover scans RepoRoots on every push and registers newly cloned or
	// created repositories automatically.
	AutoDiscover bool `json:"auto_discover"`
	// RepoRoots are directories searched recursively for git repositories.
	RepoRoots []string `json:"repo_roots,omitempty"`
}

// AddRepo records an absolute repo path if not already tracked. Returns false
// if it was already present.
func (c *Config) AddRepo(abs string) bool {
	for _, r := range c.Repos {
		if r == abs {
			return false
		}
	}
	c.Repos = append(c.Repos, abs)
	return true
}

// RemoveRepo drops a tracked repo path. Returns false if it was not tracked.
func (c *Config) RemoveRepo(abs string) bool {
	for i, r := range c.Repos {
		if r == abs {
			c.Repos = append(c.Repos[:i], c.Repos[i+1:]...)
			return true
		}
	}
	return false
}

// Path returns the config file location.
func Path() string {
	return filepath.Join(platform.ConfigDir(), "config.json")
}

// Default returns a config populated with sensible defaults.
func Default() Config {
	return Config{
		RepoPath:     filepath.Join(platform.ConfigDir(), "repo"),
		Branch:       "main",
		AutoPush:     false,
		Home:         platform.Home(),
		AutoDiscover: true,
		RepoRoots:    []string{platform.Home()},
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
	// Existing configs predate automatic discovery. Enable it during migration
	// unless the field is explicitly present and false.
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(b, &fields)
	if _, ok := fields["auto_discover"]; !ok {
		c.AutoDiscover = true
	}
	if len(c.RepoRoots) == 0 {
		c.RepoRoots = []string{platform.Home()}
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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddRemoveRepo(t *testing.T) {
	var c Config
	if !c.AddRepo("/a") {
		t.Fatal("AddRepo should return true for new path")
	}
	if c.AddRepo("/a") {
		t.Fatal("AddRepo should return false for duplicate")
	}
	c.AddRepo("/b")
	if len(c.Repos) != 2 {
		t.Fatalf("want 2 repos, got %d", len(c.Repos))
	}
	if !c.RemoveRepo("/a") {
		t.Fatal("RemoveRepo should return true when present")
	}
	if c.RemoveRepo("/a") {
		t.Fatal("RemoveRepo should return false when absent")
	}
	if len(c.Repos) != 1 || c.Repos[0] != "/b" {
		t.Fatalf("unexpected repos: %v", c.Repos)
	}
}

func TestLoadMigratesOldConfigToAutoDiscovery(t *testing.T) {
	t.Setenv("CONTIX_CONFIG_DIR", t.TempDir())
	old := []byte(`{"repo_path":"/tmp/sync","branch":"main","home":"/tmp/home"}`)
	if err := os.WriteFile(filepath.Join(os.Getenv("CONTIX_CONFIG_DIR"), "config.json"), old, 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.AutoDiscover || len(c.RepoRoots) != 1 {
		t.Fatalf("old config was not migrated: %#v", c)
	}
}

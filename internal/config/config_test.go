package config

import "testing"

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

package config

import "testing"

func TestDefault(t *testing.T) {
	c := Default()
	if c.RepoPath == "" || c.Branch != "main" || c.Home == "" {
		t.Fatalf("unexpected defaults: %#v", c)
	}
}

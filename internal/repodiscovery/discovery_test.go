package repodiscovery

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverReposAndPruneIgnoredTrees(t *testing.T) {
	root := t.TempDir()
	repoA := filepath.Join(root, "code", "a")
	repoB := filepath.Join(root, "Desktop", "b")
	ignored := filepath.Join(root, ".cache", "not-mine")
	excluded := filepath.Join(root, "sync-repo")
	for _, p := range []string{repoA, repoB, ignored, excluded} {
		if err := os.MkdirAll(filepath.Join(p, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Discover([]string{root}, []string{excluded})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{repoB, repoA}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Discover()=%v want %v", got, want)
	}
}

func TestDiscoverSupportsWorktreeGitFile(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "worktree")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: /tmp/example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Discover([]string{root}, nil)
	if err != nil || len(got) != 1 || got[0] != repo {
		t.Fatalf("Discover()=%v, %v", got, err)
	}
}

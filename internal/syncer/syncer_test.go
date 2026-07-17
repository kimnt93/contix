package syncer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"contix/internal/config"
	"contix/internal/tool"
)

func TestPushMissingSourceKeepsPreviousSnapshot(t *testing.T) {
	repo := t.TempDir()
	bundle := filepath.Join(repo, "kiro", "bundle.tar.gz")
	if err := os.MkdirAll(filepath.Dir(bundle), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("previous snapshot")
	if err := os.WriteFile(bundle, want, 0o600); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(t.TempDir(), "not-installed")
	target := tool.Tool{Name: "kiro", Home: func() string { return missing }}
	res, err := Push(config.Config{RepoPath: repo}, target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Skipped, "previous synced state kept") {
		t.Fatalf("skip reason %q does not explain preservation", res.Skipped)
	}
	got, err := os.ReadFile(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("missing source replaced the previous snapshot")
	}
}

func TestPullMissingSnapshotKeepsLocalState(t *testing.T) {
	local := t.TempDir()
	localFile := filepath.Join(local, "local-only.txt")
	if err := os.WriteFile(localFile, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := tool.Tool{Name: "kiro", Home: func() string { return local }}
	res, err := Pull(config.Config{RepoPath: t.TempDir()}, target, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Skipped, "local state kept") {
		t.Fatalf("skip reason %q does not explain preservation", res.Skipped)
	}
	got, err := os.ReadFile(localFile)
	if err != nil || string(got) != "keep me" {
		t.Fatalf("local state changed: %q, %v", got, err)
	}
}

func TestPullConflictRequiresIgnoreToOverwrite(t *testing.T) {
	repo := t.TempDir()
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "settings.json"), []byte("remote"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := tool.Tool{Name: "kiro", Home: func() string { return source }}
	if _, err := Push(config.Config{RepoPath: repo}, target); err != nil {
		t.Fatal(err)
	}

	destination := t.TempDir()
	localPath := filepath.Join(destination, "settings.json")
	if err := os.WriteFile(localPath, []byte("local"), 0o600); err != nil {
		t.Fatal(err)
	}
	target.Home = func() string { return destination }
	res, err := Pull(config.Config{RepoPath: repo}, target, nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0] != "settings.json" {
		t.Fatalf("conflicts = %v", res.Conflicts)
	}
	got, err := os.ReadFile(localPath)
	if err != nil || string(got) != "local" {
		t.Fatalf("default pull overwrote conflict: %q, %v", got, err)
	}

	res, err = Pull(config.Config{RepoPath: repo}, target, nil, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 0 {
		t.Fatalf("--ignore still returned conflicts: %v", res.Conflicts)
	}
	got, err = os.ReadFile(localPath)
	if err != nil || string(got) != "remote" {
		t.Fatalf("ignored conflict was not overwritten: %q, %v", got, err)
	}
}

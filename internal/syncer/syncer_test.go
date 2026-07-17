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
	res, err := Pull(config.Config{RepoPath: t.TempDir()}, target, nil, true)
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

func TestPullStagesProtectedTargetWithoutTouchingDestination(t *testing.T) {
	repo := t.TempDir()
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "protected"), []byte("synced"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := tool.Tool{
		Name:    "hosts",
		Home:    func() string { return source },
		Include: []string{"protected"},
	}
	if _, err := Push(config.Config{RepoPath: repo}, target); err != nil {
		t.Fatal(err)
	}

	destination := t.TempDir()
	localFile := filepath.Join(destination, "local-only")
	if err := os.WriteFile(localFile, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	staging := t.TempDir()
	target.Home = func() string { return destination }
	target.WriteProbe = "protected" // absent, so restoration must be staged
	target.RestoreFallback = func() string { return staging }
	res, err := Pull(config.Config{RepoPath: repo}, target, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.DeferredPath != filepath.Join(staging, "protected") {
		t.Fatalf("deferred path = %q", res.DeferredPath)
	}
	got, err := os.ReadFile(res.DeferredPath)
	if err != nil || string(got) != "synced" {
		t.Fatalf("staged state = %q, %v", got, err)
	}
	got, err = os.ReadFile(localFile)
	if err != nil || string(got) != "keep" {
		t.Fatalf("local destination changed: %q, %v", got, err)
	}
}

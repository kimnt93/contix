package gitutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommitSnapshotReplacesUnpublishedRootHistory(t *testing.T) {
	dir := t.TempDir()
	r, err := Init(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("config", "user.name", "contix-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run("config", "user.email", "contix@example.invalid"); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "bundle")
	if err := os.WriteFile(file, []byte("oversized-old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := r.CommitSnapshot("main", "first"); err != nil || !ok {
		t.Fatalf("first commit: ok=%v err=%v", ok, err)
	}
	if err := os.WriteFile(file, []byte("chunked-new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := r.CommitSnapshot("main", "replacement"); err != nil || !ok {
		t.Fatalf("replacement commit: ok=%v err=%v", ok, err)
	}
	count, err := r.run("rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if count != "1" {
		t.Fatalf("unpublished history was not replaced; commit count=%s", count)
	}
}

func TestCommitSnapshotSquashesCommitsAheadOfOrigin(t *testing.T) {
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	if _, err := initBareForTest(remoteDir); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	r, err := Init(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"user.name": "contix-test", "user.email": "contix@example.invalid",
	} {
		if _, err := r.run("config", key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.SetRemote(remoteDir); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "bundle")
	if err := os.WriteFile(file, []byte("published"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := r.CommitSnapshot("main", "published"); err != nil || !ok {
		t.Fatalf("published commit: ok=%v err=%v", ok, err)
	}
	if err := r.Push("main"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("rejected-large"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := r.CommitSnapshot("main", "rejected"); err != nil || !ok {
		t.Fatalf("rejected commit: ok=%v err=%v", ok, err)
	}
	if err := os.WriteFile(file, []byte("split-parts"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := r.CommitSnapshot("main", "replacement"); err != nil || !ok {
		t.Fatalf("replacement commit: ok=%v err=%v", ok, err)
	}
	ahead, err := r.run("rev-list", "--count", "refs/remotes/origin/main..HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if ahead != "1" {
		t.Fatalf("unpublished commits were not squashed; ahead=%s", ahead)
	}
}

func initBareForTest(dir string) (Repo, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Repo{}, err
	}
	r := Repo{Dir: dir}
	_, err := r.run("init", "--bare")
	return r, err
}

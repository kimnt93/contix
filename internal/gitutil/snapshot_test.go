package gitutil

import (
	"io"
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

func TestPushProgressPublishesDivergedLocalSnapshot(t *testing.T) {
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	if _, err := initBareForTest(remoteDir); err != nil {
		t.Fatal(err)
	}

	firstDir := t.TempDir()
	first, err := Init(firstDir, "main")
	if err != nil {
		t.Fatal(err)
	}
	configureTestIdentity(t, first)
	if err := first.SetRemote(remoteDir); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(firstDir, "bundle")
	if err := os.WriteFile(file, []byte("base"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := first.CommitSnapshot("main", "base"); err != nil || !ok {
		t.Fatalf("base commit: ok=%v err=%v", ok, err)
	}
	if err := first.Push("main"); err != nil {
		t.Fatal(err)
	}

	secondDir := filepath.Join(t.TempDir(), "second")
	second, err := Clone(remoteDir, secondDir, "main")
	if err != nil {
		t.Fatal(err)
	}
	configureTestIdentity(t, second)
	if err := os.WriteFile(file, []byte("local wins"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := first.CommitSnapshot("main", "local"); err != nil || !ok {
		t.Fatalf("local commit: ok=%v err=%v", ok, err)
	}
	if err := os.WriteFile(filepath.Join(secondDir, "bundle"), []byte("remote diverged"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := second.CommitSnapshot("main", "remote"); err != nil || !ok {
		t.Fatalf("remote commit: ok=%v err=%v", ok, err)
	}
	if err := second.Push("main"); err != nil {
		t.Fatal(err)
	}

	if err := first.PushProgress("main", io.Discard); err != nil {
		t.Fatal(err)
	}
	remote := Repo{Dir: remoteDir}
	got, err := remote.run("show", "main:bundle")
	if err != nil {
		t.Fatal(err)
	}
	if got != "local wins" {
		t.Fatalf("published bundle = %q, want local snapshot", got)
	}
}

func TestPullProgressAbortsConflictAndUsesRemoteSnapshot(t *testing.T) {
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	if _, err := initBareForTest(remoteDir); err != nil {
		t.Fatal(err)
	}
	publisherDir := t.TempDir()
	publisher, err := Init(publisherDir, "main")
	if err != nil {
		t.Fatal(err)
	}
	configureTestIdentity(t, publisher)
	if err := publisher.SetRemote(remoteDir); err != nil {
		t.Fatal(err)
	}
	publishedFile := filepath.Join(publisherDir, "bundle")
	if err := os.WriteFile(publishedFile, []byte("base"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := publisher.CommitSnapshot("main", "base"); err != nil || !ok {
		t.Fatalf("base commit: ok=%v err=%v", ok, err)
	}
	if err := publisher.Push("main"); err != nil {
		t.Fatal(err)
	}

	consumerDir := filepath.Join(t.TempDir(), "consumer")
	consumer, err := Clone(remoteDir, consumerDir, "main")
	if err != nil {
		t.Fatal(err)
	}
	configureTestIdentity(t, consumer)
	if err := os.WriteFile(filepath.Join(consumerDir, "bundle"), []byte("local"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := consumer.CommitSnapshot("main", "local"); err != nil || !ok {
		t.Fatalf("local commit: ok=%v err=%v", ok, err)
	}
	if err := os.WriteFile(filepath.Join(consumerDir, "untracked.tmp"), []byte("remove"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(publishedFile, []byte("remote wins"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := publisher.CommitSnapshot("main", "remote"); err != nil || !ok {
		t.Fatalf("remote commit: ok=%v err=%v", ok, err)
	}
	if err := publisher.Push("main"); err != nil {
		t.Fatal(err)
	}
	if err := consumer.Fetch("origin"); err != nil {
		t.Fatal(err)
	}
	if _, err := consumer.run("merge", "refs/remotes/origin/main"); err == nil {
		t.Fatal("expected binary snapshot merge conflict")
	}

	if err := consumer.PullProgress("main", io.Discard); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(consumerDir, "bundle"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "remote wins" {
		t.Fatalf("restored bundle = %q, want remote snapshot", got)
	}
	if _, err := os.Stat(filepath.Join(consumerDir, "untracked.tmp")); !os.IsNotExist(err) {
		t.Fatalf("untracked sync artifact was not removed: %v", err)
	}
}

func configureTestIdentity(t *testing.T, r Repo) {
	t.Helper()
	for key, value := range map[string]string{
		"user.name": "contix-test", "user.email": "contix@example.invalid",
	} {
		if _, err := r.run("config", key, value); err != nil {
			t.Fatal(err)
		}
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

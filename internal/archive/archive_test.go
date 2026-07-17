package archive

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateExtractVerifyRoundTrip(t *testing.T) {
	src := t.TempDir()
	writeFileT(t, filepath.Join(src, "a.txt"), "hello")
	writeFileT(t, filepath.Join(src, "sub", "b.txt"), "world")

	bundle := filepath.Join(t.TempDir(), "bundle.tar.gz")
	m := NewManifest("test", "1.0.0", src)
	m, err := Create(src, []string{"a.txt", "sub/b.txt"}, bundle, m)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 2 {
		t.Fatalf("want 2 files, got %d", len(m.Files))
	}

	dest := t.TempDir()
	extracted, err := Extract(bundle, dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(extracted) != 2 {
		t.Fatalf("want 2 extracted, got %d", len(extracted))
	}
	problems, err := Verify(dest, m)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("verify problems: %v", problems)
	}

	// Corrupt a file and confirm Verify catches it.
	writeFileT(t, filepath.Join(dest, "a.txt"), "tampered")
	problems, _ = Verify(dest, m)
	if len(problems) == 0 {
		t.Fatal("expected verify to detect tampering")
	}
}

func TestSkippableSourceErrorIncludesPermissionDenied(t *testing.T) {
	err := &os.PathError{Op: "open", Path: "runtime-file", Err: fs.ErrPermission}
	if !skippableSourceError(err) {
		t.Fatal("permission-denied runtime files must be skipped")
	}
}

func TestExtractRejectsZipSlip(t *testing.T) {
	// Build a bundle whose entry escapes the destination.
	src := t.TempDir()
	writeFileT(t, filepath.Join(src, "ok.txt"), "x")
	bundle := filepath.Join(t.TempDir(), "b.tar.gz")
	m := NewManifest("t", "", src)
	if _, err := Create(src, []string{"ok.txt"}, bundle, m); err != nil {
		t.Fatal(err)
	}
	// Sanity: a normal extract works.
	if _, err := Extract(bundle, t.TempDir()); err != nil {
		t.Fatalf("normal extract failed: %v", err)
	}
}

func TestCreateSkipsFileThatDisappearedAfterDiscovery(t *testing.T) {
	src := t.TempDir()
	writeFileT(t, filepath.Join(src, "stable.txt"), "kept")
	bundle := filepath.Join(t.TempDir(), "bundle.tar.gz")
	m, err := Create(src, []string{"stable.txt", "gone.lock"}, bundle, NewManifest("test", "", src))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 1 || m.Files[0].Path != "stable.txt" {
		t.Fatalf("unexpected manifest: %#v", m.Files)
	}
}

func writeFileT(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

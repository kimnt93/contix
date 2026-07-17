package archive

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
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

func TestCreateSplitsAndExtractsLargeCompressedBundle(t *testing.T) {
	oldPartSize := bundlePartSize
	bundlePartSize = 1024
	defer func() { bundlePartSize = oldPartSize }()

	src := t.TempDir()
	data := make([]byte, 8*1024)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "random.bin"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(t.TempDir(), BundleName)
	m, err := Create(src, []string{"random.bin"}, bundle, NewManifest("test", "", src))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.BundleParts) < 2 {
		t.Fatalf("want a chunked bundle, got %d parts", len(m.BundleParts))
	}
	if _, err := os.Stat(bundle); !os.IsNotExist(err) {
		t.Fatalf("single bundle should not exist after chunking: %v", err)
	}
	for _, part := range m.BundleParts {
		if part.Size > bundlePartSize {
			t.Fatalf("part %s is too large: %d", part.Name, part.Size)
		}
	}
	dest := t.TempDir()
	if _, err := Extract(bundle, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "random.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatal("chunked bundle did not round-trip")
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

func TestCreateSkipsFileThatDisappearsAfterDiscovery(t *testing.T) {
	src := t.TempDir()
	writeFileT(t, filepath.Join(src, "stable.txt"), "kept")
	bundle := filepath.Join(t.TempDir(), "bundle.tar.gz")
	m, err := Create(src, []string{"stable.txt", "gone.lock"}, bundle, NewManifest("test", "", src))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 1 || m.Files[0].Path != "stable.txt" {
		t.Fatalf("manifest files = %#v, want only stable.txt", m.Files)
	}
}

func TestCreateRetriesPermissionUntilRuntimeFileDisappears(t *testing.T) {
	src := t.TempDir()
	writeFileT(t, filepath.Join(src, "stable.txt"), "kept")
	heartbeat := filepath.Join(src, "ticker_heartbeat")
	writeFileT(t, heartbeat, "temporary")

	oldOpen := openStageSource
	oldRetries := stagePermissionRetries
	oldDelay := stagePermissionRetryDelay
	defer func() {
		openStageSource = oldOpen
		stagePermissionRetries = oldRetries
		stagePermissionRetryDelay = oldDelay
	}()
	stagePermissionRetries = 2
	stagePermissionRetryDelay = 0
	attempts := 0
	openStageSource = func(path string) (*os.File, error) {
		if path != heartbeat {
			return os.Open(path)
		}
		attempts++
		if attempts == 1 {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrPermission}
		}
		return os.Open(path)
	}

	bundle := filepath.Join(t.TempDir(), "bundle.tar.gz")
	m, err := Create(src, []string{"stable.txt", "ticker_heartbeat"}, bundle, NewManifest("test", "", src))
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("open attempts = %d, want 2", attempts)
	}
	if len(m.Files) != 1 || m.Files[0].Path != "stable.txt" {
		t.Fatalf("manifest files = %#v, want only stable.txt", m.Files)
	}
}

func TestOpenStageFileKeepsStablePermissionFailureFatal(t *testing.T) {
	oldOpen := openStageSource
	oldRetries := stagePermissionRetries
	oldDelay := stagePermissionRetryDelay
	defer func() {
		openStageSource = oldOpen
		stagePermissionRetries = oldRetries
		stagePermissionRetryDelay = oldDelay
	}()
	stagePermissionRetries = 2
	stagePermissionRetryDelay = time.Nanosecond
	attempts := 0
	openStageSource = func(path string) (*os.File, error) {
		attempts++
		return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrPermission}
	}

	_, err := openStageFile("persistent-secret")
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("error = %v, want permission denied", err)
	}
	if attempts != 3 {
		t.Fatalf("open attempts = %d, want 3", attempts)
	}
}

func TestCreateOmitsOnlyConfiguredUnreadableVolatilePath(t *testing.T) {
	src := t.TempDir()
	writeFileT(t, filepath.Join(src, "stable.txt"), "kept")
	heartbeat := filepath.Join(src, "cron", "ticker_heartbeat")
	writeFileT(t, heartbeat, "runtime timestamp")

	oldOpen := openStageSource
	oldRetries := stagePermissionRetries
	oldDelay := stagePermissionRetryDelay
	defer func() {
		openStageSource = oldOpen
		stagePermissionRetries = oldRetries
		stagePermissionRetryDelay = oldDelay
	}()
	stagePermissionRetries = 1
	stagePermissionRetryDelay = 0
	openStageSource = func(path string) (*os.File, error) {
		if path == heartbeat {
			return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrPermission}
		}
		return os.Open(path)
	}

	bundle := filepath.Join(t.TempDir(), "bundle.tar.gz")
	m, err := Create(
		src,
		[]string{"stable.txt", "cron/ticker_heartbeat"},
		bundle,
		NewManifest("hermes", "", src),
		"cron/ticker_heartbeat",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.Omitted, []string{"cron/ticker_heartbeat"}) {
		t.Fatalf("omitted = %v", m.Omitted)
	}
	if len(m.Files) != 1 || m.Files[0].Path != "stable.txt" {
		t.Fatalf("manifest files = %#v, want only stable.txt", m.Files)
	}

	if _, err := Create(
		src,
		[]string{"cron/ticker_heartbeat"},
		filepath.Join(t.TempDir(), "bundle.tar.gz"),
		NewManifest("hermes", "", src),
	); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("unconfigured permission error = %v, want permission denied", err)
	}
}

func TestSymlinkRoundTripAndConflictDetection(t *testing.T) {
	src := t.TempDir()
	writeFileT(t, filepath.Join(src, "target.txt"), "target")
	if err := os.Symlink("target.txt", filepath.Join(src, "link.txt")); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(t.TempDir(), BundleName)
	m, err := Create(src, []string{"link.txt", "target.txt"}, bundle, NewManifest("test", "", src))
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if _, err := Extract(bundle, dest); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(filepath.Join(dest, "link.txt"))
	if err != nil || target != "target.txt" {
		t.Fatalf("restored symlink target = %q, %v", target, err)
	}
	conflicts, err := Conflicts(dest, m)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %v, %v", conflicts, err)
	}
	if err := os.Remove(filepath.Join(dest, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("other.txt", filepath.Join(dest, "link.txt")); err != nil {
		t.Fatal(err)
	}
	writeFileT(t, filepath.Join(dest, "target.txt"), "local")
	conflicts, err = Conflicts(dest, m)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 2 {
		t.Fatalf("conflicts = %v, want link.txt and target.txt", conflicts)
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

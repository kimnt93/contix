package tool

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAgentTargetIncludesAllRegularFilesAndSymlinks(t *testing.T) {
	root := t.TempDir()
	files := []string{
		"auth.json",
		"cache/models.bin",
		"logs/runtime.log",
		"nested/.git/config",
		"sessions/live.lock",
	}
	for _, name := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("auth.json", filepath.Join(root, "auth-link")); err != nil {
		t.Fatal(err)
	}

	target := codex()
	target.Home = func() string { return root }
	got, err := target.IncludedFiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"auth-link",
		"auth.json",
		"cache/models.bin",
		"logs/runtime.log",
		"nested/.git/config",
		"sessions/live.lock",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("included paths = %v, want %v", got, want)
	}
}

func TestSSHIncludesKeysKnownHostsAndBackups(t *testing.T) {
	root := t.TempDir()
	files := []string{"config", "id_ed25519", "id_ed25519.pub", "known_hosts", "backup/config"}
	for _, name := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	target := sshConfig()
	target.Home = func() string { return root }
	got, err := target.IncludedFiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"backup/config", "config", "id_ed25519", "id_ed25519.pub", "known_hosts"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("included SSH paths = %v, want %v", got, want)
	}
}

func TestHostsTargetOnlyIncludesHostsFile(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"hosts", "passwd"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	target := hosts()
	target.Home = func() string { return root }
	got, err := target.IncludedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"hosts"}) {
		t.Fatalf("hosts paths = %v", got)
	}
}

func TestRegistryContainsAgentsButNoIDEs(t *testing.T) {
	want := []string{"antigravity", "claude", "codex", "hermes", "hosts", "kiro", "openclaw", "ssh"}
	if got := Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for _, name := range RetiredNames() {
		if _, ok := Lookup(name); ok {
			t.Errorf("retired IDE target %q is still registered", name)
		}
	}
}

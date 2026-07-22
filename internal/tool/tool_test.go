package tool

import (
	"net"
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

func TestCursorIncludesOnlyPortableAgentFiles(t *testing.T) {
	root := t.TempDir()
	files := []string{"mcp.json", "cli-config.json", "rules/global.mdc", "extensions/huge.bin", "cache/index.db"}
	for _, name := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	target := cursor()
	target.Home = func() string { return root }
	got, err := target.IncludedFiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"cli-config.json", "mcp.json", "rules/global.mdc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("included Cursor paths = %v, want %v", got, want)
	}
}

func TestRegistryContainsOnlyCodingAgents(t *testing.T) {
	want := []string{
		"aider", "amp", "antigravity", "auggie", "claude", "cline", "codex", "continue", "copilot",
		"cursor", "droid", "goose", "goose-config", "kiro", "opencode", "opencode-config", "qwen",
	}
	if got := Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for _, name := range RetiredNames() {
		if _, ok := Lookup(name); ok {
			t.Errorf("retired IDE target %q is still registered", name)
		}
	}
}

func TestAntigravityDoesNotProbeDesktopLauncher(t *testing.T) {
	target := antigravity()
	if target.Binary != "" {
		t.Fatalf("Antigravity binary probe = %q; probing the desktop launcher opens the IDE", target.Binary)
	}
}

func TestIncludedFilesSkipsRuntimeSockets(t *testing.T) {
	root := t.TempDir()
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(root, "runtime.sock"), Net: "unix"})
	if err != nil {
		t.Skipf("Unix sockets unavailable: %v", err)
	}
	defer listener.Close()
	if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := Tool{Name: "socket-test", Home: func() string { return root }}
	got, err := target.IncludedFiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"settings.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("included paths = %v, want %v", got, want)
	}
}

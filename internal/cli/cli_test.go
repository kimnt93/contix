package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseToolsSelectsAgentsAndDeduplicates(t *testing.T) {
	targets, err := parseTools("opencode,codex,opencode")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, target := range targets {
		names = append(names, target.Name)
	}
	want := []string{"opencode", "opencode-config", "codex"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("targets = %v, want %v", names, want)
	}
}

func TestParseToolsRejectsGeneralAgent(t *testing.T) {
	if _, err := parseTools("hermes"); err == nil {
		t.Fatal("general agent target must not be selectable")
	}
}

func TestRemoveRetiredSnapshotsKeepsAgentState(t *testing.T) {
	repo := t.TempDir()
	for _, name := range []string{"cursor", "vscode-home", "hermes", "openclaw", "codex"} {
		if err := os.MkdirAll(filepath.Join(repo, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := removeRetiredSnapshots(repo)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 3 {
		t.Fatalf("removed = %d, want 3", removed)
	}
	if _, err := os.Stat(filepath.Join(repo, "codex")); err != nil {
		t.Fatalf("agent state was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "cursor")); err != nil {
		t.Fatalf("coding-agent state was removed: %v", err)
	}
}

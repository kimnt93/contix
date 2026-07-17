package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseToolsSelectsAgentsAndDeduplicates(t *testing.T) {
	targets, err := parseTools("openclaw,codex,openclaw")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, target := range targets {
		names = append(names, target.Name)
	}
	want := []string{"openclaw", "codex"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("targets = %v, want %v", names, want)
	}
}

func TestParseToolsRejectsRetiredIDE(t *testing.T) {
	if _, err := parseTools("cursor"); err == nil {
		t.Fatal("retired IDE target must not be selectable")
	}
}

func TestRemoveRetiredSnapshotsKeepsAgentState(t *testing.T) {
	repo := t.TempDir()
	for _, name := range []string{"cursor", "vscode-home", "codex"} {
		if err := os.MkdirAll(filepath.Join(repo, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := removeRetiredSnapshots(repo)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	if _, err := os.Stat(filepath.Join(repo, "codex")); err != nil {
		t.Fatalf("agent state was removed: %v", err)
	}
}

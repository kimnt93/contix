package cli

import (
	"reflect"
	"testing"
)

func TestParseToolsExpandsEditorGroupsAndDeduplicates(t *testing.T) {
	targets, err := parseTools("cursor,cursor-home,codex")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, target := range targets {
		names = append(names, target.Name)
	}
	want := []string{"cursor", "cursor-home", "codex"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("targets = %v, want %v", names, want)
	}
}

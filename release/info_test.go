package releaseinfo

import (
	"strings"
	"testing"
)

func TestEmbeddedReleaseMetadata(t *testing.T) {
	if Version() == "" {
		t.Fatal("embedded version is empty")
	}
	if Notes() == "" {
		t.Fatal("embedded release notes are empty")
	}
	for index, line := range strings.Split(Notes(), "\n") {
		if !strings.HasPrefix(line, "- [x] ") {
			t.Fatalf("feature line %d is not a completed checklist item: %q", index+1, line)
		}
	}
}

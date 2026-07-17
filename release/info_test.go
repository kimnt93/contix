package releaseinfo

import "testing"

func TestEmbeddedReleaseMetadata(t *testing.T) {
	if Version() == "" {
		t.Fatal("embedded version is empty")
	}
	if Notes() == "" {
		t.Fatal("embedded release notes are empty")
	}
}

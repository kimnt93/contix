package platform

import (
	"path/filepath"
	"testing"
)

func TestOpenClawStateDirOverride(t *testing.T) {
	want := t.TempDir()
	t.Setenv("OPENCLAW_STATE_DIR", want)
	t.Setenv("OPENCLAW_HOME", t.TempDir())
	if got := OpenClawHome(); got != want {
		t.Fatalf("OpenClawHome() = %q, want %q", got, want)
	}
}

func TestOpenClawHomeAndProfile(t *testing.T) {
	base := t.TempDir()
	t.Setenv("OPENCLAW_STATE_DIR", "")
	t.Setenv("OPENCLAW_HOME", base)
	t.Setenv("OPENCLAW_PROFILE", "work")
	want := filepath.Join(base, ".openclaw-work")
	if got := OpenClawHome(); got != want {
		t.Fatalf("OpenClawHome() = %q, want %q", got, want)
	}
}

func TestOpenClawStateDirExpandsTilde(t *testing.T) {
	t.Setenv("OPENCLAW_STATE_DIR", filepath.Join("~", "openclaw-state"))
	want := filepath.Join(Home(), "openclaw-state")
	if got := OpenClawHome(); got != want {
		t.Fatalf("OpenClawHome() = %q, want %q", got, want)
	}
}

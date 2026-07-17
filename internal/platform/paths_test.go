package platform

import (
	"path/filepath"
	"testing"
)

func TestCodingAgentLocationOverrides(t *testing.T) {
	want := t.TempDir()
	t.Setenv("COPILOT_HOME", want)
	if got := CopilotHome(); got != want {
		t.Fatalf("CopilotHome() = %q, want %q", got, want)
	}
}

func TestAdditionalCodingAgentLocationOverrides(t *testing.T) {
	for name, test := range map[string]struct {
		env string
		get func() string
	}{
		"amp":    {"CONTIX_AMP_HOME", AmpHome},
		"auggie": {"CONTIX_AUGGIE_HOME", AuggieHome},
		"droid":  {"CONTIX_DROID_HOME", DroidHome},
		"qwen":   {"CONTIX_QWEN_HOME", QwenHome},
	} {
		t.Run(name, func(t *testing.T) {
			want := t.TempDir()
			t.Setenv(test.env, want)
			if got := test.get(); got != want {
				t.Fatalf("state home = %q, want %q", got, want)
			}
		})
	}
}

func TestGeminiCLIHome(t *testing.T) {
	base := t.TempDir()
	t.Setenv("ANTIGRAVITY_HOME", "")
	t.Setenv("GEMINI_CLI_HOME", base)
	if got, want := AntigravityHome(), filepath.Join(base, ".gemini"); got != want {
		t.Fatalf("AntigravityHome() = %q, want %q", got, want)
	}
}

func TestOpenCodeXDGPaths(t *testing.T) {
	base := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "data"))
	if got, want := OpenCodeConfigHome(), filepath.Join(base, "config", "opencode"); got != want {
		t.Fatalf("OpenCodeConfigHome() = %q, want %q", got, want)
	}
	if got, want := OpenCodeDataHome(), filepath.Join(base, "data", "opencode"); got != want {
		t.Fatalf("OpenCodeDataHome() = %q, want %q", got, want)
	}
}

func TestGoosePathRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOOSE_PATH_ROOT", root)
	if got := GooseDataHome(); got != root {
		t.Fatalf("GooseDataHome() = %q, want %q", got, root)
	}
	if got, want := GooseConfigHome(), filepath.Join(root, "config"); got != want {
		t.Fatalf("GooseConfigHome() = %q, want %q", got, want)
	}
}

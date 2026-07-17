package platform

import "testing"

func TestEditorLocationOverrides(t *testing.T) {
	tests := []struct {
		env string
		get func() string
	}{
		{"CONTIX_CURSOR_DATA_HOME", CursorDataHome},
		{"CONTIX_CURSOR_HOME", CursorHome},
		{"CONTIX_WINDSURF_DATA_HOME", WindsurfDataHome},
		{"CONTIX_WINDSURF_HOME", WindsurfHome},
		{"CONTIX_WINDSURF_AGENT_HOME", WindsurfAgentHome},
		{"CONTIX_VSCODE_DATA_HOME", VSCodeDataHome},
		{"CONTIX_VSCODE_HOME", VSCodeHome},
		{"CONTIX_VSCODIUM_DATA_HOME", VSCodiumDataHome},
		{"CONTIX_VSCODIUM_HOME", VSCodiumHome},
		{"CONTIX_VOID_DATA_HOME", VoidDataHome},
		{"CONTIX_VOID_HOME", VoidHome},
		{"CONTIX_KIRO_IDE_HOME", KiroIDEHome},
		{"CONTIX_ANTIGRAVITY_IDE_HOME", AntigravityIDEHome},
		{"CONTIX_ANTIGRAVITY_EXTENSIONS_HOME", AntigravityExtensionsHome},
	}
	for _, test := range tests {
		t.Run(test.env, func(t *testing.T) {
			want := t.TempDir()
			t.Setenv(test.env, want)
			if got := test.get(); got != want {
				t.Fatalf("%s = %q, want %q", test.env, got, want)
			}
		})
	}
}

package tool

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMatch(t *testing.T) {
	cases := []struct {
		rel, pat string
		want     bool
	}{
		{"config.toml", "config.toml", true},
		{"sessions/s1.jsonl", "sessions/", true},
		{"sessions", "sessions/", true},
		{"projects/a/b.jsonl", "projects/", true},
		{"openai.config.toml", "*.config.toml", true},
		{"nested/openai.config.toml", "*.config.toml", true}, // basename glob
		{"auth.json", "auth.json", true},
		{"cache/models.json", "cache/", true},
		{"logs_2.sqlite", "logs_*.sqlite", true},
		{"memories_1.sqlite", "logs_*.sqlite", false},
		{"CLAUDE.md", "sessions/", false},
		{"plugins/config.json", "plugins/config.json", true},
		{"plugins/marketplaces/x", "plugins/marketplaces/", true},
	}
	for _, c := range cases {
		if got := match(c.rel, c.pat); got != c.want {
			t.Errorf("match(%q,%q)=%v want %v", c.rel, c.pat, got, c.want)
		}
	}
}

func TestCredentialsAlwaysExcluded(t *testing.T) {
	// Credentials must never be synced, even under the sync-all model.
	if !matchAny("auth.json", codex().Exclude) {
		t.Fatal("codex auth.json must match an exclude pattern")
	}
	if !matchAny(".credentials.json", claude().Exclude) {
		t.Fatal("claude .credentials.json must match an exclude pattern")
	}
	// Nested git repos must be pruned to avoid corrupting the sync repo.
	if !matchAny(".git", codex().Exclude) {
		t.Fatal("nested .git must match an exclude pattern")
	}
	if !matchAny("auth.json", hermes().Exclude) || !matchAny(".env", hermes().Exclude) {
		t.Fatal("hermes credentials must match exclude patterns")
	}
	if !matchAny("hermes-agent/venv/bin/python", hermes().Exclude) {
		t.Fatal("hermes installation tree must be excluded")
	}
	if !matchAny("cron/ticker_heartbeat", hermes().Exclude) {
		t.Fatal("hermes runtime heartbeat must be excluded")
	}
	if !matchAny("sessions/a.lock", kiro().Exclude) {
		t.Fatal("kiro runtime locks must be excluded")
	}
	if !matchAny("antigravity/installation_id", antigravity().Exclude) {
		t.Fatal("antigravity machine identity must be excluded")
	}
}

func TestSSHConfigAllowlistNeverIncludesKeysOrNestedBackups(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"config":                     "Host dev\n",
		"config.d/work.conf":         "Host work\n",
		"config.d/accidental.key":    "PRIVATE KEY",
		"config.d/disguised.conf":    "-----BEGIN OPENSSH PRIVATE KEY-----\nsecret",
		"conf.d/personal.conf":       "Host personal\n",
		"id_ed25519":                 "PRIVATE KEY",
		"id_ed25519.pub":             "public key",
		"known_hosts":                "host key",
		"backup/config":              "Host old\n",
		"backup/config.d/secret.key": "PRIVATE KEY",
	}
	for name, data := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	target := sshConfig()
	target.Home = func() string { return root }
	got, err := target.IncludedFiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"conf.d/personal.conf", "config", "config.d/work.conf"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("included SSH paths = %v, want %v", got, want)
	}
}

func TestAntigravityAllowlist(t *testing.T) {
	root := t.TempDir()
	files := []string{
		"GEMINI.md",
		"antigravity/conversations/session.pb",
		"antigravity/installation_id",
		"oauth_creds.json",
		"tmp/cache.bin",
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

	target := antigravity()
	target.Home = func() string { return root }
	got, err := target.IncludedFiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"GEMINI.md", "antigravity/conversations/session.pb"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("included Antigravity paths = %v, want %v", got, want)
	}
}

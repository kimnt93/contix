package tool

import "testing"

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
}

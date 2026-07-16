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

func TestExcludeWinsOverInclude(t *testing.T) {
	tl := codex()
	// auth.json is excluded even though it lives directly in home.
	if matchAny("auth.json", tl.Include) && !matchAny("auth.json", tl.Exclude) {
		t.Fatal("auth.json should be excluded")
	}
	if !matchAny("auth.json", tl.Exclude) {
		t.Fatal("auth.json must match an exclude pattern")
	}
}

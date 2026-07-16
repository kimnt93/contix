package pathrewrite

import (
	"strings"
	"testing"
)

func TestContentPOSIX(t *testing.T) {
	r := &Rewriter{maps: []Mapping{{Old: "/home/alice", New: "/home/bob"}}, targetOS: "linux"}
	in := []byte(`{"cwd":"/home/alice/project"}`)
	out := string(r.Content(in, true))
	if !strings.Contains(out, "/home/bob/project") {
		t.Fatalf("expected rewrite, got %s", out)
	}
}

func TestContentWindowsJSONEscaping(t *testing.T) {
	r := &Rewriter{maps: []Mapping{{Old: "/home/alice", New: `C:\Users\bob`}}, targetOS: "windows"}
	in := []byte(`{"cwd":"/home/alice/p"}`)
	out := string(r.Content(in, true))
	// Backslashes must be doubled to keep JSON valid.
	if !strings.Contains(out, `C:\\Users\\bob`) {
		t.Fatalf("expected escaped windows path, got %s", out)
	}
}

func TestVariantsCoversBackslash(t *testing.T) {
	vs := variants("/home/alice")
	found := false
	for _, v := range vs {
		if v == `\home\alice` {
			found = true
		}
	}
	if !found {
		t.Fatalf("variants missing backslash form: %v", vs)
	}
}

func TestEncodeMatchesClaudeScheme(t *testing.T) {
	if got := encode("/home/kim/proj"); got != "-home-kim-proj" {
		t.Fatalf("encode = %q", got)
	}
}

func TestParseMapping(t *testing.T) {
	m, ok := ParseMapping("/a=/b")
	if !ok || m.Old != "/a" || m.New != "/b" {
		t.Fatalf("ParseMapping bad: %+v ok=%v", m, ok)
	}
	if _, ok := ParseMapping("noequals"); ok {
		t.Fatal("expected failure for missing '='")
	}
}

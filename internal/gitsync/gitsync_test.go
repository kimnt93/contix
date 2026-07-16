package gitsync

import (
	"strings"
	"testing"
)

func TestKeyStableAndUnique(t *testing.T) {
	k1 := Key("/home/kim/code/proj")
	k2 := Key("/home/kim/code/proj")
	k3 := Key("/home/kim/other/proj")
	if k1 != k2 {
		t.Fatalf("Key not stable: %s vs %s", k1, k2)
	}
	if k1 == k3 {
		t.Fatal("Key should differ for different paths with same basename")
	}
	if !strings.HasPrefix(k1, "proj-") {
		t.Fatalf("Key should start with slug: %s", k1)
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"My Project": "my-project",
		"proj_v2":    "proj-v2",
		"UPPER":      "upper",
		"--edge--":   "edge",
		"a.b.c":      "a-b-c",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q)=%q want %q", in, got, want)
		}
	}
}

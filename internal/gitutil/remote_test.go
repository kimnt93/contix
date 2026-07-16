package gitutil

import "testing"

func TestClassifyRemote(t *testing.T) {
	cases := []struct {
		url  string
		want RemoteKind
	}{
		{"https://github.com/kimnt93/dev-state.git", RemoteHTTP},
		{"http://example.com/x/y.git", RemoteHTTP},
		{"git@github.com:kimnt93/dev-state.git", RemoteSSH},
		{"ssh://git@github.com/kimnt93/dev-state.git", RemoteSSH},
		{"git://github.com/kimnt93/dev-state.git", RemoteGit},
		{"file:///srv/repos/dev-state.git", RemoteLocal},
		{"/srv/repos/dev-state.git", RemoteLocal},
		{"./relative/repo", RemoteLocal},
		{"", RemoteUnknown},
	}
	for _, c := range cases {
		if got := ClassifyRemote(c.url); got != c.want {
			t.Errorf("ClassifyRemote(%q)=%v want %v", c.url, got, c.want)
		}
	}
}

func TestValidateRemote(t *testing.T) {
	ok := []string{
		"https://github.com/kimnt93/dev-state.git",
		"git@github.com:kimnt93/dev-state.git",
		"ssh://git@github.com/kimnt93/dev-state.git",
	}
	for _, u := range ok {
		if err := ValidateRemote(u); err != nil {
			t.Errorf("ValidateRemote(%q) unexpected error: %v", u, err)
		}
	}
	bad := []string{"", "   ", "has space/repo.git"}
	for _, u := range bad {
		if err := ValidateRemote(u); err == nil {
			t.Errorf("ValidateRemote(%q) expected error, got nil", u)
		}
	}
}

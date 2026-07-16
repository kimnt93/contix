package gitutil

import (
	"fmt"
	"strings"
)

// RemoteKind classifies the transport of a git remote URL.
type RemoteKind int

const (
	// RemoteUnknown means the URL didn't match a recognised form.
	RemoteUnknown RemoteKind = iota
	// RemoteSSH is scp-like (git@host:owner/repo.git) or ssh:// syntax.
	RemoteSSH
	// RemoteHTTP is http:// or https:// syntax.
	RemoteHTTP
	// RemoteGit is the git:// protocol.
	RemoteGit
	// RemoteLocal is a filesystem path or file:// URL.
	RemoteLocal
)

func (k RemoteKind) String() string {
	switch k {
	case RemoteSSH:
		return "ssh"
	case RemoteHTTP:
		return "http(s)"
	case RemoteGit:
		return "git"
	case RemoteLocal:
		return "local"
	default:
		return "unknown"
	}
}

// ClassifyRemote reports the transport used by a git remote URL. Both SSH
// (git@github.com:you/repo.git and ssh://…) and HTTP(S)
// (https://github.com/you/repo.git) forms are recognised.
func ClassifyRemote(url string) RemoteKind {
	u := strings.TrimSpace(url)
	switch {
	case u == "":
		return RemoteUnknown
	case strings.HasPrefix(u, "https://"), strings.HasPrefix(u, "http://"):
		return RemoteHTTP
	case strings.HasPrefix(u, "ssh://"):
		return RemoteSSH
	case strings.HasPrefix(u, "git://"):
		return RemoteGit
	case strings.HasPrefix(u, "file://"):
		return RemoteLocal
	case isSCPLike(u):
		// scp-like SSH shorthand: user@host:path (no scheme, colon before a
		// path, and the part before the colon is not a drive letter).
		return RemoteSSH
	default:
		// Anything else that has no scheme is treated as a local path
		// (relative or absolute), which git also accepts as a remote.
		return RemoteLocal
	}
}

// isSCPLike reports whether s is git's scp-like SSH shorthand,
// e.g. git@github.com:you/repo.git or host:path.
func isSCPLike(s string) bool {
	if strings.Contains(s, "://") {
		return false
	}
	colon := strings.Index(s, ":")
	if colon <= 0 {
		return false
	}
	host := s[:colon]
	// A leading slash means it's a path like /a:b, not scp syntax.
	if strings.HasPrefix(s, "/") {
		return false
	}
	// Reject Windows drive letters (C:\...) which are local paths.
	if len(host) == 1 {
		return false
	}
	// The host part must not contain a slash (that would be a path with a colon).
	return !strings.Contains(host, "/")
}

// ValidateRemote checks that url is a usable git remote. It accepts SSH,
// HTTP(S), git:// and local paths. It only rejects clearly invalid input
// (empty or containing whitespace), leaving anything git might accept alone.
func ValidateRemote(url string) error {
	u := strings.TrimSpace(url)
	if u == "" {
		return fmt.Errorf("empty remote URL")
	}
	if strings.ContainsAny(u, " \t\n") {
		return fmt.Errorf("remote URL must not contain whitespace: %q", url)
	}
	return nil
}

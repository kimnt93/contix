package gitutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Branch describes a local branch and its tracking information.
type Branch struct {
	Name     string `json:"name"`
	SHA      string `json:"sha"`
	Upstream string `json:"upstream,omitempty"`
}

// runRaw runs git and returns stdout bytes without trimming (needed for
// patches, which must be preserved byte-for-byte).
func (r Repo) runRaw(args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", r.Dir}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}

// runInput runs git feeding data to stdin.
func (r Repo) runInput(input []byte, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", r.Dir}, args...)...)
	cmd.Stdin = bytes.NewReader(input)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, msg)
	}
	return strings.TrimSpace(out.String()), nil
}

// Toplevel returns the absolute root of the working tree.
func (r Repo) Toplevel() (string, error) {
	return r.run("rev-parse", "--show-toplevel")
}

// CurrentBranch returns the checked-out branch name, or "HEAD" when detached.
func (r Repo) CurrentBranch() (string, error) {
	return r.run("rev-parse", "--abbrev-ref", "HEAD")
}

// HeadSHA returns the commit SHA at HEAD.
func (r Repo) HeadSHA() (string, error) {
	return r.run("rev-parse", "HEAD")
}

// RemoteURL returns the URL for a named remote (e.g. "origin").
func (r Repo) RemoteURL(name string) (string, error) {
	return r.run("remote", "get-url", name)
}

// LocalBranches lists local branches with their commit and upstream.
func (r Repo) LocalBranches() ([]Branch, error) {
	// A tab is safe: branch names, ref names and hex object IDs never contain
	// tabs. A NUL byte cannot be used here because it truncates the argv string.
	const sep = "\t"
	out, err := r.run("for-each-ref",
		"--format=%(refname:short)"+sep+"%(objectname)"+sep+"%(upstream:short)",
		"refs/heads")
	if err != nil {
		return nil, err
	}
	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, sep)
		if len(parts) < 2 {
			continue
		}
		b := Branch{Name: parts[0], SHA: parts[1]}
		if len(parts) >= 3 {
			b.Upstream = parts[2]
		}
		branches = append(branches, b)
	}
	return branches, nil
}

// DiffPatch returns a patch of all tracked changes relative to HEAD
// (staged and unstaged). Binary diffs are included so they can be reapplied.
func (r Repo) DiffPatch() ([]byte, error) {
	return r.runRaw("diff", "--binary", "HEAD")
}

// Untracked lists untracked files that are not covered by .gitignore.
func (r Repo) Untracked() ([]string, error) {
	out, err := r.run("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			files = append(files, s)
		}
	}
	return files, nil
}

// ApplyPatch applies a diff produced by DiffPatch using a 3-way merge so it can
// tolerate minor drift. Returns nil for an empty patch.
func (r Repo) ApplyPatch(patch []byte) error {
	if len(bytes.TrimSpace(patch)) == 0 {
		return nil
	}
	_, err := r.runInput(patch, "apply", "--3way", "--whitespace=nowarn")
	return err
}

// Fetch fetches all branches and tags from a remote.
func (r Repo) Fetch(remote string) error {
	_, err := r.run("fetch", "--tags", remote)
	return err
}

// BranchExists reports whether a local branch exists.
func (r Repo) BranchExists(name string) bool {
	_, err := r.run("rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

// CommitExists reports whether an object is present in the local repo.
func (r Repo) CommitExists(sha string) bool {
	_, err := r.run("cat-file", "-e", sha+"^{commit}")
	return err == nil
}

// CreateBranch creates a branch at the given start point (SHA or ref).
func (r Repo) CreateBranch(name, start string) error {
	_, err := r.run("branch", name, start)
	return err
}

// Checkout switches to an existing branch.
func (r Repo) Checkout(name string) error {
	_, err := r.run("checkout", name)
	return err
}

// HasCommits reports whether HEAD points at a real commit (not an unborn
// branch on a fresh repo).
func (r Repo) HasCommits() bool {
	_, err := r.run("rev-parse", "--verify", "-q", "HEAD")
	return err == nil
}

// EnsureBranch makes sure the working branch is named branch. On a repo with no
// commits yet (freshly initialised or cloned from an empty remote) it renames
// the unborn HEAD so the first commit lands on the expected branch.
func (r Repo) EnsureBranch(branch string) error {
	cur, _ := r.run("symbolic-ref", "--short", "-q", "HEAD")
	if cur == branch {
		return nil
	}
	if !r.HasCommits() {
		_, err := r.run("symbolic-ref", "HEAD", "refs/heads/"+branch)
		return err
	}
	if r.BranchExists(branch) {
		return r.Checkout(branch)
	}
	_, err := r.run("checkout", "-b", branch)
	return err
}

// RemoteHasBranch reports whether origin already publishes the given branch.
func (r Repo) RemoteHasBranch(branch string) bool {
	out, err := r.run("ls-remote", "--heads", "origin", branch)
	return err == nil && strings.TrimSpace(out) != ""
}

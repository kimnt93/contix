// Package gitutil is a thin wrapper over the system git CLI. It relies on the
// user's existing git configuration (identity, SSH keys, credential helpers).
package gitutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Repo represents a local git working tree.
type Repo struct {
	Dir string
}

// Available reports whether the git binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func (r Repo) run(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", r.Dir}, args...)...)
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

func (r Repo) runStreaming(out io.Writer, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", r.Dir}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

// IsRepo reports whether Dir is inside a git working tree.
func (r Repo) IsRepo() bool {
	_, err := r.run("rev-parse", "--is-inside-work-tree")
	return err == nil
}

// Init initialises a repo on the given branch, creating the directory.
func Init(dir, branch string) (Repo, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Repo{}, err
	}
	r := Repo{Dir: dir}
	if r.IsRepo() {
		return r, nil
	}
	if _, err := r.run("init", "-b", branch); err != nil {
		// Older git without -b: fall back.
		if _, e2 := r.run("init"); e2 != nil {
			return Repo{}, err
		}
		_, _ = r.run("checkout", "-b", branch)
	}
	return r, nil
}

// Clone clones remote into dir. If dir already contains the repo it is reused.
func Clone(remote, dir, branch string) (Repo, error) {
	r := Repo{Dir: dir}
	if r.IsRepo() {
		return r, nil
	}
	args := []string{"clone", remote, dir}
	if branch != "" {
		args = []string{"clone", "-b", branch, remote, dir}
	}
	cmd := exec.Command("git", args...)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return Repo{}, fmt.Errorf("git clone: %v: %s", err, strings.TrimSpace(errb.String()))
	}
	return r, nil
}

// SetRemote adds or updates the "origin" remote.
func (r Repo) SetRemote(url string) error {
	if _, err := r.run("remote", "get-url", "origin"); err == nil {
		_, err := r.run("remote", "set-url", "origin", url)
		return err
	}
	_, err := r.run("remote", "add", "origin", url)
	return err
}

// HasRemote reports whether origin is configured.
func (r Repo) HasRemote() bool {
	_, err := r.run("remote", "get-url", "origin")
	return err == nil
}

// AddAll stages all changes.
func (r Repo) AddAll() error {
	_, err := r.run("add", "-A")
	return err
}

// IsClean reports whether the working tree has no staged/unstaged changes.
func (r Repo) IsClean() (bool, error) {
	out, err := r.run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out == "", nil
}

// Commit records a commit. Returns false if there was nothing to commit.
func (r Repo) Commit(message string) (bool, error) {
	clean, err := r.IsClean()
	if err != nil {
		return false, err
	}
	if clean {
		return false, nil
	}
	if err := r.AddAll(); err != nil {
		return false, err
	}
	if _, err := r.run("commit", "-m", message); err != nil {
		return false, err
	}
	return true, nil
}

// CommitSnapshot commits the current latest-only snapshot. If local snapshot
// commits have never reached origin, they are replaced/squashed so rejected
// large blobs do not remain in the history sent by the next push.
func (r Repo) CommitSnapshot(branch, message string) (bool, error) {
	clean, err := r.IsClean()
	if err != nil {
		return false, err
	}
	if clean {
		return false, nil
	}
	if err := r.AddAll(); err != nil {
		return false, err
	}
	if !r.HasCommits() {
		_, err := r.run("commit", "-m", message)
		return err == nil, err
	}
	remoteRef := "refs/remotes/origin/" + branch
	if _, err := r.run("rev-parse", "--verify", "--quiet", remoteRef); err == nil {
		// HEAD at or behind origin means there are no unpublished snapshot
		// commits to rewrite; create a normal child commit.
		if _, err := r.run("merge-base", "--is-ancestor", "HEAD", remoteRef); err == nil {
			_, err := r.run("commit", "-m", message)
			return err == nil, err
		}
		// Squash every unpublished snapshot on top of the last published state.
		if _, err := r.run("reset", "--soft", remoteRef); err != nil {
			return false, err
		}
		_, err := r.run("commit", "-m", message)
		return err == nil, err
	}

	// An empty/unpublished remote has no base ref. Build a new root commit from
	// the staged snapshot, making all previously rejected objects unreachable.
	tree, err := r.run("write-tree")
	if err != nil {
		return false, err
	}
	commit, err := r.runInput([]byte(message+"\n"), "commit-tree", tree, "-F", "-")
	if err != nil {
		return false, err
	}
	if _, err := r.run("reset", "--soft", commit); err != nil {
		return false, err
	}
	return true, nil
}

// Pull fetches and rebases from origin on the given branch.
func (r Repo) Pull(branch string) error {
	if !r.HasRemote() {
		return nil
	}
	_, err := r.run("pull", "--rebase", "--autostash", "origin", branch)
	return err
}

// PullProgress pulls while streaming Git's native transfer progress.
func (r Repo) PullProgress(branch string, out io.Writer) error {
	if !r.HasRemote() {
		return nil
	}
	return r.runStreaming(out, "pull", "--progress", "--rebase", "--autostash", "origin", branch)
}

// Push pushes the branch to origin, setting upstream on first push.
func (r Repo) Push(branch string) error {
	if !r.HasRemote() {
		return fmt.Errorf("no git remote configured")
	}
	_, err := r.run("push", "-u", "origin", branch)
	return err
}

// PushProgress pushes while streaming Git's native transfer progress.
func (r Repo) PushProgress(branch string, out io.Writer) error {
	if !r.HasRemote() {
		return fmt.Errorf("no git remote configured")
	}
	return r.runStreaming(out, "push", "--progress", "-u", "origin", branch)
}

// Status returns short porcelain status lines.
func (r Repo) Status() (string, error) {
	return r.run("status", "--short")
}

// Package gitsync snapshots the state of the user's git working repositories
// (remote, branches, current branch, uncommitted changes, untracked files)
// into the sync repo, and restores them on another machine. It captures enough
// to resume work: on restore it clones missing repos, recreates local branches,
// checks out the previous branch, and reapplies uncommitted/untracked work.
package gitsync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"contix/internal/archive"
	"contix/internal/gitutil"
)

// SubDir is the directory inside the sync repo that holds git repo snapshots.
const SubDir = "git"

const (
	stateName     = "state.json"
	patchName     = "changes.patch"
	untrackedName = "untracked.tar.gz"
)

// State is the persisted snapshot of one working repository.
type State struct {
	Key           string           `json:"key"`            // storage key (stable per source path)
	Name          string           `json:"name"`           // repo basename, for display
	RelPath       string           `json:"rel_path"`       // path relative to source home (portable)
	SourcePath    string           `json:"source_path"`    // absolute path at snapshot time
	SourceHome    string           `json:"source_home"`    // home dir at snapshot time
	SourceOS      string           `json:"source_os"`      // linux | darwin | windows
	Remote        string           `json:"remote"`         // origin URL ("" if none)
	CurrentBranch string           `json:"current_branch"` // branch at snapshot time
	HeadSHA       string           `json:"head_sha"`       // HEAD commit
	Branches      []gitutil.Branch `json:"branches"`       // local branches
	HasPatch      bool             `json:"has_patch"`      // uncommitted tracked changes captured
	Untracked     []string         `json:"untracked"`      // untracked files captured
	CreatedAt     time.Time        `json:"created_at"`     //
}

// SnapshotResult summarises what was captured for one repo.
type SnapshotResult struct {
	State     State
	PatchSize int64
	Untracked int
	Skipped   string
}

// RestoreResult summarises what was applied for one repo.
type RestoreResult struct {
	Name            string
	Path            string
	Cloned          bool
	BranchesCreated int
	PatchApplied    bool
	UntrackedFiles  int
	Warnings        []string
	Skipped         string
}

// Key derives a stable storage key for a source path: "<basename>-<hash8>".
func Key(sourcePath string) string {
	base := slug(filepath.Base(sourcePath))
	if base == "" {
		base = "repo"
	}
	sum := sha256.Sum256([]byte(filepath.ToSlash(sourcePath)))
	return base + "-" + hex.EncodeToString(sum[:])[:8]
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32) // lowercase
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// Snapshot captures repoPath's working state into repoRoot/git/<key>/.
// homeDir is used to compute a portable relative path.
func Snapshot(repoRoot, homeDir, repoPath string) (SnapshotResult, error) {
	var res SnapshotResult
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return res, err
	}
	r := gitutil.Repo{Dir: abs}
	if !r.IsRepo() {
		res.Skipped = "not a git repository"
		return res, nil
	}
	if top, err := r.Toplevel(); err == nil && top != "" {
		abs = top
		r = gitutil.Repo{Dir: abs}
	}

	st := State{
		Name:       filepath.Base(abs),
		SourcePath: abs,
		SourceHome: homeDir,
		SourceOS:   runtime.GOOS,
		CreatedAt:  time.Now().UTC(),
	}
	st.Key = Key(abs)
	if rel, err := filepath.Rel(homeDir, abs); err == nil && !strings.HasPrefix(rel, "..") {
		st.RelPath = filepath.ToSlash(rel)
	}
	st.Remote, _ = r.RemoteURL("origin")
	st.CurrentBranch, _ = r.CurrentBranch()
	st.HeadSHA, _ = r.HeadSHA()
	st.Branches, _ = r.LocalBranches()

	destDir := filepath.Join(repoRoot, SubDir, st.Key)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return res, err
	}

	// Uncommitted tracked changes.
	patch, err := r.DiffPatch()
	if err != nil {
		return res, fmt.Errorf("%s: diff: %w", st.Name, err)
	}
	patchPath := filepath.Join(destDir, patchName)
	if len(strings.TrimSpace(string(patch))) > 0 {
		if err := os.WriteFile(patchPath, patch, 0o644); err != nil {
			return res, err
		}
		st.HasPatch = true
		res.PatchSize = int64(len(patch))
	} else {
		_ = os.Remove(patchPath) // clear a stale patch from a previous snapshot
	}

	// Untracked (non-ignored) files.
	untracked, err := r.Untracked()
	if err != nil {
		return res, fmt.Errorf("%s: untracked: %w", st.Name, err)
	}
	untrackedPath := filepath.Join(destDir, untrackedName)
	if len(untracked) > 0 {
		m := archive.NewManifest("untracked", "", abs)
		m, err = archive.Create(abs, untracked, untrackedPath, m)
		if err != nil {
			return res, fmt.Errorf("%s: bundle untracked: %w", st.Name, err)
		}
		for _, fe := range m.Files {
			st.Untracked = append(st.Untracked, fe.Path)
		}
		res.Untracked = len(st.Untracked)
	} else {
		_ = os.Remove(untrackedPath)
	}

	if err := writeState(filepath.Join(destDir, stateName), st); err != nil {
		return res, err
	}
	res.State = st
	return res, nil
}

// List returns all repo snapshots stored under repoRoot/git.
func List(repoRoot string) ([]State, error) {
	base := filepath.Join(repoRoot, SubDir)
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var states []State
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		st, err := readState(filepath.Join(base, e.Name(), stateName))
		if err != nil {
			continue
		}
		states = append(states, st)
	}
	return states, nil
}

// Restore recreates one repo snapshot onto this machine. destHome is the
// current machine's home dir, used to relocate repos stored by relative path.
func Restore(repoRoot, destHome string, st State) (RestoreResult, error) {
	res := RestoreResult{Name: st.Name}
	destDir := filepath.Join(repoRoot, SubDir, st.Key)

	// Resolve target path: prefer the portable relative path under the new home.
	target := st.SourcePath
	if st.RelPath != "" {
		target = filepath.Join(destHome, filepath.FromSlash(st.RelPath))
	}
	res.Path = target

	r := gitutil.Repo{Dir: target}
	if !r.IsRepo() {
		if st.Remote == "" {
			res.Skipped = "repo missing locally and no remote recorded"
			return res, nil
		}
		if _, err := gitutil.Clone(st.Remote, target, ""); err != nil {
			return res, fmt.Errorf("clone %s: %w", st.Remote, err)
		}
		res.Cloned = true
		r = gitutil.Repo{Dir: target}
	}

	// Best-effort fetch so recorded commits/branches become reachable.
	if r.HasRemote() {
		if err := r.Fetch("origin"); err != nil {
			res.Warnings = append(res.Warnings, "fetch: "+err.Error())
		}
	}

	// Recreate local branches.
	for _, b := range st.Branches {
		if r.BranchExists(b.Name) {
			continue
		}
		switch {
		case b.SHA != "" && r.CommitExists(b.SHA):
			if err := r.CreateBranch(b.Name, b.SHA); err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("branch %s: %v", b.Name, err))
			} else {
				res.BranchesCreated++
			}
		case b.Upstream != "" && r.CommitExists(b.Upstream):
			if err := r.CreateBranch(b.Name, b.Upstream); err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("branch %s: %v", b.Name, err))
			} else {
				res.BranchesCreated++
			}
		default:
			res.Warnings = append(res.Warnings, fmt.Sprintf("branch %s: commit %s not available; skipped", b.Name, short(b.SHA)))
		}
	}

	// Restore the previously checked-out branch.
	if st.CurrentBranch != "" && st.CurrentBranch != "HEAD" {
		if err := r.Checkout(st.CurrentBranch); err != nil {
			res.Warnings = append(res.Warnings, "checkout "+st.CurrentBranch+": "+err.Error())
		}
	}

	// Reapply untracked files first, then the tracked-change patch.
	if len(st.Untracked) > 0 {
		untrackedPath := filepath.Join(destDir, untrackedName)
		if _, err := os.Stat(untrackedPath); err == nil {
			files, err := archive.Extract(untrackedPath, target)
			if err != nil {
				res.Warnings = append(res.Warnings, "untracked: "+err.Error())
			} else {
				res.UntrackedFiles = len(files)
			}
		}
	}
	if st.HasPatch {
		patch, err := os.ReadFile(filepath.Join(destDir, patchName))
		if err != nil {
			res.Warnings = append(res.Warnings, "read patch: "+err.Error())
		} else if err := r.ApplyPatch(patch); err != nil {
			res.Warnings = append(res.Warnings, "apply patch: "+err.Error()+" (saved as changes.patch in the sync repo)")
		} else {
			res.PatchApplied = true
		}
	}
	return res, nil
}

func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

func writeState(path string, st State) error {
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readState(path string) (State, error) {
	var st State
	b, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	return st, json.Unmarshal(b, &st)
}

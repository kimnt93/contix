// Package cli implements contix's command-line interface using only the Go
// standard library, so the tool ships as a single dependency-free binary.
package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"contix/internal/config"
	"contix/internal/gitsync"
	"contix/internal/gitutil"
	"contix/internal/platform"
	"contix/internal/tool"
)

// Version is the contix release, overridable at build time with
// -ldflags "-X contix/internal/cli.Version=x.y.z".
var Version = "0.1.0"

const usage = `contix — sync your Codex, Claude Code and git working state to one GitHub repo.

USAGE
  contix <command> [flags]

COMMANDS
  init      Configure the sync repo (clones the remote on a new machine)
  status    Show configuration and what would be synced
  push      Collect state + repo snapshots, commit, and push to the remote
  pull      Pull from the remote and restore state + repos onto this machine
  list      List what is currently stored in the sync repo
  verify    Check archive integrity of everything in the sync repo
  repos     Manage tracked git working repositories (add/remove/list)
  doctor    Diagnose the environment and configuration
  version   Print the contix version
  help      Show this help

Run "contix <command> -h" for command-specific flags.

QUICK START
  # First machine — the remote may be SSH or HTTPS:
  contix init --remote git@github.com:you/dev-state.git
  #   or: contix init --remote https://github.com/you/dev-state.git
  contix repos add ~/code/project-a ~/code/project-b
  contix push

  # New machine
  contix init --remote https://github.com/you/dev-state.git
  contix pull
`

// Run dispatches a command and returns a process exit code.
func Run(args []string) int {
	if len(args) == 0 {
		fmt.Print(usage)
		return 0
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "init":
		return cmdInit(rest)
	case "status":
		return cmdStatus(rest)
	case "push":
		return cmdPush(rest)
	case "pull":
		return cmdPull(rest)
	case "list":
		return cmdList(rest)
	case "verify":
		return cmdVerify(rest)
	case "repos":
		return cmdRepos(rest)
	case "doctor":
		return cmdDoctor(rest)
	case "version", "--version", "-v":
		fmt.Printf("contix %s\n", Version)
		return 0
	case "help", "-h", "--help":
		fmt.Print(usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "contix: unknown command %q\n\n", cmd)
		fmt.Print(usage)
		return 2
	}
}

// fail prints an error to stderr and returns exit code 1.
func fail(err error) int {
	fmt.Fprintln(os.Stderr, "error:", err)
	return 1
}

// mustConfig loads the config or prints a helpful error.
func mustConfig() (config.Config, bool) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return config.Config{}, false
	}
	return cfg, true
}

func cmdInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	repo := fs.String("repo", "", "local sync repo path (default: config dir/repo)")
	remote := fs.String("remote", "", "git remote URL of the sync repo")
	branch := fs.String("branch", "main", "git branch to sync on")
	autoPush := fs.Bool("auto-push", false, "push to the remote automatically after each 'push'")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !gitutil.Available() {
		return fail(fmt.Errorf("git is not installed or not on PATH"))
	}

	cfg := config.Default()
	if config.Exists() {
		if existing, err := config.Load(); err == nil {
			cfg = existing
		}
	}
	if *repo != "" {
		abs, err := filepath.Abs(*repo)
		if err != nil {
			return fail(err)
		}
		cfg.RepoPath = abs
	}
	if *remote != "" {
		if err := gitutil.ValidateRemote(*remote); err != nil {
			return fail(err)
		}
		cfg.Remote = *remote
	}
	if *branch != "" {
		cfg.Branch = *branch
	}
	cfg.AutoPush = *autoPush
	cfg.Home = platform.Home()

	if err := ensureRepo(cfg); err != nil {
		return fail(err)
	}
	if err := cfg.Save(); err != nil {
		return fail(err)
	}

	fmt.Printf("Configured contix.\n")
	fmt.Printf("  sync repo : %s\n", cfg.RepoPath)
	fmt.Printf("  remote    : %s\n", orNone(cfg.Remote))
	fmt.Printf("  branch    : %s\n", cfg.Branch)
	fmt.Printf("  config    : %s\n\n", config.Path())
	if cfg.Remote == "" {
		fmt.Println("No remote set. Add one later with: contix init --remote <url>")
	} else if gitutil.ClassifyRemote(cfg.Remote) == gitutil.RemoteHTTP {
		fmt.Println("Using an HTTPS remote: pushing needs a git credential helper or a")
		fmt.Println("Personal Access Token. SSH remotes (git@github.com:…) use your SSH key.")
	}
	fmt.Println("Next: 'contix repos add <path>' to track projects, then 'contix push'.")
	fmt.Println("On a new machine after init: 'contix pull' to restore everything.")
	return 0
}

// ensureRepo makes cfg.RepoPath a git repo, cloning the remote when possible so
// a freshly-initialised machine starts from existing synced state.
func ensureRepo(cfg config.Config) error {
	r := gitutil.Repo{Dir: cfg.RepoPath}
	if r.IsRepo() {
		if cfg.Remote != "" {
			if err := r.SetRemote(cfg.Remote); err != nil {
				return err
			}
		}
		return ensureBranch(r, cfg.Branch)
	}
	if cfg.Remote != "" {
		// Try to clone existing state (branch first, then default branch).
		if _, err := gitutil.Clone(cfg.Remote, cfg.RepoPath, cfg.Branch); err == nil {
			return ensureBranch(gitutil.Repo{Dir: cfg.RepoPath}, cfg.Branch)
		}
		if _, err := gitutil.Clone(cfg.Remote, cfg.RepoPath, ""); err == nil {
			return ensureBranch(gitutil.Repo{Dir: cfg.RepoPath}, cfg.Branch)
		}
	}
	// Fresh, empty sync repo.
	if _, err := gitutil.Init(cfg.RepoPath, cfg.Branch); err != nil {
		return err
	}
	nr := gitutil.Repo{Dir: cfg.RepoPath}
	if cfg.Remote != "" {
		if err := nr.SetRemote(cfg.Remote); err != nil {
			return err
		}
	}
	writeRepoReadme(cfg.RepoPath)
	return ensureBranch(nr, cfg.Branch)
}

// ensureBranch aligns the sync repo's branch with the configured one, but only
// when there are no commits yet, so an existing populated clone is left intact.
func ensureBranch(r gitutil.Repo, branch string) error {
	if r.HasCommits() {
		return nil
	}
	return r.EnsureBranch(branch)
}

func writeRepoReadme(dir string) {
	readme := filepath.Join(dir, "README.md")
	if _, err := os.Stat(readme); err == nil {
		return
	}
	_ = os.WriteFile(readme, []byte(
		"# contix sync repo\n\n"+
			"This repository is managed by [contix](https://github.com/). It stores the\n"+
			"latest snapshot of Codex / Claude Code state and tracked git working repos.\n\n"+
			"Do not edit by hand. Use `contix push` and `contix pull`.\n"), 0o644)
}

func cmdStatus(args []string) int {
	cfg, ok := mustConfig()
	if !ok {
		return 1
	}
	fmt.Printf("contix %s\n\n", Version)
	fmt.Printf("sync repo : %s\n", cfg.RepoPath)
	fmt.Printf("remote    : %s\n", orNone(cfg.Remote))
	fmt.Printf("branch    : %s\n", cfg.Branch)
	fmt.Printf("auto-push : %v\n\n", cfg.AutoPush)

	fmt.Println("AI tools:")
	names := tool.Names()
	for _, name := range names {
		t, _ := tool.Lookup(name)
		home := t.Home()
		if fi, err := os.Stat(home); err != nil || !fi.IsDir() {
			fmt.Printf("  %-8s not found (%s)\n", name, home)
			continue
		}
		files, _ := t.IncludedFiles()
		fmt.Printf("  %-8s %d files to sync (%s)\n", name, len(files), home)
	}

	fmt.Println("\nTracked git repos:")
	if len(cfg.Repos) == 0 {
		fmt.Println("  (none — add with 'contix repos add <path>')")
	}
	for _, p := range cfg.Repos {
		r := gitutil.Repo{Dir: p}
		if !r.IsRepo() {
			fmt.Printf("  %s  (missing / not a repo)\n", p)
			continue
		}
		branch, _ := r.CurrentBranch()
		clean, _ := r.IsClean()
		state := "clean"
		if !clean {
			state = "uncommitted changes"
		}
		fmt.Printf("  %s  [%s, %s]\n", p, branch, state)
	}

	// Sync repo git status.
	r := gitutil.Repo{Dir: cfg.RepoPath}
	if r.IsRepo() {
		if s, _ := r.Status(); strings.TrimSpace(s) != "" {
			fmt.Println("\nSync repo has uncommitted changes (run 'contix push').")
		} else {
			fmt.Println("\nSync repo is clean.")
		}
	}
	return 0
}

func cmdDoctor(args []string) int {
	fmt.Printf("contix %s\n\n", Version)
	ok := true

	// git
	if gitutil.Available() {
		fmt.Println("[ok]   git is installed")
	} else {
		fmt.Println("[FAIL] git not found on PATH")
		ok = false
	}

	// config
	if config.Exists() {
		fmt.Printf("[ok]   config present (%s)\n", config.Path())
	} else {
		fmt.Println("[warn] not configured — run 'contix init'")
	}

	cfg, err := config.Load()
	if err == nil {
		r := gitutil.Repo{Dir: cfg.RepoPath}
		if r.IsRepo() {
			fmt.Printf("[ok]   sync repo initialised (%s)\n", cfg.RepoPath)
		} else {
			fmt.Printf("[warn] sync repo not initialised (%s)\n", cfg.RepoPath)
		}
		if cfg.Remote != "" {
			fmt.Printf("[ok]   remote configured (%s)\n", cfg.Remote)
		} else {
			fmt.Println("[warn] no git remote configured")
		}
	}

	// tools
	for _, name := range tool.Names() {
		t, _ := tool.Lookup(name)
		home := t.Home()
		if fi, err := os.Stat(home); err == nil && fi.IsDir() {
			fmt.Printf("[ok]   %s state found (%s)\n", name, home)
		} else {
			fmt.Printf("[warn] %s state not found (%s)\n", name, home)
		}
	}

	if ok {
		return 0
	}
	return 1
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// parseTools resolves a comma-separated tool list into Tool values. An empty
// selection means "all known tools".
func parseTools(csv string) ([]tool.Tool, error) {
	if strings.TrimSpace(csv) == "" {
		var all []tool.Tool
		for _, n := range tool.Names() {
			t, _ := tool.Lookup(n)
			all = append(all, t)
		}
		return all, nil
	}
	var out []tool.Tool
	for _, name := range strings.Split(csv, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		t, ok := tool.Lookup(name)
		if !ok {
			return nil, fmt.Errorf("unknown tool %q (known: %s)", name, strings.Join(tool.Names(), ", "))
		}
		out = append(out, t)
	}
	return out, nil
}

// listStates returns tracked git snapshots sorted by name for stable output.
func listStates(repoRoot string) ([]gitsync.State, error) {
	states, err := gitsync.List(repoRoot)
	if err != nil {
		return nil, err
	}
	sort.Slice(states, func(i, j int) bool { return states[i].Name < states[j].Name })
	return states, nil
}

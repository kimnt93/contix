// Package cli implements contix's command-line interface using only the Go
// standard library, so the tool ships as a single dependency-free binary.
package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"contix/internal/config"
	"contix/internal/gitutil"
	"contix/internal/platform"
	"contix/internal/tool"
	"contix/release"
)

// Version is the contix release, overridable at build time with
// -ldflags "-X contix/internal/cli.Version=x.y.z".
var Version = releaseinfo.Version()

const usage = `contix — sync AI coding-agent state to one GitHub repo.

USAGE
  contix <command> [flags]

COMMANDS
  init      Configure the sync repo (clones the remote on a new machine)
  collect   Collect available state and commit it locally
  push      Upload the collected state to the remote
  pull      Pull from the remote and restore available state

Run "contix <command> -h" for command-specific flags.

QUICK START
  # First machine — the remote may be SSH or HTTPS:
  contix init --remote git@github.com:you/dev-state.git
  #   or: contix init --remote https://github.com/you/dev-state.git
  contix collect
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
	case "collect":
		return cmdCollect(rest)
	case "push":
		return cmdPush(rest)
	case "pull":
		return cmdPull(rest)
	case "--version", "-v":
		fmt.Printf("contix %s\n", Version)
		if notes := releaseinfo.Notes(); notes != "" {
			fmt.Println("features:")
			for _, line := range strings.Split(notes, "\n") {
				fmt.Printf("  %s\n", line)
			}
		}
		return 0
	case "-h", "--help":
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
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
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
	fmt.Println("Next: run 'contix collect', then 'contix push'.")
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
			"latest snapshot of supported AI coding-agent state.\n\n"+
			"Do not edit by hand. Use `contix collect`, `contix push` and `contix pull`.\n"), 0o644)
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// parseTools resolves a comma-separated target list into Tool values. An empty
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
	seen := make(map[string]bool)
	for _, name := range strings.Split(csv, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		names := []string{name}
		if group, ok := tool.Group(name); ok {
			names = group
		}
		for _, targetName := range names {
			if seen[targetName] {
				continue
			}
			t, ok := tool.Lookup(targetName)
			if !ok {
				return nil, fmt.Errorf("unknown target %q (known: %s)", name, strings.Join(tool.Names(), ", "))
			}
			seen[targetName] = true
			out = append(out, t)
		}
	}
	return out, nil
}

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"contix/internal/config"
	"contix/internal/gitutil"
)

const reposUsage = `contix repos — manage tracked git working repositories.

USAGE
  contix repos add <path>...    Track one or more git repositories
  contix repos remove <path>    Stop tracking a repository
  contix repos list             Show tracked repositories
  contix repos scan [root]...   Find and track repositories now
`

func cmdRepos(args []string) int {
	if len(args) == 0 {
		fmt.Print(reposUsage)
		return 0
	}
	sub, rest := args[0], args[1:]
	cfg, ok := mustConfig()
	if !ok {
		return 1
	}
	switch sub {
	case "add":
		return reposAdd(cfg, rest)
	case "remove", "rm":
		return reposRemove(cfg, rest)
	case "list", "ls":
		return reposList(cfg)
	case "scan":
		return reposScan(cfg, rest)
	case "-h", "--help", "help":
		fmt.Print(reposUsage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "contix repos: unknown subcommand %q\n\n", sub)
		fmt.Print(reposUsage)
		return 2
	}
}

func reposScan(cfg config.Config, roots []string) int {
	added, err := discoverAndAdd(&cfg, roots)
	if err != nil {
		return fail(err)
	}
	if len(added) == 0 {
		fmt.Println("No new git repositories found.")
		return 0
	}
	if err := cfg.Save(); err != nil {
		return fail(err)
	}
	for _, p := range added {
		fmt.Println("tracking", p)
	}
	fmt.Printf("Added %d repositories.\n", len(added))
	return 0
}

func reposAdd(cfg config.Config, paths []string) int {
	if len(paths) == 0 {
		return fail(fmt.Errorf("usage: contix repos add <path>..."))
	}
	added := 0
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", p, err)
			continue
		}
		r := gitutil.Repo{Dir: abs}
		if !r.IsRepo() {
			fmt.Fprintf(os.Stderr, "skip %s: not a git repository\n", abs)
			continue
		}
		if top, err := r.Toplevel(); err == nil && top != "" {
			abs = top
		}
		if cfg.AddRepo(abs) {
			fmt.Printf("tracking %s\n", abs)
			if url, err := r.RemoteURL("origin"); err != nil || url == "" {
				fmt.Fprintf(os.Stderr, "  note: %s has no 'origin' remote; its commit history\n"+
					"        cannot be restored on another machine (only uncommitted and\n"+
					"        untracked files will be synced).\n", filepath.Base(abs))
			}
			added++
		} else {
			fmt.Printf("already tracked: %s\n", abs)
		}
	}
	if added > 0 {
		if err := cfg.Save(); err != nil {
			return fail(err)
		}
	}
	return 0
}

func reposRemove(cfg config.Config, paths []string) int {
	if len(paths) == 0 {
		return fail(fmt.Errorf("usage: contix repos remove <path>"))
	}
	changed := false
	for _, p := range paths {
		abs, _ := filepath.Abs(p)
		if cfg.RemoveRepo(abs) || cfg.RemoveRepo(p) {
			fmt.Printf("untracked %s\n", abs)
			changed = true
		} else {
			fmt.Printf("not tracked: %s\n", abs)
		}
	}
	if changed {
		if err := cfg.Save(); err != nil {
			return fail(err)
		}
	}
	return 0
}

func reposList(cfg config.Config) int {
	if len(cfg.Repos) == 0 {
		fmt.Println("No repositories tracked. Add one with 'contix repos add <path>'.")
		return 0
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
	return 0
}

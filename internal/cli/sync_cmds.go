package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"contix/internal/gitutil"
	"contix/internal/syncer"
)

func cmdCollect(args []string) int {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	tools := fs.String("tools", "", "comma-separated tools to collect (default: all)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		return fail(fmt.Errorf("collect does not accept positional arguments"))
	}
	cfg, ok := mustConfig()
	if !ok {
		return 1
	}
	if err := ensureRepo(cfg); err != nil {
		return fail(err)
	}
	legacyGit := filepath.Join(cfg.RepoPath, "git")
	if _, err := os.Stat(legacyGit); err == nil {
		if err := os.RemoveAll(legacyGit); err != nil {
			return fail(fmt.Errorf("remove legacy git snapshots: %w", err))
		}
		fmt.Println("Removed legacy git working-repository snapshots.")
	}

	tls, err := parseTools(*tools)
	if err != nil {
		return fail(err)
	}

	fmt.Println("Collecting AI tool state:")
	for _, t := range tls {
		res, err := syncer.Push(cfg, t)
		if err != nil {
			return fail(fmt.Errorf("push %s: %w", t.Name, err))
		}
		if res.Skipped != "" {
			fmt.Printf("  %-8s skipped (%s)\n", t.Name, res.Skipped)
			continue
		}
		parts := ""
		if res.Parts > 1 {
			parts = fmt.Sprintf(", %d compressed parts", res.Parts)
		}
		fmt.Printf("  %-8s %d files, %s%s%s\n", t.Name, res.Files, humanBytes(res.Bytes), versionSuffix(res.Version), parts)
	}

	// Commit the collected snapshot locally.
	r := gitutil.Repo{Dir: cfg.RepoPath}
	host, _ := os.Hostname()
	msg := fmt.Sprintf("contix sync %s from %s", time.Now().Format("2006-01-02 15:04"), host)
	committed, err := r.CommitSnapshot(cfg.Branch, msg)
	if err != nil {
		return fail(err)
	}
	if committed {
		fmt.Printf("\nCommitted: %s\n", msg)
	} else {
		fmt.Println("\nNothing changed since last collection.")
	}

	if cfg.Remote != "" {
		fmt.Println("Run 'contix push' to upload the collected state.")
	}
	return 0
}

func cmdPush(args []string) int {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		return fail(fmt.Errorf("push does not accept positional arguments"))
	}
	cfg, ok := mustConfig()
	if !ok {
		return 1
	}
	if cfg.Remote == "" {
		return fail(fmt.Errorf("no git remote configured; run 'contix init --remote <url>'"))
	}
	r := gitutil.Repo{Dir: cfg.RepoPath}
	if !r.IsRepo() {
		return fail(fmt.Errorf("sync repo not initialised; run 'contix init' first"))
	}
	clean, err := r.IsClean()
	if err != nil {
		return fail(err)
	}
	if !clean {
		return fail(fmt.Errorf("sync repo has uncommitted changes; run 'contix collect' first"))
	}
	if r.RemoteHasBranch(cfg.Branch) {
		fmt.Println("Updating from remote before push...")
		if err := r.Pull(cfg.Branch); err != nil {
			return fail(err)
		}
	}
	if err := r.Push(cfg.Branch); err != nil {
		return fail(err)
	}
	fmt.Println("Pushed collected state to", cfg.Remote)
	return 0
}

func cmdPull(args []string) int {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		return fail(fmt.Errorf("pull does not accept positional arguments"))
	}
	cfg, ok := mustConfig()
	if !ok {
		return 1
	}

	// Pull latest from the remote first.
	r := gitutil.Repo{Dir: cfg.RepoPath}
	if !r.IsRepo() {
		return fail(fmt.Errorf("sync repo not initialised; run 'contix init' first"))
	}
	if cfg.Remote != "" {
		fmt.Println("Pulling latest from remote...")
		if err := r.Pull(cfg.Branch); err != nil {
			return fail(err)
		}
	}

	tls, err := parseTools("")
	if err != nil {
		return fail(err)
	}

	fmt.Println("\nRestoring AI tool state:")
	for _, t := range tls {
		res, err := syncer.Pull(cfg, t, nil, true)
		if err != nil {
			return fail(fmt.Errorf("pull %s: %w", t.Name, err))
		}
		if res.Skipped != "" {
			fmt.Printf("  %-8s skipped (%s)\n", t.Name, res.Skipped)
			continue
		}
		line := fmt.Sprintf("  %-8s %d files restored", t.Name, res.Files)
		if res.FilesRewrite > 0 || res.DirsRenamed > 0 {
			line += fmt.Sprintf(", %d rewritten, %d dirs renamed", res.FilesRewrite, res.DirsRenamed)
		}
		fmt.Println(line)
		if !res.VersionOK {
			fmt.Printf("           version mismatch: synced %s, local %s — update the tool to match\n",
				orUnknown(res.SourceVersion), orUnknown(res.LocalVersion))
		}
		if len(res.Mismatches) > 0 {
			fmt.Printf("           %d integrity warning(s):\n", len(res.Mismatches))
			for _, m := range res.Mismatches {
				fmt.Printf("             - %s\n", m)
			}
		}
	}

	fmt.Println("\nDone. Your AI agents should resume where you left off.")
	return 0
}

func versionSuffix(v string) string {
	if v == "" {
		return ""
	}
	return " (v" + v + ")"
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"contix/internal/gitutil"
	"contix/internal/procstop"
	"contix/internal/syncer"
	"contix/internal/tool"
)

func cmdCollect(args []string) int {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	tools := fs.String("tools", "", "comma-separated targets to collect (default: all)")
	forceClose := fs.Bool("force-close", false, "close selected tools before collecting their state")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
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
	tls, err := parseTools(*tools)
	if err != nil {
		return fail(err)
	}
	legacyGit := filepath.Join(cfg.RepoPath, "git")
	if _, err := os.Stat(legacyGit); err == nil {
		if err := os.RemoveAll(legacyGit); err != nil {
			return fail(fmt.Errorf("remove legacy git snapshots: %w", err))
		}
		fmt.Println("Removed legacy git working-repository snapshots.")
	}
	removedIDE, err := removeRetiredSnapshots(cfg.RepoPath)
	if err != nil {
		return fail(err)
	}
	if removedIDE > 0 {
		fmt.Printf("Removed %d retired IDE snapshot(s) from the sync repo.\n", removedIDE)
	}

	if *forceClose {
		var processes []string
		for _, t := range tls {
			processes = append(processes, t.Processes...)
		}
		fmt.Println("Closing selected applications before collection...")
		stopped, err := procstop.Close(processes)
		if err != nil {
			return fail(err)
		}
		if len(stopped) == 0 {
			fmt.Println("  no matching processes were running")
		} else {
			for _, name := range stopped {
				fmt.Printf("  stopped %s\n", name)
			}
		}
	}

	fmt.Println("Collecting state:")
	for _, t := range tls {
		stop := startActivity(fmt.Sprintf("  %-24s compressing", t.Name))
		res, err := syncer.Push(cfg, t)
		stop()
		if err != nil {
			return fail(fmt.Errorf("collect %s: %w", t.Name, err))
		}
		if res.Skipped != "" {
			fmt.Printf("  %-24s skipped (%s)\n", t.Name, res.Skipped)
			continue
		}
		parts := ""
		if res.Parts > 1 {
			parts = fmt.Sprintf(", %d compressed parts", res.Parts)
		}
		fmt.Printf("  %-24s done: %d files, %s%s%s\n", t.Name, res.Files, humanBytes(res.Bytes), versionSuffix(res.Version), parts)
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

func removeRetiredSnapshots(repoPath string) (int, error) {
	removed := 0
	for _, name := range tool.RetiredNames() {
		retired := filepath.Join(repoPath, name)
		if _, err := os.Lstat(retired); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return removed, fmt.Errorf("inspect retired IDE snapshot %s: %w", name, err)
		}
		if err := os.RemoveAll(retired); err != nil {
			return removed, fmt.Errorf("remove retired IDE snapshot %s: %w", name, err)
		}
		removed++
	}
	return removed, nil
}

func cmdPush(args []string) int {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
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
	fmt.Println("Preparing collected state for upload...")
	clean, err := r.IsClean()
	if err != nil {
		return fail(err)
	}
	if !clean {
		return fail(fmt.Errorf("sync repo has uncommitted changes; run 'contix collect' first"))
	}
	fmt.Println("Checking remote branch...")
	if r.RemoteHasBranch(cfg.Branch) {
		fmt.Println("Updating from remote before push...")
		if err := r.PullProgress(cfg.Branch, os.Stdout); err != nil {
			return fail(err)
		}
	}
	fmt.Println("Pushing collected state...")
	if err := r.PushProgress(cfg.Branch, os.Stdout); err != nil {
		return fail(err)
	}
	fmt.Println("Pushed collected state to", cfg.Remote)
	return 0
}

func cmdPull(args []string) int {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	ignore := fs.Bool("ignore", false, "ignore local conflicts and overwrite with synced state")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
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
		if err := r.PullProgress(cfg.Branch, os.Stdout); err != nil {
			return fail(err)
		}
	}

	tls, err := parseTools("")
	if err != nil {
		return fail(err)
	}

	fmt.Println("\nRestoring state:")
	hadConflicts := false
	for _, t := range tls {
		stop := startActivity(fmt.Sprintf("  %-24s restoring and verifying", t.Name))
		res, err := syncer.Pull(cfg, t, nil, true, *ignore)
		stop()
		if err != nil {
			return fail(fmt.Errorf("pull %s: %w", t.Name, err))
		}
		if res.Skipped != "" {
			fmt.Printf("  %-24s skipped (%s)\n", t.Name, res.Skipped)
			continue
		}
		if len(res.Conflicts) > 0 {
			hadConflicts = true
			fmt.Printf("  %-24s conflict: %d local file(s) differ; target kept\n", t.Name, len(res.Conflicts))
			for _, path := range res.Conflicts {
				fmt.Printf("               - %s\n", path)
			}
			continue
		}
		action := "restored"
		if res.DeferredPath != "" {
			action = "staged"
		}
		line := fmt.Sprintf("  %-24s done: %d files %s", t.Name, res.Files, action)
		if res.FilesRewrite > 0 || res.DirsRenamed > 0 {
			line += fmt.Sprintf(", %d rewritten, %d dirs renamed", res.FilesRewrite, res.DirsRenamed)
		}
		fmt.Println(line)
		if res.DeferredPath != "" {
			fmt.Println("               destination needs permission; local file kept")
			fmt.Printf("               synced copy: %s\n", res.DeferredPath)
			fmt.Printf("               destination: %s\n", res.Destination)
		}
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

	if hadConflicts {
		fmt.Println("\nConflicting targets were not overwritten.")
		fmt.Println("Review the files above, or run 'contix pull --ignore' to overwrite them.")
		return 1
	}
	fmt.Println("\nDone. Available agent and machine state is ready.")
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

package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"contix/internal/archive"
	"contix/internal/config"
	"contix/internal/gitutil"
	"contix/internal/pathrewrite"
	"contix/internal/syncer"
	"contix/internal/tool"
)

// mapList collects repeatable --map OLD=NEW flags.
type mapList []pathrewrite.Mapping

func (m *mapList) String() string { return fmt.Sprintf("%v", []pathrewrite.Mapping(*m)) }
func (m *mapList) Set(v string) error {
	mp, ok := pathrewrite.ParseMapping(v)
	if !ok {
		return fmt.Errorf("invalid mapping %q (want OLD=NEW)", v)
	}
	*m = append(*m, mp)
	return nil
}

func cmdPush(args []string) int {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	tools := fs.String("tools", "", "comma-separated tools to push (default: all)")
	days := fs.Int("days", 0, "only include session transcripts newer than N days (0 = all)")
	message := fs.String("message", "", "commit message (default: auto)")
	push := fs.Bool("push", false, "push to the git remote after committing")
	if err := fs.Parse(args); err != nil {
		return 2
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
		res, err := syncer.Push(cfg, t, *days)
		if err != nil {
			return fail(fmt.Errorf("push %s: %w", t.Name, err))
		}
		if res.Skipped != "" {
			fmt.Printf("  %-8s skipped (%s)\n", t.Name, res.Skipped)
			continue
		}
		fmt.Printf("  %-8s %d files, %s%s\n", t.Name, res.Files, humanBytes(res.Bytes), versionSuffix(res.Version))
	}

	// Commit + optional push.
	r := gitutil.Repo{Dir: cfg.RepoPath}
	msg := *message
	if msg == "" {
		host, _ := os.Hostname()
		msg = fmt.Sprintf("contix sync %s from %s", time.Now().Format("2006-01-02 15:04"), host)
	}
	committed, err := r.Commit(msg)
	if err != nil {
		return fail(err)
	}
	if committed {
		fmt.Printf("\nCommitted: %s\n", msg)
	} else {
		fmt.Println("\nNothing changed since last push.")
	}

	if (*push || cfg.AutoPush) && cfg.Remote != "" {
		fmt.Println("Pushing to remote...")
		if r.RemoteHasBranch(cfg.Branch) {
			if err := r.Pull(cfg.Branch); err != nil {
				fmt.Fprintln(os.Stderr, "warning: pull before push failed:", err)
			}
		}
		if err := r.Push(cfg.Branch); err != nil {
			return fail(err)
		}
		fmt.Println("Pushed to", cfg.Remote)
	} else if cfg.Remote != "" {
		fmt.Println("Run 'contix push --push' (or set auto-push) to upload to the remote.")
	}
	return 0
}

func cmdPull(args []string) int {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	tools := fs.String("tools", "", "comma-separated tools to restore (default: all)")
	noRewrite := fs.Bool("no-rewrite", false, "do not rewrite machine paths in restored state")
	var maps mapList
	fs.Var(&maps, "map", "extra path mapping OLD=NEW (repeatable)")
	if err := fs.Parse(args); err != nil {
		return 2
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

	tls, err := parseTools(*tools)
	if err != nil {
		return fail(err)
	}

	fmt.Println("\nRestoring AI tool state:")
	for _, t := range tls {
		res, err := syncer.Pull(cfg, t, maps, !*noRewrite)
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

func cmdList(args []string) int {
	cfg, ok := mustConfig()
	if !ok {
		return 1
	}
	fmt.Printf("Sync repo: %s\n\n", cfg.RepoPath)

	fmt.Println("AI tool bundles:")
	found := false
	for _, name := range toolNamesInRepo(cfg) {
		mpath := filepath.Join(cfg.RepoPath, name, archive.ManifestName)
		m, err := archive.ReadManifest(mpath)
		if err != nil {
			continue
		}
		found = true
		var total int64
		for _, fe := range m.Files {
			total += fe.Size
		}
		fmt.Printf("  %-8s %d files, %s, %s%s, from %s\n",
			name, len(m.Files), humanBytes(total),
			m.SourceOS, versionSuffix(m.ToolVersion), m.CreatedAt.Local().Format("2006-01-02 15:04"))
	}
	if !found {
		fmt.Println("  (none)")
	}

	return 0
}

func cmdVerify(args []string) int {
	cfg, ok := mustConfig()
	if !ok {
		return 1
	}
	tmp, err := os.MkdirTemp("", "contix-verify-")
	if err != nil {
		return fail(err)
	}
	defer os.RemoveAll(tmp)

	problems := 0
	fmt.Println("Verifying AI tool bundles:")
	any := false
	for _, name := range toolNamesInRepo(cfg) {
		dir := filepath.Join(cfg.RepoPath, name)
		mpath := filepath.Join(dir, archive.ManifestName)
		bpath := filepath.Join(dir, archive.BundleName)
		m, err := archive.ReadManifest(mpath)
		if err != nil {
			continue
		}
		any = true
		dest := filepath.Join(tmp, name)
		if _, err := archive.Extract(bpath, dest); err != nil {
			fmt.Printf("  %-8s FAIL: extract: %v\n", name, err)
			problems++
			continue
		}
		mism, err := archive.Verify(dest, m)
		if err != nil {
			fmt.Printf("  %-8s FAIL: %v\n", name, err)
			problems++
			continue
		}
		if len(mism) == 0 {
			fmt.Printf("  %-8s ok (%d files)\n", name, len(m.Files))
		} else {
			fmt.Printf("  %-8s %d mismatch(es)\n", name, len(mism))
			problems += len(mism)
		}
	}
	if !any {
		fmt.Println("  (none)")
	}

	if problems == 0 {
		fmt.Println("\nAll bundles verified.")
		return 0
	}
	fmt.Printf("\n%d problem(s) found.\n", problems)
	return 1
}

// toolNamesInRepo returns the tool names that have a directory in the sync repo.
func toolNamesInRepo(cfg config.Config) []string {
	var out []string
	for _, name := range tool.Names() {
		if _, err := os.Stat(filepath.Join(cfg.RepoPath, name, archive.ManifestName)); err == nil {
			out = append(out, name)
		}
	}
	return out
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

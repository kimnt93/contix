# contix — usage & internals

This document is the full reference for `contix`. For a quick overview see the
[README](../README.md).

## Contents

1. [Concepts](#concepts)
2. [Command reference](#command-reference)
3. [Configuration](#configuration)
4. [What gets synced (and skipped)](#what-gets-synced-and-skipped)
5. [How git working repos are synced](#how-git-working-repos-are-synced)
6. [Cross-machine path rewriting](#cross-machine-path-rewriting)
7. [Layout of the sync repo](#layout-of-the-sync-repo)
8. [Typical workflows](#typical-workflows)
9. [Troubleshooting](#troubleshooting)

---

## Concepts

- **Sync repo** — a single git repository you own (usually private on GitHub)
  that stores the *latest* snapshot of everything. `contix` keeps a local clone
  and pushes/pulls it.
- **Tools** — the AI coding agents `contix` understands: `codex` and `claude`.
  Each has a curated include/exclude list so only meaningful state is synced.
- **Tracked repos** — your own git working repositories whose branches and
  uncommitted work you want to carry between machines.
- **Snapshot** — one `contix push`: it rebuilds the bundles, commits them to the
  sync repo, and optionally pushes to the remote.

`contix` never keeps history of its own; each push overwrites the previous
snapshot's bundles. The *git history* of the sync repo still records every push,
so you can inspect or roll back manually if needed.

---

## Command reference

### `contix init`

Configure `contix` and prepare the local sync repo.

```
--repo <path>      Local sync repo directory (default: <config-dir>/repo)
--remote <url>     Git remote URL of the sync repo
--branch <name>    Branch to sync on (default: main)
--auto-push        Push to the remote automatically after every 'push'
```

If `--remote` points at a repo that already contains data, `init` **clones** it,
so a new machine starts from your existing state. If the remote is empty (or
unset), it initialises a fresh local repo.

Re-running `init` updates the configuration in place (e.g. to add a remote or
turn on auto-push).

### `contix status`

Shows configuration, each tool's detected state directory and file count, the
list of tracked git repos with their current branch/cleanliness, and whether the
sync repo has uncommitted snapshots.

### `contix push`

```
--tools <list>     Comma-separated tools to push (default: all)
--days <N>         Only include session transcripts newer than N days (0 = all)
--message <msg>    Commit message (default: "contix sync <time> from <host>")
--push             Push to the git remote after committing
--no-repos         Skip tracked git working repositories
```

Steps: collect each tool's whitelisted files into `tarball + manifest`,
snapshot each tracked git repo, `git add -A && git commit`, then (with `--push`
or `auto_push`) `git pull --rebase` and `git push`.

### `contix pull`

```
--tools <list>     Comma-separated tools to restore (default: all)
--no-rewrite       Do not rewrite machine paths in restored state
--no-repos         Skip restoring tracked git working repositories
--map OLD=NEW      Extra path mapping (repeatable)
```

Steps: `git pull` the sync repo, extract each tool bundle into its home dir,
verify every file against its recorded SHA-256, rewrite embedded paths for this
machine, then restore each git repo (clone if missing, recreate branches,
checkout the previous branch, reapply uncommitted + untracked work).

### `contix list`

Lists tool bundles (file counts, size, source OS, tool version, timestamp) and
git repo snapshots (branch, branch count, whether uncommitted/untracked data is
present) currently stored in the sync repo.

### `contix verify`

Extracts every tool bundle to a temp directory and checks all files against the
manifest SHA-256 digests. Confirms the archives are intact and restorable.

### `contix repos`

```
contix repos add <path>...    Track git repositories (skips non-repos)
contix repos remove <path>    Stop tracking a repository
contix repos list             Show tracked repositories and their state
```

`add` records the repository's top-level directory. If a repo has no `origin`
remote, `contix` warns you: without a remote its commit history can't be cloned
on another machine (only its uncommitted/untracked files will be synced).

### `contix doctor`

Environment diagnostics: git availability, config presence, sync-repo state,
remote configuration, and detection of each tool's state directory.

---

## Configuration

Config is a small JSON file:

| OS | Location |
|---|---|
| Linux | `$XDG_CONFIG_HOME/contix/config.json` or `~/.config/contix/config.json` |
| macOS | `~/.config/contix/config.json` |
| Windows | `%AppData%\contix\config.json` |

Override the whole config directory with `CONTIX_CONFIG_DIR`.

```json
{
  "repo_path": "/home/you/.config/contix/repo",
  "remote": "git@github.com:you/dev-state.git",
  "branch": "main",
  "auto_push": false,
  "home": "/home/you",
  "repos": ["/home/you/code/project-a"]
}
```

Environment overrides for tool locations:

- `CODEX_HOME` — Codex state dir (default `~/.codex`)
- `CLAUDE_CONFIG_DIR` — Claude Code state dir (default `~/.claude`)

---

## What gets synced (and skipped)

Matching rules (relative to each tool's home):

- `dir/` — the directory and everything under it
- `*.ext` — glob on the file's basename
- `name` — any path segment equal to `name`
- `a/b/c` — an exact path or a prefix directory

**Exclude always wins over include.**

### Codex (`~/.codex`)

Included: `config.toml`, `*.config.toml` provider configs, `AGENTS.md`,
`version.json`, `history.jsonl`, `session_index.jsonl`, `rules/`, `skills/`,
`memories/`, `sessions/`, `archived_sessions/`, `attachments/`, and the small
`memories_*.sqlite` / `goals_*.sqlite` / `state_*.sqlite` stores (with their
`-wal` sidecars).

Excluded: `auth.json`, `.credentials.json`, `logs_*.sqlite` (large telemetry),
`*.sqlite-shm`, `cache/`, `models_cache.json`, `shell_snapshots/`, `log/`,
`tmp/`, `plugins/`, `vendor_imports/`, `installation_id`, `.git`.

### Claude Code (`~/.claude`)

Included: `CLAUDE.md`, `settings.json`, `.claude.json` (project registry),
`history.jsonl`, `projects/` (per-project transcripts), `skills/`, `rules/`,
and plugin *config* files (`plugins/config.json`, `installed_plugins.json`,
`known_marketplaces.json`, `blocklist.json`).

Excluded: `.credentials.json`, `cache/`, `downloads/`, `backups/`, `ide/`,
`plugins/marketplaces/`, `plugins/repos/`, `.last-cleanup`, `*.bak`, `.git`.

> To trim large bundles, use `--days N` so old session transcripts under
> `sessions/`, `archived_sessions/` and `projects/` are dropped. Config,
> memory, rules and skills are always kept regardless of age.

---

## How git working repos are synced

For each tracked repo, `push` records (into `git/<key>/state.json`):

- `origin` remote URL
- current branch and `HEAD` commit
- every local branch with its commit SHA and upstream
- whether uncommitted tracked changes exist → stored as `git/<key>/changes.patch`
  (`git diff --binary HEAD`)
- untracked, non-ignored files → bundled into `git/<key>/untracked.tar.gz`

On `pull`, for each snapshot:

1. If the working directory is missing and a remote is known → `git clone` it.
2. `git fetch` so recorded commits/branches are reachable.
3. Recreate any missing local branch at its recorded commit (falling back to its
   upstream). Branches whose commit was never pushed can't be recreated and are
   reported.
4. `git checkout` the branch that was active at snapshot time.
5. Restore untracked files, then reapply `changes.patch` with `git apply --3way`.

If the patch doesn't apply cleanly (diverged base), the failure is reported and
the patch remains in the sync repo so nothing is lost.

---

## Cross-machine path rewriting

Session and project files often embed absolute paths (e.g.
`/home/alice/project`). On restore, `contix` rewrites the *source* machine's home
directory to the *current* machine's home inside text files (`.json`, `.jsonl`,
`.md`, `.toml`, `.txt`, `.yaml`, `.yml`). It handles:

- POSIX and Windows separators (`/` vs `\`)
- JSON escaping on Windows (`\` → `\\`) so files stay valid
- Claude Code's dash-encoded project directory names
  (`-home-alice-proj` → `-Users-bob-proj`), renaming the directories too

Add extra rules with `--map OLD=NEW` (repeatable), or disable rewriting with
`--no-rewrite`.

---

## Layout of the sync repo

```
<repo>/
├── README.md                 # generated note
├── claude/
│   ├── bundle.tar.gz         # compressed Claude state
│   └── manifest.json         # per-file SHA-256, tool version, source machine
├── codex/
│   ├── bundle.tar.gz
│   └── manifest.json
└── git/
    └── <name>-<hash>/        # one dir per tracked repo
        ├── state.json        # remote, branches, current branch, HEAD
        ├── changes.patch     # uncommitted tracked changes (if any)
        └── untracked.tar.gz  # untracked non-ignored files (if any)
```

---

## Typical workflows

**Daily checkpoint before switching machines**

```bash
contix push --push
```

**New laptop setup**

```bash
# install git + make first (no Go needed — prebuilt binary), then:
make install
contix init --remote git@github.com:you/dev-state.git
contix pull
```

**Only sync Codex, keep bundles small**

```bash
contix push --tools codex --days 14 --push
```

**Restore AI state but leave my local repos alone**

```bash
contix pull --no-repos
```

---

## Troubleshooting

**`git is not installed or not on PATH`** — install git; `contix` relies on it.

**Push rejected / non-fast-forward** — someone else (another machine) pushed
first. `contix push --push` runs `git pull --rebase` before pushing; if a genuine
conflict exists, resolve it in `repo_path` and push again.

**A repo shows `skipped (repo missing locally and no remote recorded)`** — the
tracked repo had no `origin` remote when snapshotted, so there's nothing to clone
on this machine. Add a remote and re-push, or create the directory manually.

**`version mismatch` on pull** — the tool version that produced the state differs
from the one installed here. State usually still loads; update the tool to match
if you hit format issues.

**Patch didn't apply** — the repo's base commits diverged too far. The patch is
preserved at `git/<name>/changes.patch` in the sync repo; apply it by hand with
`git apply --3way`.

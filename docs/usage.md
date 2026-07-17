# contix — usage & internals

This document is the full reference for `contix`. For a quick overview see the
[README](../README.md).

## Contents

1. [Concepts](#concepts)
2. [Command reference](#command-reference)
3. [Configuration](#configuration)
4. [What gets synced (and skipped)](#what-gets-synced-and-skipped)
5. [Cross-machine path rewriting](#cross-machine-path-rewriting)
6. [Layout of the sync repo](#layout-of-the-sync-repo)
7. [Typical workflows](#typical-workflows)
8. [Troubleshooting](#troubleshooting)

---

## Concepts

- **Sync repo** — a single git repository you own (usually private on GitHub)
  that stores the *latest* snapshot of everything. `contix` keeps a local clone
  and pushes/pulls it.
- **Tools** — the AI coding agents `contix` understands: `codex`, `claude`, and
  `hermes`.
  Each has a curated include/exclude list so only meaningful state is synced.
- **Snapshot** — one `contix collect`: it rebuilds the bundles, commits them to the
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
```

If `--remote` points at a repo that already contains data, `init` **clones** it,
so a new machine starts from your existing state. If the remote is empty (or
unset), it initialises a fresh local repo.

Re-running `init` updates the configuration in place, such as adding a remote.

### `contix collect`

```
--tools <list>     Comma-separated tools to collect (default: all)
```

Steps: collect each tool's portable files into `tarball + manifest`, remove any
legacy working-repository snapshots, then run `git add -A && git commit` locally.
Compressed bundles larger than 5 MiB are split into GitHub-safe parts. If a
previous snapshot was never accepted by the remote, `collect` replaces/squashes
that unpublished history so rejected large objects are no longer pushed.
The commit message, hostname and timestamp are generated automatically.

### `contix push`

Uploads the state previously committed by `contix collect`. Before pushing, it
pulls and rebases the configured sync branch. It refuses to run when the sync
repo contains uncommitted changes. Remote Git transfer progress is streamed
while it works.

### `contix pull`

Steps: `git pull` the sync repo, extract each tool bundle into its home dir,
verify every file against its recorded SHA-256, then rewrite embedded paths for
this machine. Pull always restores all three tools and verifies their checksums.
Git transfer progress and per-tool restore activity remain visible throughout.

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
  "home": "/home/you"
}
```

Environment overrides for tool locations:

- `CODEX_HOME` — Codex state dir (default `~/.codex`)
- `CLAUDE_CONFIG_DIR` — Claude Code state dir (default `~/.claude`)
- `HERMES_HOME` — Hermes Agent state dir (default `~/.hermes`)

---

## What gets synced (and skipped)

`contix` syncs **everything** under each tool's home directory, except for a
small set of items that are unsafe or pointless to sync. Matching rules for the
exclude patterns (relative to each tool's home):

- `dir/` — the directory and everything under it
- `*.ext` — glob on the file's basename
- `name` — any path segment equal to `name`
- `a/b/c` — an exact path or a prefix directory

Archives use gzip's maximum compression level. A compressed bundle larger than
5 MiB is stored as ordered `bundle.tar.gz.part-NNN` files. `pull` reads those
parts as one continuous archive and verifies restored files; older single-file
bundles remain compatible.

### Codex (`~/.codex`)

Everything is synced **except**:

- `auth.json`, `.credentials.json` — machine-locked credentials (never synced)
- `logs_*.sqlite` — large (300MB+) telemetry log that regenerates on its own
- `*.sqlite-shm` — SQLite shared-memory sidecar, rebuilt on open
- `tmp/`, `.tmp/`, `*.lock` — volatile runtime scratch files
- `.git` — nested git repos, which would corrupt the sync repo if embedded

### Claude Code (`~/.claude`)

Everything is synced **except**:

- `.credentials.json` — machine-locked credentials (never synced)
- `*.sqlite-shm` — SQLite shared-memory sidecar, rebuilt on open
- `tmp/`, `*.lock` — volatile runtime scratch files
- `.git` — nested git repos (e.g. plugin marketplaces), which would corrupt the
  sync repo if embedded

### Hermes Agent (`~/.hermes`)

Portable config and working context are synced: `config.yaml`, `SOUL.md`,
memories, skills, sessions, cron definitions and the state database. The
following are excluded:

- `auth.json`, `auth.lock`, `.env`, `.credentials.json`, `pairing/`
- `hermes-agent/`, `bin/`, caches, logs, sandbox/runtime data and
  `cron/ticker_heartbeat`
- lock files and SQLite sidecars

## Cross-machine path rewriting

Session and project files often embed absolute paths (e.g.
`/home/alice/project`). On restore, `contix` rewrites the *source* machine's home
directory to the *current* machine's home inside text files (`.json`, `.jsonl`,
`.md`, `.toml`, `.txt`, `.yaml`, `.yml`). It handles:

- POSIX and Windows separators (`/` vs `\`)
- JSON escaping on Windows (`\` → `\\`) so files stay valid
- Claude Code's dash-encoded project directory names
  (`-home-alice-proj` → `-Users-bob-proj`), renaming the directories too

Path rewriting is automatic during `contix pull`.

---

## Layout of the sync repo

```
<repo>/
├── README.md                 # generated note
├── claude/
│   ├── bundle.tar.gz         # compressed Claude state
│   └── manifest.json         # per-file SHA-256, tool version, source machine
├── codex/
│   ├── bundle.tar.gz.part-000 # large bundles are split into 5 MiB parts
│   ├── bundle.tar.gz.part-001
│   └── manifest.json
└── hermes/
│   ├── bundle.tar.gz
│   └── manifest.json
```

---

## Typical workflows

**Daily checkpoint before switching machines**

```bash
contix collect
contix push
```

**New laptop setup**

```bash
# install git + make first (no Go needed — prebuilt binary), then:
make install
contix init --remote git@github.com:you/dev-state.git
contix pull
```

**Upgrade contix from the checkout**

```bash
make upgrade
```

`make install`, `make upgrade`, and `contix --version` show the version and short
release notes. Change [`release/VERSION`](../release/VERSION) and
[`release/NOTES`](../release/NOTES), then run `make release`; both values are
embedded into the resulting binaries.

**Only collect Codex**

```bash
contix collect --tools codex
contix push
```

---

## Troubleshooting

**`git is not installed or not on PATH`** — install git; `contix` relies on it.

**Push rejected / non-fast-forward** — someone else (another machine) pushed
first. `contix push` runs `git pull --rebase` before pushing; if a genuine
conflict exists, resolve it in `repo_path` and push again.

**GitHub reports a file larger than 100 MB** — upgrade contix, then run
`contix collect` again followed by `contix push`. The new collection splits the
archive and rewrites the unpublished rejected snapshot.

**`version mismatch` on pull** — the tool version that produced the state differs
from the one installed here. State usually still loads; update the tool to match
if you hit format issues.

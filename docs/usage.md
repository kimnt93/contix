# contix — usage & internals

This document is the full reference for `contix`. For a quick overview see the
[README](../README.md).

## Contents

1. [Concepts](#concepts)
2. [Command reference](#command-reference)
3. [Configuration](#configuration)
4. [What gets synced](#what-gets-synced)
5. [Cross-machine path rewriting](#cross-machine-path-rewriting)
6. [Layout of the sync repo](#layout-of-the-sync-repo)
7. [Typical workflows](#typical-workflows)
8. [Troubleshooting](#troubleshooting)

---

## Concepts

- **Sync repo** — a single git repository you own (usually private on GitHub)
  that stores the *latest* snapshot of everything. `contix` keeps a local clone
  and pushes/pulls it.
- **Targets** — agent state, VS Code-family editor data/extensions, SSH and
  hosts. Product groups include `antigravity`, `cursor`, `kiro`, `vscode`,
  `vscodium`, `void` and `windsurf`.
  Agent and SSH roots are complete snapshots; hosts selects only the hosts file
  rather than the entire system configuration directory.
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
--tools <list>     Comma-separated targets to collect (default: all)
```

Steps: collect each tool's portable files into `tarball + manifest`, remove any
legacy working-repository snapshots, then run `git add -A && git commit` locally.
Compressed bundles larger than 5 MiB are split into GitHub-safe parts. If a
previous snapshot was never accepted by the remote, `collect` replaces/squashes
that unpublished history so rejected large objects are no longer pushed.
The commit message, hostname and timestamp are generated automatically.
When a target is missing or has no matching files, any previous bundle already
in the sync repo is kept.

### `contix push`

Uploads the state previously committed by `contix collect`. Before pushing, it
pulls and rebases the configured sync branch. It refuses to run when the sync
repo contains uncommitted changes. Remote Git transfer progress is streamed
while it works.

### `contix pull`

```
--ignore           Ignore conflicts and overwrite with the synced snapshot
```

Steps: `git pull` the sync repo, extract each tool bundle into its home dir,
verify every file against its recorded SHA-256, then rewrite embedded paths for
this machine. Pull restores every available target and verifies its checksums.
Git transfer progress and per-tool restore activity remain visible throughout.
If no remote bundle exists for a target, its local files are left unchanged.
When an archived path already exists with different content or a different file
type, normal pull reports the paths and leaves that entire target untouched.
`--ignore` explicitly permits those conflicts to be overwritten.

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
- `KIRO_HOME` — Kiro state dir (default `~/.kiro`)
- `ANTIGRAVITY_HOME` — Antigravity/Gemini state root (default `~/.gemini`)
- `CONTIX_SSH_HOME` — SSH config dir (default `~/.ssh`)
- `CONTIX_HOSTS_DIR` — system hosts directory (default `/etc` on Unix)
- `CONTIX_CURSOR_DATA_HOME`, `CONTIX_CURSOR_HOME`
- `CONTIX_WINDSURF_DATA_HOME`, `CONTIX_WINDSURF_AGENT_HOME`, `CONTIX_WINDSURF_HOME`
- `CONTIX_VSCODE_DATA_HOME`, `CONTIX_VSCODE_HOME`
- `CONTIX_VSCODIUM_DATA_HOME`, `CONTIX_VSCODIUM_HOME`
- `CONTIX_VOID_DATA_HOME`, `CONTIX_VOID_HOME`
- `CONTIX_KIRO_IDE_HOME`
- `CONTIX_ANTIGRAVITY_IDE_HOME`, `CONTIX_ANTIGRAVITY_EXTENSIONS_HOME`

Editor `*_DATA_HOME` defaults are platform aware: `~/.config/<Product>` on
Linux, `~/Library/Application Support/<Product>` on macOS and
`%APPDATA%\<Product>` on Windows.

---

## What gets synced

Agent and SSH targets sync every regular file and symlink below their configured
root. There are no credential, key, cache, log, runtime or nested-repository
exclusions. The hosts target is intentionally scoped to the single system hosts
file instead of the entire system configuration directory. Sockets, devices and
other non-portable special files cause collection to fail explicitly.

Archives use gzip's maximum compression level. A compressed bundle larger than
5 MiB is stored as ordered `bundle.tar.gz.part-NNN` files. `pull` reads those
parts as one continuous archive and verifies restored files; older single-file
bundles remain compatible.

### Codex (`~/.codex`)

Everything is synced, including credentials, databases and sidecars, telemetry,
temporary files, locks, caches, plugins and nested repositories.

### Claude Code (`~/.claude`)

Everything is synced, including credentials, database sidecars, temporary files,
locks, downloads, plugin marketplaces and their nested repositories.

### Hermes Agent (`~/.hermes`)

Everything is synced: configuration, credentials, pairing state, memories,
skills, sessions, cron data, databases and sidecars, installed source/venv,
binaries, caches, logs and sandbox/runtime data.

### Kiro (`~/.kiro`)

Everything under the state root is synced, including agents, prompts, steering,
settings, skills, CLI sessions, credentials, locks, logs, caches and nested
repositories. Kiro's official `KIRO_HOME` override is honored.

### Google Antigravity (`~/.gemini`)

Everything below the configured Gemini/Antigravity state root is synced,
including global rules, authentication, installation IDs, artifacts, knowledge,
conversations, MCP configuration, locks, temporary data, logs and caches.

### VS Code-family editors

The following product groups expand into every listed root:

| Group | Application data | Additional state |
|---|---|---|
| `cursor` | Cursor | `~/.cursor` |
| `windsurf` | Windsurf | `~/.codeium/windsurf`, `~/.windsurf` |
| `vscode` | Code | `~/.vscode` |
| `vscodium` | VSCodium | `~/.vscode-oss` |
| `void` | Void | `~/.void` |
| `kiro` | Kiro | `~/.kiro` |
| `antigravity` | Antigravity | `~/.gemini`, `~/.antigravity` |

Application-data roots contain settings, histories, workspace/global storage,
authentication, caches and extension state. Home roots contain agent config,
rules, MCP settings, installed extensions and related runtime data. As with all
targets, every regular file and symlink is included.

### SSH configuration (`~/.ssh`)

Everything below the SSH root is synced, including configuration fragments,
private/public keys, `known_hosts`, authorized keys and backups.

### System hosts file

Only `hosts` under the platform's system hosts directory is collected. On pull,
contix writes it directly only when the destination is writable. Otherwise the
local file remains unchanged and the synced copy is verified and staged at the
path printed by `contix pull` (normally
`~/.config/contix/pending/hosts/hosts`). Review it before applying it with
administrator privileges because host-specific entries may differ.

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
├── antigravity/              # agent artifacts, knowledge and rules
├── claude/
│   ├── bundle.tar.gz         # compressed Claude state
│   └── manifest.json         # per-file SHA-256, tool version, source machine
├── codex/
│   ├── bundle.tar.gz.part-000 # large bundles are split into 5 MiB parts
│   ├── bundle.tar.gz.part-001
│   └── manifest.json
├── cursor/                   # OS-specific Cursor application data
├── cursor-home/              # ~/.cursor
├── hermes/                   # same bundle + manifest layout
├── hosts/
├── kiro/
└── ssh/
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

`make install`, `make upgrade`, and `contix --version` show the version and
feature checklist. Change [`release/VERSION`](../release/VERSION) and
[`release/NOTES`](../release/NOTES), then run `make release`; both values are
embedded into the resulting binaries.

**Only collect Codex**

```bash
contix collect --tools codex
contix push
```

Multiple targets can be selected together:

```bash
contix collect --tools kiro,antigravity,ssh,hosts
```

Editor product names collect all associated roots:

```bash
contix collect --tools cursor,windsurf
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

**Hosts was staged instead of restored** — the system hosts file needs elevated
permissions. Contix prints the staged and destination paths and deliberately
keeps the local file untouched for review.

**Pull reports conflicts** — the local and synced versions differ. The listed
target is not modified. Review the paths, or use `contix pull --ignore` to
explicitly overwrite them with the synced snapshot.

**Collection fails on a runtime/permission file** — all-files mode refuses to
silently omit it. Stop the related application or correct the file permissions,
then run `contix collect` again.

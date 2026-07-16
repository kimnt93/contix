# contix

**Sync your AI coding agents and git working state across machines through one GitHub repo.**

`contix` snapshots the state of [Codex](https://github.com/openai/codex) and
[Claude Code](https://docs.anthropic.com/en/docs/claude-code) — their memory,
settings, rules, skills and session history — together with the branches and
uncommitted work in your git repositories, compresses it, and pushes it to a
single git repo you own. On a new machine you run one command to pull it all
back and pick up exactly where you left off.

It is a single, dependency-free binary written in Go (1.26). It shells out to
your existing `git`, so it reuses your SSH keys, credentials and identity.

```
  Machine A                    GitHub                    Machine B
  ┌──────────────┐                                     ┌──────────────┐
  │ ~/.codex     │──┐        ┌───────────┐        ┌──▶ │ ~/.codex     │
  │ ~/.claude    │──┼─push──▶│ one repo  │──pull──┤    │ ~/.claude    │
  │ ~/code/*     │──┘        │ (latest)  │        └──▶ │ ~/code/*     │
  └──────────────┘           └───────────┘             └──────────────┘
```

---

## Why

Moving to a new laptop means losing your agents' accumulated memory, your
carefully tuned settings, and the half-finished branches scattered across your
projects. Existing dotfile/sync tools compress and upload files, but none
understand multiple AI coding agents. `contix` does:

- **Codex + Claude Code aware** — it knows which files are worth syncing
  (memory, rules, skills, sessions, settings) and which to skip
  (caches, logs, machine-locked credentials).
- **Git working state** — it records each tracked repo's remote, all local
  branches, the current branch, uncommitted changes, and untracked files, then
  reconstructs them on the other machine.
- **One repo, latest wins** — everything lands in a single git repo that always
  holds the latest snapshot. No servers, no accounts, no lock-in.
- **Cross-platform & portable** — Linux, macOS, Windows. Paths embedded in
  session files are rewritten to the new machine's home directory automatically.

---

## Install

### From source (recommended)

Requires Go 1.26+, git, and `make`. Works the same on Linux, macOS and Windows.

```bash
git clone <this-repo> contix && cd contix
make install      # builds and installs into $(go env GOPATH)/bin
```

Make sure `$(go env GOPATH)/bin` is on your `PATH`.

### Prebuilt binaries

```bash
make release      # cross-compiles into ./dist for all platforms
```

Download the archive for your OS/arch from `dist/`, extract it, and put `contix`
(`contix.exe` on Windows) somewhere on your `PATH`.

---

## Quick start

### 1. First machine

Create an **empty, private** git repo on GitHub (e.g. `you/dev-state`), then:

```bash
contix init --remote git@github.com:you/dev-state.git
contix repos add ~/code/project-a ~/code/project-b   # track your working repos
contix push                                           # collect + commit locally
contix push --push                                    # ...and upload to GitHub
```

Tip: `contix init --remote <url> --auto-push` uploads automatically on every push.

### 2. New machine

Install `contix`, then:

```bash
contix init --remote git@github.com:you/dev-state.git   # clones existing state
contix pull                                             # restores everything
```

Your Codex/Claude memory and settings are back, your projects are cloned to the
same relative path under your home directory, their branches are recreated, and
your uncommitted work is reapplied.

---

## Commands

| Command | What it does |
|---|---|
| `contix init` | Configure the sync repo. Clones the remote if it already has data. |
| `contix status` | Show config, what each tool would sync, and tracked repos. |
| `contix push [--push]` | Collect state + repo snapshots, commit, and (with `--push`) upload. |
| `contix pull` | Pull from the remote and restore state + repos onto this machine. |
| `contix list` | List what is currently stored in the sync repo. |
| `contix verify` | Extract and checksum every bundle to confirm integrity. |
| `contix repos add/remove/list` | Manage tracked git working repositories. |
| `contix doctor` | Diagnose environment and configuration. |
| `contix version` | Print the version. |

Common flags:

- `contix push --days 30` — only include session transcripts from the last 30
  days (memory, rules and skills are always kept). Keeps bundles small.
- `contix push --tools codex` — sync only one tool.
- `contix push --message "before reinstall"` — custom commit message.
- `contix pull --no-repos` — restore only AI state, not git repos.
- `contix pull --map /old/home=/new/home` — extra path rewrite rule.

See [docs/usage.md](docs/usage.md) for the full reference and internals.

---

## What gets synced

**Codex** (`~/.codex`, or `$CODEX_HOME`): `config.toml`, provider configs,
`AGENTS.md`, history, rules, skills, memories, sessions, and small SQLite
memory/state stores.
**Skipped:** `auth.json`/credentials, telemetry logs, caches, shell snapshots,
plugins.

**Claude Code** (`~/.claude`, or `$CLAUDE_CONFIG_DIR`): `CLAUDE.md`,
`settings.json`, project registry, `history.jsonl`, per-project transcripts,
skills, rules, and plugin *config*.
**Skipped:** `.credentials.json`, caches, downloads, backups, marketplace repos.

**Git repos** you track with `contix repos add`: origin URL, all local branches,
current branch, uncommitted tracked changes (as a patch), and untracked
non-ignored files.

---

## Security notes

- **Use a private repository.** Your synced state contains project context,
  session history and settings.
- **Credentials are never synced.** `auth.json`, `.credentials.json` and similar
  machine-locked secrets are explicitly excluded. You re-authenticate each tool
  on the new machine as usual.
- `contix` uses your existing `git` and its credential setup; it never handles
  tokens itself.

---

## Limitations

- A tracked repo **without an `origin` remote** can't have its commit history
  restored on a new machine (there's nowhere to clone it from). `contix` warns
  you at `repos add` time; only its uncommitted/untracked files are synced.
- Uncommitted changes are stored as a `git diff` patch and reapplied with a
  3-way merge. If the base commits diverge wildly the patch may not apply
  cleanly; it is kept in the sync repo as `git/<repo>/changes.patch` so nothing
  is lost.
- `contix` syncs the **latest** snapshot. History lives in the git repo's commits,
  but `contix` itself always restores the most recent push.

---

## Development

```bash
make build      # build ./contix
make test       # run tests
make vet        # go vet
make fmt        # gofmt -w .
make release    # cross-compile all platforms into ./dist
```

## License

MIT (see LICENSE).

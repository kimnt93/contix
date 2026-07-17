# contix

**Sync your coding agents and portable machine configuration through one GitHub repo.**

`contix` snapshots the state of [Codex](https://github.com/openai/codex),
[Claude Code](https://docs.anthropic.com/en/docs/claude-code), Hermes Agent,
[Kiro](https://kiro.dev/) and [Google Antigravity](https://antigravity.google/),
plus SSH configuration and the system hosts file. It compresses everything into
a single git repo you own. On a new machine, one pull restores the available
state so you can pick up where you left off.

It is a single, dependency-free binary written in Go (1.26). It shells out to
your existing `git`, so it reuses your SSH keys, credentials and identity.

> **The name** — *contix* blends **cont**ext and **-x** (sync/exchange): it keeps
> the working *context* of your AI agents in sync across machines.

```
  Machine A                    GitHub                    Machine B
  ┌────────────────┐                                   ┌────────────────┐
  │ agent state    │──┐      ┌───────────┐        ┌──▶ │ agent state    │
  │ SSH config     │──┼─push▶│ one repo  │──pull──┤    │ SSH config     │
  │ hosts          │──┘      │ (latest)  │        └──▶ │ hosts          │
  └────────────────┘         └───────────┘             └────────────────┘
```

---

## Why

Moving to a new laptop means losing your agents' accumulated memory and your
carefully tuned settings. Existing dotfile/sync tools compress and upload files,
but none understand multiple AI coding agents. `contix` does:

- **Seven built-in targets** — Codex, Claude Code, Hermes, Kiro, Antigravity,
  SSH config and hosts are collected together. It syncs portable state
  (memory, rules, skills, sessions, settings and more), skipping only what's
  unsafe or pointless: machine-locked credentials, huge regenerating telemetry
  logs, and nested `.git` repos.
- **One repo, latest wins** — everything lands in a single git repo that always
  holds the latest snapshot. No servers, no accounts, no lock-in.
- **GitHub-size safe** — archives use maximum gzip compression and are split
  into 5 MiB parts when necessary, staying below GitHub's per-file limit.
- **Cross-platform & portable** — Linux, macOS, Windows. Paths embedded in
  session files are rewritten to the new machine's home directory automatically.

---

## Install

Prebuilt binaries for every supported platform are committed under [`dist/`](dist/),
so **you don't need Go to install** — just `make`, `git` and a shell.

```bash
git clone https://github.com/kimnt93/contix.git && cd contix
make install      # installs the prebuilt binary for your OS/arch
make upgrade      # later: fast-forward this checkout and reinstall latest
```

Both commands print the installed version and its feature checklist. Release
metadata is kept in two easy-to-edit files:

- [`release/VERSION`](release/VERSION) — one version string, such as `0.5.0`
- [`release/NOTES`](release/NOTES) — one `- [x]` line per shipped feature

The files are embedded into the binary during the build, so they are not needed
at runtime. Edit them before running `make release` for a new version.

`make install` auto-detects your platform (Linux/macOS/Windows, amd64/arm64),
copies the matching binary from `dist/`, and installs it to a bin directory:

- Linux/macOS: `/usr/local/bin` if writable, else `~/.local/bin`
- Windows: `%LOCALAPPDATA%\contix\bin`

Override the location with `make install PREFIX=/usr/local`. Make sure the
target directory is on your `PATH`.

### Building from source (optional)

If you'd rather build yourself (requires Go 1.26+):

```bash
make build        # compile ./contix for the host platform
make release      # cross-compile all platforms into ./dist and refresh binaries
```

---

## Quick start

### 1. First machine

Create an **empty, private** git repo on GitHub (e.g. `you/dev-state`), then:

```bash
contix init --remote git@github.com:you/dev-state.git
contix collect       # collect available state and commit locally
contix push          # upload the collected state to GitHub
```

The remote may be **SSH** (`git@github.com:you/dev-state.git`) or **HTTPS**
(`https://github.com/you/dev-state.git`). SSH uses your existing SSH key; HTTPS
uses your git credential helper or a Personal Access Token.

### 2. New machine

Install `contix`, then:

```bash
contix init --remote git@github.com:you/dev-state.git   # clones existing state
contix pull                                             # restores everything
```

Your available agent state and portable machine configuration are back.

---

## Commands

| Command | What it does |
|---|---|
| `contix init` | Configure the sync repo. Clones the remote if it already has data. |
| `contix collect` | Collect available agent and machine state locally. |
| `contix push` | Upload the collected state to the configured remote. |
| `contix pull` | Pull from the remote and restore available state. |

Long operations stay visible: collection and restoration show an activity
spinner, while Git uploads and downloads stream their native percentage output.

Optional collection filter:

- `contix collect --tools codex,kiro,ssh` — collect selected targets only.

See [docs/usage.md](docs/usage.md) for the full reference and internals.

---

## What gets synced

`contix` syncs each tool's portable state, skipping only what's unsafe,
volatile, or reproducible.

**Codex** (`~/.codex`, or `$CODEX_HOME`): everything — `config.toml`, provider
configs, `AGENTS.md`, history, rules, skills, memories, sessions, SQLite
memory/state stores, caches, plugins, and more.
**Skipped:** `auth.json`/`.credentials.json` (credentials), `logs_*.sqlite`
(large regenerating telemetry), `*.sqlite-shm` sidecars, and nested `.git` dirs.

**Claude Code** (`~/.claude`, or `$CLAUDE_CONFIG_DIR`): everything — `CLAUDE.md`,
`settings.json`, project registry, `history.jsonl`, per-project transcripts,
skills, rules, plugins, downloads, backups, and more.
**Skipped:** `.credentials.json` (credentials), `*.sqlite-shm` sidecars, and
nested `.git` dirs (e.g. plugin marketplaces).

**Hermes Agent** (`~/.hermes`, or `$HERMES_HOME`): portable state including
`config.yaml`, `SOUL.md`, memories, skills, sessions, cron definitions and its
state database. Credentials (`auth.json`, `.env`), pairing data, caches, logs,
sandbox state, runtime binaries and the installed `hermes-agent` source/venv are
skipped.

**Kiro** (`~/.kiro`, or `$KIRO_HOME`): global settings, agents, prompts,
steering, skills and CLI sessions. Credentials, locks, logs and caches are
skipped.

**Google Antigravity** (`~/.gemini`, or `$ANTIGRAVITY_HOME`): global
`GEMINI.md` rules plus Antigravity artifacts, knowledge, conversations and MCP
configuration under `~/.gemini/antigravity`. Authentication, installation IDs,
locks, logs and caches are skipped.

**SSH config** (`~/.ssh`): only `config`, `config.d/` and `conf.d/` are synced.
Private keys, public keys, `known_hosts` and backup directories are never added.

**Hosts** (`/etc/hosts`, or the Windows equivalent): collected as its own
bundle. If the destination is not writable, `pull` keeps the local hosts file
unchanged and stages the synced copy under contix's `pending/hosts` directory,
printing both paths for manual review.

Each target is stored as one recursive `tar.gz` stream. When its compressed size
exceeds 5 MiB, contix writes `bundle.tar.gz.part-000`, `part-001`, and so on.
`pull` reassembles and verifies the parts automatically.

---

## Security notes

- **Use a private repository.** Your synced state contains project context,
  session history and settings.
- **Credentials are never synced.** `auth.json`, `.credentials.json` and similar
  machine-locked secrets are explicitly excluded. You re-authenticate each tool
  on the new machine as usual.
- **SSH keys are never synced.** Only allowlisted SSH configuration files are
  collected.
- `contix` uses your existing `git` and its credential setup; it never handles
  tokens itself.

---

## Limitations

- `contix` syncs the **latest** snapshot. History lives in the git repo's commits,
  but `contix` itself always restores the most recent push.
- If a target is not installed or has no matching files, its previous remote
  snapshot is retained. If the remote has no bundle for a target, `pull` keeps
  that target's existing local state.

If GitHub previously rejected an oversized bundle, upgrade and run
`contix collect` again. Contix rewrites unpublished snapshot history so the old
oversized object is not included by the following `contix push`.

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

# contix

**Sync your AI coding agents across machines through one GitHub repo.**

`contix` snapshots the state of [Codex](https://github.com/openai/codex),
[Claude Code](https://docs.anthropic.com/en/docs/claude-code), and Hermes Agent
— their memory, settings, rules, skills and session history — compresses it,
and pushes it to a single git repo you own. On a new machine you run one command
to pull it all back and pick up exactly where you left off.

It is a single, dependency-free binary written in Go (1.26). It shells out to
your existing `git`, so it reuses your SSH keys, credentials and identity.

> **The name** — *contix* blends **cont**ext and **-x** (sync/exchange): it keeps
> the working *context* of your AI agents in sync across machines.

```
  Machine A                    GitHub                    Machine B
  ┌──────────────┐                                     ┌──────────────┐
  │ ~/.codex     │──┐        ┌───────────┐        ┌──▶ │ ~/.codex     │
  │ ~/.claude    │──┼─push──▶│ one repo  │──pull──┤    │ ~/.claude    │
  │ ~/.hermes    │──┘        │ (latest)  │        └──▶ │ ~/.hermes    │
  └──────────────┘           └───────────┘             └──────────────┘
```

---

## Why

Moving to a new laptop means losing your agents' accumulated memory and your
carefully tuned settings. Existing dotfile/sync tools compress and upload files,
but none understand multiple AI coding agents. `contix` does:

- **Codex + Claude Code + Hermes aware** — it syncs portable tool state
  (memory, rules, skills, sessions, settings and more), skipping only what's
  unsafe or pointless: machine-locked credentials, huge regenerating telemetry
  logs, and nested `.git` repos.
- **One repo, latest wins** — everything lands in a single git repo that always
  holds the latest snapshot. No servers, no accounts, no lock-in.
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
contix push          # collect tool state and commit locally
contix push --push   # ...and upload to GitHub
```

The remote may be **SSH** (`git@github.com:you/dev-state.git`) or **HTTPS**
(`https://github.com/you/dev-state.git`). SSH uses your existing SSH key; HTTPS
uses your git credential helper or a Personal Access Token.

Tip: `contix init --remote <url> --auto-push` uploads automatically on every push.

### 2. New machine

Install `contix`, then:

```bash
contix init --remote git@github.com:you/dev-state.git   # clones existing state
contix pull                                             # restores everything
```

Your Codex/Claude/Hermes memory and settings are back.

---

## Commands

| Command | What it does |
|---|---|
| `contix init` | Configure the sync repo. Clones the remote if it already has data. |
| `contix status` | Show config and what each tool would sync. |
| `contix push [--push]` | Collect AI state, commit, and (with `--push`) upload. |
| `contix pull` | Pull from the remote and restore AI state onto this machine. |
| `contix list` | List what is currently stored in the sync repo. |
| `contix verify` | Extract and checksum every bundle to confirm integrity. |
| `contix doctor` | Diagnose environment and configuration. |
| `contix version` | Print the version. |

Common flags:

- `contix push --days 30` — only include session transcripts from the last 30
  days (memory, rules and skills are always kept). Keeps bundles small.
- `contix push --tools codex` — sync only one tool.
- `contix push --message "before reinstall"` — custom commit message.
- `contix pull --map /old/home=/new/home` — extra path rewrite rule.

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

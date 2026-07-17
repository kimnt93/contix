# contix

**Sync your coding agents and portable machine configuration through one GitHub repo.**

`contix` snapshots the state of [Codex](https://github.com/openai/codex),
[Claude Code](https://docs.anthropic.com/en/docs/claude-code), Hermes Agent,
[Kiro](https://kiro.dev/), [Google Antigravity](https://antigravity.google/) and
[OpenClaw](https://openclaw.ai/), along with SSH state and the system hosts file.
IDE application data, extensions, caches and workspace storage are not synced.
Everything selected is compressed into a single git repo you own. On a new
machine, one pull restores the available state so you can pick up where you
left off.

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

- **Agent-only coverage** — Codex, Claude Code, Hermes, Kiro, Antigravity and
  OpenClaw agent state are collected together with SSH and hosts. IDE state is
  deliberately excluded. Every regular file and symlink below each configured
  agent root is included, without credential/cache/runtime exclusion lists.
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

- [`release/VERSION`](release/VERSION) — one version string, such as `0.9.0`
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
- `contix collect --force-close` — stop running synced applications before
  collecting all targets.
- `contix collect --tools codex --force-close` — stop and collect only Codex.

`--force-close` may terminate active agent sessions, so it is never enabled by
default. Applications are asked to exit first and are forcibly terminated after
two seconds if they remain running.

`contix pull` protects differing local files by reporting conflicts and leaving
that target unchanged. Use `contix pull --ignore` when you intentionally want
the synced snapshot to overwrite those conflicts.

See [docs/usage.md](docs/usage.md) for the full reference and internals.

---

## What gets synced

`contix` syncs every regular file and symlink under each configured target root.
Credentials, keys, logs, caches, locks, runtime files and nested `.git`
directories are included. Non-portable special files such as sockets and device
nodes cause collection to fail rather than being silently skipped.

**Codex** (`~/.codex`, or `$CODEX_HOME`): everything, including configuration,
credentials, histories, rules, skills, memories, sessions, databases, logs,
caches, plugins, temporary state and nested repositories.

**Claude Code** (`~/.claude`, or `$CLAUDE_CONFIG_DIR`): everything, including
credentials, `CLAUDE.md`, settings, project registries, histories, transcripts,
skills, rules, plugins, downloads, backups, caches and runtime files.

**Hermes Agent** (`~/.hermes`, or `$HERMES_HOME`): everything, including
configuration, credentials, pairing data, memories, skills, sessions, cron,
databases, caches, logs, sandboxes, runtime binaries and installed source/venv.

**Kiro** (`~/.kiro`, or `$KIRO_HOME`): everything, including global settings,
agents, prompts, steering, skills, CLI sessions, credentials, locks, logs and
caches.

**Google Antigravity** (`~/.gemini`, or `$ANTIGRAVITY_HOME`): everything under
the Gemini/Antigravity state root, including global rules, authentication,
installation IDs, artifacts, knowledge, conversations, MCP configuration,
temporary files, locks, logs and caches.

**OpenClaw** (`~/.openclaw`, or `$OPENCLAW_STATE_DIR`): everything under its
mutable state root, including `openclaw.json`, credentials, secrets, per-agent
state, SQLite stores, sessions, transcripts, memories, skills, automation and
workspaces. `OPENCLAW_HOME` and named profiles are also honored.

**IDEs are not synced:** Cursor, Windsurf, VS Code, VSCodium, Void, Kiro IDE and
Antigravity IDE application data/extensions are excluded. Upgrading to this
version removes their old top-level bundles from the sync repository during the
next `contix collect`; it never deletes local IDE files.

**SSH** (`~/.ssh`, or `$CONTIX_SSH_HOME`): everything, including configuration,
private/public keys, `known_hosts`, authorized keys and backup directories.

**Hosts** (`/etc/hosts`, or the Windows equivalent): collected as its own
bundle. If the destination is not writable, `pull` keeps the local hosts file
unchanged and stages the synced copy under contix's `pending/hosts` directory,
printing both paths for manual review.

Each target is stored as one recursive `tar.gz` stream. When its compressed size
exceeds 5 MiB, contix writes `bundle.tar.gz.part-000`, `part-001`, and so on.
`pull` reassembles and verifies the parts automatically.

---

## Security notes

- **Use only a private, trusted repository.** Credentials, access tokens, SSH
  private keys, project context, session contents and machine state are
  intentionally synced.
- Anyone who can read the sync repository can potentially access your accounts
  and machines. Contix does not encrypt archive contents beyond Git transport.
- `contix` uses your existing `git` and its credential setup; it never handles
  the remote's Git authentication itself.
- Stop synced applications before `collect` when possible, or explicitly use
  `contix collect --force-close`. Files that disappear during collection are
  omitted because they no longer exist; permission and other read errors remain
  fatal.

---

## Limitations

- `contix` syncs the **latest** snapshot. History lives in the git repo's commits,
  but `contix` itself always restores the most recent push.
- If a target is not installed or has no matching files, its previous remote
  snapshot is retained. If the remote has no bundle for a target, `pull` keeps
  that target's existing local state.
- A normal `pull` does not overwrite differing local files. It reports every
  conflict and keeps that target unchanged; `pull --ignore` explicitly accepts
  overwriting with the synced snapshot.

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

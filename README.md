# contix

**Sync AI coding-agent state between computers through one private Git repository.**

Contix collects the configuration, credentials, instructions, skills, memories
and sessions of supported coding agents. It deliberately does not sync general
personal agents, SSH/hosts configuration, full IDE application data, extension
binaries, indexes or caches.

Supported coding agents:

- Codex
- Claude Code
- Gemini CLI / Google Antigravity
- Kiro CLI
- Cursor Agent and Cursor CLI
- OpenCode
- GitHub Copilot CLI
- Cline CLI
- Continue CLI (`cn`)
- Aider
- Goose
- Qwen Code
- Factory Droid
- Amp
- Auggie CLI

Each target becomes a compressed `tar.gz` stream. Streams larger than 5 MiB are
split into Git-safe parts and reassembled automatically during pull.

## Install

Prebuilt Linux, macOS and Windows binaries for amd64 and arm64 are committed in
[`dist/`](dist/), so installation does not require Go:

```bash
git clone https://github.com/kimnt93/contix.git
cd contix
make install
```

Upgrade later with:

```bash
make upgrade
```

Installation and upgrade print the embedded version and feature checklist from
[`release/VERSION`](release/VERSION) and [`release/NOTES`](release/NOTES).

## Usage

First computer:

```bash
contix init --remote git@github.com:you/dev-state.git
contix collect
contix push
```

New computer:

```bash
contix init --remote git@github.com:you/dev-state.git
contix pull
```

The four commands are intentionally simple:

| Command | Purpose |
|---|---|
| `contix init` | Configure or clone the private sync repository. |
| `contix collect` | Collect, compress and commit coding-agent state locally. |
| `contix push` | Upload the collected commit. |
| `contix pull` | Download and restore coding-agent state. |

Optional filters:

```bash
contix collect --tools codex,claude,cursor
contix collect --tools opencode,goose
contix collect --tools codex --force-close
```

`opencode` expands to its official data and configuration roots. `goose`
expands to its platform data root and legacy/global config root.

Normal pull reports differing local files and leaves that entire target
unchanged. Use `contix pull --ignore` only when the synced snapshot should
overwrite local conflicts.

## What is collected

- **Codex** — complete `$CODEX_HOME` or `~/.codex` state.
- **Claude Code** — complete `$CLAUDE_CONFIG_DIR` or `~/.claude` state.
- **Gemini CLI / Antigravity** — complete `$ANTIGRAVITY_HOME`,
  `$GEMINI_CLI_HOME/.gemini` or `~/.gemini` state, honoring Gemini's user
  settings, global instructions and sessions.
- **Kiro CLI** — complete `$KIRO_HOME` or `~/.kiro` state.
- **Cursor Agent** — only `mcp.json`, `cli-config.json`, rules, commands, skills
  and hooks below `$CONTIX_CURSOR_HOME` or `~/.cursor`. Full Cursor IDE data,
  extensions and caches are excluded.
- **OpenCode** — `$OPENCODE_CONFIG_DIR` or
  `$XDG_CONFIG_HOME/opencode`, plus `$XDG_DATA_HOME/opencode` for credentials,
  sessions and databases.
- **GitHub Copilot CLI** — complete `$COPILOT_HOME` or `~/.copilot` state. Its
  separately located platform cache is not collected.
- **Cline CLI** — settings, teams, sessions, plugins and hooks from
  `$CONTIX_CLINE_HOME` or `~/.cline`; logs and unrelated IDE data are excluded.
- **Continue CLI** — portable config, permissions, rules, models, MCP servers,
  prompts, agents and sessions from `$CONTIX_CONTINUE_HOME` or `~/.continue`.
- **Aider** — user-level `.aider.*` configuration and history files from the
  home directory. Contix does not scan Git repositories for project files.
- **Goose** — `$GOOSE_PATH_ROOT` when set; otherwise its platform data root and
  global config root, including coding sessions and provider configuration.
- **Qwen Code** — complete `$CONTIX_QWEN_HOME` or `~/.qwen` state, including
  user settings, instructions, skills and recorded sessions.
- **Factory Droid** — complete `$CONTIX_DROID_HOME` or `~/.factory` state,
  including settings, MCP, droids, commands, hooks, skills and sessions.
- **Amp** — complete `$CONTIX_AMP_HOME` or `~/.config/amp` state, including
  user settings, instructions, skills and plugins.
- **Auggie CLI** — complete `$CONTIX_AUGGIE_HOME` or `~/.augment` coding-agent
  state.

If a coding agent is absent, Contix keeps its previous snapshot. The next
collection removes old Hermes, OpenClaw, SSH, hosts and retired IDE bundles from
the sync repository; local source directories are never deleted.

## Security

Use a private, trusted repository. Coding-agent roots can contain access tokens,
OAuth credentials, prompts, transcripts and source context. Contix does not
encrypt archives beyond Git transport.

Transient files that disappear during collection are omitted because no bytes
remain. Permission failures are retried briefly, but a persistent unreadable
coding-agent file remains an error rather than being silently lost.

See [docs/usage.md](docs/usage.md) for path details, repository layout and
troubleshooting.

## Development

```bash
make build
make test
make vet
make fmt-check
make release
```

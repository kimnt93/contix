# contix usage and internals

Contix synchronizes user-level AI coding-agent state. It does not discover Git
repositories and does not synchronize general personal agents or machine
configuration.

## Commands

### `contix init`

```text
--repo <path>      Local sync repository (default: <config-dir>/repo)
--remote <url>     SSH or HTTPS Git remote
--branch <name>    Sync branch (default: main)
```

An existing remote is cloned. An empty or omitted remote creates a local Git
repository. Running `init` again updates the saved configuration.

### `contix collect`

```text
--tools <list>     Comma-separated coding-agent targets (default: all)
--force-close      Close selected coding-agent processes before collection
```

Collection walks only registered coding-agent roots, creates a gzip archive and
SHA-256 manifest for each available target, removes retired non-coding bundles,
and commits the result locally with an automatic timestamp/hostname message.

Product groups:

- `opencode` → `opencode` data plus `opencode-config`
- `goose` → `goose` platform data plus `goose-config`

Temporary files deleted between discovery and staging are omitted. Permission
failures are retried for two seconds; a file that remains unreadable fails the
collection. `--force-close` sends normal termination, waits two seconds, then
force-kills matching processes still running.

### `contix push`

Requires a clean sync repository produced by `collect`. It rebases from the
remote branch first, then streams native Git upload progress.

### `contix pull`

```text
--ignore           Overwrite conflicting local files
```

Pull downloads the latest commit, checks local conflicts, extracts each archive,
verifies its SHA-256 manifest, and rewrites the source machine's home path for
the destination machine. Without `--ignore`, any differing local file preserves
that entire target and is reported to the user.

## Coding-agent paths

| Target | User-level state |
|---|---|
| `codex` | `$CODEX_HOME` or `~/.codex` |
| `claude` | `$CLAUDE_CONFIG_DIR` or `~/.claude` |
| `antigravity` | `$ANTIGRAVITY_HOME`, `$GEMINI_CLI_HOME/.gemini` or `~/.gemini` |
| `kiro` | `$KIRO_HOME` or `~/.kiro` |
| `cursor` | `$CONTIX_CURSOR_HOME` or `~/.cursor`, portable agent allowlist only |
| `opencode` | `$XDG_DATA_HOME/opencode` or `~/.local/share/opencode` |
| `opencode-config` | `$OPENCODE_CONFIG_DIR` or `$XDG_CONFIG_HOME/opencode` |
| `copilot` | `$COPILOT_HOME` or `~/.copilot` |
| `cline` | `$CONTIX_CLINE_HOME` or `~/.cline`, portable agent allowlist only |
| `continue` | `$CONTIX_CONTINUE_HOME` or `~/.continue`, portable agent allowlist only |
| `aider` | user-level `.aider.*` files under the home directory |
| `goose` | `$GOOSE_PATH_ROOT` or the platform Goose data root |
| `goose-config` | `$GOOSE_PATH_ROOT/config` or the global Goose config root |
| `qwen` | `$CONTIX_QWEN_HOME` or `~/.qwen` |
| `droid` | `$CONTIX_DROID_HOME` or `~/.factory` |
| `amp` | `$CONTIX_AMP_HOME` or `~/.config/amp` |
| `auggie` | `$CONTIX_AUGGIE_HOME` or `~/.augment` |

Cursor's allowlist contains `mcp.json`, `cli-config.json`, `rules/`, `commands/`,
`skills/`, `hooks/` and `hooks.json`. It intentionally excludes full Cursor IDE
application data and extensions.

Cline includes `data/settings`, `data/teams`, `data/sessions`, `plugins` and
`hooks`. Continue includes its configuration files, `.env`, permissions, rules,
models, MCP servers, prompts, agents and sessions. Aider includes only global
user configuration/history because Contix never scans project repositories.

## Removed targets

The following are not coding-agent targets and are removed from the sync
repository during the next collection:

- Hermes and OpenClaw general-agent state
- SSH and system hosts state
- full Cursor/VS Code/VSCodium/Void/Windsurf application data
- Kiro and Antigravity IDE application data/extensions

Removal affects only bundle directories inside the Contix sync repository. It
never deletes local application state.

## Repository layout

```text
<repo>/
├── aider/
├── amp/
├── antigravity/
├── auggie/
├── claude/
├── cline/
├── codex/
├── continue/
├── copilot/
├── cursor/
├── droid/
├── goose/
├── goose-config/
├── kiro/
├── opencode/
├── opencode-config/
└── qwen/
```

Each directory contains `manifest.json` and either `bundle.tar.gz` or numbered
`bundle.tar.gz.part-NNN` files. Five-MiB parts avoid Git hosting file limits.

## Typical workflow

```bash
contix collect
contix push
```

On another machine:

```bash
contix init --remote git@github.com:you/dev-state.git
contix pull
```

## Troubleshooting

- **Unknown target:** run `contix collect -h`; supported names are also printed
  in an unknown-target error.
- **Permission denied:** close that coding agent or run collection with
  `--force-close`. Persistent unreadable state is not silently discarded.
- **GitHub large-file rejection:** upgrade, collect again, then push. Current
  archives are split into five-MiB parts.
- **Pull conflict:** review reported files or explicitly use `pull --ignore`.
- **Missing agent:** its previous remote snapshot is retained and local state is
  unchanged during pull.

## Release metadata

Edit [`../release/VERSION`](../release/VERSION) and
[`../release/NOTES`](../release/NOTES), then run `make release`. Both values are
embedded in every prebuilt binary and displayed by install, upgrade and
`contix --version`.

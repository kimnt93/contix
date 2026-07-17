# contix

Contix synchronizes AI coding-agent state between computers through one private
Git repository. It collects agent configuration, credentials, instructions,
skills, memories and sessions—without copying Git working repositories, general
personal agents, SSH/hosts files or complete IDE installations.

## Quick start

```bash
git clone https://github.com/kimnt93/contix.git
cd contix
make install
```

On the first computer:

```bash
contix init --remote git@github.com:you/dev-state.git
contix collect
contix push
```

On another computer:

```bash
contix init --remote git@github.com:you/dev-state.git
contix pull
```

Upgrade later with `make upgrade`.

## Supported coding agents

- [x] Codex
- [x] Claude Code
- [x] Gemini CLI / Google Antigravity
- [x] Kiro CLI
- [x] Cursor Agent / Cursor CLI
- [x] OpenCode
- [x] GitHub Copilot CLI
- [x] Cline CLI
- [x] Continue CLI
- [x] Aider
- [x] Goose
- [x] Qwen Code
- [x] Factory Droid
- [x] Amp
- [x] Auggie CLI

## Feature checklist

- [x] Four commands: `init`, `collect`, `push`, and `pull`
- [x] Automatic timestamped local commits
- [x] Recursive `tar.gz` compression with 5 MiB Git-safe parts
- [x] Collection, restore, and Git transfer progress
- [x] Previous snapshots preserved when an agent is missing
- [x] Pull conflict notification; `pull --ignore` explicitly overwrites
- [x] Optional `collect --tools ...` and `collect --force-close`
- [x] Version and feature notes embedded in install and upgrade output
- [x] Coding-agent-only scope with retired non-coding bundles cleaned up

## Documentation

See [docs/usage.md](docs/usage.md) for installation options, commands, synced
paths, security, archive layout, troubleshooting, and development/release
instructions.

## License

Contix is available under the [MIT License](LICENSE).

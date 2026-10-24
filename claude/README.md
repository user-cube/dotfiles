# Claude Code

Synced to `~/.claude/` by `dotfiles-sync` and the playbook. It lives in
`claude/`, not `.claude/`: Claude Code would read that as this repo's own
project settings. Change these here, not with `/config`: a sync overwrites
`~/.claude/settings.json`.

| File | |
|---|---|
| `settings.json` | no co-author trailer (`attribution`), `Read(...)` deny rules for keys, `.env` and credential files, the hooks below, plugins |
| `CLAUDE.md` | global instructions (it includes `RTK.md`) |
| `RTK.md` | how to use rtk (Rust Token Killer), the token-saving CLI proxy the playbook installs |
| `hooks/deny-sensitive-reads/` | a Go PreToolUse hook, below |

## Hooks (PreToolUse)

- **deny-sensitive-reads** (Bash and Read): blocks what is really private
  - ssh private keys, AWS credentials, env files (not `.env.example`),
    `*.pem` / `*.key`, age keys, app credential files
  - files under `secrets/` / `credentials/` unless `sops filestatus` says they are encrypted
  - `~/.ssh/config`, `~/.aws/config` and docker-compose files stay readable
  - the `Read(...)` deny rules don't cover Bash (`cat file`), hence the hook;
    for Bash it reads the paths from the command text, so it is a safety net, not a sandbox
  - built by `install.sh`; until it is, Claude Code's Bash and Read are blocked
    with "run dotfiles-sync"
  - `go test ./...` in its folder
- **rtk hook claude** (Bash): rewrites commands through rtk

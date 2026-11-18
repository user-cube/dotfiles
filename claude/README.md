# Claude Code

> The guide: [Claude Code](https://github.com/user-cube/dotfiles/wiki/Claude-Code) in the wiki. This README is the reference.

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
| `hooks/stop-check/` | a Go Stop hook, below |
| `hooks/git-guard/` | a Go PreToolUse hook, below |
| `hooks/write-guard/` | a Go PreToolUse hook, below |
| `statusline/` | the status line (a Go binary), below |

## Status line

One line under the prompt, what a session needs to know at a glance:

```
dotfiles  main*  ⎈ prod-eu/api PROD │ Opus · ctx 42% · $1.23 · 5h 18% · PR #7
```

- the directory, the git branch (`*` when tracked files are modified; untracked files are not looked for)
- the Kubernetes context and namespace from [kube-guard](../.config/zsh/kube-guard.zsh) (`kube-guard context`),
  so **PROD** in red means what kube-guard and its Claude hook mean by production
- the model, the context used (yellow from 70 %, red from 90 %), the cost, the 5-hour rate limit and the open PR
- what is not there is left out: no repository, no kube context, no cost yet, a field Claude Code did not send;
  git and kube-guard get 400 ms each and are left out when slower, so the line never holds the session up
- `refreshInterval` is 15 s so a context switched in another terminal shows up without a new message;
  `NO_COLOR=1` turns the colors off
- built by `install.sh`; until it is, the line is blank
- `go test ./...` in its folder

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
- **write-guard** (Edit, Write, MultiEdit, NotebookEdit and Bash): the other half of deny-sensitive-reads,
  which stops Claude reading what is private; this stops it changing what keeps it safe
  - denies a write (an edit, a redirection, `tee`, `sed -i`, `rm`, `mv`, `cp`, `ln`, `chmod`, `dd of=`...) to
    - `~/.ssh` and `~/.aws`
    - env files (not `.env.example` and other templates), `*.pem`, `*.key`, age keys
    - `.git/hooks` and `.git/config`: a `core.hooksPath` there turns every git hook off
    - `.claude/settings*.json` and `.claude/hooks/`, in `~/.claude` and in any project: the guardrails themselves
    - `~/.gitconfig*`, `~/.config/git/`, `~/git_hooks/`: the global git hooks and their switches
    - `~/.config/kube-guard/prod-regex`: what kube-guard calls production
  - reading them is not its business; this repo's own `claude/` directory is not `~/.claude`, so
    Claude edits the source here and a sync (your step) puts it in place
  - looks inside `sh -c`, `eval`, `$(...)`, backticks and `sudo` / `env` / `rtk` / `xargs`
  - for Bash it reads the paths from the command text: a script or an interpreter (`python -c`) that writes
    a path built at runtime is not seen, so it is a safety net, not a sandbox
  - built by `install.sh`; until it is, Claude Code's Edit, Write and Bash are blocked with "run dotfiles-sync"
  - `go test ./...` in its folder
- **git-guard** (Bash): checks the commits and PRs Claude writes, before they exist
  - `git commit`: no `Co-Authored-By` / "Generated with Claude", no `--no-verify` / `-n`
    (it would skip commit-msg and sops-guard), a header under 100 characters, in English,
    without a final full stop
  - `gh pr create` / `gh pr edit`: the same attribution rule and the same title rule;
    `create` also needs `--title` and `--body` / `--body-file` (or `--fill`), or gh stops to ask
  - `git push`: no force-push to, and no deleting of, `main` / `master` / `develop`
    (`--force`, `-f`, `--force-with-lease`, `+main`, `:main`, `HEAD:main`; a bare force-push
    looks up the current branch)
  - the git hooks can't be switched off: `--no-verify` (and `-n` on commit) on commit / push /
    merge / rebase / am, including its short forms (`--no-verif`: git takes any unambiguous start
    of an option), `-c core.hooksPath=` / `-c hook.*`, `GIT_CONFIG_*` variables and
    `git config` that sets them (reading them is fine)
  - looks inside `sh -c` / `bash -c`, `eval`, `$(...)`, backticks, `sudo` / `env` / `nohup`,
    and git aliases (`git config --get alias.<name>`), so none of the above hides there
  - asks the user (not a flat deny) for `gh pr merge`, `git reset --hard` and `git clean -f`
  - reads the message from the command text (`-m`, a heredoc, `-F -`); a message in a file
    or built at runtime is not seen, the git `commit-msg` hook is the one that always runs
  - built by `install.sh`; until it is, Claude Code's Bash is blocked with "run dotfiles-sync"
  - `go test ./...` in its folder
- **kube-guard claude-hook** (Bash): the zsh wrappers never see what Claude runs, so Claude gets the same rule
  - asks the user (not a flat deny) before `kubectl` / `kubecolor` / `helm` changes a production
    cluster: same parser, same production regex (`KUBE_PROD_REGEX` or `~/.config/kube-guard/prod-regex`)
    and same reads-pass rule as [kube-guard](../.config/zsh/kube-guard.zsh)
  - looks inside `sh -c`, `eval`, `$(...)`, backticks, `sudo` / `env` / `rtk` / `xargs`
  - `KUBE_GUARD=0` does not switch it off: that is the user's switch, not Claude's
  - an invalid production regex asks instead of allowing
  - a command built at runtime (`$CMD`, a script) is not seen: a safety net, not a sandbox
  - the binary is the one `install.sh` already builds in `.config/kube-guard`; until it is,
    Claude Code's Bash is blocked with "run dotfiles-sync"
  - `go test ./...` in `.config/kube-guard`
- **rtk hook claude** (Bash): rewrites commands through rtk

## Hooks (Stop)

The PreToolUse hooks stop what Claude must not do; this checks what it leaves behind.

- **stop-check**: when Claude is about to say it is done, it checks the files changed in the working tree
  (`git status`, untracked included) and, if a check fails, tells Claude why and Claude carries on
  - `*.go`: `go vet ./...` and `go test ./...` in the module of each changed file
  - `*.sh`: `shellcheck`, or `bash -n` without it; `*.zsh`: `zsh -n`
  - `*.json`: valid JSON (not under `vscode/`: those are JSONC)
  - no loop: once Claude has been sent back (`stop_hook_active`) it may stop
  - the same state is not checked twice: a fingerprint of the changed files (path, size, time) is kept per
    repository in `~/Library/Caches/stop-check`, so a turn that only answers a question costs nothing and
    a failure is reported once, not on every turn
  - 100 s for all the checks: a check that runs out of time is skipped, never reported as a failure
  - it runs the repository's own tests (`go test`), as Claude would
  - `STOP_CHECK=0` in the environment Claude Code starts from switches it off
  - built by `install.sh`; until it is, the hook does nothing and Claude just stops
  - `go test ./...` in its folder

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
| `hooks/readonly/`, `agents/` | the review agents and their Go hook, below |
| `statusline/` | the status line (a Go binary), below |
| `skills/` | my slash commands, below |

## Skills

`skills/<name>/SKILL.md`, run as `/<name>`. `dotfiles-sync` copies them with the rest of `claude/` (`rsync`
without `--delete`: a skill removed here stays in `~/.claude/skills` until removed there too).
`tests/test_claude_skills.py` checks every header (`name` is the directory, a description and a body exist).

| Skill | |
|---|---|
| `/commit [hint]` | commits the working tree on a feature branch (never on `main`: the commit-msg hook blocks it): files staged by name, a conventional header in English, no full stop, no attribution, no `--no-verify`. A hook that rejects it means a **new** commit, not an amend. It does not push |
| `/pr [draft] [base]` | pushes and opens a PR ready for review (`draft` for a draft): the body goes through `--body-file` (a heredoc leaves stray backslashes before backticks), with what changed, why and what a reviewer should know. An open PR for the branch is only pushed to. A `CONFLICTING` PR gets the base merged in. It never merges or turns on auto-merge |

Both are `disable-model-invocation: true`: only the user runs them, Claude does not decide to commit or open a PR
by itself, and each has an `allowed-tools` list of just the git and `gh pr` commands it needs.

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
- shown by Claude Code in a terminal (the VS Code integrated terminal too); the desktop app and the VS Code
  panel ignore `statusLine`, the hooks below work everywhere
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

## Agents

`agents/<name>.md`, copied with the rest of `claude/` (`rsync` without `--delete`: an agent removed here stays in
`~/.claude/agents`). A subagent's `tools:` line takes whole tools, not patterns, so least privilege is a short
tool list plus, where a tool is too wide, a hook in the agent's own header.
`tests/test_claude_skills.py` checks every header (and that an agent lists its `tools`).

| Agent | |
|---|---|
| `security-reviewer` | read-only review of a change (`git diff HEAD`, then the branch against `main`) for secrets, injection, trust, dependencies and CI, and for anything that weakens a guardrail. Reports findings ranked by severity with `file:line`, and what it did not check; it fixes nothing. Bash narrowed by `readonly git` |
| `helm-reviewer` | read-only review of a Helm chart or a change to one: `helm lint --strict`, `helm template` with the defaults and each environment's values (production first), then pod security, reliability, access, values and secrets, template and chart hygiene problems, with the rendered snippet as evidence. Never talks to a cluster or the network. Bash narrowed by `readonly git helm` |
| `k8s-reviewer` | read-only review of Kubernetes manifests and kustomize overlays or a change to them (for Helm charts: `helm-reviewer`): `kubeconform -strict` against the API schemas, `trivy config`, `kube-linter` and `kube-score` where installed, `kustomize build` per overlay (production first), then pod security, reliability, access, secrets in the repository (base64 is not encryption) and correctness (selectors that match nothing, an immutable `spec.selector` changed, patches that drop a base setting, removed `apiVersion`s). Never talks to a cluster: no `kubectl`, not even `--dry-run=client`. A tool that is not installed is reported, not guessed. Bash narrowed by `readonly git k8s` |
| `terraform-reviewer` | read-only review of Terraform or OpenTofu code or a change to it: `fmt -check`, `validate`, `tflint` and `trivy config`, then security, blast radius and lifecycle (replacements, renames with no `moved` block, `count` over a list), state and backend, version pinning and hygiene. It says what a change would create, replace or destroy from reading it, since it cannot plan. Never touches an account, a backend or the state. Bash narrowed by `readonly git terraform` |
| `ansible-reviewer` | read-only review of Ansible playbooks, roles and inventories or a change to them: `ansible-lint --offline`, `ansible-playbook --syntax-check` / `--list-tasks` / `--list-tags`, `yamllint`, then security (clear secrets, a vault-named file that is not encrypted, missing `no_log`, loose `mode`, unchecked downloads), idempotency (`shell` with no `changed_when`, `state: latest`), blast radius (`hosts: all` with no `serial`, destructive tasks built from variables), structure and inventories. Never connects to a host, runs a task or decrypts a vault. Bash narrowed by `readonly git ansible` |

They are delegated to by Claude (their descriptions say "use proactively"), or `@agent-<name>`. Tools: `Read`,
`Grep`, `Glob` and `Bash`, which the header's hook narrows.

### readonly (the review agents' Bash)

A PreToolUse hook named in an agent's own header, with the profiles it needs, so it applies to that agent only:

```yaml
hooks:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "~/.claude/hooks/readonly/readonly git helm || { echo 'readonly is not built: run dotfiles-sync' >&2; exit 2; }"
```

A profile (`hooks/readonly/profiles.go`) lists, per program, the subcommands that read, the flags that don't and the
arguments that may not be passed. Profiles add up (`git helm` allows what either allows); one it does not know
denies everything, so a typo in a header cannot widen what an agent may run. Adding a domain is a table entry and
its tests.

| Profile | Allows | Denies |
|---|---|---|
| `git` | `diff`, `log`, `show`, `status`, `ls-files`, `ls-tree`, `blame`, `rev-parse`, `merge-base`, `shortlog`, `describe` | every other subcommand; `--output` (writes a file), `--ext-diff`, `--textconv`, `-O` (run a program) |
| `helm` | `lint`, `template`, `version`, `dependency list` | `install`, `upgrade`, `get`, `repo`, `pull`, `dependency build` and the rest; `--validate` and `--dry-run` (ask the cluster), `--kube*` (point at one), `--output-dir`, `--post-renderer`, `--set-file`, `--repo`, key and credential flags; a remote chart or URL; a path to an env file, a key, an age key, an ssh or AWS directory, or a cleartext file under `secrets/` or `credentials/` (an `.enc.` / `.sops.` one is fine) |
| `k8s` | `kubeconform` (not `-cache`; it fetches the API schemas over the network, a read), `kustomize build` and `version`, `kube-linter lint`, `kube-score score`, `trivy config` | `kubectl` in every form (even `--dry-run=client` and `kustomize`, which can reach the API server), `helm`, `k9s`; `kustomize edit`, `create`, `cfg`, `fn`; `kustomize build` with `--enable-*` (plugins and helm run programs), `--load-restrictor`, `-o` / `--output`, `--helm-command`, `--network`, `--mount`, `--env`; a remote base (`github.com/org/repo//path`, `?ref=`, `git@`, a URL); the same argument rules as `helm` |
| `terraform` | `terraform` and `tofu`: `fmt` (only with `-check` or `-write=false`, else it rewrites the files), `validate`, `version`, with `-chdir=<dir>` as the one option before the subcommand; `tflint`; `trivy config` | `plan`, `apply`, `destroy`, `init`, `get`, `import`, `state`, `show`, `output`, `console`, `test`, `workspace`, `providers` and the rest; `tflint --init`, `--fix`, `--langserver`; `trivy` other than `config`, `-o` / `--output`, `--server`; the same argument rules as `helm` |
| `ansible` | `ansible-playbook` only with `--syntax-check`, `--list-tasks`, `--list-tags`, `--list-hosts` or `--version`; `ansible-lint` only with `--offline` (else it installs roles and collections from the network); `yamllint`; `ansible-doc` | a playbook run; `--check` / `-C` (connects to the hosts); `-i` / `--inventory` (an inventory can be a script, run to be read); vault flags, password and key flags, connection flags, `-M`; `ansible-lint --fix`, `--write` and `-r` (a rules directory is code); `ansible`, `ansible-vault`, `ansible-inventory`, `ansible-galaxy`, `ansible-config`, `ansible-pull`; an `-e @file` that names a secret |

Limit of the `terraform` and `ansible` profiles: they read what the repository configures. `terraform validate`
loads the provider plugins a previous `init` installed, and Ansible reads the repository's `ansible.cfg`, which can
point the inventory at a script or add plugin directories. That is the same trust as running the repository's
own tests; the profile cannot see it, so run the reviewers on code you would run.

For every profile:

- one simple command, with `rtk` in front when the global rtk hook rewrites it; an option before the subcommand
  (`git -c`, `helm --kube-context` can run a program or move to another target) and shell syntax (`;` `&` `|` `<`
  `>`, backticks, `$(`, new lines) are denied
- the global hooks still apply on top (`deny-sensitive-reads` stops it reading keys, `write-guard` stops writes):
  `readonly` exists because `helm template -f secrets.yaml` reads a file that the Read tool's hooks never see
- the arguments are judged by their text, so a symlink to a secret is not seen: a safety net, not a sandbox
- built by `install.sh`; until it is, the agents' Bash is blocked with "run dotfiles-sync"
- `go test ./...` in `hooks/readonly`

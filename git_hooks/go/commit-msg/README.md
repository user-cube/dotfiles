# commit-msg hook

Git hook that automatically formats commit messages with a Conventional Commit prefix and the issue key extracted from the branch name.

## Install

`dotfiles-sync` (and the playbook) copy this folder to `~/git_hooks` and build it:

```bash
go build -o commit-msg commit-msg.go
```

`.gitconfig` runs it in every repo as a config-based hook (git 2.54+), next to
each repo's own `.git/hooks`:

```ini
[hook "commit-msg-go"]
    event = commit-msg
    command = ~/git_hooks/go/commit-msg/commit-msg
```

`.gitconfig-personal` sets `enabled = false` for personal repos. To skip it in
another repo: `git config hook.commit-msg-go.enabled false`.

## Behavior

The issue key is extracted from the branch name (e.g. `features/PROJECT-123-foo` → `PROJECT-123`).

Commits on protected branches (`main`, `master`, `develop`) are blocked.

Only the first line is rewritten; the body is kept. Left unchanged:

- headers that already carry the issue key (`--amend`, rebase reword)
- `fixup!` / `squash!` / `amend!` (so `rebase --autosquash` still matches), `Merge …`, `Revert "…"`
- merges, conflicted cherry-picks and reverts (`MERGE_HEAD`, `CHERRY_PICK_HEAD`, `REVERT_HEAD`), also on protected branches
- detached HEAD outside a rebase (during a rebase the key comes from the branch being rebased)

`go test ./...` covers these cases.

### Output formats

| Input | Output |
|---|---|
| `fix something` | `fix: PROJECT-123 something` |
| `no-prefix: something` | `PROJECT-123 something` |
| `no-track: something` | `something` |

### Conventional Commit prefixes

If the message has no recognized prefix, the repo's `default_action` is used (default: `feat`).

Recognized prefixes: `feat`, `fix`, `chore`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`

### Mode keywords

- **`no-prefix:`** — skips the semantic prefix, keeps the issue key
- **`no-track:`** — bypasses everything, writes the message as-is

## Configuration

`~/.gitconfig-hook` defines per-repository behaviour using the repository's full path:

```yaml
repos:
  - path: /Users/you/projects/infrastructure-terraform
    default_action: disabled
  - path: /Users/you/projects/base
    default_action: feat
```

- If the repo is **not listed**, `feat` is used as the default action.
- If `default_action` is **`disabled`**, the hook writes the message as-is (equivalent to `no-track:`).
- Any valid Conventional Commit prefix can be used as `default_action`.

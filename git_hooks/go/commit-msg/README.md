# commit-msg hook

Git hook that automatically formats commit messages with a Conventional Commit prefix and the issue key extracted from the branch name.

## Build

```bash
go build -o commit-msg commit-msg.go
```

## Behavior

The issue key is extracted from the branch name (e.g. `features/PROJECT-123-foo` → `PROJECT-123`).

Commits on protected branches (`main`, `master`, `develop`) are blocked.

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

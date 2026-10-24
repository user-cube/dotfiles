# Global git hooks

Synced to `~/git_hooks` and run in every repo as config-based hooks (git 2.54+),
next to each repo's own `.git/hooks`. They are wired in `../.gitconfig`; skip
one once with `git commit --no-verify`, or for a repo with
`git config hook.<name>.enabled false`.

| Hook | Event | |
|---|---|---|
| `gitleaks` (`hook.gitleaks`) | pre-commit | scans the staged changes for secrets |
| `actionlint/pre-commit` (`hook.actionlint`) | pre-commit | the staged GitHub Actions workflows, as staged, through actionlint (their `run:` steps through shellcheck); nothing without actionlint |
| `gitlab-ci-lint/pre-commit` (`hook.gitlab-ci-lint`) | pre-commit | the staged `.gitlab-ci.yml` through the GitLab API (`glab ci lint`, resolves `include:` and templates), in repos whose host glab is logged in to; GitLab unreachable only warns |
| `go/commit-msg/` (`hook.commit-msg-go`) | commit-msg | Conventional Commit prefix and the issue key from the branch; blocks commits on main/master/develop. Off for personal repos (`.gitconfig-personal`). [Its README](go/commit-msg/README.md) |

The commit-msg hook is Go, built by `.config/dotfiles/install.sh`; CI runs its
`go vet` and `go test`.

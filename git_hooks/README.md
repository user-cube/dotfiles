# Global git hooks

Synced to `~/git_hooks` and run in every repo as config-based hooks (git 2.54+),
next to each repo's own `.git/hooks`. They are wired in `../.gitconfig-hooks`
(included from `.gitconfig`); skip one once with `git commit --no-verify`, or
for a repo with
`git config hook.<name>.enabled false`.

| Hook | Event | |
|---|---|---|
| `gitleaks` (`hook.gitleaks`) | pre-commit | scans the staged changes for secrets |
| `actionlint/pre-commit` (`hook.actionlint`) | pre-commit | the staged GitHub Actions workflows, as staged, through actionlint (their `run:` steps through shellcheck); nothing without actionlint |
| `gitlab-ci-lint/pre-commit` (`hook.gitlab-ci-lint`) | pre-commit | the staged `.gitlab-ci.yml` through the GitLab API (`glab ci lint`, resolves `include:` and templates), in repos whose host glab is logged in to; GitLab unreachable only warns |
| `terraform-fmt/pre-commit` (`hook.terraform-fmt`) | pre-commit | the staged `*.tf`, `*.tfvars` and `*.tftest.hcl`, as staged, through `tofu fmt` (`terraform fmt` without tofu): shows the diff of an unformatted file and stops the commit; run `tofu fmt` and stage again |
| `shellcheck/pre-commit` (`hook.shellcheck`) | pre-commit | the staged shell scripts (`*.sh`, `*.bash` or a sh / bash shebang; not zsh), as staged, through shellcheck at warning level, as CI; nothing without shellcheck |
| `hadolint/pre-commit` (`hook.hadolint`) | pre-commit | the staged Dockerfiles and Containerfiles, as staged, through hadolint (the repo's `.hadolint.yaml` applies): warnings and errors stop the commit, info and style are only shown; nothing without hadolint |
| `sops-guard/pre-commit` (`hook.sops-guard`) | pre-commit | stops a file sops should have encrypted that is staged in cleartext: named `*.sops.*` / `*.enc.*`, or matched by a `path_regex` of the repo's `.sops.yaml`; encrypted means it carries sops's MAC. Needs no sops |
| `go/commit-msg/` (`hook.commit-msg-go`) | commit-msg | Conventional Commit prefix and the issue key from the branch; blocks commits on main/master/develop. Off in my own repos (`.gitconfig-personal-github`, `.gitconfig-personal-gitlab`). [Its README](go/commit-msg/README.md) |

The commit-msg hook is Go, built by `.config/dotfiles/install.sh`; CI runs its
`go vet` and `go test`.

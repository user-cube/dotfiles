# Dotfiles

macOS configuration files. Installed by the
[devops-daily-driver](https://github.com/user-cube/devops-daily-driver) Ansible
playbook, which clones this repo to `~/dotfiles`, syncs `.config/` to
`~/.config/` and copies `.zshrc`, `.p10k.zsh` and the git configs to `~`.

## Directory Structure

- `.config/`: tool configs, synced to `~/.config/`
  - `aerospace/`: tiling window manager, plus `CHEATSHEET.md` (`⌥ i`)
  - `borders/`: JankyBorders (focused window border)
  - `sketchybar/`: bottom status bar; `plugins/icon_map.sh` (app icons) is not in git: `fetch-icon-map.sh` downloads it from the sketchybar-app-font release of the installed font; `plugins/check/` (connectivity checks: kube contexts or `~/.config/sketchybar/checks`) is a Go program built by `dotfiles-sync`
  - `ghostty/`, `tmux/`: terminal; every Ghostty window runs inside tmux
  - `nvim/`, `yazi/`: editor, file manager
  - `zsh/`: shell functions sourced by `.zshrc` (`bw`, `dotfiles-sync`, `ws`: VS Code workspaces in `~/workspaces`, `sops-on`: age keys from Bitwarden for a limited time)
  - `vscode/templates/`: settings and recommended extensions (terraform, opentofu, ansible, golang, helm) merged into a workspace by `ws -n NAME -t terraform DIR...`
  - `vscode/extensions/hue/`: Hue, a small VS Code extension that tints the window with the color of the workspace folder you are working in (`ws -n NAME DIR:blue ...`; folders without a color get one from their name, and "Hue: Set Color" changes it; the named colors, the strength and which parts are tinted are `hue.*` settings; `DIR!` (all of it) or `DIR!hetzner/prod` (only that path) marks production with a red title bar, a PROD marker and a red terminal tab, the status bar turns amber on `main`/`master`, and "Hue: Open Terminal in Folder" (⌘⌥T) opens a terminal with a tab in the folder's color, "Go to Folder" jumps to a folder's last file, "Clear" removes the colors and turns it off); installed by `dotfiles-sync` through `install.sh`, no marketplace
  - `ssh/`: ssh defaults (ControlMaster), included from the top of the per-machine ssh config by the playbook
  - `git/ignore`: global gitignore
- `.zshrc`, `.p10k.zsh`: Zsh with Oh My Zsh and Powerlevel10k
- `.gitconfig*`: git config, with personal/work identities
- `git_hooks/`: global git hooks, wired in `.gitconfig`: commit-msg (Go, `hook.commit-msg-go`) and two pre-commit checks of the staged CI config: actionlint for GitHub Actions workflows (`hook.actionlint`) and `glab ci lint` for `.gitlab-ci.yml`, through the GitLab API, in repos whose host glab is logged in to (`hook.gitlab-ci-lint`)
- `claude/`: Claude Code config, synced to `~/.claude/`: `settings.json` (no co-author trailer, deny rules, hooks, plugins), global `CLAUDE.md`, and `hooks/deny-sensitive-reads/` (a Go PreToolUse hook built by `dotfiles-sync` and the playbook: blocks Bash commands that touch cleartext credentials, and reads of files under `secrets/`/`credentials/` that sops does not report as encrypted; while it is not built, Claude Code's Bash and Read are blocked). Not `.claude/`: Claude Code would read that as this repo's project settings. Change these here, not with `/config`: a sync overwrites `~/.claude/settings.json`

## Zsh

`.zshrc` is copied to `~/.zshrc`, overwriting it.
Never edit `~/.zshrc` by hand: put anything specific to one machine (work-only
tools, aliases) in `~/.zshrc.local`, which is sourced at the end and
never committed.

Shared shell functions live in `.config/zsh/*.zsh` and are all sourced before
`~/.zshrc.local`, e.g. `bw.zsh` (Bitwarden CLI session wrapper; needs
`export BW_USER=<email>` in `~/.zshrc.local`).

Startup is kept fast: nvm loads on first use, `pyenv init` skips the rehash
(run `pyenv rehash` after installing a package with commands), and completions
are cached in `~/.cache/zsh` (regenerated when the tool is upgraded; delete the
folder to force it). Measure with `time zsh -i -c exit`.

## SOPS age keys

The keys for SOPS-encrypted files live in one Bitwarden secure note, `SOPS age keys`
(a plain age keys file in the notes field), never on disk by default.
`sops-on [60m]` writes them to sops's default keys file for a limited time (also
what the VS Code SOPS extension needs), then removes them with a notice at the
next prompt and a macOS notification. `sops-off` removes them now, `sops-status`
shows the time left, and `sops-save FILE` puts a keys file in the note. A keys file
that `sops-on` did not create is never overwritten or removed. Default time:
`SOPS_ON_MINUTES`, else 60. See `.config/zsh/sops.zsh`.

## Secrets (tokens, API keys)

Secrets never go in this repo, and preferably not in `~/.zshrc.local` either:
plain-text files end up in backups, screen shares and shell history. Bitwarden
is the source of truth; the shell fetches a secret only when the tool that needs
it is started, so it never sits in a file or in every shell's environment.

1. Store the secret in Bitwarden: one item per service, the secret in the
   password field (or in custom fields when there are several).
2. Wrap the command that needs it in a function in `~/.zshrc.local` (per
   machine) or `.config/zsh/<name>.zsh` (shared, as long as it holds no secret
   and no machine-specific item names):

   ```zsh
   # Secrets live only in this process, not in the shell environment
   mytool() {
     local token
     token=$(bw get password <item>) || return
     MYTOOL_TOKEN=$token command mytool "$@"
   }
   ```

   With several values in one item:

   ```zsh
   bw get item <item> | jq -r '.fields[] | select(.name == "<field>").value'
   ```

`bw` is the wrapper in `.config/zsh/bw.zsh`: the first call per shell asks for
Touch ID/sudo to read the session key, later calls reuse it. Never fetch secrets
at the top level of `.zshrc*`: every new shell would block on the vault.

A token needed once on the command line (`claude mcp add --header ...`,
`gh auth login --with-token`): `secret T` asks for it without echoing it
(`secret T <item>` takes it from Bitwarden), then use `"$T"` in the next command;
it is cleared right after and never exported (`.config/zsh/secrets.zsh`). Lines
that look like they hold a token (GitHub/GitLab/Slack/OpenAI keys, AWS access
keys, JWTs, a long `Bearer`, `--password=...`) are not saved to the history, and
a leading space keeps any command out of it.

Check nothing leaked: `git grep -nIE '(token|secret|password|api_?key)\s*='`
before committing (gitleaks also runs as a pre-commit hook).

## Per-machine files (never committed)

| File | Purpose |
|---|---|
| `~/.zshrc.local` | machine-specific shell config (secrets: see [Secrets](#secrets-tokens-api-keys)) |
| `~/.gitconfig-hook` | optional: per-repo `default_action` of the commit-msg hook (see `git_hooks/go/commit-msg/README.md`) |
| `~/.config/nvim/lua/plugins_local/*.lua` | optional: nvim plugins only for this machine (lazy.nvim specs, e.g. vim-be-good) |
| `~/.config/tmux/sessions.local` | repos preloaded as tmux sessions (see `.config/tmux/sessions.local.example`; `tmux-sessions` after editing) |
| `~/.config/sketchybar/aws_profiles` | AWS profiles shown in the bar |
| `~/workspaces/*.code-workspace` | VS Code workspaces, created and opened with `ws` |
| `~/.config/sketchybar/checks` | optional: connectivity checks shown in the bar, instead of every kube context (see `checks.example`) |

### Backup in Bitwarden

These files, plus `~/.gitconfig-work`, `~/.kube/config`, `~/.terraformrc` and `~/.terraform.d/credentials.tfrc.json` (`terraform login` tokens), are backed up as one
`.tar.gz` attached to a Bitwarden secure note `Local files (<machine>)`
(`.config/dotfiles/local-files/`, a Go tool built by `dotfiles-sync` and the
playbook; uses the `bw` wrapper's session).
Claude Code's MCP servers (their tokens sit in `env`) go too: only the
`mcpServers` key of `~/.claude.json`, which also holds the login and per-project
state. Restore merges them back; a server already set up on the machine is kept
unless `--force`:

```sh
local-backup                 # this machine -> Bitwarden (replaces the previous copy)
local-restore                # Bitwarden -> this machine; existing files are kept
local-restore other-mac      # restore another machine's backup (e.g. a new Mac)
local-restore --force        # overwrite existing files
```

On a new Mac the playbook does the same with `--tags local-restore`.

## Updating without Ansible

```sh
dotfiles-sync
```

Defined in `.config/zsh/dotfiles.zsh`: pulls `~/dotfiles` (or `$DOTFILES_DIR`),
syncs `.config/` (building `local-files` and the SketchyBar checks, downloading the SketchyBar icon map), `git_hooks/` (building the commit-msg hook) and `claude/` (to `~/.claude/`, building its hook), copies `.zshrc`, `.p10k.zsh` and the git configs
(`.gitconfig-work` only when missing), removes the files the repo no longer installs
(`.config/dotfiles/obsolete`: rsync never deletes; add a line when a file leaves the repo), then reloads tmux, AeroSpace, SketchyBar and
borders. Run `exec zsh` afterwards; Ghostty keybind changes need ⌘Q.

## CI

`.github/workflows/lint.yml` runs on macOS (for `/bin/bash` 3.2): `bash -n` and
`shellcheck` on every shell script, `zsh -n` on the zsh files, `go vet` and `go test`
on the commit-msg hook, `local-files`, the SketchyBar checks and the Claude Code hook, and `gitleaks` over
the whole history. Run the same checks locally before pushing:

```sh
git ls-files -z | xargs -0 grep -lIE '^#!.*\b(ba)?sh\b' | xargs shellcheck -x -S warning
(cd git_hooks/go/commit-msg && go vet ./... && go test ./...)
(cd .config/dotfiles/local-files && go vet ./... && go test ./...)
(cd .config/sketchybar/plugins/check && go vet ./... && go test ./...)
(cd claude/hooks/deny-sensitive-reads && go vet ./... && go test ./...)
gitleaks git --redact --no-banner .
```

# Dotfiles

macOS configuration files. Installed by the
[devops-daily-driver](https://github.com/user-cube/devops-daily-driver) Ansible
playbook, which clones this repo to `~/dotfiles`, syncs `.config/` to
`~/.config/` and copies `.zshrc`, `.p10k.zsh` and the git configs to `~`.

## Directory Structure

- `.config/`: tool configs, synced to `~/.config/`
  - `aerospace/`: tiling window manager, plus `CHEATSHEET.md` (`⌥ i`)
  - `borders/`: JankyBorders (focused window border)
  - `sketchybar/`: bottom status bar
  - `ghostty/`, `tmux/`: terminal; every Ghostty window runs inside tmux
  - `nvim/`, `yazi/`, `cmux/`: editor, file manager, cmux
  - `zsh/`: shell functions sourced by `.zshrc` (`bw`, `dotfiles-sync`)
  - `ssh/`: ssh defaults (ControlMaster), included from the top of the per-machine ssh config by the playbook
  - `git/ignore`: global gitignore
- `.zshrc`, `.p10k.zsh`: Zsh with Oh My Zsh and Powerlevel10k
- `.gitconfig*`: git config, with personal/work identities
- `git_hooks/`: commit-msg hooks (bash, go)
- `vscode/`: VS Code snippets/settings (golang, helmchart, terraform)

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

Check nothing leaked: `git grep -nIE '(token|secret|password|api_?key)\s*='`
before committing (gitleaks also runs as a pre-commit hook).

## Per-machine files (never committed)

| File | Purpose |
|---|---|
| `~/.zshrc.local` | machine-specific shell config (secrets: see [Secrets](#secrets-tokens-api-keys)) |
| `~/.config/tmux/sessions.local` | repos preloaded as tmux sessions |
| `~/.config/sketchybar/aws_profiles` | AWS profiles shown in the bar |
| `~/.config/sketchybar/checks` | optional: connectivity checks shown in the bar, instead of every kube context (see `checks.example`) |

### Backup in Bitwarden

These files, plus `~/.gitconfig-work` and `~/.kube/config`, are backed up as one
`.tar.gz` attached to a Bitwarden secure note `Local files (<machine>)`
(`.config/dotfiles/local-files.sh`; uses the `bw` wrapper's session):

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
syncs `.config/` and `git_hooks/`, copies `.zshrc`, `.p10k.zsh` and the git configs
(`.gitconfig-work` only when missing), then reloads tmux, AeroSpace, SketchyBar and
borders. Run `exec zsh` afterwards; Ghostty keybind changes need ⌘Q.

## CI

`.github/workflows/lint.yml` runs on macOS (for `/bin/bash` 3.2): `bash -n` and
`shellcheck` on every shell script, `zsh -n` on the zsh files, and `gitleaks` over
the whole history. Run the same checks locally before pushing:

```sh
git ls-files -z | xargs -0 grep -lIE '^#!.*\b(ba)?sh\b' | xargs shellcheck -x -S warning
gitleaks git --redact --no-banner .
```

# Dotfiles

macOS configuration files. Installed by the
[devops-daily-driver](https://github.com/user-cube/devops-daily-driver) Ansible
playbook, which clones this repo to `~/dotfiles`, syncs `.config/` to
`~/.config/` and copies `.zshrc`, `.p10k.zsh` and the git configs to `~`.

## Directory Structure

The folders with more than a config file have their own README.

- `.config/`: tool configs, synced to `~/.config/`
  - `aerospace/`: tiling window manager, plus `CHEATSHEET.md` (`⌥ i`)
  - `borders/`: JankyBorders (focused window border)
  - [`sketchybar/`](.config/sketchybar/README.md): bottom status bar; its plugins, and the per-machine `aws_profiles` and `checks`
  - [`ghostty/`](.config/ghostty/README.md), [`tmux/`](.config/tmux/README.md): terminal; every Ghostty window runs inside tmux, and the ⌘ shortcuts are tmux's (the tmux README lists them, and how to copy)
  - [`nvim/`](.config/nvim/README.md), `yazi/`: editor, file manager
  - `kubectl/complete/`: Tab completion for the krew plugins (`kubectl_complete-<plugin>`, which kubectl runs), mostly through `kubectl get`'s own
  - [`k9s/`](.config/k9s/README.md): the Kubernetes TUI's config, plugins (stern, `kubectl neat`, dive) and read-only on production
  - [`zsh/`](.config/zsh/README.md): shell functions sourced by `.zshrc` (`bw`, `secret`, `sops-on`, `dotfiles-sync`, `local-backup`, `ws`, `plugins-outdated`)
  - [`vscode/`](.config/vscode/README.md): workspace templates for `ws`, and Hue, a small extension that colors the window per folder (red for production)
  - [`dotfiles/`](.config/dotfiles/README.md): `install.sh` (the steps after a sync), the obsolete files list, `local-files`
  - `ssh/`: ssh defaults (ControlMaster), included from the top of the per-machine ssh config by the playbook
  - `git/ignore`: global gitignore: macOS and editor files, tool state (`.direnv/`, `.terraform/`), and env files as a safety net (`.env`, `.env.*`, but not `.env.example`; `git add -f` overrides)
- `.zshrc`, `.p10k.zsh`: Zsh with Oh My Zsh and Powerlevel10k
- `.gitconfig*`: git config, with personal/work identities. Defaults: `pull` rebases (autostash), `fetch` prunes, conflicts in `zdiff3` with `rerere`, histogram diffs with moved lines colored, branches and tags newest first, `main` for new repos
- [`git_hooks/`](git_hooks/README.md): global git hooks, wired in `.gitconfig`: gitleaks, actionlint and `glab ci lint` on the staged CI config, `tofu fmt` on the staged Terraform files, the Go commit-msg hook
- [`claude/`](claude/README.md): Claude Code config, synced to `~/.claude/` (not `.claude/`: Claude Code would read that as this repo's project settings). Change these here, not with `/config`: a sync overwrites `~/.claude/settings.json`
  - `settings.json`: no co-author trailer, deny rules, hooks, plugins
  - `hooks/deny-sensitive-reads/`: a Go PreToolUse hook that blocks Bash commands and reads of what is really private (keys, credentials, env files); while it is not built, Claude Code's Bash and Read are blocked

## Zsh

`.zshrc` is copied to `~/.zshrc`, overwriting it.
Never edit `~/.zshrc` by hand: put anything specific to one machine (work-only
tools, aliases) in `~/.zshrc.local`, which is sourced at the end and
never committed.

Shared shell functions live in `.config/zsh/*.zsh` and are all sourced before
`~/.zshrc.local`, e.g. `bw.zsh` (Bitwarden CLI session wrapper; needs
`export BW_USER=<email>` in `~/.zshrc.local`).

Startup is kept around 0.1 s: nvm loads on first use, `pyenv init` skips the rehash
(run `pyenv rehash` after installing a package with commands), and the
pyenv/zoxide/fzf inits and the completions are cached in `~/.cache/zsh`
(regenerated when the tool is upgraded or its command line changes; delete the
folder to force it). Measure with `time zsh -i -c exit`.

Day to day (details in the [zsh README](.config/zsh/README.md)):

- Tab completes in an fzf list (fzf-tab), hidden files included
- Ctrl+T picks a file, Ctrl+G a folder, Ctrl+R a history entry, with previews
- `cd` is zoxide: `cd dotfiles` from anywhere, `cdi` to pick
- production clusters: `kubectl` / `k` / `helm` ask before a command that changes one, and k9s opens read-only (`kube-guard.zsh`)
- Esc Esc adds `sudo`, Ctrl+O copies the line, `copypath`, `extract`, `ofd` / `cdf` (Finder)
- the Oh My Zsh plugins (zsh-autosuggestions, fzf-tab, zsh-syntax-highlighting)
  are cloned at pinned commits by the playbook; `plugins-outdated` shows which
  ones master has moved past, and every 14 days a notification says so
- p10k marks each prompt and command output, so tmux can jump between commands
  and copy one's output (⌘⇧↑ / ⌘⇧↓, ⌘⇧C: see the [tmux README](.config/tmux/README.md))

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

These files (but not the nvim plugins or the workspaces), plus `~/.gitconfig-work`, `~/.kube/config`, `~/.terraformrc` and `~/.terraform.d/credentials.tfrc.json` (`terraform login` tokens), are backed up as one
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
syncs `.config/`, `git_hooks/` and `claude/` (to `~/.claude/`), copies `.zshrc`,
`.p10k.zsh` and the git configs (`.gitconfig-work` only when missing), runs
`.config/dotfiles/install.sh`, then reloads tmux, AeroSpace, SketchyBar and borders.
Run `exec zsh` afterwards; Ghostty keybind changes need ⌘Q.

To see first what it would change, changing nothing:

```sh
dotfiles-diff      # files it would overwrite (changed here, or older), add, or remove (obsolete)
dotfiles-diff -p   # and their diffs: - is this machine, + what the sync brings
```

It compares with the upstream `dotfiles-sync` would pull (fetched, not merged), and
says how many commits the clone is behind.

`install.sh` holds the steps after the files are in place, the same for
`dotfiles-sync` and the playbook (a new step goes there, in one repo):

- builds the Go tools when their sources changed: the commit-msg hook,
  `local-files`, the SketchyBar checks and the Claude Code hook
- downloads the SketchyBar icon map for the installed font version
- installs the Hue VS Code extension when it changed
- removes the files the repo no longer installs (`.config/dotfiles/obsolete`:
  rsync never deletes; add a line when a file leaves the repo)

## CI

`.github/workflows/lint.yml` runs on macOS (for `/bin/bash` 3.2): `bash -n` and
`shellcheck` on every shell script (unquoted variables included: where splitting is
wanted, use an array), `zsh -n` on the zsh files, `jq` on every JSON
file (a broken `claude/settings.json` would break Claude Code), nvim started headless
with the plugins at `lazy-lock.json` (any output on startup is an error), `go vet` and `go test`
on the commit-msg hook, `local-files`, the SketchyBar checks and the Claude Code hook,
`.github/scripts/test-install.sh` (`dotfiles-sync` into an empty fake HOME: everything
installed, a second run that changes nothing, the built tools working, `dotfiles-diff`), and
`gitleaks` over the whole history. Run the same checks locally before pushing:

```sh
git ls-files -z | xargs -0 grep -lIE '^#!.*\b(ba)?sh\b' | xargs shellcheck -x -S warning
git ls-files -z | xargs -0 grep -lIE '^#!.*\b(ba)?sh\b' | xargs shellcheck -x -S info -i SC2086
(cd git_hooks/go/commit-msg && go vet ./... && go test ./...)
(cd .config/dotfiles/local-files && go vet ./... && go test ./...)
(cd .config/sketchybar/plugins/check && go vet ./... && go test ./...)
(cd claude/hooks/deny-sensitive-reads && go vet ./... && go test ./...)
.github/scripts/test-install.sh
gitleaks git --redact --no-banner .
```

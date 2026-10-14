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
  - `zsh/`: shell functions sourced by `.zshrc`
- `.zshrc`, `.p10k.zsh`: Zsh with Oh My Zsh and Powerlevel10k
- `.gitconfig*`: git config, with personal/work identities
- `git_hooks/`: commit-msg hooks (bash, go)
- `vscode/`: VS Code snippets/settings (golang, helmchart, terraform)
- `.warp/themes/`: Warp themes

## Zsh

`.zshrc` is copied to `~/.zshrc`, overwriting it.
Never edit `~/.zshrc` by hand: put anything specific to one machine (work-only
tools, tokens, aliases) in `~/.zshrc.local`, which is sourced at the end and
never committed.

Shared shell functions live in `.config/zsh/*.zsh` and are all sourced before
`~/.zshrc.local`, e.g. `bw.zsh` (Bitwarden CLI session wrapper; needs
`export BW_USER=<email>` in `~/.zshrc.local`).

Startup is kept fast: nvm loads on first use, `pyenv init` skips the rehash
(run `pyenv rehash` after installing a package with commands), and completions
are cached in `~/.cache/zsh` (regenerated when the tool is upgraded; delete the
folder to force it). Measure with `time zsh -i -c exit`.

## Per-machine files (never committed)

| File | Purpose |
|---|---|
| `~/.zshrc.local` | machine-specific shell config and secrets |
| `~/.config/tmux/sessions.local` | repos preloaded as tmux sessions |
| `~/.config/sketchybar/aws_profiles` | AWS profiles shown in the bar |

## Updating without Ansible

```sh
git -C ~/dotfiles pull
rsync -a ~/dotfiles/.config/ ~/.config/
cp ~/dotfiles/.zshrc ~/.zshrc
```

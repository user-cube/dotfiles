# Dotfiles

macOS configuration: zsh, Ghostty + tmux (or Rex), AeroSpace, SketchyBar,
Neovim, VS Code, Kubernetes guardrails, secrets kept in Bitwarden, global git
hooks and Claude Code.

**The guide is the [wiki](https://github.com/user-cube/dotfiles/wiki)**:
installing and updating, the shell, Kubernetes, secrets, git, development and
troubleshooting. The READMEs in the folders hold the reference tables
(shortcuts, plugins, items).

## Install and update

A new Mac: the [devops-daily-driver](https://github.com/user-cube/devops-daily-driver)
Ansible playbook clones this repo to `~/dotfiles` and installs it. Then, on any
machine:

```sh
dotfiles-diff -p   # what a sync would change, changing nothing
dotfiles-sync      # pull and install; then `exec zsh`
```

Never edit the installed files: what is specific to one machine goes in files
that are never committed (`~/.zshrc.local` and the others in
[the wiki](https://github.com/user-cube/dotfiles/wiki/Installing-and-updating#per-machine-files)).

## Layout

- `.config/`: tool configs, synced to `~/.config/`
  - [`aerospace/`](.config/aerospace/CHEATSHEET.md), `borders/`, [`sketchybar/`](.config/sketchybar/README.md): the desktop
  - [`ghostty/`](.config/ghostty/README.md), [`tmux/`](.config/tmux/README.md), `rex/`: the terminal
  - [`nvim/`](.config/nvim/README.md), [`vscode/`](.config/vscode/README.md), `lazygit/`, `yazi/`, `bat/`: editors and viewers
  - [`kube-guard/`](.config/kube-guard), [`k9s/`](.config/k9s/README.md), [`sofka/`](.config/sofka/README.md), `kubectl/`: Kubernetes
  - [`zsh/`](.config/zsh/README.md): the shell functions `.zshrc` sources
  - [`dotfiles/`](.config/dotfiles/README.md): `install.sh`, the lists of files copied to `~` and of obsolete ones, the LaunchAgents, `local-files`
  - `git/`, `ssh/`: global gitignore, allowed signers, ssh defaults
- `.zshrc`, `.p10k.zsh`, `.gitconfig*`: copied to `~` (`.config/dotfiles/home-files`)
- [`git_hooks/`](git_hooks/README.md): the global git hooks
- [`claude/`](claude/README.md): Claude Code, synced to `~/.claude/`
- [`tests/`](tests/README.md): pytest, run by CI

## Before pushing

CI (`.github/workflows/lint.yml`, on macOS) runs shellcheck, bash 3.2 and zsh
syntax, JSON, nvim, pytest, Hue's and the Go tools' tests, and gitleaks. The same
locally: [Development](https://github.com/user-cube/dotfiles/wiki/Development#tests).

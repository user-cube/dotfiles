# VS Code

Used through `ws` (`../zsh/vscode-workspaces.zsh`): workspaces live in
`~/workspaces/*.code-workspace` (per machine, never committed).

```sh
ws                                   # pick a workspace with fzf
ws NAME                              # open NAME.code-workspace
ws -n NAME -t terraform DIR:blue ... # create one from folders, then open it
```

- `settings.json`, `keybindings.json`: the user settings.
  `../dotfiles/install.sh` links `~/Library/Application Support/Code/User` to
  them in the clone itself (`DOTFILES_DIR`, else `~/dotfiles`), not the copy
  in `~/.config`: a setting changed in VS Code's UI lands in the repo, shows
  in `git status` to commit, and no `dotfiles-sync` copies over it (a file
  VS Code had there before is kept as `.bak`). Plain JSON, no comments: the
  tests parse it
  - formatting is global, the same in every workspace: format on save, 2
    spaces (4 for Python), LF, final newline, no trailing whitespace (kept in
    Markdown), gofmt + organized imports for Go, Red Hat's formatter for YAML
- `extensions.txt`: the extensions `install-extensions.sh` (run by
  `../dotfiles/install.sh`) installs when missing; it never uninstalls, so
  add one you installed by hand to have it on every machine
  (`code --list-extensions` prints the ids)
- the integrated terminal is VS Code's own, plain zsh: no tmux there (tabs,
  splits and ⌘ shortcuts are VS Code's). Same font and cursor as Ghostty:
  JetBrainsMono Nerd Font 14 (weight 500 stands in for `font-thicken`),
  blinking bar cursor, copy on select; the colours stay the VS Code theme's
- VS Code's ⌘K = Terminal: Clear is taken out, so chords that start with ⌘K
  (⌘K H Hue: Search, ⌘K ⌘S Hue's jump between workspace folders) work with
  the terminal focused too; clear it with C-l or `clear`
- Shift+Enter in the terminal sends ESC + Enter (Alt+Enter): VS Code would send
  a plain Enter, and Ghostty's extended Shift+Enter doesn't exist in VS Code.
  Claude Code (and zsh) take it as a newline
- `templates/`: settings (tooling only: linters, schemas, file associations;
  never formatting) and recommended extensions `ws -n ... -t TPL` merges
  into a new workspace: `ansible`, `golang`, `helm`, `opentofu`, `terraform`;
  repeat `-t` to combine
- `extensions/hue/`: Hue, a small extension outside the marketplace, installed
  by `install.sh` when it changes ([its README](extensions/hue/README.md))
  - the window takes the color of the folder you work in (`DIR:COLOR`, or one
    from the folder's name)
  - production: `DIR!` (all of it) or `DIR!hetzner/prod` (that path only)
    gives a red title bar, a PROD marker and a red terminal tab
  - the status bar turns amber on `main` / `master`

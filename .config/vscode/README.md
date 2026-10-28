# VS Code

Used through `ws` (`../zsh/vscode-workspaces.zsh`): workspaces live in
`~/workspaces/*.code-workspace` (per machine, never committed).

```sh
ws                                   # pick a workspace with fzf
ws NAME                              # open NAME.code-workspace
ws -n NAME -t terraform DIR:blue ... # create one from folders, then open it
```

- `settings.json`, `keybindings.json`: the user settings, linked into
  `~/Library/Application Support/Code/User` by `../dotfiles/install.sh` (a file
  VS Code had there is kept as `.bak`). Plain JSON, no comments: the tests
  parse it
  - formatting is global, the same in every workspace: format on save, 2
    spaces (4 for Python), LF, final newline, no trailing whitespace (kept in
    Markdown), gofmt + organized imports for Go, Red Hat's formatter for YAML
- `extensions.txt`: the extensions `install-extensions.sh` (run by
  `../dotfiles/install.sh`) installs when missing; it never uninstalls, so
  add one you installed by hand to have it on every machine
  (`code --list-extensions` prints the ids)
- `tmux.sh`: the default terminal profile (`tmux`), so the integrated terminal
  runs tmux with the same `tmux.conf` as Ghostty
  - one session per workspace folder, named like ⌘O names it; a second VS Code
    terminal on it gets a throwaway session
  - starts tmux through `../tmux/start-server.sh` when it isn't running, so the
    tmux-resurrect save is restored whoever opens first
  - the other profiles stay in the terminal's + menu
  - same font and cursor as Ghostty: JetBrainsMono Nerd Font 14 (weight 500
    stands in for `font-thicken`), blinking bar cursor, copy on select; the
    colours stay the VS Code theme's
- ⌘ shortcuts in the terminal send the tmux prefix like Ghostty's (only while
  the terminal has focus; `tests/test_shortcuts.py` checks they match). Left
  to VS Code: ⌘J (panel), ⌘K (chords, e.g. ⌘K ⌘S), ⌘O, ⌘⇧E, ⌘⇧F; in tmux use
  `C-b K`, `C-b f`, `C-b E`, `C-b z`. ⌘K ⌘S: Hue's jump between workspace folders.
  VS Code's ⌘K = Terminal: Clear is taken out, so chords that start with ⌘K
  (⌘K H Hue: Search, ⌘K ⌘S) work with the terminal focused too; clear tmux
  with C-l or `clear`
- Shift+Enter in the terminal sends ESC + Enter (Alt+Enter): VS Code would send
  a plain Enter, and Ghostty's extended Shift+Enter doesn't exist in VS Code.
  Through tmux it reaches the app, a newline in Claude Code (and in zsh)
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

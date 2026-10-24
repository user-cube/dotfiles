# Ghostty

The terminal. Tabs, splits and sessions come from tmux, not from Ghostty: its
native tabs are separate macOS windows, which AeroSpace would tile one by one.

- `config`: font, Monokai Soda theme, and the ⌘ shortcuts, which send the tmux
  prefix (`\x02` = `C-b`) plus a key; the list is in [the tmux README](../tmux/README.md)
- `tmux-start.sh`: Ghostty's `command`, so every window starts inside tmux
  - when tmux isn't running yet, restores the last tmux-resurrect save, then
    preloads `~/.config/tmux/sessions.local`
  - the first window attaches the persistent `main` session (survives closing
    Ghostty); further windows get a throwaway session, gone with the window
  - without tmux, a plain login shell

Other settings: the quick terminal on ctrl+\` (anywhere), non-native fullscreen
(no separate Space, for AeroSpace), the `tip` update channel.

Reload with ⌘⇧, (`dotfiles-sync` does not reload Ghostty); a changed keybind
may need ⌘Q.

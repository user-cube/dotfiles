#!/bin/bash
# Make sure the tmux server runs with the "main" session, whoever starts it
# (ghostty/tmux-start.sh, vscode/tmux.sh). When tmux isn't running yet: restore
# the last tmux-resurrect save (sessions, tabs, splits, folders) first, so
# tmux-continuum never autosaves a fresh, empty server over it; then preload
# the sessions in ~/.config/tmux/sessions.local (preload-sessions.sh).
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

RESTORE="$HOME/.config/tmux/plugins/tmux-resurrect/scripts/restore.sh"
SAVE="${XDG_DATA_HOME:-$HOME/.local/share}/tmux/resurrect/last"

tmux has-session -t main 2>/dev/null && exit 0

if ! tmux list-sessions >/dev/null 2>&1 && [ -e "$SAVE" ] && [ -x "$RESTORE" ]; then
  # Fresh server: restore into a scratch session, which becomes "main" if the
  # save didn't have one
  tmux new-session -d -s __restore
  tmux run-shell "$RESTORE"
  if tmux has-session -t main 2>/dev/null; then
    tmux kill-session -t __restore
  else
    tmux rename-session -t __restore main
  fi
fi
tmux has-session -t main 2>/dev/null || tmux new-session -d -s main

"$HOME/.config/tmux/preload-sessions.sh"

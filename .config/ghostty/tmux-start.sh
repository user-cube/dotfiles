#!/bin/bash
# Ghostty's `command`: start every window inside tmux.
# - When tmux isn't running yet: restore the last tmux-resurrect save (sessions,
#   tabs, splits, folders), then preload the sessions in ~/.config/tmux/sessions.local
#   (tmux/preload-sessions.sh).
# - First window attaches the persistent "main" session (survives closing Ghostty).
# - Further windows get their own throwaway session, destroyed when the window closes.
# - Without tmux, fall back to a plain login shell so Ghostty never just closes.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

command -v tmux >/dev/null || exec "${SHELL:-/bin/zsh}" -l

RESTORE="$HOME/.config/tmux/plugins/tmux-resurrect/scripts/restore.sh"
SAVE="${XDG_DATA_HOME:-$HOME/.local/share}/tmux/resurrect/last"

if ! tmux has-session -t main 2>/dev/null; then
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
  exec tmux attach-session -t main
elif [ -z "$(tmux list-clients -t main 2>/dev/null)" ]; then
  exec tmux attach-session -t main
else
  exec tmux new-session \; set-option destroy-unattached on
fi

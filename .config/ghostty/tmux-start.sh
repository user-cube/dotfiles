#!/bin/bash
# Ghostty's `command`: start every window inside tmux.
# - First window attaches the persistent "main" session (survives closing Ghostty).
# - Further windows get their own throwaway session, destroyed when the window closes.
# - Without tmux, fall back to a plain login shell so Ghostty never just closes.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

command -v tmux >/dev/null || exec "${SHELL:-/bin/zsh}" -l

if ! tmux has-session -t main 2>/dev/null; then
  exec tmux new-session -s main
elif [ -z "$(tmux list-clients -t main 2>/dev/null)" ]; then
  exec tmux attach-session -t main
else
  exec tmux new-session \; set-option destroy-unattached on
fi

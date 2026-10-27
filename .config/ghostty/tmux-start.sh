#!/bin/bash
# Ghostty's `command`: start every window inside tmux.
# - tmux/start-server.sh starts tmux when it isn't running (restoring the last
#   tmux-resurrect save and preloading ~/.config/tmux/sessions.local).
# - First window attaches the persistent "main" session (survives closing Ghostty).
# - Further windows get their own throwaway session, destroyed when the window closes.
# - Without tmux, fall back to a plain login shell so Ghostty never just closes.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

command -v tmux >/dev/null || exec "${SHELL:-/bin/zsh}" -l

"$HOME/.config/tmux/start-server.sh"

if [ -z "$(tmux list-clients -t main 2>/dev/null)" ]; then
  exec tmux attach-session -t main
else
  exec tmux new-session \; set-option destroy-unattached on
fi

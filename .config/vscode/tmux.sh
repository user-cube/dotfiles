#!/bin/bash
# VS Code's terminal profile (settings.json, "tmux"): the integrated terminal
# runs tmux with the same tmux.conf as Ghostty.
# - tmux/start-server.sh starts tmux when it isn't running, as Ghostty does.
# - One session per workspace folder, named after it like ⌘O does
#   (tmux/sessionizer.sh), so ⌘O on the same repo lands in the same session.
# - A second VS Code terminal while that session is shown gets its own
#   throwaway session, destroyed when the terminal closes.
# - Without tmux, fall back to a plain login shell.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

command -v tmux >/dev/null || exec "${SHELL:-/bin/zsh}" -l

"$HOME/.config/tmux/start-server.sh"

NAME=$(basename "$PWD" | tr '.:' '__')
if ! tmux has-session -t="$NAME" 2>/dev/null; then
  exec tmux new-session -s "$NAME" -c "$PWD"
elif [ -z "$(tmux list-clients -t="$NAME" 2>/dev/null)" ]; then
  exec tmux attach-session -t="$NAME"
else
  exec tmux new-session -c "$PWD" \; set-option destroy-unattached on
fi

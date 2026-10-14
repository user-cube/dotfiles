#!/bin/bash
# Ghostty's `command`: start every window inside tmux.
# - First window attaches the persistent "main" session (survives closing Ghostty).
#   When tmux isn't running yet, the repos listed in ~/.config/tmux/sessions.local
#   (one path per line, per machine, not in dotfiles) are preloaded as sessions.
# - Further windows get their own throwaway session, destroyed when the window closes.
# - Without tmux, fall back to a plain login shell so Ghostty never just closes.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

command -v tmux >/dev/null || exec "${SHELL:-/bin/zsh}" -l

if ! tmux has-session -t main 2>/dev/null; then
  tmux new-session -d -s main
  LIST="$HOME/.config/tmux/sessions.local"
  if [ -f "$LIST" ]; then
    grep -vE '^\s*(#|$)' "$LIST" | while read -r dir; do
      dir="${dir/#\~/$HOME}"
      [ -d "$dir" ] || continue
      name=$(basename "$dir" | tr '.:' '__')
      tmux has-session -t="$name" 2>/dev/null || tmux new-session -d -s "$name" -c "$dir"
    done
  fi
  exec tmux attach-session -t main
elif [ -z "$(tmux list-clients -t main 2>/dev/null)" ]; then
  exec tmux attach-session -t main
else
  exec tmux new-session \; set-option destroy-unattached on
fi

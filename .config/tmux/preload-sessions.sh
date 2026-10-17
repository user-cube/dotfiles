#!/bin/bash
# Create the sessions listed in ~/.config/tmux/sessions.local (per machine, not
# in dotfiles; see sessions.local.example). Run by ghostty/tmux-start.sh when
# tmux starts, and by hand (`tmux-sessions`) after editing the file: closing
# Ghostty leaves tmux running, so the file isn't read again until a reboot.
# Sessions and windows that already exist are left alone.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

LIST="$HOME/.config/tmux/sessions.local"
[ -f "$LIST" ] || exit 0

# "name=path" -> session <name>; repeat a name to add the path as another
# window of that session. A bare "path" -> session named after the folder.
grep -vE '^\s*(#|$)' "$LIST" | while IFS= read -r line; do
  if [[ "$line" == *=* ]]; then name="${line%%=*}"; dir="${line#*=}"
  else name=""; dir="$line"; fi
  name=$(echo "$name" | xargs); dir=$(echo "$dir" | xargs)
  dir="${dir/#\~/$HOME}"
  [ -d "$dir" ] || { echo "skipped, no such folder: $dir" >&2; continue; }
  [ -n "$name" ] || name=$(basename "$dir" | tr '.:' '__')
  wname=$(basename "$dir" | tr '.:' '__')
  if ! tmux has-session -t="$name" 2>/dev/null; then
    tmux new-session -d -s "$name" -n "$wname" -c "$dir"
  elif ! tmux list-panes -s -t="$name" -F '#{pane_current_path}' | grep -qxF "$dir"; then
    tmux new-window -d -t="$name": -n "$wname" -c "$dir"
  fi
done

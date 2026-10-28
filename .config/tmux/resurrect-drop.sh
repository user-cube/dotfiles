#!/bin/bash
# tmux-resurrect's post-save-layout hook (tmux.conf): takes out of the save the
# sessions that shouldn't come back after a reboot. Those VS Code's terminal
# made (option @vscode, vscode/tmux.sh), which it makes again when it opens,
# and the throwaway ones (destroy-unattached: a second Ghostty window, a
# second VS Code terminal). $1: the save file, rewritten in place.
set -euo pipefail
file=$1
[ -f "$file" ] || exit 0

# No server to ask (it is going away): the save stays as it is
# ':' between the fields: tmux prints a tab in -F as _, and a session name
# can't hold a ':'
drop=$( (tmux list-sessions -F '#{session_name}:#{@vscode}:#{destroy-unattached}' 2>/dev/null || true) |
  awk -F: '$2 == "1" || $3 == "on" || $3 == "1" { print $1 }')
[ -n "$drop" ] || exit 0

# Save lines are tab-separated, the session second: pane, window, grouped_session
tmp=$(mktemp "$file.XXXXXX")
awk -F'\t' 'NR == FNR { drop[$0] = 1; next } !(($1 == "pane" || $1 == "window" || $1 == "grouped_session") && ($2 in drop))' \
  <(printf '%s\n' "$drop") "$file" >"$tmp"
mv "$tmp" "$file"

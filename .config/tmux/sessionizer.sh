#!/bin/bash
# Open a repo as a tmux session: fuzzy-pick a git repo under ~/work (⌘O in
# Ghostty / prefix f; most-used repos first via zoxide), or pass a path. Session = repo name, started in the repo;
# reused if it already exists.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
ROOT="${TMUX_SESSIONIZER_ROOT:-$HOME/work}"

if [ -n "$1" ]; then
  DIR="$1"
else
  # Most-used first (zoxide frecency, git repos under $ROOT only), then every
  # other repo alphabetically. Same fzf list, better order.
  REL=$( {
          command -v zoxide >/dev/null && zoxide query -l 2>/dev/null \
            | while read -r d; do
                # no `case` here: macOS /bin/bash 3.2 can't parse `pat)` inside $( )
                [[ "$d" == "$ROOT"/* && -d "$d/.git" ]] && echo "$d"
              done
          find "$ROOT" -mindepth 2 -maxdepth 4 -type d -name .git -prune 2>/dev/null \
            | sed 's#/\.git$##' | sort
        } | awk '!seen[$0]++' | sed "s#^$ROOT/##" \
        | fzf --prompt='repo › ' --height=100% --reverse --border=none)
  [ -z "$REL" ] && exit 0
  DIR="$ROOT/$REL"
fi
[ -d "$DIR" ] || { echo "not a directory: $DIR" >&2; exit 1; }

NAME=$(basename "$DIR" | tr '.:' '__')
tmux has-session -t="$NAME" 2>/dev/null || tmux new-session -d -s "$NAME" -c "$DIR"

if [ -n "$TMUX" ]; then
  tmux switch-client -t="$NAME"
else
  tmux attach-session -t="$NAME"
fi

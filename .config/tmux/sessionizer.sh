#!/bin/bash
# Open a repo as a tmux session: fuzzy-pick a git repo under ~/work (⌘O in
# Ghostty / prefix f), or pass a path. Session = repo name, started in the repo;
# reused if it already exists.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
ROOT="${TMUX_SESSIONIZER_ROOT:-$HOME/work}"

if [ -n "$1" ]; then
  DIR="$1"
else
  REL=$(find "$ROOT" -mindepth 2 -maxdepth 4 -type d -name .git -prune 2>/dev/null \
        | sed -e "s#^$ROOT/##" -e 's#/\.git$##' | sort \
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

#!/bin/bash
# ⌘O in Rex: fuzzy-pick a repo under ~/work (the tmux sessionizer's list) and
# switch to its session, creating it in the repo when there isn't one. Runs in a
# floating layer (init.lua). Switching needs Remote Control on in Rex's Settings
# (Rex Server), otherwise the app refuses actions from the CLI.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

DIR=$(~/.config/tmux/sessionizer.sh --print) || exit 1
[ -z "$DIR" ] && exit 0
NAME=$(basename "$DIR" | tr '.:' '__')

ID=$(rex api call session.list '{}' |
  jq -r --arg n "$NAME" '[.sessions[] | select(.label == $n)][0].session_id // empty')
[ -z "$ID" ] && ID=$(rex new "$NAME" -c "$DIR" --json | jq -r .session_id)

rex do session.select session_id="$ID" || sleep 4

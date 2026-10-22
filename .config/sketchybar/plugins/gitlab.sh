#!/bin/bash
# Open MRs waiting for my review on the default glab host. Hidden when glab is
# missing or not logged in (e.g. machines without GitLab access).
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"

HOST=$(run glab config get host 2>/dev/null)
HOST=${HOST:-gitlab.com}

if [ "$SENDER" = "mouse.clicked" ]; then
  open "https://$HOST/dashboard/merge_requests?reviewer_username=$(run glab api user 2>/dev/null | jq -r .username)"
  exit 0
fi

if ! command -v glab >/dev/null || ! run glab auth status >/dev/null 2>&1; then
  sketchybar --set "$NAME" drawing=off
  exit 0
fi

USER_ID=$(run glab api user 2>/dev/null | jq -r .id 2>/dev/null)
COUNT=$(run glab api "merge_requests?scope=all&state=opened&reviewer_id=$USER_ID&per_page=100" 2>/dev/null |
  jq length 2>/dev/null)
[ -z "$COUNT" ] && COUNT="?"

if [ "$COUNT" = "0" ]; then COLOR=$DIM; else COLOR=$ORANGE; fi
sketchybar --set "$NAME" drawing=on label="$COUNT" icon.color=$COLOR label.color=$COLOR

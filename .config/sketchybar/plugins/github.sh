#!/bin/bash
# Open PRs waiting for my review. Click opens the list on GitHub.
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"

COUNT=$(run gh search prs --review-requested=@me --state=open --json number --jq length 2>/dev/null)
[ -z "$COUNT" ] && COUNT="?"

if [ "$COUNT" = "0" ]; then COLOR=$DIM; else COLOR=$YELLOW; fi
sketchybar --set "$NAME" label="$COUNT" icon.color="$COLOR" label.color="$COLOR"

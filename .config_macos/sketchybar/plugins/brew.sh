#!/bin/bash
# Count outdated Homebrew packages (refreshes the index first; runs hourly).
# brew needs SIGCHLD reset (see helpers.sh), otherwise it fails with
# "undefined method 'exitstatus' for nil".
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"

run brew update --quiet >/dev/null 2>&1
COUNT=$(run brew outdated --quiet 2>/dev/null | wc -l | tr -d ' ')
COLOR=$GREEN
[ "$COUNT" -gt 0 ] && COLOR=$YELLOW
[ "$COUNT" -gt 20 ] && COLOR=$ORANGE
sketchybar --set "$NAME" label="$COUNT" icon.color=$COLOR

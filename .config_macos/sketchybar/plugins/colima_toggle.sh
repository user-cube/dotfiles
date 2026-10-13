#!/bin/bash
# $1 = profile, $2 = start|stop. Runs in the background, then refreshes the item.
source "$CONFIG_DIR/plugins/helpers.sh"
sketchybar --set colima popup.drawing=off label="$2ing $1…"
( run colima "$2" --profile "$1" >/dev/null 2>&1; sketchybar --trigger colima_update ) &

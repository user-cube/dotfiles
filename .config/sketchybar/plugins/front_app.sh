#!/bin/bash
[ "$SENDER" = "front_app_switched" ] || exit 0
# Not in git: downloaded by fetch-icon-map.sh; __icon_map sets icon_result
# shellcheck source=/dev/null
source "$CONFIG_DIR/plugins/icon_map.sh"
icon_result=""
__icon_map "$INFO"
sketchybar --set "$NAME" icon="$icon_result" label="$INFO"

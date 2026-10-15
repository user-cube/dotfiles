#!/bin/bash
[ "$SENDER" = "front_app_switched" ] || exit 0
# shellcheck source=SCRIPTDIR/icon_map.sh
source "$CONFIG_DIR/plugins/icon_map.sh"
__icon_map "$INFO"
sketchybar --set "$NAME" icon="$icon_result" label="$INFO"

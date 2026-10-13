#!/bin/bash
# $1 = workspace this item represents.
# Focused: green name + green icon pill. Visible on another monitor: bright.
# Has windows: dim. Empty: hidden.
export PATH="/opt/homebrew/bin:$PATH"
source "$CONFIG_DIR/colors.sh"
source "$CONFIG_DIR/plugins/icon_map.sh"

WS="$1"
FOCUSED="${FOCUSED_WORKSPACE:-$(aerospace list-workspaces --focused)}"
VISIBLE="$(aerospace list-workspaces --monitor all --visible)"

ICONS=""
while IFS= read -r app; do
  [ -z "$app" ] && continue
  __icon_map "$app"
  ICONS+="$icon_result "
done < <(aerospace list-windows --workspace "$WS" --format '%{app-name}' | sort -u)
ICONS="${ICONS% }"

if [ "$WS" = "$FOCUSED" ]; then
  sketchybar --set "$NAME" drawing=on label="$ICONS" label.drawing=$([ -n "$ICONS" ] && echo on || echo off) \
    icon.color=$GREEN label.color=$DARK label.background.color=$GREEN
elif [ -n "$ICONS" ] || echo "$VISIBLE" | grep -qx "$WS"; then
  if echo "$VISIBLE" | grep -qx "$WS"; then COLOR=$FG; else COLOR=$DIM; fi
  sketchybar --set "$NAME" drawing=on label="$ICONS" label.drawing=$([ -n "$ICONS" ] && echo on || echo off) \
    icon.color=$COLOR label.color=$COLOR label.background.color=$PILL_BG
else
  sketchybar --set "$NAME" drawing=off
fi

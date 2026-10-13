#!/bin/bash
VOL="${INFO:-$(osascript -e 'output volume of (get volume settings)')}"
case "$VOL" in
  [6-9][0-9]|100) ICON=$'\xf3\xb0\x95\xbe' ;;
  [3-5][0-9])     ICON=$'\xf3\xb0\x96\x80' ;;
  [1-9]|[1-2][0-9]) ICON=$'\xf3\xb0\x95\xbf' ;;
  *)              ICON=$'\xf3\xb0\x96\x81' ;;
esac
sketchybar --set "$NAME" icon="$ICON" label="${VOL}%"

#!/bin/bash
# $1 = app name as shown in the Dock (e.g. Slack, Discord). Reads the app's
# Dock badge via Accessibility (needs Accessibility + Automation→System Events
# for sketchybar; `lsappinfo` no longer exposes badges on macOS 27).
# Red count = mentions/DMs, yellow dot = unread, dim = nothing, hidden = closed.
source "$CONFIG_DIR/colors.sh"
APP="$1"

if [ "$SENDER" = "mouse.clicked" ]; then
  open -a "$APP"
  exit 0
fi

if ! pgrep -xq "$APP"; then
  sketchybar --set "$NAME" drawing=off
  exit 0
fi

BADGE=$(osascript -e "tell application \"System Events\" to tell process \"Dock\" to \
  get value of attribute \"AXStatusLabel\" of UI element \"$APP\" of list 1" 2>/dev/null)
[ "$BADGE" = "missing value" ] && BADGE=""

case "$BADGE" in
  "")      sketchybar --set "$NAME" drawing=on label.drawing=off icon.color=$DIM ;;
  *[0-9]*) sketchybar --set "$NAME" drawing=on label.drawing=on label="$BADGE" icon.color=$RED label.color=$RED ;;
  *)       sketchybar --set "$NAME" drawing=on label.drawing=on label="•" icon.color=$YELLOW label.color=$YELLOW ;;
esac

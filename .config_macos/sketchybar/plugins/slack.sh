#!/bin/bash
# Slack Dock badge: red count for mentions/DMs, yellow dot for unread channels,
# dim icon when nothing new, hidden when Slack isn't running. Click opens Slack.
# (`lsappinfo info -only StatusLabel` returns NULL on macOS 27, so parse the
# full output instead.)
source "$CONFIG_DIR/colors.sh"

if [ "$SENDER" = "mouse.clicked" ]; then
  open -a Slack
  exit 0
fi

INFO_OUT=$(lsappinfo info Slack 2>/dev/null)
if [ -z "$INFO_OUT" ] || ! echo "$INFO_OUT" | grep -q 'com.tinyspeck.slackmacgap'; then
  sketchybar --set "$NAME" drawing=off
  exit 0
fi

BADGE=$(echo "$INFO_OUT" | grep -o '"label"="[^"]*"' | head -1 | cut -d'"' -f4)

case "$BADGE" in
  "")        sketchybar --set "$NAME" drawing=on label.drawing=off icon.color=$DIM ;;
  *[0-9]*)   sketchybar --set "$NAME" drawing=on label.drawing=on label="$BADGE" icon.color=$RED label.color=$RED ;;
  *)         sketchybar --set "$NAME" drawing=on label.drawing=on label="•" icon.color=$YELLOW label.color=$YELLOW ;;
esac

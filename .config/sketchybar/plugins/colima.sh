#!/bin/bash
# Colima profiles: label lists the running ones; popup toggles each profile.
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"

case "$SENDER" in
  mouse.clicked)       sketchybar --set "$NAME" popup.drawing=toggle; exit 0 ;;
  mouse.exited.global) sketchybar --set "$NAME" popup.drawing=off; exit 0 ;;
esac

LIST=$(run colima list --json 2>/dev/null)
# colima list --json: one JSON object per line
RUNNING=$(echo "$LIST" | jq -rs 'map(select(.status == "Running").name) | join(" ")' 2>/dev/null)

if [ -n "$RUNNING" ]; then
  sketchybar --set "$NAME" label="$RUNNING" icon.color=$GREEN label.color=$FG
else
  sketchybar --set "$NAME" label="off" icon.color=$DIM label.color=$DIM
fi

# Rebuild popup: one entry per profile, click starts/stops it
sketchybar --remove '/colima\.profile\..*/' >/dev/null 2>&1
echo "$LIST" | jq -r '"\(.name) \(.status)"' 2>/dev/null |
while read -r PROFILE STATUS; do
  if [ "$STATUS" = "Running" ]; then ACTION=stop; DOT=$GREEN; else ACTION=start; DOT=$DIM; fi
  sketchybar --add item "colima.profile.$PROFILE" popup.colima \
    --set "colima.profile.$PROFILE" icon=$'\xef\x84\x91' icon.color=$DOT icon.font="Hack Nerd Font:Bold:9.0" \
          label="$PROFILE  ($ACTION)" \
          click_script="$CONFIG_DIR/plugins/colima_toggle.sh $PROFILE $ACTION"
done

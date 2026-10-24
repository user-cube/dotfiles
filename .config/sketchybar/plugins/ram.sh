#!/bin/bash
# Memory used (100 - system-wide free %). Yellow above 70%, red above 85%.
source "$CONFIG_DIR/colors.sh"
FREE=$(memory_pressure -Q | awk -F': ' '/free percentage/ {gsub("%","",$2); print $2}')
USED=$((100 - FREE))
COLOR=$GREEN
[ "$USED" -ge 70 ] && COLOR=$YELLOW
[ "$USED" -ge 85 ] && COLOR=$RED
sketchybar --set "$NAME" label="${USED}%" icon.color="$COLOR"

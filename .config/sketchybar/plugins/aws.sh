#!/bin/bash
# $1 = AWS profile. Shows the profile name when its session is valid, red
# "expired" when it needs `aws sso login --profile $1`. Uses STS only, never
# reads credential files.
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"
PROFILE="$1"

OUT=$(run aws sts get-caller-identity --profile "$PROFILE" --output json 2>&1)
if [ $? -eq 0 ]; then
  sketchybar --set "$NAME" label="$PROFILE" label.color="$FG" icon.color="$ORANGE"
elif echo "$OUT" | grep -qiE 'expired|sso|refresh|token'; then
  sketchybar --set "$NAME" label="$PROFILE expired" label.color="$RED" icon.color="$RED"
else
  sketchybar --set "$NAME" label="$PROFILE error" label.color="$RED" icon.color="$RED"
fi

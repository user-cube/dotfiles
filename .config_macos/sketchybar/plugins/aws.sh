#!/bin/bash
# AWS session shown as role@account; red "expired" when the SSO session needs
# `aws sso login`. Uses STS only, never reads credential files.
#
# Profile, in order: $CONFIG_DIR/aws_profile (one line, per machine, not in
# dotfiles) → "default" → the only profile there is. With several profiles and
# none chosen it shows a dim hint. Hidden without the aws CLI or any profile.
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"

command -v aws >/dev/null || { sketchybar --set "$NAME" drawing=off; exit 0; }

PROFILES=$(run aws configure list-profiles 2>/dev/null)
[ -z "$PROFILES" ] && { sketchybar --set "$NAME" drawing=off; exit 0; }

PROFILE=""
[ -f "$CONFIG_DIR/aws_profile" ] && PROFILE=$(tr -d '[:space:]' < "$CONFIG_DIR/aws_profile")
if [ -z "$PROFILE" ]; then
  if echo "$PROFILES" | grep -qx default; then PROFILE=default
  elif [ "$(echo "$PROFILES" | wc -l | tr -d ' ')" = "1" ]; then PROFILE=$PROFILES
  else
    sketchybar --set "$NAME" drawing=on label="set ~/.config/sketchybar/aws_profile" \
      icon.color=$DIM label.color=$DIM
    exit 0
  fi
fi

OUT=$(run aws sts get-caller-identity --profile "$PROFILE" --output json 2>&1)
if [ $? -eq 0 ]; then
  LABEL=$(echo "$OUT" | python3 -c 'import sys,json
d=json.load(sys.stdin); arn=d["Arn"]
role=arn.split("/")[1] if ":assumed-role/" in arn else arn.split("/")[-1]
print(f"{role}@{d[\"Account\"]}")')
  sketchybar --set "$NAME" drawing=on label="$LABEL" icon.color=$ORANGE label.color=$FG
elif echo "$OUT" | grep -qiE 'expired|sso|refresh|token'; then
  sketchybar --set "$NAME" drawing=on label="$PROFILE expired" icon.color=$RED label.color=$RED
else
  sketchybar --set "$NAME" drawing=on label="$PROFILE: error" icon.color=$RED label.color=$RED
fi

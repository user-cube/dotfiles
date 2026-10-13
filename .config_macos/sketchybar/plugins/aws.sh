#!/bin/bash
# AWS session of the default profile (or PROFILE below). Shows role@account;
# red "expired" when the SSO session needs `aws sso login`. Hidden without
# the aws CLI or any credentials. Uses STS only, never reads credential files.
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"

PROFILE=""   # empty = default profile

command -v aws >/dev/null || { sketchybar --set "$NAME" drawing=off; exit 0; }

ARGS=(sts get-caller-identity --output json)
[ -n "$PROFILE" ] && ARGS+=(--profile "$PROFILE")

OUT=$(run aws "${ARGS[@]}" 2>&1)
if [ $? -eq 0 ]; then
  LABEL=$(echo "$OUT" | python3 -c 'import sys,json
d=json.load(sys.stdin); arn=d["Arn"]
role=arn.split("/")[1] if ":assumed-role/" in arn else arn.split("/")[-1]
print(f"{role}@{d[\"Account\"]}")')
  sketchybar --set "$NAME" drawing=on label="$LABEL" icon.color=$ORANGE label.color=$FG
elif echo "$OUT" | grep -qiE 'expired|sso|refresh'; then
  sketchybar --set "$NAME" drawing=on label="expired" icon.color=$RED label.color=$RED
else
  sketchybar --set "$NAME" drawing=off   # no credentials configured
fi

#!/bin/bash
# GCP: active project; red "expired" when the login needs `gcloud auth login`.
# Hidden without gcloud or an active account.
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"

command -v gcloud >/dev/null || { sketchybar --set "$NAME" drawing=off; exit 0; }

ACCOUNT=$(run gcloud config get-value account 2>/dev/null)
[ -z "$ACCOUNT" ] && { sketchybar --set "$NAME" drawing=off; exit 0; }
PROJECT=$(run gcloud config get-value project 2>/dev/null)

# Refreshes the token if possible; output discarded
if run gcloud auth print-access-token >/dev/null 2>&1; then
  sketchybar --set "$NAME" drawing=on label="${PROJECT:-$ACCOUNT}" icon.color="$BLUE" label.color="$FG"
else
  sketchybar --set "$NAME" drawing=on label="expired" icon.color="$RED" label.color="$RED"
fi

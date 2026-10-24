#!/bin/bash
# Current kubectl context / namespace. Red when the context looks like prod.
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"

PROD_REGEX='prod|prd'

CTX=$(run kubectl config current-context 2>/dev/null)
if [ -z "$CTX" ]; then
  sketchybar --set "$NAME" label="none" icon.color="$DIM" label.color="$DIM"
  exit 0
fi
NS=$(run kubectl config view --minify -o jsonpath='{..namespace}' 2>/dev/null)

if echo "$CTX" | grep -qiE "$PROD_REGEX"; then COLOR=$RED; else COLOR=$BLUE; fi
sketchybar --set "$NAME" label="$CTX/${NS:-default}" icon.color="$COLOR" label.color="$COLOR"

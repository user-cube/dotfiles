#!/bin/bash
# Current kubectl context / namespace. Red when the context is production.
# Asks kube-guard (.config/kube-guard), which reads the kubeconfig in ~3 ms and
# says production as kube-guard.zsh does (KUBE_PROD_REGEX, default prod|prd);
# kubectl only until it is built (two kubectl runs every 5 s: ~0.1 s of CPU)
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"

KUBE_GUARD="$HOME/.config/kube-guard/kube-guard"

if [ -x "$KUBE_GUARD" ]; then
  # no `run`: kube-guard starts no subprocess
  { read -r CTX; read -r NS; read -r KIND; } < <("$KUBE_GUARD" context 2>/dev/null)
else
  CTX=$(run kubectl config current-context 2>/dev/null)
  [ -n "$CTX" ] && NS=$(run kubectl config view --minify -o jsonpath='{..namespace}' 2>/dev/null)
  KIND=other
  echo "$CTX" | grep -qiE "${KUBE_PROD_REGEX:-prod|prd}" && KIND=prod
fi

if [ -z "$CTX" ]; then
  sketchybar --set "$NAME" label="none" icon.color="$DIM" label.color="$DIM"
  exit 0
fi

if [ "$KIND" = prod ]; then COLOR=$RED; else COLOR=$BLUE; fi
sketchybar --set "$NAME" label="$CTX/${NS:-default}" icon.color="$COLOR" label.color="$COLOR"

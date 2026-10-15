#!/bin/bash
# $1 = check name from $CONFIG_DIR/checks. Runs its command with no stdin and a
# 10 s timeout: green label when it exits 0, red when it fails. Click re-runs it.
# `kube <context>` is shorthand for "the context's credentials are accepted":
# /api needs an authenticated user, unlike /healthz or /version.
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"
CHECK="$1"
TIMEOUT=10

CMD=$(grep -vE '^\s*#' "$CONFIG_DIR/checks" 2>/dev/null \
  | awk -F'|' -v n="$CHECK" '{k=$1; gsub(/[[:blank:]]/, "", k)} k == n {sub(/^[^|]*\|[[:blank:]]*/, ""); print; exit}')
case "$CMD" in
  kube\ *) CMD="kubectl --context ${CMD#kube } get --raw /api --request-timeout=5s" ;;
esac

if [ -z "$CMD" ]; then
  sketchybar --set "$NAME" label="$CHECK ?" label.color=$DIM
  exit 0
fi

# Exec credential plugins (kubelogin) open a browser when the login expired; a
# fake `open` first in PATH keeps the bar from popping one every few minutes.
SHIM=$(mktemp -d)
trap 'rm -rf "$SHIM"' EXIT
printf '#!/bin/sh\nexit 1\n' >"$SHIM/open"
chmod +x "$SHIM/open"

sketchybar --set "$NAME" label.color=$YELLOW
if PATH="$SHIM:$PATH" run perl -e 'alarm shift; exec @ARGV' "$TIMEOUT" \
  bash -c "$CMD" </dev/null >/dev/null 2>&1; then
  sketchybar --set "$NAME" label="$CHECK" label.color=$GREEN
else
  sketchybar --set "$NAME" label="$CHECK" label.color=$RED
fi

#!/bin/bash
# Per-machine checks from $CONFIG_DIR/checks, one `name | command` per line.
# Runs them all in parallel (no stdin, 10 s timeout each); the label shows
# passed/total, red when any fails. Click toggles a popup with each check,
# failures first, with a short reason.
# `kube <context>` is shorthand for "the context's credentials are accepted":
# /api needs an authenticated user, unlike /healthz or /version.
source "$CONFIG_DIR/plugins/helpers.sh"
source "$CONFIG_DIR/colors.sh"
CHECKS_FILE="$CONFIG_DIR/checks"
TIMEOUT=10
OK_ICON=$'\xef\x80\x8c'   # nf-fa-check
FAIL_ICON=$'\xef\x80\x8d' # nf-fa-times

case "$SENDER" in
  mouse.clicked)       sketchybar --set "$NAME" popup.drawing=toggle; exit 0 ;;
  mouse.exited.global) sketchybar --set "$NAME" popup.drawing=off; exit 0 ;;
esac

# Exec credential plugins (kubelogin) open a browser when the login expired; a
# fake `open` first in PATH keeps the bar from popping one every few minutes.
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
mkdir "$WORK/bin" "$WORK/results"
printf '#!/bin/sh\nexit 1\n' >"$WORK/bin/open"
chmod +x "$WORK/bin/open"

sketchybar --set "$NAME" label.color=$YELLOW
NAMES=()
while IFS='|' read -r CHECK CMD; do
  CHECK=$(echo "$CHECK" | tr -d '[:blank:]')
  case "$CHECK" in '' | \#*) continue ;; esac
  CMD="${CMD#"${CMD%%[![:blank:]]*}"}"
  case "$CMD" in
    kube\ *) CMD="kubectl --context ${CMD#kube } get --raw /api --request-timeout=5s" ;;
  esac
  NAMES+=("$CHECK")
  (
    PATH="$WORK/bin:$PATH" run perl -e 'alarm shift; exec @ARGV' "$TIMEOUT" \
      bash -c "$CMD" </dev/null >/dev/null 2>"$WORK/results/$CHECK.err"
    echo $? >"$WORK/results/$CHECK"
  ) &
done <"$CHECKS_FILE"
wait

# Short reason for a failed check, from its exit code and stderr
reason() {
  local code err
  code=$(cat "$WORK/results/$1" 2>/dev/null)
  err=$(tr '\n' ' ' <"$WORK/results/$1.err" 2>/dev/null)
  [ "$code" = 142 ] && { echo "timeout"; return; } # 128 + SIGALRM
  case "$err" in
    *Unauthorized* | *401* | *expired* | *"token"*) echo "expired" ;;
    *Forbidden* | *403*) echo "forbidden" ;;
    *refused*) echo "refused" ;;
    *"does not exist"*) echo "no context" ;;
    *"no such host"* | *"Could not resolve"*) echo "dns" ;;
    *"timed out"* | *Timeout* | *timeout*) echo "timeout" ;;
    *) echo "exit $code" ;;
  esac
}

# Sort: failures first, then by file order
FAILED=() PASSED=()
for CHECK in "${NAMES[@]}"; do
  if [ "$(cat "$WORK/results/$CHECK" 2>/dev/null)" = 0 ]; then PASSED+=("$CHECK"); else FAILED+=("$CHECK"); fi
done

# Name column fits the longest name (Hack Nerd Font is monospaced)
WIDTH=0
for CHECK in "${NAMES[@]}"; do [ ${#CHECK} -gt $WIDTH ] && WIDTH=${#CHECK}; done
NAME_WIDTH=$((WIDTH * 8 + 40))

# Rebuild popup: header, then one row per check
sketchybar --remove '/checks\.item\..*/' >/dev/null 2>&1
sketchybar --add item checks.item.header "popup.$NAME" \
  --set checks.item.header icon="Checks" icon.font="SF Pro:Bold:13.0" icon.color=$FG \
        icon.width=$NAME_WIDTH label="$(date +%H:%M)" label.color=$DIM \
        label.font="SF Pro:Regular:12.0" icon.padding_left=12 label.padding_right=14 \
        background.height=28
for CHECK in ${FAILED[@]+"${FAILED[@]}"}; do
  sketchybar --add item "checks.item.$CHECK" "popup.$NAME" \
    --set "checks.item.$CHECK" icon="$FAIL_ICON  $CHECK" icon.color=$RED \
          icon.font="Hack Nerd Font:Bold:13.0" icon.width=$NAME_WIDTH \
          label="$(reason "$CHECK")" label.color=$DIM \
          label.font="SF Pro:Regular:12.0" icon.padding_left=12 label.padding_right=14
done
for CHECK in ${PASSED[@]+"${PASSED[@]}"}; do
  sketchybar --add item "checks.item.$CHECK" "popup.$NAME" \
    --set "checks.item.$CHECK" icon="$OK_ICON  $CHECK" icon.color=$GREEN \
          icon.font="Hack Nerd Font:Bold:13.0" icon.width=$NAME_WIDTH \
          label="ok" label.color=$DIM \
          label.font="SF Pro:Regular:12.0" icon.padding_left=12 label.padding_right=14
done

if [ ${#FAILED[@]} -eq 0 ]; then COLOR=$GREEN; else COLOR=$RED; fi
sketchybar --set "$NAME" label="${#PASSED[@]}/${#NAMES[@]}" label.color=$COLOR

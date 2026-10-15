#!/bin/bash
# Connectivity checks: one per line of $CONFIG_DIR/checks (`name | command`,
# per machine, not in dotfiles) or, without that file, every kube context in the
# kubeconfig.
# Runs them all in parallel (no stdin, 10 s timeout each); the label shows
# passed/total, red when any fails, hidden when there is nothing to check.
# Click toggles a popup with each check, failures first, with a short reason.
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

# `name | kube <context>` for every context in the kubeconfig
kube_contexts() {
  command -v kubectl >/dev/null || return 0
  run kubectl config get-contexts -o name 2>/dev/null | sed 's/.*/& | kube &/'
}

if grep -qvE '^\s*(#|$)' "$CHECKS_FILE" 2>/dev/null; then
  LINES=$(cat "$CHECKS_FILE")
else
  LINES=$(kube_contexts)
fi
if ! echo "$LINES" | grep -qvE '^\s*(#|$)'; then
  sketchybar --set "$NAME" drawing=off
  exit 0
fi

# Exec credential plugins (kubelogin) open a browser when the login expired; a
# fake `open` first in PATH keeps the bar from popping one every few minutes.
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
mkdir "$WORK/bin" "$WORK/results"
printf '#!/bin/sh\nexit 1\n' >"$WORK/bin/open"
chmod +x "$WORK/bin/open"

sketchybar --set "$NAME" drawing=on label.color=$YELLOW
# Results go by index: context names can hold / and : (EKS ARNs)
NAMES=()
while IFS='|' read -r CHECK CMD; do
  CHECK=$(echo "$CHECK" | tr -d '[:blank:]')
  case "$CHECK" in '' | \#*) continue ;; esac
  CMD="${CMD#"${CMD%%[![:blank:]]*}"}"
  case "$CMD" in
    kube\ *) CMD="kubectl --context $(printf %q "${CMD#kube }") get --raw /api --request-timeout=5s" ;;
  esac
  I=${#NAMES[@]}
  NAMES+=("$CHECK")
  (
    PATH="$WORK/bin:$PATH" run perl -e 'alarm shift; exec @ARGV' "$TIMEOUT" \
      bash -c "$CMD" </dev/null >/dev/null 2>"$WORK/results/$I.err"
    echo $? >"$WORK/results/$I"
  ) &
done <<<"$LINES"
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

# Sort: failures first, then in order; EKS ARNs show only the cluster name
FAILED=() PASSED=()
WIDTH=0
for I in "${!NAMES[@]}"; do
  if [ "$(cat "$WORK/results/$I" 2>/dev/null)" = 0 ]; then PASSED+=("$I"); else FAILED+=("$I"); fi
  SHORT=${NAMES[$I]##*/}
  [ ${#SHORT} -gt $WIDTH ] && WIDTH=${#SHORT}
done
NAME_WIDTH=$((WIDTH * 8 + 40)) # Hack Nerd Font is monospaced

row() { # row <index> <icon> <color> <label>
  sketchybar --add item "checks.item.$1" "popup.$NAME" \
    --set "checks.item.$1" icon="$2  ${NAMES[$1]##*/}" icon.color="$3" \
          icon.font="Hack Nerd Font:Bold:13.0" icon.width=$NAME_WIDTH \
          label="$4" label.color=$DIM \
          label.font="SF Pro:Regular:12.0" icon.padding_left=12 label.padding_right=14
}

# Rebuild popup: header, then one row per check
sketchybar --remove '/checks\.item\..*/' >/dev/null 2>&1
sketchybar --add item checks.item.header "popup.$NAME" \
  --set checks.item.header icon="Checks" icon.font="SF Pro:Bold:13.0" icon.color=$FG \
        icon.width=$NAME_WIDTH label="$(date +%H:%M)" label.color=$DIM \
        label.font="SF Pro:Regular:12.0" icon.padding_left=12 label.padding_right=14 \
        background.height=28
for I in ${FAILED[@]+"${FAILED[@]}"}; do row "$I" "$FAIL_ICON" "$RED" "$(reason "$I")"; done
for I in ${PASSED[@]+"${PASSED[@]}"}; do row "$I" "$OK_ICON" "$GREEN" "ok"; done

if [ ${#FAILED[@]} -eq 0 ]; then COLOR=$GREEN; else COLOR=$RED; fi
sketchybar --set "$NAME" label="${#PASSED[@]}/${#NAMES[@]}" label.color=$COLOR

#!/bin/bash
# Spotify: green while playing, dim with a play symbol while paused, hidden when
# Spotify isn't running. Click toggles play/pause.
source "$CONFIG_DIR/colors.sh"

if [ "$SENDER" = "mouse.clicked" ]; then
  osascript -e 'tell application "Spotify" to playpause'
  exit 0
fi

# Spotify quit (or never opened): hide, without launching it via osascript
if ! pgrep -xq Spotify; then
  sketchybar --set "$NAME" drawing=off
  exit 0
fi

STATE=$(echo "$INFO" | python3 -c 'import sys,json
try: print(json.load(sys.stdin).get("Player State","").lower())
except Exception: print("")')
[ -z "$STATE" ] && STATE=$(osascript -e 'tell application "Spotify" to player state as string' 2>/dev/null)

TRACK=$(osascript -e 'tell application "Spotify" to (artist of current track) & " — " & (name of current track)' 2>/dev/null)

case "$STATE" in
  playing)
    sketchybar --set "$NAME" drawing=on label="$TRACK" icon.color=$GREEN label.color=$FG ;;
  paused)
    sketchybar --set "$NAME" drawing=on label=$'\xe2\x96\xb6\xef\xb8\x8e'"  $TRACK" icon.color=$DIM label.color=$DIM ;;
  *)
    sketchybar --set "$NAME" drawing=off ;;
esac

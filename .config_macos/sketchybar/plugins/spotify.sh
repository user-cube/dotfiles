#!/bin/bash
# Now playing on Spotify; hidden when paused/stopped. Click toggles play/pause.
if [ "$SENDER" = "mouse.clicked" ]; then
  osascript -e 'tell application "Spotify" to playpause'
  exit 0
fi

STATE=$(echo "$INFO" | python3 -c 'import sys,json
try: d=json.load(sys.stdin); print(d.get("Player State",""))
except Exception: print("")')

if [ "$STATE" != "Playing" ]; then
  # On startup (no event yet) ask Spotify directly, without launching it
  if [ -z "$STATE" ] && pgrep -xq Spotify && \
     [ "$(osascript -e 'tell application "Spotify" to player state as string')" = "playing" ]; then
    TRACK=$(osascript -e 'tell application "Spotify" to (artist of current track) & " — " & (name of current track)')
    sketchybar --set "$NAME" drawing=on label="$TRACK"
  else
    sketchybar --set "$NAME" drawing=off
  fi
  exit 0
fi

TRACK=$(echo "$INFO" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("Artist","")+" — "+d.get("Name",""))')
sketchybar --set "$NAME" drawing=on label="$TRACK"

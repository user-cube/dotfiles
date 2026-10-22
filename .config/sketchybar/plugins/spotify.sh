#!/bin/bash
# Spotify: green while playing, dim with a play symbol while paused, hidden when
# Spotify isn't running. Click toggles play/pause.
# Never `tell application "Spotify"` unguarded: that relaunches it, e.g. right
# after ⌘Q when the quit event arrives. Prefer the event payload ($INFO).
source "$CONFIG_DIR/plugins/helpers.sh" # PATH: jq
source "$CONFIG_DIR/colors.sh"

if [ "$SENDER" = "mouse.clicked" ]; then
  osascript -e 'tell application "Spotify" to playpause'
  exit 0
fi

if [ -n "$INFO" ]; then
  # Event payload: {"Player State": "Playing|Paused|Stopped", "Artist": ..., "Name": ...}
  OUT=$(echo "$INFO" | jq -r '((."Player State" // "") | ascii_downcase) + "|" + (.Artist // "") + " — " + (.Name // "")' 2>/dev/null)
else
  # Startup / periodic: ask Spotify only if it is already running
  OUT=$(osascript -e 'if application "Spotify" is running then
    tell application "Spotify" to return (player state as string) & "|" & (artist of current track) & " — " & (name of current track)
  end if' 2>/dev/null)
fi

STATE="${OUT%%|*}"
TRACK="${OUT#*|}"

case "$STATE" in
  playing)
    sketchybar --set "$NAME" drawing=on label="$TRACK" icon.color=$GREEN label.color=$FG ;;
  paused)
    sketchybar --set "$NAME" drawing=on label=$'\xe2\x96\xb6\xef\xb8\x8e'"  $TRACK" icon.color=$DIM label.color=$DIM ;;
  *)
    sketchybar --set "$NAME" drawing=off ;;
esac

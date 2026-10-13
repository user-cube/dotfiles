#!/bin/bash
# Weather via wttr.in (location from IP). Click opens the full forecast.
if [ "$SENDER" = "mouse.clicked" ]; then
  open "https://wttr.in"
  exit 0
fi

OUT=$(curl -s --max-time 10 "https://wttr.in/?format=%c|%t")
# Keep the last value on network errors / rate limiting
echo "$OUT" | grep -q '°' || exit 0

ICON=$(echo "${OUT%%|*}" | tr -d ' ')
TEMP=$(echo "${OUT##*|}" | tr -d ' +')
sketchybar --set "$NAME" icon="$ICON" label="$TEMP"

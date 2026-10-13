#!/bin/bash
INFO="$(pmset -g batt)"
PCT="$(echo "$INFO" | grep -Eo '[0-9]+%' | head -1 | tr -d '%')"
[ -z "$PCT" ] && exit 0

case "$PCT" in
  9[0-9]|100) ICON=$'\xef\x89\x80' ;;
  [6-8][0-9]) ICON=$'\xef\x89\x81' ;;
  [3-5][0-9]) ICON=$'\xef\x89\x82' ;;
  [1-2][0-9]) ICON=$'\xef\x89\x83' ;;
  *)          ICON=$'\xef\x89\x84' ;;
esac
echo "$INFO" | grep -q 'AC Power' && ICON=$'\xef\x83\xa7'

COLOR=0xffa6e22e
[ "$PCT" -le 50 ] && COLOR=0xffe6db74
[ "$PCT" -le 20 ] && COLOR=0xfff92672
sketchybar --set "$NAME" icon="$ICON" icon.color=$COLOR label="${PCT}%"

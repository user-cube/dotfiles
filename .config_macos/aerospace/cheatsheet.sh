#!/bin/bash
# alt-i: show the cheat sheet (rendered by glow) in a Ghostty window that fills
# the focused monitor (AeroSpace fullscreen, so it respects the SketchyBar gap).
# `q` closes it. on-window-detected can't run `fullscreen`, hence this script.
export PATH="/opt/homebrew/bin:$PATH"
TITLE="AeroSpace cheatsheet"

# Already open: just focus it
ID=$(aerospace list-windows --all --format '%{window-id}|%{window-title}' | grep "$TITLE" | head -1 | cut -d'|' -f1)
if [ -n "$ID" ]; then
  aerospace layout --window-id "$ID" tiling 2>/dev/null
  aerospace fullscreen on --window-id "$ID" 2>/dev/null
  aerospace focus --window-id "$ID"
  exit 0
fi

open -na Ghostty --args --title="$TITLE" \
  -e glow --pager --style=dracula --width=120 "$HOME/.config/aerospace/CHEATSHEET.md"

for _ in $(seq 1 30); do
  ID=$(aerospace list-windows --all --format '%{window-id}|%{window-title}' | grep "$TITLE" | head -1 | cut -d'|' -f1)
  [ -n "$ID" ] && break
  sleep 0.1
done
[ -n "$ID" ] && aerospace fullscreen on --window-id "$ID"

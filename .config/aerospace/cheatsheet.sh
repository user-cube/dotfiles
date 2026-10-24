#!/bin/bash
# alt-i: show the cheat sheet (rendered by glow) in a Ghostty window that fills
# the focused monitor (AeroSpace fullscreen, so it respects the SketchyBar gap).
# `q` closes it. on-window-detected can't run `fullscreen`, hence this script.
export PATH="/opt/homebrew/bin:$PATH"
TITLE="AeroSpace cheatsheet"

# Already open: just focus it. AeroSpace can still list a window Ghostty has
# closed (after a ⌘Q): when the focus fails, open a new one instead
ID=$(aerospace list-windows --all --format '%{window-id}|%{window-title}' | grep "$TITLE" | head -1 | cut -d'|' -f1)
if [ -n "$ID" ] && aerospace focus --window-id "$ID" 2>/dev/null; then
  aerospace layout --window-id "$ID" tiling 2>/dev/null
  aerospace fullscreen on --window-id "$ID" 2>/dev/null
  exit 0
fi

open -na Ghostty --args --title="$TITLE" \
  -e glow --pager --style=dracula --width=120 "$HOME/.config/aerospace/CHEATSHEET.md"

# A new Ghostty instance takes a few seconds to show its window: up to 10 s
for _ in $(seq 1 100); do
  ID=$(aerospace list-windows --all --format '%{window-id}|%{window-title}' | grep "$TITLE" | head -1 | cut -d'|' -f1)
  [ -n "$ID" ] && break
  sleep 0.1
done
[ -n "$ID" ] || exit 0
aerospace fullscreen on --window-id "$ID"
aerospace focus --window-id "$ID" # a new instance doesn't always come to the front

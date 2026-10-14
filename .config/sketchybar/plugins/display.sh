#!/bin/bash
# Rebuild the bar only on real monitor changes (not on the initial forced run)
[ "$SENDER" = "display_change" ] && sketchybar --reload

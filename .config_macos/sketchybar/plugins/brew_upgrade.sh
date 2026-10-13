#!/bin/bash
# Click on the brew item: run the upgrade in a new Ghostty window
open -na Ghostty --args -e /bin/zsh -lc \
  'brew upgrade; sketchybar --trigger brew_update; echo; read -k1 "?Done. Press any key to close"'

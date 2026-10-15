#!/bin/bash
# Prints the check names from $CONFIG_DIR/checks (per machine, not in dotfiles),
# one per line, in display order. Prints nothing without the file.
# Line format: `name | command`, see plugins/check.sh.
CHECKS_FILE="$CONFIG_DIR/checks"
[ -s "$CHECKS_FILE" ] || exit 0
grep -vE '^\s*(#|$)' "$CHECKS_FILE" | cut -d'|' -f1 | tr -d '[:blank:]'

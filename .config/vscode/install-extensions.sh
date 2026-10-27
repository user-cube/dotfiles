#!/bin/bash
# Installs the VS Code extensions in extensions.txt that are missing (`code` on
# PATH). Prints a line per extension it installed; nothing when all are there.
# One that fails to install (offline, renamed) only warns. Run by dotfiles-sync.
set -euo pipefail

dir=$(cd "$(dirname "$0")" && pwd)
command -v code >/dev/null || { echo "VS Code extensions: 'code' not on PATH, skipped"; exit 0; }

# Ids are case-insensitive: compare them lowercased
installed=$(code --list-extensions | tr '[:upper:]' '[:lower:]')
grep -vE '^[[:space:]]*(#|$)' "$dir/extensions.txt" | while read -r id; do
  lower=$(printf '%s' "$id" | tr '[:upper:]' '[:lower:]')
  grep -qxF "$lower" <<<"$installed" && continue
  if code --install-extension "$id" >/dev/null 2>&1; then
    echo "VS Code: installed $id"
  else
    echo "VS Code: could not install $id" >&2
  fi
done

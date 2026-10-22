#!/bin/bash
# Removes the files listed in ./obsolete (relative to $HOME) that this repo no
# longer installs, since rsync never deletes. Only regular files and symlinks;
# a line with .., an absolute path or a glob is skipped. The directory is
# removed too when that leaves it empty. Run by dotfiles-sync and the playbook.
set -euo pipefail

list="$(cd "$(dirname "$0")" && pwd)/obsolete"
while IFS= read -r f || [ -n "$f" ]; do
  case "$f" in
    '' | \#*) continue ;;
    /* | *..* | *'*'* | *'?'* | *'['*)
      echo "obsolete: skipped '$f' (not a plain path under \$HOME)" >&2
      continue
      ;;
  esac
  path="$HOME/$f"
  if [ -f "$path" ] || [ -L "$path" ]; then
    rm -f -- "$path"
    echo "removed ~/$f"
    rmdir -- "$(dirname "$path")" 2>/dev/null || true # only when empty
  fi
done <"$list"

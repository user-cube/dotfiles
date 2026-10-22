#!/bin/bash
# Downloads plugins/icon_map.sh (app name -> sketchybar-app-font glyph) from the
# sketchybar-app-font release that matches the installed font (Homebrew cask), so
# the map and the font never drift apart. Not in git. Run by dotfiles-sync and the
# playbook; skips when that version is already there.
set -euo pipefail
export PATH="/opt/homebrew/bin:$PATH"

dir=$(cd "$(dirname "$0")" && pwd)
out="$dir/plugins/icon_map.sh"
stamp="$out.version"
repo=https://github.com/kvndrsslr/sketchybar-app-font

version=$(brew list --cask --versions font-sketchybar-app-font 2>/dev/null | awk '{print $2}' || true)
if [[ -n $version ]]; then
  url="$repo/releases/download/v$version/icon_map.sh"
else
  # Font not installed yet (first playbook run): the latest one, replaced once it is
  version=latest
  url="$repo/releases/latest/download/icon_map.sh"
fi

if [[ -s $out && -f $stamp && $(cat "$stamp") == "$version" ]]; then
  exit 0
fi

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
curl -fsSL --retry 2 -o "$tmp" "$url" || { echo "icon_map.sh: download of $version failed" >&2; exit 1; }
grep -q '__icon_map()' "$tmp" || { echo "icon_map.sh: $url is not an icon map" >&2; exit 1; }

chmod 644 "$tmp"
mv "$tmp" "$out"
echo "$version" > "$stamp"
echo "icon_map.sh: downloaded $version"

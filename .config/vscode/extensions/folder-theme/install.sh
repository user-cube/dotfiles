#!/bin/bash
# Packages this extension as a .vsix and installs it into VS Code (`code` on PATH).
# Skips the install when nothing changed since the last one. Run by dotfiles-sync.
set -euo pipefail

dir=$(cd "$(dirname "$0")" && pwd)
command -v code >/dev/null || { echo "folder-theme: 'code' not on PATH, skipped"; exit 0; }

stamp_file="$HOME/.cache/folder-theme.sha"
sha=$(cat "$dir/package.json" "$dir/extension.js" | shasum | cut -d' ' -f1)
if [[ -f $stamp_file && $(cat "$stamp_file") == "$sha" ]] &&
  code --list-extensions | grep -qix 'rui.folder-theme'; then
  exit 0
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
version=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "$dir/package.json")

python3 - "$dir" "$tmp/folder-theme.vsix" "$version" <<'PY'
import sys, zipfile
src, out, version = sys.argv[1:4]
manifest = f'''<?xml version="1.0" encoding="utf-8"?>
<PackageManifest Version="2.0.0" xmlns="http://schemas.microsoft.com/developer/vsx-schema/2011">
  <Metadata>
    <Identity Language="en-US" Id="folder-theme" Version="{version}" Publisher="rui"/>
    <DisplayName>Folder Theme</DisplayName>
    <Description xml:space="preserve">Tints the window with the color of the active workspace folder</Description>
  </Metadata>
  <Installation><InstallationTarget Id="Microsoft.VisualStudio.Code"/></Installation>
  <Dependencies/>
  <Assets><Asset Type="Microsoft.VisualStudio.Code.Manifest" Path="extension/package.json" Addressable="true"/></Assets>
</PackageManifest>
'''
types = '''<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension=".json" ContentType="application/json"/>
  <Default Extension=".js" ContentType="application/javascript"/>
  <Default Extension=".vsixmanifest" ContentType="text/xml"/>
</Types>
'''
with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
    z.writestr("[Content_Types].xml", types)
    z.writestr("extension.vsixmanifest", manifest)
    for f in ("package.json", "extension.js"):
        z.write(f"{src}/{f}", f"extension/{f}")
PY

code --install-extension "$tmp/folder-theme.vsix" --force
mkdir -p "$(dirname "$stamp_file")"
echo "$sha" > "$stamp_file"
echo "folder-theme: installed $version; reload VS Code windows to pick it up"

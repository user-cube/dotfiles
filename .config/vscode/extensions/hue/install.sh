#!/bin/bash
# Packages this extension as a .vsix and installs it into VS Code (`code` on PATH).
# Skips the install when nothing changed since the last one. Run by dotfiles-sync.
set -euo pipefail

dir=$(cd "$(dirname "$0")" && pwd)
command -v code >/dev/null || { echo "hue: 'code' not on PATH, skipped"; exit 0; }

stamp_file="$HOME/.cache/hue.sha"
sha=$(cat "$dir/package.json" "$dir/extension.js" "$dir/README.md" "$dir/icon.png" | shasum | cut -d' ' -f1)
installed=$(code --list-extensions)
if [[ -f $stamp_file && $(cat "$stamp_file") == "$sha" ]] && grep -qix 'rui.hue' <<<"$installed"; then
  exit 0
fi

# This extension was called folder-theme before it was renamed
if grep -qix 'rui.folder-theme' <<<"$installed"; then
  code --uninstall-extension rui.folder-theme
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
version=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "$dir/package.json")

python3 - "$dir" "$tmp/hue.vsix" "$version" <<'PY'
import json, sys, zipfile
from xml.sax.saxutils import escape
src, out, version = sys.argv[1:4]
pkg = json.load(open(f"{src}/package.json"))
manifest = f'''<?xml version="1.0" encoding="utf-8"?>
<PackageManifest Version="2.0.0" xmlns="http://schemas.microsoft.com/developer/vsx-schema/2011">
  <Metadata>
    <Identity Language="en-US" Id="{pkg["name"]}" Version="{version}" Publisher="{pkg["publisher"]}"/>
    <DisplayName>{escape(pkg["displayName"])}</DisplayName>
    <Description xml:space="preserve">{escape(pkg["description"])}</Description>
    <Categories>{",".join(pkg.get("categories", []))}</Categories>
  </Metadata>
  <Installation><InstallationTarget Id="Microsoft.VisualStudio.Code"/></Installation>
  <Dependencies/>
  <Assets>
    <Asset Type="Microsoft.VisualStudio.Code.Manifest" Path="extension/package.json" Addressable="true"/>
    <Asset Type="Microsoft.VisualStudio.Services.Content.Details" Path="extension/README.md" Addressable="true"/>
    <Asset Type="Microsoft.VisualStudio.Services.Icons.Default" Path="extension/icon.png" Addressable="true"/>
  </Assets>
</PackageManifest>
'''
types = '''<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension=".json" ContentType="application/json"/>
  <Default Extension=".js" ContentType="application/javascript"/>
  <Default Extension=".md" ContentType="text/markdown"/>
  <Default Extension=".png" ContentType="image/png"/>
  <Default Extension=".vsixmanifest" ContentType="text/xml"/>
</Types>
'''
with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
    z.writestr("[Content_Types].xml", types)
    z.writestr("extension.vsixmanifest", manifest)
    for f in ("package.json", "extension.js", "README.md", "icon.png"):
        z.write(f"{src}/{f}", f"extension/{f}")
PY

code --install-extension "$tmp/hue.vsix" --force
mkdir -p "$(dirname "$stamp_file")"
echo "$sha" > "$stamp_file"
rm -f "$HOME/.cache/folder-theme.sha"
echo "hue: installed $version; reload VS Code windows to pick it up"

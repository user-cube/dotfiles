#!/bin/bash
# Packages this extension as a .vsix and installs it into VS Code (`code` on PATH).
# Skips the install when nothing changed since the last one. Run by dotfiles-sync.
set -euo pipefail

dir=$(cd "$(dirname "$0")" && pwd)
command -v code >/dev/null || { echo "hue: 'code' not on PATH, skipped"; exit 0; }

stamp_file="$HOME/.cache/hue.sha"
sha=$(cat "$dir/package.json" "$dir/extension.js" "$dir/README.md" "$dir/icon.png" | shasum | cut -d' ' -f1)
if [[ -f $stamp_file && $(cat "$stamp_file") == "$sha" ]] && code --list-extensions | grep -ix 'rui.hue' >/dev/null; then
  exit 0
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
version=$(jq -r .version "$dir/package.json")

# A .vsix is a zip: the manifest, the content types and the files under extension/
# (@html escapes the text that lands in XML)
mkdir -p "$tmp/vsix/extension"
cp "$dir/package.json" "$dir/extension.js" "$dir/README.md" "$dir/icon.png" "$tmp/vsix/extension/"
jq -r '"<?xml version=\"1.0\" encoding=\"utf-8\"?>
<PackageManifest Version=\"2.0.0\" xmlns=\"http://schemas.microsoft.com/developer/vsx-schema/2011\">
  <Metadata>
    <Identity Language=\"en-US\" Id=\"\(.name)\" Version=\"\(.version)\" Publisher=\"\(.publisher)\"/>
    <DisplayName>\(.displayName | @html)</DisplayName>
    <Description xml:space=\"preserve\">\(.description | @html)</Description>
    <Categories>\(.categories // [] | join(","))</Categories>
  </Metadata>
  <Installation><InstallationTarget Id=\"Microsoft.VisualStudio.Code\"/></Installation>
  <Dependencies/>
  <Assets>
    <Asset Type=\"Microsoft.VisualStudio.Code.Manifest\" Path=\"extension/package.json\" Addressable=\"true\"/>
    <Asset Type=\"Microsoft.VisualStudio.Services.Content.Details\" Path=\"extension/README.md\" Addressable=\"true\"/>
    <Asset Type=\"Microsoft.VisualStudio.Services.Icons.Default\" Path=\"extension/icon.png\" Addressable=\"true\"/>
  </Assets>
</PackageManifest>"' "$dir/package.json" > "$tmp/vsix/extension.vsixmanifest"
cat > "$tmp/vsix/[Content_Types].xml" <<'XML'
<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension=".json" ContentType="application/json"/>
  <Default Extension=".js" ContentType="application/javascript"/>
  <Default Extension=".md" ContentType="text/markdown"/>
  <Default Extension=".png" ContentType="image/png"/>
  <Default Extension=".vsixmanifest" ContentType="text/xml"/>
</Types>
XML
(cd "$tmp/vsix" && zip -qrXD "$tmp/hue.vsix" "[Content_Types].xml" extension.vsixmanifest extension)

code --install-extension "$tmp/hue.vsix" --force
mkdir -p "$(dirname "$stamp_file")"
echo "$sha" > "$stamp_file"
echo "hue: installed $version; reload VS Code windows to pick it up"

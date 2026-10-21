# ws: open a VS Code workspace from ~/workspaces (VSCODE_WORKSPACES overrides).
#   ws                              pick one with fzf
#   ws NAME                         open NAME.code-workspace
#   ws -n NAME [-t TPL]... DIR[:TAG]...   create NAME.code-workspace with DIR(s) as folders, then open it
# TAG marks a folder in the Explorer: red, green, blue, yellow, purple, orange, brown,
# white or black color its name (Folder Path Color extension, recommended in the
# workspace); anything else (an emoji, "TF") is put in front of its name.
# -t merges settings and recommended extensions from ~/.config/vscode/templates/TPL.json
# (VSCODE_TEMPLATES overrides); repeat it to combine, e.g. -t terraform -t helm.
# In VS Code, ⌘R switches between recent workspaces.
ws() {
  local dir=${VSCODE_WORKSPACES:-$HOME/workspaces} name
  local tpldir=${VSCODE_TEMPLATES:-$HOME/.config/vscode/templates}
  mkdir -p "$dir"
  if [[ $1 == -n ]]; then
    name=$2
    [[ -n $name && $# -ge 3 ]] || { echo "usage: ws -n NAME [-t TPL]... DIR[:TAG]..."; return 1; }
    [[ -e $dir/$name.code-workspace ]] && { echo "ws: $name already exists"; return 1; }
    shift 2
    local tpls=()
    while [[ $1 == -t ]]; do
      [[ -f $tpldir/$2.json ]] || { echo "ws: no template '$2' (have: ${(j:, :)${(@)${(f)"$(print -rl -- $tpldir/*.json(N:t:r))"}}})"; return 1; }
      tpls+=("$tpldir/$2.json"); shift 2
    done
    [[ $# -ge 1 ]] || { echo "usage: ws -n NAME [-t TPL]... DIR[:TAG]..."; return 1; }
    # Folder paths are relative to the workspace file, so the file stays portable
    python3 - "$dir" "${#tpls}" "${tpls[@]}" "$@" > "$dir/$name.code-workspace" <<'PY' || { rm -f "$dir/$name.code-workspace"; return 1; }
import json, os, sys
dir, n = sys.argv[1], int(sys.argv[2])
tpls, folders = sys.argv[3:3 + n], sys.argv[3 + n:]

def merge(a, b):
    for k, v in b.items():
        if isinstance(v, dict) and isinstance(a.get(k), dict):
            merge(a[k], v)
        else:
            a[k] = v

settings, recs = {}, []
for t in tpls:
    doc = json.load(open(t))
    merge(settings, doc.get("settings", {}))
    recs += [r for r in doc.get("extensions", {}).get("recommendations", []) if r not in recs]

# Palette colors color the folder name (Folder Path Color extension); any other
# TAG (an emoji, "TF") is prefixed to the name. The extension has 6 named colors
# and 6 hex slots, used here for the rest.
named = {"red": "red", "green": "green", "blue": "blue", "yellow": "yellow", "purple": "magenta"}
slots = {"orange": ("custom1", "#f0883e"), "brown": ("custom2", "#b07d56"),
         "white": ("custom3", "#e6e6e6"), "black": ("custom4", "#7a7a7a")}
folder_colors, hexes, entries = [], {}, []

for arg in folders:
    path, _, tag = arg.rpartition(":")
    if not path or os.path.exists(arg):
        path, tag = arg, ""
    real = os.path.realpath(path)
    base = os.path.basename(real)
    entry = {"path": os.path.relpath(real, dir)}
    if tag in named or tag in slots:
        color = named.get(tag) or slots[tag][0]
        if tag in slots:
            hexes["folderPathColor." + slots[tag][0]] = slots[tag][1]
        # The glob is what makes the extension match a workspace root
        folder_colors.append({"path": base + "*", "color": color})
    elif tag:
        entry["name"] = tag + " " + base
    entries.append(entry)

if folder_colors:
    settings["folder-path-color.folders"] = folder_colors
    if hexes:
        settings.setdefault("workbench.colorCustomizations", {}).update(hexes)
    if "visbydev.folder-path-color" not in recs:
        recs.append("visbydev.folder-path-color")

ws = {"folders": entries}
if settings:
    ws["settings"] = settings
if recs:
    ws["extensions"] = {"recommendations": recs}
print(json.dumps(ws, indent=2, ensure_ascii=False))
PY
  else
    name=${1:-$(print -rl -- $dir/*.code-workspace(N:t:r) | fzf --prompt='workspace> ')}
    [[ -n $name ]] || return
  fi
  code "$dir/$name.code-workspace"
}

_ws() { _values workspace ${${(f)"$(print -rl -- ${VSCODE_WORKSPACES:-$HOME/workspaces}/*.code-workspace(N:t:r))"}} }
compdef _ws ws

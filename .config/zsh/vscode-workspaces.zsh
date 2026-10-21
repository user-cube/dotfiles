# ws: open a VS Code workspace from ~/workspaces (VSCODE_WORKSPACES overrides).
#   ws                                          pick one with fzf
#   ws NAME                                     open NAME.code-workspace
#   ws -n NAME [-t TPL]... [-c COLOR] DIR...    create NAME.code-workspace with DIR(s) as folders, then open it
# -t merges settings and recommended extensions from ~/.config/vscode/templates/TPL.json
# (VSCODE_TEMPLATES overrides); repeat it to combine, e.g. -t terraform -t helm.
# -c colors the window's title bar so workspaces are told apart at a glance:
# red, orange, yellow, green, teal, blue, purple, pink, or a #rrggbb.
# In VS Code, ⌘R switches between recent workspaces.
ws() {
  local dir=${VSCODE_WORKSPACES:-$HOME/workspaces} name
  local tpldir=${VSCODE_TEMPLATES:-$HOME/.config/vscode/templates}
  local usage="usage: ws -n NAME [-t TPL]... [-c COLOR] DIR..."
  mkdir -p "$dir"
  if [[ $1 == -n ]]; then
    name=$2
    [[ -n $name && $# -ge 3 ]] || { echo $usage; return 1; }
    [[ -e $dir/$name.code-workspace ]] && { echo "ws: $name already exists"; return 1; }
    shift 2
    local tpls=() color=
    while [[ $1 == -t || $1 == -c ]]; do
      [[ -n $2 ]] || { echo $usage; return 1; }
      if [[ $1 == -t ]]; then
        [[ -f $tpldir/$2.json ]] || { echo "ws: no template '$2' (have: ${(j:, :)${(@)${(f)"$(print -rl -- $tpldir/*.json(N:t:r))"}}})"; return 1; }
        tpls+=("$tpldir/$2.json")
      else
        color=$2
      fi
      shift 2
    done
    [[ $# -ge 1 ]] || { echo $usage; return 1; }
    # Folder paths are relative to the workspace file, so the file stays portable
    python3 - "$dir" "$color" "${#tpls}" "${tpls[@]}" "$@" > "$dir/$name.code-workspace" <<'PY' || { rm -f "$dir/$name.code-workspace"; return 1; }
import json, os, re, sys
dir, color, n = sys.argv[1], sys.argv[2], int(sys.argv[3])
tpls, folders = sys.argv[4:4 + n], sys.argv[4 + n:]

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

# Title bar: (active, inactive) backgrounds, dark enough for white text
palette = {"red": ("#7a2b2b", "#4a1a1a"), "orange": ("#8a4b14", "#5a3010"),
           "yellow": ("#7a6a10", "#4d4309"), "green": ("#1f5f33", "#133a20"),
           "teal": ("#176b6b", "#0e4141"), "blue": ("#1f4f8a", "#132f52"),
           "purple": ("#5a2f8a", "#371c55"), "pink": ("#8a2f5f", "#551c3a")}
if color:
    if color in palette:
        active, inactive = palette[color]
    elif re.fullmatch(r"#[0-9a-fA-F]{6}", color):
        active = inactive = color
    else:
        sys.exit("ws: unknown color '%s' (have: %s, or #rrggbb)" % (color, ", ".join(palette)))
    merge(settings, {"workbench.colorCustomizations": {
        "titleBar.activeBackground": active, "titleBar.inactiveBackground": inactive,
        "titleBar.activeForeground": "#ffffff", "titleBar.inactiveForeground": "#ffffffaa"}})

ws = {"folders": [{"path": os.path.relpath(os.path.realpath(f), dir)} for f in folders]}
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

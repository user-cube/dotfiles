# ws: open a VS Code workspace from ~/workspaces (VSCODE_WORKSPACES overrides).
#   ws                                    pick one with fzf
#   ws NAME                               open NAME.code-workspace
#   ws -n NAME [-t TPL]... DIR[:COLOR]... create NAME.code-workspace with DIR(s) as folders, then open it
# -t merges settings and recommended extensions from ~/.config/vscode/templates/TPL.json
# (VSCODE_TEMPLATES overrides); repeat it to combine, e.g. -t terraform -t helm.
# COLOR tints the whole window with that color while you work in the folder (folder-theme
# extension, installed by dotfiles-sync): red, orange, yellow, green, teal, blue, purple,
# pink or #rrggbb. Without COLOR the folder gets one from its name; change it later with
# the "Folder Theme: Set Color" command. ! marks production: DIR! all of the folder, DIR:COLOR!hetzner/prod,other/path only
# those paths (globs allowed): red title bar and a PROD marker while you work in them.
# In VS Code, ⌘R switches between recent workspaces.
ws() {
  local dir=${VSCODE_WORKSPACES:-$HOME/workspaces} name
  local tpldir=${VSCODE_TEMPLATES:-$HOME/.config/vscode/templates}
  local usage="usage: ws -n NAME [-t TPL]... DIR[:COLOR]..."
  mkdir -p "$dir"
  if [[ $1 == -n ]]; then
    name=$2
    [[ -n $name && $# -ge 3 ]] || { echo $usage; return 1; }
    [[ -e $dir/$name.code-workspace ]] && { echo "ws: $name already exists"; return 1; }
    shift 2
    local tpls=()
    while [[ $1 == -t ]]; do
      [[ -n $2 ]] || { echo $usage; return 1; }
      [[ -f $tpldir/$2.json ]] || { echo "ws: no template '$2' (have: ${(j:, :)${(@)${(f)"$(print -rl -- $tpldir/*.json(N:t:r))"}}})"; return 1; }
      tpls+=("$tpldir/$2.json"); shift 2
    done
    [[ $# -ge 1 ]] || { echo $usage; return 1; }
    # Folder paths are relative to the workspace file, so the file stays portable
    python3 - "$dir" "${#tpls}" "${tpls[@]}" "$@" > "$dir/$name.code-workspace" <<'PY' || { rm -f "$dir/$name.code-workspace"; return 1; }
import json, os, re, sys
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

# Colors the folder-theme extension knows (.config/vscode/extensions/folder-theme)
colors = ["red", "orange", "yellow", "green", "teal", "blue", "purple", "pink"]
entries, theme, danger = [], {}, []
for arg in folders:
    # DIR! (all of it) or DIR!sub/path,other/path (only those paths): dangerous (production)
    arg, bang, subs = arg.partition("!")
    path, _, tag = arg.rpartition(":")
    if not path or os.path.exists(arg):
        path, tag = arg, ""
    real = os.path.realpath(path)
    entries.append({"path": os.path.relpath(real, dir)})
    if bang:
        base = os.path.basename(real)
        danger += [base + "/" + sub.strip("/") for sub in subs.split(",") if sub.strip("/")] or [base]
    if not tag:
        continue
    if tag not in colors and not re.fullmatch(r"#[0-9a-fA-F]{6}", tag):
        sys.exit("ws: unknown color '%s' (have: %s, or #rrggbb)" % (tag, ", ".join(colors)))
    theme[os.path.basename(real)] = tag

if theme:
    settings["folderTheme.folders"] = theme
if danger:
    settings["folderTheme.danger"] = danger

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

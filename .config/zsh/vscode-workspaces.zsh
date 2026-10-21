# ws: open a VS Code workspace from ~/workspaces (VSCODE_WORKSPACES overrides).
#   ws            pick one with fzf
#   ws NAME       open NAME.code-workspace
#   ws -n NAME DIR...   create NAME.code-workspace with DIR(s) as folders, then open it
# In VS Code, ⌘R switches between recent workspaces.
ws() {
  local dir=${VSCODE_WORKSPACES:-$HOME/workspaces} name
  mkdir -p "$dir"
  if [[ $1 == -n ]]; then
    name=$2
    [[ -n $name && $# -ge 3 ]] || { echo "usage: ws -n NAME DIR..."; return 1; }
    [[ -e $dir/$name.code-workspace ]] && { echo "ws: $name already exists"; return 1; }
    shift 2
    # Folder paths are relative to the workspace file, so the file stays portable
    local f folders=()
    for f in "$@"; do
      folders+=("{\"path\": \"$(python3 -c 'import os,sys; print(os.path.relpath(os.path.realpath(sys.argv[1]), sys.argv[2]))' "$f" "$dir")\"}")
    done
    printf '{\n  "folders": [%s]\n}\n' "${(j:, :)folders}" > "$dir/$name.code-workspace"
  else
    name=${1:-$(print -rl -- $dir/*.code-workspace(N:t:r) | fzf --prompt='workspace> ')}
    [[ -n $name ]] || return
  fi
  code "$dir/$name.code-workspace"
}

_ws() { _values workspace ${${(f)"$(print -rl -- ${VSCODE_WORKSPACES:-$HOME/workspaces}/*.code-workspace(N:t:r))"}} }
compdef _ws ws

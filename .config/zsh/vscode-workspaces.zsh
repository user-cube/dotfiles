# ws: open a VS Code workspace from ~/workspaces (VSCODE_WORKSPACES overrides).
#   ws                                    pick one with fzf
#   ws NAME                               open NAME.code-workspace
#   ws -n NAME [-t TPL]... DIR[:COLOR]... create NAME.code-workspace with DIR(s) as folders, then open it
# -t merges settings and recommended extensions from ~/.config/vscode/templates/TPL.json
# (VSCODE_TEMPLATES overrides); repeat it to combine, e.g. -t terraform -t helm.
# COLOR tints the whole window with that color while you work in the folder (Hue
# extension, installed by dotfiles-sync): red, orange, yellow, green, teal, blue, purple,
# pink or #rrggbb. Without COLOR the folder gets one from its name; change it later with
# the "Hue: Set Color" command. ! marks production: DIR! all of the folder, DIR:COLOR!hetzner/prod,other/path only
# those paths (globs allowed): red title bar and a PROD marker while you work in them.
# In VS Code, ⌘R switches between recent workspaces.
ws() {
  setopt local_options extended_glob # sub=${${sub##/#}%%/#}
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
    # Colors the Hue extension knows (.config/vscode/extensions/hue)
    local colors=(red orange yellow green teal blue purple pink)
    local -a entries danger paths theme # theme: folder, color, folder, color, ...
    local arg p tag bang subs real sub
    for arg; do
      # DIR! (all of it) or DIR!sub/path,other/path (only those paths): dangerous (production)
      bang= subs=
      [[ $arg == *'!'* ]] && bang=1 subs=${arg#*!} arg=${arg%%!*}
      p=${arg%:*} tag=${arg##*:}
      [[ $arg != *:* || -z $p || -e $arg ]] && p=$arg tag=
      real=${p:A}
      # Folder paths are relative to the workspace file, so the file stays portable
      entries+=("$(_ws_relpath "$real" "${dir:a}")")
      if [[ -n $bang ]]; then
        paths=()
        for sub in ${(s:,:)subs}; do
          sub=${${sub##/#}%%/#}
          [[ -n $sub ]] && paths+=("${real:t}/$sub")
        done
        danger+=(${paths:-${real:t}})
      fi
      [[ -n $tag ]] || continue
      if [[ -z ${colors[(r)$tag]} && ! $tag =~ '^#[0-9a-fA-F]{6}$' ]]; then
        print -u2 "ws: unknown color '$tag' (have: ${(j:, :)colors}, or #rrggbb)"
        return 1
      fi
      theme+=("${real:t}" "$tag")
    done

    # Templates: settings merged in order (objects deep), extension recommendations without repeats
    local tjson='[]' json
    (( $#tpls )) && { tjson=$(jq -s . "${tpls[@]}") || return 1; }
    json=$(jq -n --argjson tpls "$tjson" \
      --argjson folders "$(jq -n '$ARGS.positional' --args -- "${entries[@]}")" \
      --argjson danger "$(jq -n '$ARGS.positional' --args -- "${danger[@]}")" \
      --argjson theme "$(jq -n '$ARGS.positional' --args -- "${theme[@]}")" '
      (reduce $tpls[] as $t ({}; . * ($t.settings // {}))) as $settings
      | (reduce ($tpls[] | .extensions.recommendations // [] | .[]) as $r ([];
          if any(.[]; . == $r) then . else . + [$r] end)) as $recs
      | (reduce range(0; $theme | length; 2) as $i ({}; .[$theme[$i]] = $theme[$i + 1])) as $hue
      | ($settings
         | if $hue != {} then .["hue.folders"] = $hue end
         | if $danger != [] then .["hue.danger"] = $danger end) as $settings
      | {folders: ($folders | map({path: .}))}
      | if $settings != {} then .settings = $settings end
      | if $recs != [] then .extensions = {recommendations: $recs} end') || return 1
    print -r -- "$json" > "$dir/$name.code-workspace"
  else
    name=${1:-$(print -rl -- $dir/*.code-workspace(N:t:r) | fzf --prompt='workspace> ')}
    [[ -n $name ]] || return
  fi
  code "$dir/$name.code-workspace"
}

# $1 relative to the directory $2, both absolute: ../../work/github/infra
_ws_relpath() {
  local -a to=(${(s:/:)1}) from=(${(s:/:)2}) rel=()
  local i=1 j
  while (( i <= $#to && i <= $#from )) && [[ $to[i] == $from[i] ]]; do (( i++ )); done
  for (( j = i; j <= $#from; j++ )); do rel+=(..); done
  rel+=(${to[i,-1]})
  print -r -- ${${(j:/:)rel}:-.}
}

_ws() { _values workspace ${${(f)"$(print -rl -- ${VSCODE_WORKSPACES:-$HOME/workspaces}/*.code-workspace(N:t:r))"}} }
compdef _ws ws

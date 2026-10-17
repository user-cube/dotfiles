#!/bin/zsh
# prefix S (also in the ⌘⇧H list): manage tmux sessions in an fzf list.
# Enter switches, ctrl-x kills, ctrl-r renames, ctrl-n creates one, ctrl-l loads
# the ones in ~/.config/tmux/sessions.local (preload-sessions.sh), Esc closes.
# The list reloads after each change, so several can be killed in a row.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
k=$'\e[38;2;102;217;239m'    # Monokai cyan: keys
g=$'\e[38;2;117;113;94m'     # Monokai comment grey: details
r=$'\e[0m'

list="tmux list-sessions -F '#{session_name}	#{session_windows} tabs#{?session_attached,  · here,}' \
  | awk -F '\t' '{ printf \"%s\t%-24s ${g}%s${r}\n\", \$1, \$1, \$2 }'"

# Run by fzf's execute() in its own shell; {1} is the session name, quoted by fzf
rename='printf "rename %s to: " {1}; read -r n; [ -n "$n" ] && tmux rename-session -t={1} "$n"'
create='printf "new session name: "; read -r n; [ -n "$n" ] && tmux new-session -d -s "$n" -c "$HOME"'

pick=$(eval $list | fzf --ansi --delimiter=$'\t' --with-nth=2 --prompt='session › ' --reverse \
  --height=100% --border=none --info=hidden --no-scrollbar \
  --header="${k}enter${r} switch  ${k}ctrl-x${r} kill  ${k}ctrl-r${r} rename"$'\n'"${k}ctrl-n${r} new     ${k}ctrl-l${r} load sessions.local" \
  --bind="ctrl-x:execute-silent(tmux kill-session -t={1})+reload($list)" \
  --bind="ctrl-r:execute($rename)+reload($list)" \
  --bind="ctrl-n:execute($create)+reload($list)" \
  --bind="ctrl-l:execute-silent(~/.config/tmux/preload-sessions.sh)+reload($list)" \
  --color='bg+:#3e3d32,fg+:#f8f8f2,hl:#a6e22e,hl+:#a6e22e,pointer:#a6e22e,prompt:#a6e22e,header:#75715e') || exit 0

tmux switch-client -t="${pick%%$'\t'*}"

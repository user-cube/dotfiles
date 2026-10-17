#!/bin/zsh
# ⌘⇧H (prefix ?): every ⌘ shortcut (see ghostty/config) in an fzf list. Arrows
# or typing pick one, Enter runs it, Esc closes. Keep in sync when a shortcut is
# added. $1: the client to run the picked shortcut on.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
client=$1
g=$'\e[38;2;117;113;94m'     # Monokai comment grey: section
k=$'\e[38;2;102;217;239m'    # Monokai cyan: keys
r=$'\e[0m'

# tmux keys to send <TAB> section <TAB> ⌘ shortcut <TAB> what it does
entries=(
  'C-b c	tabs	⌘T	new tab'
  'C-b X	tabs	⌘W	close pane / tab'
  'C-b ,	tabs	⌘⇧R	rename tab'
  'C-b w	tabs	⌘1-9	go to tab (pick from a list)'
  'C-b p	tabs	⌘⇧←	previous tab'
  'C-b n	tabs	⌘⇧→	next tab'
  'C-b %	panes	⌘D	split right'
  'C-b "	panes	⌘⇧D	split down'
  'C-b q	panes	⌘⌥←↑↓→	move between panes (pick by number)'
  'C-b z	panes	⌘⇧F	zoom pane'
  'C-b E	panes	⌘⇧E	even pane sizes'
  'C-b f	sessions	⌘O	open repo'
  'C-b O	sessions	⌘⇧O	open ssh host'
  'C-b s	sessions	⌘⇧S	switch session'
  'C-b S	sessions	C-b S	manage sessions (kill, rename, new, load list)'
  'C-b g	popups	⌘G	lazygit'
  'C-b G	popups	⌘⇧G	diff'
  'C-b e	popups	⌘E	files (yazi)'
  'C-b D	popups	⌘⇧L	docker'
  'C-b K	popups	⌘K	k8s (k9s)'
  'C-b J	popups	⌘J	scratch terminal'
  'C-b Y	copy	⌘Y	copy a token on screen'
  'C-b u	copy	⌘U	open a URL from the pane'
  'C-b a	copy	⌘A	select all'
  'F12	other	F12	nested tmux on / off'
)

pick=$(for e in $entries; do
  IFS=$'\t' read -r keys section shortcut desc <<< $e
  printf "%s\t${g}%-9s${k}%-8s${r}%s\n" $keys $section $shortcut $desc
done | fzf --ansi --delimiter=$'\t' --with-nth=2 --prompt='› ' --reverse \
  --height=100% --border=none --info=hidden --no-scrollbar \
  --color='bg+:#3e3d32,fg+:#f8f8f2,hl:#a6e22e,hl+:#a6e22e,pointer:#a6e22e,prompt:#a6e22e') || exit 0

# Run it after the popup has closed, so the keys reach the pane, not the popup
keys=(${=pick%%$'\t'*})
tmux run-shell -b -d 0.1 -C "send-keys -K -c ${(qq)client} ${(j: :)${(@qq)keys}}"

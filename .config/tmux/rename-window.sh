#!/bin/zsh
# ⌘⇧R (prefix ,): rename a tab in a small popup, prefilled with its current
# name. Enter saves, Ctrl-U clears, Esc or Ctrl-C cancels, empty changes nothing.
win=$1
name=$(tmux display -p -t "$win" '#W')
bindkey -e
KEYTIMEOUT=1
bindkey '\e' send-break
vared -p '%F{green}❯%f ' name || exit 0
[[ -n $name ]] && tmux rename-window -t "$win" -- "$name"

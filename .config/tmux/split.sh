#!/bin/bash
# ⌘D / ⌘⇧D (prefix % / "): split the pane. When the pane is in an interactive
# ssh session (`ssh [options] host`, no remote command), the new pane opens the
# same ssh (instant with ControlMaster, see .config/ssh/controlmaster.conf);
# anything else, including `ssh host <command>`, gets a local shell as before.
# Usage: split.sh -h|-v <pane_id> <pane_tty> <pane_current_path>
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
DIR_FLAG="$1" PANE="$2" TTY="$3" CWD="$4"

# Foreground ssh on the pane's tty (stat has "+"), if any
SSH_ARGS=$(ps -o stat=,args= -t "$TTY" 2>/dev/null \
  | awk '$1 ~ /\+/ && $2 == "ssh" {$1 = ""; sub(/^ +/, ""); print; exit}')

# Only `ssh [options] host`: options that take a value skip it, -N/-f/-O/-W
# mean no interactive shell, and any word after the host is a remote command.
interactive_host() {
  local host="" arg
  set -- $SSH_ARGS
  shift # "ssh"
  while [ $# -gt 0 ]; do
    arg="$1"
    shift
    if [ -n "$host" ]; then return 1; fi
    case "$arg" in
      -N* | -f* | -O* | -W* | -G* | -V*) return 1 ;;
      -[BbcDEeFIiJLlmopQRSw]) shift ;; # value in the next word
      -*) ;;
      *) host="$arg" ;;
    esac
  done
  [ -n "$host" ]
}

if [ -n "$SSH_ARGS" ] && interactive_host; then
  tmux split-window "$DIR_FLAG" -t "$PANE" -c "$CWD" "$SSH_ARGS"
else
  tmux split-window "$DIR_FLAG" -t "$PANE" -c "$CWD"
fi

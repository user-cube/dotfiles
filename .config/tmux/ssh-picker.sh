#!/bin/bash
# ⌘⇧O (prefix O): fuzzy-pick an ssh host and open it in a new tmux tab named
# after the host; the tab closes when the ssh ends. Hosts: the `Host` names (no
# wildcards) in the ssh config, following its `Include` lines, then every host
# already connected to (known_hosts, unless hashed). Env overrides for tests:
# SSH_PICKER_CONFIG, SSH_PICKER_KNOWN_HOSTS. --rex opens it in a Rex tab instead.
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
CONFIG=${SSH_PICKER_CONFIG:-$HOME/.ssh/config}
KNOWN_HOSTS=${SSH_PICKER_KNOWN_HOSTS:-$HOME/.ssh/known_hosts}

# Host names from a config file and, recursively, the files it Includes
# (relative paths are under ~/.ssh, globs allowed, as ssh does)
config_hosts() { # config_hosts <file> <depth>
  local file="$1" depth="$2" inc path
  [ -f "$file" ] && [ "$depth" -lt 5 ] || return 0
  awk 'tolower($1) == "host" { for (i = 2; i <= NF; i++) print $i }' "$file"
  awk 'tolower($1) == "include" { for (i = 2; i <= NF; i++) print $i }' "$file" |
    while read -r inc; do
      inc="${inc/#\~/$HOME}"
      [ "${inc:0:1}" = / ] || inc="$HOME/.ssh/$inc"
      for path in $inc; do config_hosts "$path" $((depth + 1)); done
    done
}

# known_hosts: first field, comma-separated names; [host]:port -> host
known_hosts() {
  [ -f "$KNOWN_HOSTS" ] || return 0
  awk '$1 !~ /^[|@#]/ { print $1 }' "$KNOWN_HOSTS" | tr ',' '\n' |
    sed -E 's/^\[([^]]+)\]:[0-9]+$/\1/'
}

# Skip wildcards, IPv6 (link-local noise) and git forges (no shell there)
HOSTS=$({ config_hosts "$CONFIG" 0; known_hosts; } | grep -v '[*?!:]' |
  grep -vxE 'github\.com|gitlab\.com|bitbucket\.org|ssh\.dev\.azure\.com|codeberg\.org' |
  awk 'NF && !seen[$0]++')
[ -z "$HOSTS" ] && { echo "no hosts in $CONFIG (and its Includes) or $KNOWN_HOSTS" >&2; sleep 3; exit 1; }

HOST=$(echo "$HOSTS" | fzf --prompt='ssh › ' --height=100% --reverse --border=none)
[ -z "$HOST" ] && exit 0
[ "$1" = --rex ] && exec rex run --label "$HOST" -- ssh "$HOST"
tmux new-window -n "$HOST" "ssh $(printf %q "$HOST")"

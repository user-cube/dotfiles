#!/bin/bash
# Backup / restore of the per-machine files that never go to git, as one
# .tar.gz attached to a Bitwarden secure note "Local files (<machine>)".
#   local-files.sh backup              this machine -> Bitwarden (replaces the old copy)
#   local-files.sh restore [machine]   Bitwarden -> this machine; never overwrites
#                                      existing files (--force to overwrite)
#   local-files.sh list                machines with a backup
# Bitwarden session: BW_SESSION if exported, else the one kept by the dotfiles
# `bw` wrapper (/var/root/.bitwarden.session, read with sudo / Touch ID).
set -euo pipefail
export PATH="$PATH:/opt/homebrew/bin:/usr/local/bin"

# Paths relative to $HOME; missing ones are skipped
FILES=(
  .zshrc.local
  .gitconfig-work
  .gitconfig-hook
  .config/tmux/sessions.local
  .config/sketchybar/checks
  .config/sketchybar/aws_profiles
  .kube/config
  .terraformrc
  .terraform.d/credentials.tfrc.json
)
# Claude Code's MCP servers (tokens in their env). They live in ~/.claude.json
# next to the login and per-project state, so only the mcpServers key is saved
# (as this path inside the archive) and merged back on restore.
CLAUDE_JSON="$HOME/.claude.json"
MCP=".claude/mcp-servers.json"
PREFIX="Local files"
ATTACHMENT="local-files.tar.gz"
MACHINE=$(scutil --get LocalHostName 2>/dev/null || hostname -s)

die() { echo "local-files: $*" >&2; exit 1; }
command -v bw >/dev/null || die "bitwarden-cli (bw) is not installed"
command -v jq >/dev/null || die "jq is not installed"

if [ -z "${BW_SESSION:-}" ]; then
  BW_SESSION=$(sudo cat /var/root/.bitwarden.session 2>/dev/null) ||
    die 'no Bitwarden session: run `bw --regenerate-session-key` or export BW_SESSION'
  sudo -k
fi
export BW_SESSION

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

item_id() { # item_id <name> -> id of the item with exactly that name, or empty
  bw list items --search "$1" | jq -r --arg n "$1" '[.[] | select(.name == $n)][0].id // empty'
}

backup() {
  local name="$PREFIX ($MACHINE)" present=() f id
  for f in "${FILES[@]}"; do [ -f "$HOME/$f" ] && present+=("$f"); done
  [ ${#present[@]} -gt 0 ] || die "none of the files exist on this machine"
  # The export is written under $WORK only (removed on exit), never in ~/.claude
  if [ -f "$CLAUDE_JSON" ] && jq -e '(.mcpServers // {}) | length > 0' "$CLAUDE_JSON" >/dev/null; then
    mkdir -p "$WORK/mcp/.claude"
    (umask 077 && jq '{mcpServers}' "$CLAUDE_JSON" > "$WORK/mcp/$MCP")
    tar czf "$WORK/$ATTACHMENT" -C "$HOME" "${present[@]}" -C "$WORK/mcp" "$MCP"
    present+=(".claude.json  (MCP servers only: $(jq -r '.mcpServers | keys | join(", ")' "$WORK/mcp/$MCP"))")
  else
    tar czf "$WORK/$ATTACHMENT" -C "$HOME" "${present[@]}"
  fi

  bw sync >/dev/null
  id=$(item_id "$name")
  if [ -z "$id" ]; then
    id=$(bw get template item | jq --arg n "$name" \
      '.type = 2 | .secureNote = {type: 0} | .name = $n | .login = null
       | .notes = "Per-machine dotfiles, restore with: local-files.sh restore"' |
      bw encode | bw create item | jq -r .id)
    echo "created Bitwarden item: $name"
  fi
  # Replace the previous copy
  bw get item "$id" | jq -r --arg a "$ATTACHMENT" '.attachments // [] | .[] | select(.fileName == $a) | .id' |
    while read -r old; do bw delete attachment "$old" --itemid "$id" >/dev/null; done
  bw create attachment --file "$WORK/$ATTACHMENT" --itemid "$id" >/dev/null
  echo "backed up to \"$name\":"
  printf '  ~/%s\n' "${present[@]}"
}

list() {
  bw sync >/dev/null
  bw list items --search "$PREFIX (" | jq -r --arg p "$PREFIX (" \
    '.[] | select(.name | startswith($p)) | .name | ltrimstr($p) | rtrimstr(")")'
}

restore_mcp() { # merge the saved MCP servers into ~/.claude.json, the rest untouched
  local force="$1" saved="$WORK/mcp/$MCP" tmp kept
  mkdir -p "$WORK/mcp"
  tar xzf "$WORK/$ATTACHMENT" -C "$WORK/mcp" "$MCP"
  [ -f "$CLAUDE_JSON" ] || (umask 077 && echo '{}' > "$CLAUDE_JSON")
  # Without --force a server already set up here wins over the saved one
  kept=$(jq -r --slurpfile s "$saved" --arg force "$force" \
    '(.mcpServers // {}) as $here
     | [$s[0].mcpServers | keys[] | select(. as $k | $force != "--force" and ($here | has($k)))] | join(", ")' \
    "$CLAUDE_JSON")
  tmp=$(mktemp "$CLAUDE_JSON.XXXXXX")
  jq --slurpfile s "$saved" --arg force "$force" \
    '.mcpServers = (if $force == "--force" then (.mcpServers // {}) + $s[0].mcpServers
                    else $s[0].mcpServers + (.mcpServers // {}) end)' \
    "$CLAUDE_JSON" > "$tmp"
  chmod 600 "$tmp" && mv "$tmp" "$CLAUDE_JSON"
  echo "  ~/.claude.json  (MCP servers: $(jq -r '.mcpServers | keys | join(", ")' "$saved"))"
  [ -z "$kept" ] || echo "    already here, kept: $kept (--force to overwrite)"
}

restore() {
  local machine="${1:-$MACHINE}" force="${2:-}" name id f
  name="$PREFIX ($machine)"
  bw sync >/dev/null
  id=$(item_id "$name")
  [ -n "$id" ] || die "no backup \"$name\"; machines with one: $(list | tr '\n' ' ')"
  bw get attachment "$ATTACHMENT" --itemid "$id" --output "$WORK/$ATTACHMENT" >/dev/null
  echo "restoring \"$name\":"
  for f in $(tar tzf "$WORK/$ATTACHMENT"); do
    if [ "$f" = "$MCP" ]; then
      restore_mcp "$force"
    elif [ -e "$HOME/$f" ] && [ "$force" != --force ]; then
      echo "  ~/$f  (exists, kept; --force to overwrite)"
    else
      tar xzf "$WORK/$ATTACHMENT" -C "$HOME" "$f"
      echo "  ~/$f"
    fi
  done
}

case "${1:-}" in
  backup) backup ;;
  restore) shift; [ "${1:-}" = --force ] && set -- "" --force; restore "${1:-}" "${2:-}" ;;
  list) list ;;
  *) sed -n '2,9p' "$0" | sed 's/^# \{0,1\}//'; exit 1 ;;
esac

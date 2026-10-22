#!/bin/bash
# The steps after the dotfiles are copied into place (~/.config, ~/git_hooks,
# ~/.claude): build the Go tools, download the SketchyBar icon map, install the
# Hue VS Code extension, remove what the repo no longer installs. The one list
# of them, run by dotfiles-sync and by the playbook: a new step goes here.
#
# Prints a line for each thing it changed ("built", "downloaded", "installed",
# "removed"); nothing when all was up to date. Fails when a build fails.
set -euo pipefail
# The playbook's shell lacks Homebrew; kept where it is when already on PATH
case ":$PATH:" in *:/opt/homebrew/bin:*) ;; *) export PATH="/opt/homebrew/bin:$PATH" ;; esac

# Go tools: source directory -> binary in it. Rebuilt when a .go file or
# go.mod is newer than the binary.
tools=(
  "$HOME/git_hooks/go/commit-msg"                # global commit-msg hook (hook.commit-msg-go in .gitconfig)
  "$HOME/.config/dotfiles/local-files"           # local-backup / local-restore (Bitwarden)
  "$HOME/.config/sketchybar/plugins/check"       # SketchyBar connectivity checks
  "$HOME/.claude/hooks/deny-sensitive-reads"     # Claude Code PreToolUse hook (Bash and Read are blocked until built)
)
for dir in "${tools[@]}"; do
  bin="$dir/$(basename "$dir")"
  if [ -x "$bin" ] && [ -z "$(find "$dir" \( -name '*.go' -o -name go.mod \) -newer "$bin")" ]; then
    continue
  fi
  go build -C "$dir" -o "$bin" .
  echo "built $(basename "$dir")"
done

# SketchyBar app icons: the map of the installed font version (not in git);
# offline keeps the current one
"$HOME/.config/sketchybar/fetch-icon-map.sh" || echo "icon_map.sh: kept the current one" >&2

# VS Code: the Hue extension (skips when unchanged or without `code`)
"$HOME/.config/vscode/extensions/hue/install.sh"

# Files the repo no longer installs (rsync never deletes): ./obsolete
"$HOME/.config/dotfiles/remove-obsolete.sh"

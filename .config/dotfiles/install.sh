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

# lazygit reads ~/Library/Application Support/lazygit on macOS, also from the
# tmux popup, which never sees an env var from .zshrc: a link to the dotfiles'
# config. A config written there by hand is left alone (and said so)
lg_link="$HOME/Library/Application Support/lazygit/config.yml"
if [ ! -L "$lg_link" ]; then
  if [ -s "$lg_link" ]; then
    echo "lazygit: $lg_link has its own config, not linked to ~/.config/lazygit" >&2
  else
    mkdir -p "$(dirname "$lg_link")"
    ln -sf "$HOME/.config/lazygit/config.yml" "$lg_link"
    echo "linked lazygit config"
  fi
fi

# VS Code user settings and keybindings (terminal runs tmux, ⌘ shortcuts send
# the tmux prefix like Ghostty's): links to ~/.config/vscode. A file VS Code
# wrote there before is kept as .bak
for f in settings.json keybindings.json; do
  link="$HOME/Library/Application Support/Code/User/$f"
  [ -L "$link" ] && continue
  mkdir -p "$(dirname "$link")"
  if [ -s "$link" ]; then
    mv "$link" "$link.bak"
    echo "VS Code: kept the old $f as $f.bak"
  fi
  ln -sf "$HOME/.config/vscode/$f" "$link"
  echo "linked VS Code $f"
done

# Files the repo no longer installs (rsync never deletes): ./obsolete
"$HOME/.config/dotfiles/remove-obsolete.sh"

# dotfiles-sync: update this machine from the dotfiles repo without Ansible.
# Mirrors the playbook's `dotfiles` tag: pull, sync .config, copy the home
# files, then reload whatever is running. DOTFILES_DIR overrides ~/dotfiles.
dotfiles-sync() {
  local repo=${DOTFILES_DIR:-$HOME/dotfiles}
  [[ -d $repo/.git ]] || { echo "dotfiles-sync: $repo is not a git clone"; return 1; }

  git -C "$repo" pull --ff-only || return
  rsync -a "$repo/.config/" "$HOME/.config/" || return
  # SketchyBar app icons: the map of the installed font version (not in git); offline keeps the old one
  "$HOME/.config/sketchybar/fetch-icon-map.sh" || echo "icon_map.sh: kept the current one"
  rsync -a "$repo/git_hooks/" "$HOME/git_hooks/" || return
  # Global commit-msg hook (hook.commit-msg-go in .gitconfig)
  go build -C "$HOME/git_hooks/go/commit-msg" -o commit-msg commit-msg.go || return
  # local-backup / local-restore (Bitwarden)
  go build -C "$HOME/.config/dotfiles/local-files" -o local-files . || return
  # SketchyBar connectivity checks (the `checks` item)
  go build -C "$HOME/.config/sketchybar/plugins/check" -o check . || return
  cp "$repo/.zshrc" "$repo/.p10k.zsh" "$repo/.gitconfig" "$repo/.gitconfig-personal" "$HOME/" || return
  # Claude Code: settings, global instructions, hooks (claude/ and not .claude/,
  # which Claude Code would read as this repo's own project settings)
  mkdir -p "$HOME/.claude" && rsync -a "$repo/claude/" "$HOME/.claude/" || return
  # Claude Code's PreToolUse hook (settings.json blocks Bash and Read until it is built)
  go build -C "$HOME/.claude/hooks/deny-sensitive-reads" -o deny-sensitive-reads . || return
  # Files the repo no longer installs (rsync never deletes): .config/dotfiles/obsolete
  "$HOME/.config/dotfiles/remove-obsolete.sh" || return
  # .gitconfig-work holds a placeholder in the repo: only seed it, never overwrite
  [[ -f $HOME/.gitconfig-work ]] || cp "$repo/.gitconfig-work" "$HOME/"

  # VS Code: the Hue extension (skips when unchanged)
  "$HOME/.config/vscode/extensions/hue/install.sh" || return

  tmux source-file "$HOME/.config/tmux/tmux.conf" 2>/dev/null && echo "tmux: reloaded"
  pgrep -xq AeroSpace && aerospace reload-config && echo "AeroSpace: reloaded"
  pgrep -xq sketchybar && sketchybar --reload && echo "SketchyBar: reloaded"
  pgrep -xq borders && brew services restart borders >/dev/null && echo "borders: restarted"

  echo "Done. Run 'exec zsh' to load the new .zshrc; Ghostty keybinds need ⌘Q."
}

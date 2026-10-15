# dotfiles-sync: update this machine from the dotfiles repo without Ansible.
# Mirrors the playbook's `dotfiles` tag: pull, sync .config, copy the home
# files, then reload whatever is running. DOTFILES_DIR overrides ~/dotfiles.
dotfiles-sync() {
  local repo=${DOTFILES_DIR:-$HOME/dotfiles}
  [[ -d $repo/.git ]] || { echo "dotfiles-sync: $repo is not a git clone"; return 1; }

  git -C "$repo" pull --ff-only || return
  rsync -a "$repo/.config/" "$HOME/.config/" || return
  rsync -a "$repo/git_hooks/" "$HOME/git_hooks/" || return
  cp "$repo/.zshrc" "$repo/.p10k.zsh" "$repo/.gitconfig" "$repo/.gitconfig-personal" "$HOME/" || return
  # .gitconfig-work holds a placeholder in the repo: only seed it, never overwrite
  [[ -f $HOME/.gitconfig-work ]] || cp "$repo/.gitconfig-work" "$HOME/"

  tmux source-file "$HOME/.config/tmux/tmux.conf" 2>/dev/null && echo "tmux: reloaded"
  pgrep -xq AeroSpace && aerospace reload-config && echo "AeroSpace: reloaded"
  pgrep -xq sketchybar && sketchybar --reload && echo "SketchyBar: reloaded"
  pgrep -xq borders && brew services restart borders >/dev/null && echo "borders: restarted"

  echo "Done. Run 'exec zsh' to load the new .zshrc; Ghostty keybinds need ⌘Q."
}

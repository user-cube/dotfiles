# dotfiles-sync: update this machine from the dotfiles repo without Ansible.
# Mirrors the playbook's `dotfiles` tag: pull, sync .config, copy the home
# files, then reload whatever is running. DOTFILES_DIR overrides ~/dotfiles: it
# can be the clone being edited, set in ~/.zshrc.local (a branch or uncommitted
# changes there are installed as they are, and said so).
dotfiles-sync() {
  local repo=${DOTFILES_DIR:-$HOME/dotfiles}
  [[ -d $repo/.git ]] || { echo "dotfiles-sync: $repo is not a git clone"; return 1; }

  git -C "$repo" pull --ff-only || return
  # DOTFILES_DIR can be the clone being edited: it installs what is there, so
  # say so when that is a branch or uncommitted changes
  local branch=$(git -C "$repo" branch --show-current)
  local default=$(git -C "$repo" symbolic-ref -q --short refs/remotes/origin/HEAD)
  default=${default#origin/}
  [[ $branch == ${default:-main} ]] || echo "dotfiles-sync: installing branch '${branch:-detached HEAD}' of $repo, not ${default:-main}" >&2
  [[ -z $(git -C "$repo" status --porcelain) ]] || echo "dotfiles-sync: $repo has uncommitted changes, installed as they are" >&2
  rsync -a "$repo/.config/" "$HOME/.config/" || return
  rsync -a "$repo/git_hooks/" "$HOME/git_hooks/" || return
  cp "$repo/.zshrc" "$repo/.p10k.zsh" "$repo/.gitconfig" "$repo/.gitconfig-personal" "$HOME/" || return
  # Claude Code: settings, global instructions, hooks (claude/ and not .claude/,
  # which Claude Code would read as this repo's own project settings)
  mkdir -p "$HOME/.claude" && rsync -a "$repo/claude/" "$HOME/.claude/" || return
  # .gitconfig-work holds a placeholder in the repo: only seed it, never overwrite
  [[ -f $HOME/.gitconfig-work ]] || cp "$repo/.gitconfig-work" "$HOME/"

  # Go tools, SketchyBar icon map, Hue, obsolete files: the same steps as the
  # playbook; it gets the clone, as VS Code's settings link into it
  DOTFILES_DIR=$repo "$HOME/.config/dotfiles/install.sh" || return

  tmux source-file "$HOME/.config/tmux/tmux.conf" 2>/dev/null && echo "tmux: reloaded"
  pgrep -xq AeroSpace && aerospace reload-config && echo "AeroSpace: reloaded"
  pgrep -xq sketchybar && sketchybar --reload && echo "SketchyBar: reloaded"
  pgrep -xq borders && brew services restart borders >/dev/null && echo "borders: restarted"

  echo "Done. Run 'exec zsh' to load the new .zshrc; Ghostty keybinds need ⌘Q."
}

# dotfiles-diff [-p]: what dotfiles-sync would change on this machine, changing
# nothing: the files it would overwrite (changed here, or older), add (new) or
# remove (obsolete). -p also shows the diffs, from this machine to the repo
# (+ is what the sync brings). Compares with the upstream the sync would pull
# (fetched, never merged), or HEAD when there is none or the clone is ahead of
# it. Uncommitted changes are not compared. The builds of install.sh are not
# listed: they follow the sources.
dotfiles-diff() {
  local repo=${DOTFILES_DIR:-$HOME/dotfiles} patch=0
  [[ $1 == -p ]] && patch=1
  [[ -d $repo/.git ]] || { echo "dotfiles-diff: $repo is not a git clone"; return 1; }
  git -C "$repo" rev-parse -q --verify HEAD >/dev/null || { echo "dotfiles-diff: $repo has no commits"; return 1; }

  # The upstream when the sync's pull would fast-forward to it; HEAD when the
  # clone is ahead of it (commits not pushed yet: the sync installs those)
  local ref=HEAD
  git -C "$repo" fetch -q 2>/dev/null && git -C "$repo" rev-parse -q --verify '@{u}' >/dev/null &&
    git -C "$repo" merge-base --is-ancestor HEAD '@{u}' && ref='@{u}'
  local behind=$(git -C "$repo" rev-list --count "HEAD..$ref")
  [[ -z $(git -C "$repo" status --porcelain) ]] || echo "dotfiles-diff: $repo has uncommitted changes, not compared (the sync installs them)"
  echo "Compared with $(git -C "$repo" rev-parse --abbrev-ref "$ref") ($(git -C "$repo" rev-parse --short "$ref"))$( ((behind)) && echo ", $behind commit(s) ahead of the clone: dotfiles-sync pulls them")"

  # The repo as dotfiles-sync would install it: its tracked files at $ref
  local snap=$(mktemp -d)
  {
    setopt local_options pipe_fail
    git -C "$repo" archive "$ref" | tar -x -C "$snap" || { echo "dotfiles-diff: could not read $ref from $repo"; return 1; }

    local -a rows diffs # rows: "status path"; diffs: "here repo" pairs for -p
    local src dst line f
    for src dst in .config "$HOME/.config" git_hooks "$HOME/git_hooks" claude "$HOME/.claude"; do
      while IFS= read -r line; do
        f=${line#* }
        case $line in
          '>f+++++++'*) rows+=("new      $dst/$f") ;;
          '>f'*) rows+=("changed  $dst/$f"); diffs+=("$dst/$f" "$snap/$src/$f") ;;
        esac
      done < <(rsync -nai --checksum "$snap/$src/" "$dst/")
    done
    # The files it copies to ~ (.gitconfig-work only when missing)
    for f in .zshrc .p10k.zsh .gitconfig .gitconfig-personal; do
      if [[ ! -e $HOME/$f ]]; then
        rows+=("new      $HOME/$f")
      elif ! cmp -s "$snap/$f" "$HOME/$f"; then
        rows+=("changed  $HOME/$f"); diffs+=("$HOME/$f" "$snap/$f")
      fi
    done
    [[ -e $HOME/.gitconfig-work ]] || rows+=("new      $HOME/.gitconfig-work")
    # What remove-obsolete.sh would delete
    for f in ${(f)"$(grep -vE '^[[:space:]]*(#|$)' "$snap/.config/dotfiles/obsolete" 2>/dev/null)"}; do
      [[ -f $HOME/$f || -L $HOME/$f ]] && rows+=("removed  $HOME/$f")
    done

    if (( ! $#rows )); then
      echo "This machine matches it."
    else
      print -rl -- ${rows//$HOME/\~}
      if (( patch )); then
        local i
        for (( i = 1; i < $#diffs; i += 2 )); do
          git --no-pager diff --no-index -- "$diffs[i]" "$diffs[i+1]"
        done
      fi
    fi
  } always {
    [[ -d $snap ]] && rm -rf -- "$snap"
  }
}

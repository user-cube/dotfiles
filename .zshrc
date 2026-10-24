# PATH keeps only the first of each entry: nested shells (tmux, exec zsh)
# would otherwise prepend Homebrew, go/bin and the rest again
typeset -U path PATH

# direnv loads a folder's .envrc (and prints its "direnv: loading" lines) here,
# before instant prompt, so p10k doesn't warn about console output. The hook
# itself is set later by the oh-my-zsh direnv plugin.
(( ${+commands[direnv]} )) && emulate zsh -c "$(direnv export zsh)"

# Enable Powerlevel10k instant prompt. Must stay at the top of ~/.zshrc.
if [[ -r "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh" ]]; then
  source "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh"
fi

# --- Oh My Zsh ---
export ZSH="$HOME/.oh-my-zsh"

plugins=(
  git
  zsh-autosuggestions
  colored-man-pages
  docker
  docker-compose
  aws
  terraform
  kubectl
  aliases
  ansible
  brew
  direnv
  opentofu
  gh
  sudo        # Esc Esc: sudo in front of the line (or of the last command)
  copybuffer  # Ctrl+O: the line being typed to the clipboard, not run
  copypath    # copypath [file]: the folder's (or file's) path to the clipboard
  extract     # extract FILE: any archive (.tar.gz, .zip, .7z, ...)
  macos       # ofd: this folder in Finder, cdf: cd to Finder's, quick-look FILE
  zsh-syntax-highlighting # last: it wraps the widgets defined before it
)

# Updates only noted at startup, never asked: the prompt would block a new pane
zstyle ':omz:update' mode reminder
# Suggest from completions too when the history has nothing; skip long lines (pastes)
ZSH_AUTOSUGGEST_STRATEGY=(history completion)
ZSH_AUTOSUGGEST_BUFFER_MAX_SIZE=20

# fzf-tab: Tab completes in an fzf list. Loaded when cloned into custom/plugins
# (the playbook), before the plugins that wrap widgets
if [[ -d $ZSH/custom/plugins/fzf-tab ]]; then
  plugins[(i)zsh-autosuggestions]=(fzf-tab zsh-autosuggestions)
  zstyle ':completion:*' menu no   # fzf-tab replaces the menu
  zstyle ':fzf-tab:complete:cd:*' fzf-preview 'ls -1 --color=always $realpath'
fi

source $ZSH/oh-my-zsh.sh
# Tab lists hidden files and folders too (completion only: * still skips them),
# without ./ and ../ (Oh My Zsh's special-dirs would add them to every list)
_comp_options+=(globdots)
zstyle ':completion:*' special-dirs false

# --- History ---
HISTFILE=~/.zsh_history
HISTSIZE=100000
SAVEHIST=100000
setopt SHARE_HISTORY          # every tmux pane sees the others' commands (appends too)
setopt EXTENDED_HISTORY       # start time and duration of each command: history -i
setopt HIST_IGNORE_ALL_DUPS   # a repeated command keeps only its latest entry
setopt HIST_IGNORE_SPACE
setopt HIST_REDUCE_BLANKS
setopt HIST_VERIFY            # !! and !$ expand onto the line, run on a second Enter

# --- Powerlevel10k ---
source ~/powerlevel10k/powerlevel10k.zsh-theme
[[ ! -f ~/.p10k.zsh ]] || source ~/.p10k.zsh

# --- Exports ---
# $TTY, not $(tty): under p10k instant prompt `tty` prints "not a tty"
export GPG_TTY=$TTY
export GOBIN=$HOME/go/bin
# git commit, kubectl edit, crontab -e, lazygit/k9s `e` open nvim
export EDITOR=nvim
export VISUAL=nvim

# Bitwarden SSH agent
export SSH_AUTH_SOCK="$HOME/.bitwarden-ssh-agent.sock"

export PYENV_ROOT="$HOME/.pyenv"
[[ -d $PYENV_ROOT/bin ]] && export PATH="$PYENV_ROOT/bin:$PATH"

# nvm is loaded on first use; the newest installed node is on PATH meanwhile
export NVM_DIR="$HOME/.nvm"
_nvm_bin=($NVM_DIR/versions/node/*/bin(Nn[-1]))
(( $#_nvm_bin )) && export PATH="$_nvm_bin:$PATH"
unset _nvm_bin
if [[ -s $NVM_DIR/nvm.sh ]]; then
  nvm() {
    unset -f nvm
    . "$NVM_DIR/nvm.sh"
    [[ -s $NVM_DIR/bash_completion ]] && . "$NVM_DIR/bash_completion"
    nvm "$@"
  }
fi

# --- PATH ---
export PATH=${PATH}:$GOBIN
# kubectl plugins from krew (kubectl neat, tree, view-secret, who-can; the playbook installs them),
# and their Tab completion (kubectl_complete-<plugin> scripts, .config/kubectl/complete)
[[ -d $HOME/.krew/bin ]] && path+=($HOME/.krew/bin $HOME/.config/kubectl/complete)

# --- Aliases ---
alias vim="nvim"
(( $+commands[bat] )) && alias cat="bat --pager=never"
alias lz="lazygit"
alias lzd="lazydocker"
alias tmux-sessions="~/.config/tmux/preload-sessions.sh" # after editing ~/.config/tmux/sessions.local
alias ktx="kontext"
alias z=cd # zoxide is cd now (below); z kept for the habit

# Kubernetes
alias k=kubecolor
compdef kubecolor=kubectl
setNS() { kubectl config set-context --current --namespace="$@" ; }

# --- Inits and completions ---
# Cached in ~/.cache/zsh, skipped when the tool is missing. Regenerated when the
# binary is newer than the cache, resolves to another path (a brew upgrade:
# pyenv's init sources a file under its versioned Cellar directory), or its
# command here changes
_cached_source() {
  (( $+commands[$1] )) || return
  local f="$HOME/.cache/zsh/$1.zsh" bin=${commands[$1]:A} first
  [[ -s $f ]] && read -r first < $f
  if [[ $first != "# $bin $2" || $bin -nt $f ]]; then
    mkdir -p ${f:h}
    { print -r -- "# $bin $2"; eval "$2" } >| $f
  fi
  source $f
}
_cached_source pyenv           'pyenv init - --no-rehash zsh'
_cached_source zoxide          'zoxide init zsh --cmd cd' # cd learns: `cd dotfiles` from anywhere, cdi picks
# fzf (Ctrl+T files, Ctrl+G folders, Ctrl+R history): fd lists them (dotfiles in,
# .gitignore respected), with a preview; ? shows a long history entry in full
if (( $+commands[fd] )); then
  export FZF_CTRL_T_COMMAND='fd --type f --hidden --exclude .git'
  export FZF_ALT_C_COMMAND='fd --type d --hidden --exclude .git'
fi
export FZF_CTRL_T_OPTS="--preview 'bat --color=always --style=numbers --line-range=:200 {}'"
export FZF_ALT_C_OPTS="--preview 'ls -1 --color=always {}'"
export FZF_CTRL_R_OPTS="--preview 'echo {2..}' --preview-window=up:3:hidden:wrap --bind '?:toggle-preview'"
_cached_source fzf             'fzf --zsh'
# Folders on Ctrl+G, not fzf's Alt+C: Option+letter belongs to AeroSpace
(( $+widgets[fzf-cd-widget] )) && bindkey '^G' fzf-cd-widget
_cached_source aws-cli-manager 'aws-cli-manager completion zsh'
_cached_source kontext         'kontext completion zsh'
_cached_source helm            'helm completion zsh'
_cached_source tenv            'tenv completion zsh'
_cached_source task            'task --completion zsh'
unfunction _cached_source

[ -f "$HOME/.local/bin/env" ] && . "$HOME/.local/bin/env"

# Shared functions (dotfiles .config/zsh)
for f in ~/.config/zsh/*.zsh(N); do source $f; done
unset f

# Per-machine extras (not in dotfiles): work-only tools, aliases; secrets: see README
[[ -f ~/.zshrc.local ]] && source ~/.zshrc.local

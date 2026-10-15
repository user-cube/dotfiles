# Enable Powerlevel10k instant prompt. Must stay at the top of ~/.zshrc.
if [[ -r "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh" ]]; then
  source "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh"
fi

# --- Oh My Zsh ---
export ZSH="$HOME/.oh-my-zsh"

plugins=(
  git
  zsh-autosuggestions
  zsh-syntax-highlighting
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
)

source $ZSH/oh-my-zsh.sh

# --- History ---
HISTFILE=~/.zsh_history
HISTSIZE=100000
SAVEHIST=100000
setopt APPEND_HISTORY
setopt SHARE_HISTORY          # every tmux pane sees the others' commands
setopt HIST_IGNORE_ALL_DUPS   # a repeated command keeps only its latest entry
setopt HIST_IGNORE_SPACE
setopt HIST_REDUCE_BLANKS

# --- Powerlevel10k ---
source ~/powerlevel10k/powerlevel10k.zsh-theme
[[ ! -f ~/.p10k.zsh ]] || source ~/.p10k.zsh

# --- Exports ---
export GPG_TTY=$(tty)
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

# --- Evals ---
eval "$(pyenv init - --no-rehash zsh)"
eval "$(zoxide init zsh)"
source <(fzf --zsh)

# --- Aliases ---
alias vim="nvim"
alias cat="bat --pager=never"
alias lz="lazygit"
alias lzd="lazydocker"
alias ktx="kontext"
alias redis-cli="docker run --network host --rm -it redis:alpine redis-cli"

# Kubernetes
alias k=kubecolor
compdef kubecolor=kubectl
setNS() { kubectl config set-context --current --namespace="$@" ; }

# --- Completions ---
# Cached in ~/.cache/zsh; regenerated when the binary is newer than the cache
_cached_completion() {
  (( $+commands[$1] )) || return
  local f="$HOME/.cache/zsh/$1.zsh"
  if [[ ! -s $f || ${commands[$1]} -nt $f ]]; then
    mkdir -p ${f:h}
    eval "$2" >| $f
  fi
  source $f
}
_cached_completion aws-cli-manager 'aws-cli-manager completion zsh'
_cached_completion kontext         'kontext completion zsh'
_cached_completion helm            'helm completion zsh'
_cached_completion tenv            'tenv completion zsh'
_cached_completion task            'task --completion zsh'
unfunction _cached_completion

[ -f "$HOME/.local/bin/env" ] && . "$HOME/.local/bin/env"

# Shared functions (dotfiles .config/zsh)
for f in ~/.config/zsh/*.zsh(N); do source $f; done
unset f

# Per-machine extras (not in dotfiles): work-only tools, aliases; secrets: see README
[[ -f ~/.zshrc.local ]] && source ~/.zshrc.local

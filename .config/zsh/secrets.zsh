# Keep tokens off the screen and out of ~/.zsh_history.
#
#   secret T                  asks for a value without echoing it; use it as "$T"
#                             in the next command, then it is cleared
#   secret T <bitwarden item> the same, from the item's password field
#
#   read the token, then use it:
#     secret T
#     claude mcp add ... --header "Authorization: Bearer $T"
#
# Also, a command line that looks like it holds a token (GitHub, GitLab, Slack,
# OpenAI / Anthropic keys, AWS access keys, JWTs, a long Bearer, --password=...)
# is not saved to the history. A leading space skips any command (HIST_IGNORE_SPACE).

autoload -Uz add-zsh-hook

typeset -ga _secret_vars
typeset -gi _secret_prompts=0

secret() {
  local name=$1 item=$2 value
  if [[ ! $name =~ '^[A-Za-z_][A-Za-z0-9_]*$' ]]; then
    print -u2 "usage: secret NAME [bitwarden-item]"
    return 1
  fi
  if [[ -n $item ]]; then
    value=$(bw get password "$item") || return
  else
    read -rs "value?$name: " || return
    print
  fi
  if [[ -z $value ]]; then
    print -u2 "secret: empty value, $name not set"
    return 1
  fi
  typeset -g "$name=$value" # not exported: only this shell sees it, not child processes
  _secret_vars+=($name)
  _secret_prompts=2 # cleared at the second prompt from now: after the next command
}

# Unset the secrets once the command after `secret` has run
_secret_clear() {
  (( _secret_prompts > 0 )) || return 0
  (( --_secret_prompts > 0 )) && return 0
  unset -- $_secret_vars
  _secret_vars=()
}
add-zsh-hook precmd _secret_clear

# Return 1 = the line isn't written to the history file (zshaddhistory)
_secret_history_filter() {
  local line=${1%%$'\n'}
  local re='(ghp_|gho_|ghs_|ghu_|github_pat_)[A-Za-z0-9_]{20,}'
  re+='|glpat-[A-Za-z0-9_-]{20,}'
  re+='|xox[abprs]-[A-Za-z0-9-]{10,}'
  re+='|sk-[A-Za-z0-9_-]{20,}'
  re+='|AKIA[0-9A-Z]{16}'
  re+='|eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.'
  re+='|[Bb]earer +[A-Za-z0-9._~+/=-]{20,}'
  re+='|-(password|passwd|token|secret|api-key)[= ]+[^ $"]{8,}'
  if [[ $line =~ $re ]]; then
    print -u2 "(not saved to history: the line looks like it holds a secret)"
    return 1
  fi
  return 0
}
add-zsh-hook zshaddhistory _secret_history_filter

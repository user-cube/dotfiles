# Bitwarden CLI wrapper: keeps the session key in a root-only file so `bw`
# works without re-entering the master password (unlocked with sudo/Touch ID).
# Set BW_USER (account email) in ~/.zshrc.local.
# local-files (.config/dotfiles/local-files/bw.go) reads the same file.
typeset -g _bw_session_file=/var/root/.bitwarden.session
typeset -g _bw_session # read once per shell, then reused

bw() {
  local bw_exec
  bw_exec=$(whence -p bw) || { print -u2 "bw: Bitwarden CLI not found"; return 127 }

  case $1 in
  --regenerate-session-key)
    [[ -n $BW_USER ]] || { print -u2 "BW_USER is not set; add it to ~/.zshrc.local"; return 1 }
    echo "Regenerating session key, this has invalidated all existing sessions..."
    sudo rm -f $_bw_session_file
    $bw_exec logout &>/dev/null # invalidate all existing sessions
    _bw_session=$($bw_exec login "$BW_USER" --raw) && [[ -n $_bw_session ]] || { sudo -k; return 1 }
    print -r -- $_bw_session | sudo tee $_bw_session_file >/dev/null
    sudo -k # de-elevate
    ;;
  login | logout | config)
    $bw_exec "$@"
    ;;
  --help | -h | '')
    $bw_exec "$@"
    echo "To regenerate your session key type:"
    echo "  bw --regenerate-session-key"
    ;;
  *)
    if [[ -z $_bw_session ]]; then
      _bw_session=$(sudo cat $_bw_session_file 2>/dev/null)
      sudo -k # de-elevate
      [[ -n $_bw_session ]] || { print -u2 "Token not found, please run bw --regenerate-session-key"; return 1 }
    fi
    $bw_exec "$@" --session "$_bw_session"
    ;;
  esac
}

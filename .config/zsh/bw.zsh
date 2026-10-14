# Bitwarden CLI wrapper: keeps the session key in a root-only file so `bw`
# works without re-entering the master password (unlocked with sudo/Touch ID).
# Set BW_USER (account email) in ~/.zshrc.local.
bw() {
  bw_exec=$(sh -c "which bw")
  local -r bw_session_file='/var/root/.bitwarden.session' # Only accessible as root

  _read_token_from_file() {

    local -r err_token_not_found="Token not found, please run bw --regenerate-session-key"
    case $1 in
    '--force')
      unset bw_session
      ;;
    esac

    if [ "$bw_session" = "$err_token_not_found" ]; then
      unset bw_session
    fi

    # If the session key env variable is not set, read it from the file
    # if file it not there, ask user to regenerate it

    if [ -z "$bw_session" ]; then
      bw_session="$(
        sh -c "sudo cat $bw_session_file 2> /dev/null"
        # shellcheck disable=SC2181
        if [ "$?" -ne "0" ]; then
          echo "$err_token_not_found"
          sudo -k # De-elevate privileges
          exit 1
        fi
        sudo -k # De-elevate privileges
      )"

      # shellcheck disable=SC2181
      if [ "$bw_session" = "$err_token_not_found" ]; then
        echo "$err_token_not_found"
        return 1
      fi
    fi
  }

  case $1 in
  '--regenerate-session-key')
    if [ -z "$BW_USER" ]; then
      echo "BW_USER is not set; add it to ~/.zshrc.local"
      return 1
    fi
    echo "Regenerating session key, this has invalidated all existing sessions..."
    sudo rm -f /var/root/.bitwarden.session && ${bw_exec} logout 2>/dev/null # Invalidate all existing sessions

    ${bw_exec} login "${BW_USER}" --raw | sudo tee /var/root/.bitwarden.session &>/dev/null # Generate new session key

    _read_token_from_file --force # Read the new session key for immediate use
    sudo -k                       # De-elevate privileges, only doing this now so _read_token_from_file can resuse the same sudo session
    ;;

  'login' | 'logout' | 'config')
    ${bw_exec} "$@"
    ;;


  '--help' | '-h' | '')
    ${bw_exec} "$@"
    echo "To regenerate your session key type:"
    echo "  bw --regenerate-session-key"
    ;;

  *)
    _read_token_from_file

    ${bw_exec} "$@" --session "$bw_session"
    ;;
  esac
}

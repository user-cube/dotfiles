# sops-on: the age keys SOPS needs, from Bitwarden, in a file for a limited time.
#
#   sops-on [DURATION]   fetch the keys and write them where sops (and the VS Code
#                        SOPS extension) look; removed after DURATION and a notice
#                        at the next prompt + a macOS notification.
#                        90 or 90m = minutes, 30s, 2h. Default: $SOPS_ON_MINUTES, else 60.
#                        Running it again refreshes the keys and restarts the timer.
#   sops-off             remove the keys now
#   sops-status          is it on, and for how long
#   sops-save [FILE]     put a keys file (default: the sops one) in the Bitwarden note
#
# The keys live in one secure note, "SOPS age keys" (SOPS_BW_ITEM overrides), whose
# notes field is a plain age keys file: one "AGE-SECRET-KEY-..." per line.
# File: sops's default, ~/Library/Application Support/sops/age/keys.txt
# ($XDG_CONFIG_HOME/sops/age/keys.txt when set); SOPS_KEYS_FILE overrides it
# (then sops needs SOPS_AGE_KEY_FILE pointing at the same file).
#
# A keys file that sops-on did not create is never overwritten or removed.

zmodload zsh/datetime
autoload -Uz add-zsh-hook

_sops_keys_file() { print -r -- "${SOPS_KEYS_FILE:-${XDG_CONFIG_HOME:-$HOME/Library/Application Support}/sops/age/keys.txt}" }
_sops_state_dir() { print -r -- "${XDG_CACHE_HOME:-$HOME/.cache}/sops-on" }
_sops_item() { print -r -- "${SOPS_BW_ITEM:-SOPS age keys}" }

# 90 | 90m | 30s | 2h -> seconds
_sops_seconds() {
  local d=${1:-${SOPS_ON_MINUTES:-60}}
  [[ $d =~ '^([0-9]+)([smh]?)$' ]] || return 1
  local n=$match[1] u=$match[2]
  (( n > 0 )) || return 1
  case $u in
    s) print $n ;;
    h) print $(( n * 3600 )) ;;
    *) print $(( n * 60 )) ;;
  esac
}

# Remove the keys and the state; $1 = reason, shown at the next prompt
_sops_expire() {
  local state=$(_sops_state_dir)
  rm -f -- "$(_sops_keys_file)" "$state/expiry" "$state/pid"
  [[ -n $1 ]] && print -r -- "$1" >| "$state/notice"
}

# Runs detached: waits for the deadline (re-read each round, sops-on can extend it)
_sops_reaper() {
  local state=$1 file=$2 left
  while [[ -f $state/expiry && -f $file ]]; do
    left=$(( $(<"$state/expiry") - EPOCHSECONDS ))
    (( left <= 0 )) && break
    sleep $(( left < 15 ? left : 15 ))
  done
  # Gone already (sops-off): nothing to announce
  [[ -f $state/expiry ]] || return 0
  _sops_expire "sops-on: keys removed (time is up)"
  osascript -e 'display notification "Age keys removed" with title "sops-on"' &>/dev/null
}

sops-on() {
  local secs file state item keys
  secs=$(_sops_seconds "$1") || { print -u2 "usage: sops-on [90|90m|30s|2h]"; return 1 }
  file=$(_sops_keys_file) state=$(_sops_state_dir) item=$(_sops_item)
  mkdir -p "$state" && chmod 700 "$state"

  if [[ -e $file && ! -f $state/expiry ]]; then
    print -u2 "sops-on: $file already exists and was not created by sops-on; not touching it."
    print -u2 "         Put it in Bitwarden with sops-save, then remove it."
    return 1
  fi

  bw sync &>/dev/null
  keys=$(bw get notes "$item") || { print -u2 "sops-on: could not read the Bitwarden note '$item'"; return 1 }
  if [[ $keys != *AGE-SECRET-KEY-* ]]; then
    print -u2 "sops-on: note '$item' holds no AGE-SECRET-KEY line"
    return 1
  fi

  mkdir -p -- "${file:h}"
  ( umask 077; print -r -- "$keys" >| "$file" ) || return
  print -r -- $(( EPOCHSECONDS + secs )) >| "$state/expiry"
  rm -f -- "$state/notice"

  local pid
  [[ -f $state/pid ]] && pid=$(<"$state/pid")
  if [[ -z $pid ]] || ! kill -0 "$pid" 2>/dev/null; then
    _sops_reaper "$state" "$file" &>/dev/null &!
    print -r -- $! >| "$state/pid"
  fi
  print "sops-on: keys ready until $(strftime '%H:%M' $(( EPOCHSECONDS + secs )))"
}

sops-off() {
  local state=$(_sops_state_dir) pid
  if [[ ! -f $state/expiry ]]; then
    print "sops-off: not on (a keys file sops-on did not create is left alone)"
    return 0
  fi
  [[ -f $state/pid ]] && pid=$(<"$state/pid") && kill "$pid" 2>/dev/null
  _sops_expire
  print "sops-off: keys removed"
}

sops-status() {
  local state=$(_sops_state_dir) left
  if [[ -f $state/expiry ]] && left=$(( $(<"$state/expiry") - EPOCHSECONDS )) && (( left > 0 )); then
    print "sops-on: keys present, $(( (left + 59) / 60 )) min left"
  else
    print "sops-on: off"
  fi
}

sops-save() {
  local state=$(_sops_state_dir) src=${1:-$(_sops_keys_file)} item=$(_sops_item) id
  if [[ -z $1 && -f $state/expiry ]]; then
    print -u2 "sops-save: the keys file comes from Bitwarden (sops-on); give another FILE"
    return 1
  fi
  [[ -f $src ]] || { print -u2 "sops-save: no file $src"; return 1 }
  grep -q AGE-SECRET-KEY- -- "$src" || { print -u2 "sops-save: $src holds no AGE-SECRET-KEY line"; return 1 }

  bw sync &>/dev/null
  id=$(bw get item "$item" 2>/dev/null | jq -r '.id // empty')
  if [[ -n $id ]]; then
    # --rawfile: the keys never appear in a command line
    bw get item "$id" | jq --rawfile notes "$src" '.notes = $notes' | bw encode | bw edit item "$id" >/dev/null || return
    print "sops-save: updated '$item'"
  else
    bw get template item | jq --arg name "$item" --rawfile notes "$src" \
      '.type = 2 | .secureNote = {type: 0} | .name = $name | .notes = $notes' | bw encode | bw create item >/dev/null || return
    print "sops-save: created '$item'"
  fi
}

# At every prompt: announce an expiry once, and clean up when the timer died
# with the machine (reboot) and the deadline has passed
_sops_precmd() {
  local state=$(_sops_state_dir)
  [[ -e $state/expiry || -e $state/notice ]] || return 0
  if [[ -f $state/expiry ]] && (( $(<"$state/expiry") <= EPOCHSECONDS )); then
    _sops_expire "sops-on: keys removed (time is up)"
  fi
  if [[ -f $state/notice ]]; then
    print -r -- "$(<"$state/notice")"
    rm -f -- "$state/notice"
  fi
}
add-zsh-hook precmd _sops_precmd

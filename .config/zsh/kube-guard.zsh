# kube-guard: asks before a kubectl / helm command that changes a production
# cluster, and k9s and sofka open read-only there (--write to change things;
# a context switched to inside them keeps the mode they started with). The
# decision is the Go binary's (.config/kube-guard: the line's subcommand and
# context, what production is, the question); these are its wrappers.
# Production on this machine: ~/.config/kube-guard/prod-regex (one line, e.g.
# prod|prd|live); KUBE_PROD_REGEX overrides it in a shell.
#
#   KUBE_GUARD=0 kubectl apply -f x.yaml   skip it for one command
#
# Covers kubectl, kubecolor (k) and helm, and so the Oh My Zsh aliases (kaf,
# kdel...); scripts don't go through it. Until the binary is built
# (dotfiles-sync), every kubectl / helm command asks and the TUIs open read-only.

: ${KUBE_GUARD_BIN:=$HOME/.config/kube-guard/kube-guard}

_kube_guard() { # tool args...
  if [[ -x $KUBE_GUARD_BIN ]]; then
    # not exported when set in ~/.zshrc.local or before the function
    KUBE_GUARD=${KUBE_GUARD-} KUBE_PROD_REGEX=${KUBE_PROD_REGEX-} $KUBE_GUARD_BIN check "$@"
    return
  fi
  [[ ${KUBE_GUARD:-1} == 0 ]] && return 0
  # From the terminal: stdin may be the manifest (kubectl apply -f -)
  if read -q "?kube-guard is not built (dotfiles-sync). $* ? [y/N] " </dev/tty; then
    print
    return 0
  fi
  print "\nkube-guard: not run"
  return 1
}

kubectl() { _kube_guard kubectl "$@" && command kubectl "$@"; }
kubecolor() { _kube_guard kubectl "$@" && command kubecolor "$@"; }
helm() { _kube_guard helm "$@" && command helm "$@"; }

# The TUIs open read-only on a production context: the binary says so (exit 0)
_kube_guard_tui() { # tool args...
  local tool=$1 rc
  shift
  if [[ -x $KUBE_GUARD_BIN ]]; then
    KUBE_GUARD=${KUBE_GUARD-} KUBE_PROD_REGEX=${KUBE_PROD_REGEX-} $KUBE_GUARD_BIN tui $tool "$@"
    rc=$?
  elif [[ ${KUBE_GUARD:-1} != 0 && " $* " != *" --write "* && " $* " != *" --readonly "* ]]; then
    print -u2 "kube-guard is not built (dotfiles-sync): $tool read-only ($tool --write to change things)"
    rc=0
  else
    rc=1
  fi
  if (( rc != 1 )); then # 2: the binary failed, read-only too
    command $tool --readonly "$@"
  else
    command $tool "$@"
  fi
}

# k9s: config from the dotfiles (.config/k9s)
export K9S_CONFIG_DIR=${K9S_CONFIG_DIR:-$HOME/.config/k9s}
k9s() { _kube_guard_tui k9s "$@"; }

# ⌘K follows the shell it was pressed in: a `ktx -s` shell has a KUBECONFIG of
# its own, which a popup (started by the tmux server or Rex) never sees. Each
# interactive shell writes its KUBECONFIG to a file per pane (tmux's
# TMUX_PANE, Rex's REX_BLOCK) when it changes, and puts back the one of the
# shell it was started from when it exits (the pane's first shell removes the
# file: Rex's block IDs are never reused); _kube_popup reads it.
: ${KUBE_PANE_DIR:=${XDG_CACHE_HOME:-$HOME/.cache}/kube-pane}
if [[ -o interactive && -n $TMUX_PANE$REX_BLOCK ]]; then
  _kube_pane_files=(${TMUX_PANE:+$KUBE_PANE_DIR/tmux-$TMUX_PANE} ${REX_BLOCK:+$KUBE_PANE_DIR/rex-$REX_BLOCK})
  mkdir -p $KUBE_PANE_DIR
  _kube_pane_prev=$(cat $_kube_pane_files[1] 2>/dev/null)
  _kube_pane_kc=$'\0' # nothing written yet
  _kube_pane_write() { local f; for f in $_kube_pane_files; do print -r -- $1 >| $f; done }
  _kube_pane_publish() {
    [[ ${KUBECONFIG-} == $_kube_pane_kc ]] && return
    _kube_pane_kc=${KUBECONFIG-}
    _kube_pane_write $_kube_pane_kc
  }
  _kube_pane_restore() {
    if [[ -n $_kube_pane_prev ]]; then _kube_pane_write $_kube_pane_prev; else rm -f $_kube_pane_files; fi
  }
  autoload -Uz add-zsh-hook
  add-zsh-hook precmd _kube_pane_publish
  add-zsh-hook zshexit _kube_pane_restore
fi

_kube_popup() { # tool [pane key]: the TUI with the KUBECONFIG of the pane it was opened from
  local key=$2 kc
  [[ -z $key && -n $TMUX ]] && key=tmux-$(command tmux display -p '#{pane_id}')
  [[ -n $key ]] && kc=$(cat $KUBE_PANE_DIR/$key 2>/dev/null)
  # a file left behind by a shell that didn't exit cleanly can name a deleted one
  [[ -n $kc && -f ${kc%%:*} ]] && export KUBECONFIG=$kc
  $1
}

# sofka (the ⌘K popup of Rex and tmux sources this file): its subcommands
# (info, import, plugin...) don't open the TUI. Its guardrails (.config/sofka)
# also block deletes on a context switched to inside
sofka() {
  case $1 in
    completion | info | plugin | import | check-update | help) command sofka "$@" ;;
    *) _kube_guard_tui sofka "$@" ;;
  esac
}

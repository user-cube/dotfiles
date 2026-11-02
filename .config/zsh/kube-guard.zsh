# kube-guard: asks before a kubectl / helm command that changes a production
# cluster. Production: the context (--context / --kube-context, else the
# current one) matches KUBE_PROD_REGEX, default prod|prd like the SketchyBar
# kube item. Reads (get, describe, logs, diff, rollout status, helm template...)
# pass without asking.
#
#   KUBE_GUARD=0 kubectl apply -f x.yaml   skip it for one command
#
# Covers kubectl, kubecolor (k) and helm, and so the Oh My Zsh aliases (kaf,
# kdel...); scripts don't go through it. k9s and sofka on a production context
# open read-only (--write to change things); a context switched to inside them
# keeps the mode they started with.

: ${KUBE_PROD_REGEX:='prod|prd'}

# Every subcommand of kubectl and helm (and helm diff, a read): the subcommand is
# the first word that is one, so the value of a flag before it (--as admin,
# --request-timeout 5s, -v 6) isn't taken for it
typeset -gA _kube_guard_cmds=(
  kubectl ' alpha annotate api-resources api-versions apply attach auth autoscale certificate cluster-info completion config cordon cp create debug delete describe diff drain edit events exec explain expose get kustomize label logs options patch plugin port-forward proxy replace rollout run scale set taint top uncordon version wait '
  helm ' completion create del delete dependency diff env get help history install lint list ls package plugin pull push registry repo rollback search show status template test un uninstall upgrade verify version '
)

# tool args...: sets REPLY to the subcommand (and its sub-subcommand for
# rollout), _kube_ctx to the --context and _kube_kc to the --kubeconfig given
# anywhere on the line, skipping flag values. A kubectl plugin (kubectl-<name>
# on the PATH, e.g. who-can) is a subcommand too: its arguments (who-can delete
# pods) are not kubectl's
_kube_guard_parse() {
  local tool=$1 arg skip=
  shift
  REPLY= _kube_ctx= _kube_kc=
  for arg in "$@"; do
    if [[ -n $skip ]]; then
      case $skip in
        ctx) _kube_ctx=$arg ;;
        kc) _kube_kc=$arg ;;
      esac
      skip=
      continue
    fi
    case $arg in
      --context=* | --kube-context=*) _kube_ctx=${arg#*=} ;;
      --context | --kube-context) skip=ctx ;;
      --kubeconfig=*) _kube_kc=${arg#*=} ;;
      --kubeconfig) skip=kc ;;
      -n | --namespace | --cluster | --user | -s | --server | -f | --filename | -l | --selector | -o | --output | -c | --container) skip=value ;;
      -*) ;;
      *)
        if [[ -z $REPLY ]]; then
          if [[ $_kube_guard_cmds[$tool] == *" $arg "* ]] ||
            [[ $tool == kubectl && -n $arg && $+commands[kubectl-${arg//-/_}] == 1 ]]; then
            REPLY=$arg
          fi
        elif [[ $REPLY == rollout ]]; then
          REPLY+=" $arg"
        fi
        ;;
    esac
  done
}

# The context the parsed line runs on: its --context, else the current one of
# its --kubeconfig (else of KUBECONFIG / ~/.kube/config)
_kube_guard_ctx() {
  REPLY=${_kube_ctx:-$(command kubectl ${_kube_kc:+--kubeconfig=$_kube_kc} config current-context 2>/dev/null)}
}

_kube_guard() { # tool args...
  local tool=$1 ctx
  shift
  [[ ${KUBE_GUARD:-1} == 0 ]] && return 0
  _kube_guard_parse $tool "$@"
  case $tool:$REPLY in
    kubectl:(apply|create|delete|edit|patch|replace|scale|set|label|annotate|taint|drain|cordon|uncordon|autoscale|expose|run|exec|debug|cp)) ;;
    kubectl:"rollout "(restart|undo|pause|resume)) ;;
    helm:(install|upgrade|uninstall|delete|rollback)) ;;
    *) return 0 ;;
  esac
  _kube_guard_ctx
  ctx=$REPLY
  [[ ${(L)ctx} =~ ($KUBE_PROD_REGEX) ]] || return 0 # any case, like grep -i
  # From the terminal: stdin may be the manifest (kubectl apply -f -)
  if read -q "?kube-guard: $ctx is production. $tool $* ? [y/N] " </dev/tty; then
    print
    return 0
  fi
  print "\nkube-guard: not run"
  return 1
}

kubectl() { _kube_guard kubectl "$@" && command kubectl "$@"; }
kubecolor() { _kube_guard kubectl "$@" && command kubecolor "$@"; }
helm() { _kube_guard helm "$@" && command helm "$@"; }

# The TUIs open read-only on a production context; --write overrides it
_kube_guard_tui() { # tool args...
  local tool=$1 ctx
  shift
  _kube_guard_parse $tool "$@"
  _kube_guard_ctx
  ctx=$REPLY
  if [[ ${KUBE_GUARD:-1} != 0 && ${(L)ctx} =~ ($KUBE_PROD_REGEX) && " $* " != *" --write "* && " $* " != *" --readonly "* ]]; then
    print "kube-guard: $ctx is production, $tool read-only ($tool --write to change things)"
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

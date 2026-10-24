# kube-guard: asks before a kubectl / helm command that changes a production
# cluster. Production: the context (--context / --kube-context, else the
# current one) matches KUBE_PROD_REGEX, default prod|prd like the SketchyBar
# kube item. Reads (get, describe, logs, diff, rollout status, helm template...)
# pass without asking.
#
#   KUBE_GUARD=0 kubectl apply -f x.yaml   skip it for one command
#
# Covers kubectl, kubecolor (k) and helm, and so the Oh My Zsh aliases (kaf,
# kdel...); scripts don't go through it. k9s on a production context opens
# read-only (k9s --write to change things); a context switched to inside k9s
# keeps the mode it started with.

: ${KUBE_PROD_REGEX:='prod|prd'}

# Sets REPLY to the subcommand (and its sub-subcommand for rollout) and
# _kube_ctx to the --context given anywhere on the line, skipping flag values
_kube_guard_parse() {
  local arg skip=
  REPLY= _kube_ctx=
  for arg in "$@"; do
    if [[ -n $skip ]]; then
      [[ $skip == ctx ]] && _kube_ctx=$arg
      skip=
      continue
    fi
    case $arg in
      --context=* | --kube-context=*) _kube_ctx=${arg#*=} ;;
      --context | --kube-context) skip=ctx ;;
      -n | --namespace | --kubeconfig | --cluster | --user | -s | --server | -f | --filename | -l | --selector | -o | --output | -c | --container) skip=value ;;
      -*) ;;
      *)
        if [[ -z $REPLY ]]; then
          REPLY=$arg
        elif [[ $REPLY == rollout ]]; then
          REPLY+=" $arg"
        fi
        ;;
    esac
  done
}

_kube_guard() { # tool args...
  local tool=$1 ctx
  shift
  [[ ${KUBE_GUARD:-1} == 0 ]] && return 0
  _kube_guard_parse "$@"
  case $tool:$REPLY in
    kubectl:(apply|create|delete|edit|patch|replace|scale|set|label|annotate|taint|drain|cordon|uncordon|autoscale|expose|run|exec|debug|cp)) ;;
    kubectl:"rollout "(restart|undo|pause|resume)) ;;
    helm:(install|upgrade|uninstall|delete|rollback)) ;;
    *) return 0 ;;
  esac
  ctx=${_kube_ctx:-$(command kubectl config current-context 2>/dev/null)}
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

# k9s: config from the dotfiles (.config/k9s; the tmux ⌘K popup sources this file too)
export K9S_CONFIG_DIR=${K9S_CONFIG_DIR:-$HOME/.config/k9s}
k9s() {
  local ctx
  _kube_guard_parse "$@"
  ctx=${_kube_ctx:-$(command kubectl config current-context 2>/dev/null)}
  if [[ ${KUBE_GUARD:-1} != 0 && ${(L)ctx} =~ ($KUBE_PROD_REGEX) && " $* " != *" --write "* && " $* " != *" --readonly "* ]]; then
    print "kube-guard: $ctx is production, k9s read-only (k9s --write to change things)"
    command k9s --readonly "$@"
  else
    command k9s "$@"
  fi
}

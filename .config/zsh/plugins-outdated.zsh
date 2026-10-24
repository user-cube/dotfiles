# plugins-outdated: the zsh and tmux plugins the playbook pins to a commit
# (devops-daily-driver, roles/terminal/defaults/main.yml) that master has moved
# past. Updating them is the playbook's: change version there (or merge
# Renovate's PR), then run it. DAILY_DRIVER_DIR overrides where the playbook is
# cloned. Every 14 days the first shell checks in the background and sends a
# macOS notification when some are behind.

# "name repo commit" for each pinned plugin
_plugins_pins() {
  local f=${DAILY_DRIVER_DIR:-$HOME/work/github/devops-daily-driver}/roles/terminal/defaults/main.yml
  [[ -r $f ]] || return 1
  awk '/^ +- name:/ { n = $3 } /^ +repo:/ { r = $2 } /^ +version:/ { print n, r, $2 }' $f
}

plugins-outdated() {
  local pins name repo pin head behind=0
  pins=$(_plugins_pins) || {
    echo "plugins-outdated: no playbook in ${DAILY_DRIVER_DIR:-$HOME/work/github/devops-daily-driver} (set DAILY_DRIVER_DIR)"
    return 2
  }
  while read -r name repo pin; do
    head=$(git ls-remote $repo refs/heads/master 2>/dev/null)
    head=${head%%[[:space:]]*}
    if [[ -z $head ]]; then
      printf '%-24s ?  %s did not answer\n' $name $repo
    elif [[ $head != $pin ]]; then
      printf '%-24s %s -> %s\n' $name ${pin[1,7]} ${head[1,7]}
      (( behind++ ))
    fi
  done <<<$pins
  if (( behind )); then
    echo "Update: version in roles/terminal/defaults/main.yml (or Renovate's PR), then the playbook"
    return 1
  fi
  echo "All $(wc -l <<<$pins | tr -d ' ') plugins are at master."
}

# The 14-day check, in the background: never slows the prompt
() {
  local stamp=$HOME/.cache/zsh/plugins-outdated.stamp
  local recent=($stamp(N.mh-336)) # touched in the last 336 h
  (( $#recent )) && return
  _plugins_pins >/dev/null || return # no playbook on this machine
  mkdir -p ${stamp:h} && : >| $stamp # first, so other shells skip it
  {
    local n=$(plugins-outdated 2>/dev/null | grep -c -- ' -> ')
    (( n )) && osascript -e "display notification \"$n behind master: run plugins-outdated\" with title \"zsh/tmux plugins\"" &>/dev/null
  } &!
}

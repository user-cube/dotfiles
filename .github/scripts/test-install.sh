#!/bin/bash
# End-to-end test of the install path: dotfiles-sync (and so install.sh) into
# an empty fake HOME, from a clone of this checkout with its uncommitted
# changes. Run by CI (lint.yml); locally: .github/scripts/test-install.sh
#
# Checks that a first run installs everything, a second run changes nothing,
# a changed .go file rebuilds only its tool, a broken build fails, and that
# the binaries it built work.
set -uo pipefail

repo=$(cd "$(dirname "$0")/../.." && pwd)
t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT
H=$t/home
mkdir -p "$H" "$t/bin"
failed=0
fail() { echo "FAIL: $*"; failed=1; }
pass() { echo "ok:   $*"; }

# The "repo" dotfiles-sync pulls: a clone of this checkout, with the working tree on top
git clone -q "$repo" "$t/dotfiles"
rsync -a --exclude .git "$repo/" "$t/dotfiles/"

# A fake VS Code CLI (Hue's install.sh): lists rui.hue once installed
cat >"$t/bin/code" <<EOF
#!/bin/sh
case "\$1" in
  --list-extensions) [ -f "$t/hue-installed" ] && echo rui.hue; exit 0 ;;
  --install-extension) touch "$t/hue-installed" ;;
esac
EOF
chmod +x "$t/bin/code"

# An obsolete file, as an old install would have left it
mkdir -p "$H/.config/cmux" && echo old >"$H/.config/cmux/settings.json"

sync() { # stdout of dotfiles-sync's own steps, without its reload lines
  HOME=$H PATH="$t/bin:$PATH" DOTFILES_DIR=$t/dotfiles GOCACHE=$(go env GOCACHE) \
    zsh -f -c 'source "$DOTFILES_DIR/.config/zsh/dotfiles.zsh" && dotfiles-sync' 2>"$t/stderr" |
    grep -vE '^(Already up to date\.|Done\. |tmux: |AeroSpace: |SketchyBar: |borders: )'
  return "${PIPESTATUS[0]}"
}

echo "== first run: an empty HOME"
if out=$(sync); then pass "dotfiles-sync exits 0"; else fail "dotfiles-sync failed:"; cat "$t/stderr"; fi
echo "$out" | sed 's/^/      /'
tools="git_hooks/go/commit-msg/commit-msg .config/dotfiles/local-files/local-files .config/sketchybar/plugins/check/check .claude/hooks/deny-sensitive-reads/deny-sensitive-reads"
for b in $tools; do
  if [ -x "$H/$b" ]; then pass "built ~/$b"; else fail "not built: ~/$b"; fi
done
for f in .zshrc .p10k.zsh .gitconfig .gitconfig-personal .gitconfig-work .claude/settings.json; do
  [ -f "$H/$f" ] && pass "copied ~/$f" || fail "not copied: ~/$f"
done
grep -q '__icon_map()' "$H/.config/sketchybar/plugins/icon_map.sh" 2>/dev/null && pass "icon map downloaded" || fail "no icon map"
[ -f "$t/hue-installed" ] && pass "Hue installed" || fail "Hue not installed"
[ ! -e "$H/.config/cmux/settings.json" ] && pass "obsolete file removed" || fail "obsolete file still there"

echo "== second run: nothing to do"
out=$(sync); rc=$?
[ $rc -eq 0 ] && [ -z "$out" ] && pass "no changes" || fail "rc=$rc, output: $out"

echo "== a .go file changes: only its tool is rebuilt"
touch "$t/dotfiles/.config/sketchybar/plugins/check/main.go"
out=$(sync)
[ "$out" = "built check" ] && pass "$out" || fail "expected 'built check', got: $out"

echo "== the binaries work"
ev='{"tool_name":"Bash","cwd":"/tmp","tool_input":{"command":"cat .env"}}'
echo "$ev" | "$H/.claude/hooks/deny-sensitive-reads/deny-sensitive-reads" | grep -q '"deny"' &&
  pass "hook denies a private path" || fail "hook did not deny"
echo "${ev/.env/README.md}" | "$H/.claude/hooks/deny-sensitive-reads/deny-sensitive-reads" | grep -q . &&
  fail "hook denied a public path" || pass "hook allows a public path"
printf '#!/bin/sh\necho "$@" >>"%s/bar"\n' "$t" >"$t/bin/sketchybar" && chmod +x "$t/bin/sketchybar"
PATH="$t/bin:$PATH" NAME=checks SENDER=mouse.clicked CONFIG_DIR=$H/.config/sketchybar "$H/.config/sketchybar/plugins/check/check" &&
  grep -q 'popup.drawing=toggle' "$t/bar" && pass "check toggles its popup" || fail "check did not run"
usage=$("$H/.config/dotfiles/local-files/local-files" 2>&1) # exits 1 without a command
[[ $usage == usage:* ]] && pass "local-files prints its usage" || fail "local-files did not run: $usage"

echo "== a broken build fails the sync"
echo 'not go' >>"$t/dotfiles/.config/dotfiles/local-files/main.go"
if sync >/dev/null; then fail "dotfiles-sync passed with a broken build"; else pass "dotfiles-sync fails"; fi

[ $failed -eq 0 ] && echo "PASS" || echo "FAILED"
exit $failed

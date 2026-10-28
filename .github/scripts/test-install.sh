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

# The "repo" dotfiles-sync pulls: a clone of this checkout with the working tree
# on top, committed, and its own bare origin (dotfiles-diff compares with it)
# git without the user's config: no global hooks (commit-msg blocks main), no signing
g() { HOME=$t GIT_CONFIG_NOSYSTEM=1 git -c user.name=test -c user.email=test@example.com "$@"; }
g clone -q "$repo" "$t/dotfiles"
rsync -a --exclude .git "$repo/" "$t/dotfiles/"
g -C "$t/dotfiles" add -A
g -C "$t/dotfiles" commit -q --allow-empty -m "working tree" || fail "could not commit the working tree"
g clone -q --bare "$t/dotfiles" "$t/origin.git"
g -C "$t/dotfiles" remote set-url origin "$t/origin.git"
g -C "$t/dotfiles" fetch -q
g -C "$t/dotfiles" branch -q -u "origin/$(g -C "$t/dotfiles" branch --show-current)"

# A fake VS Code CLI: lists rui.hue once Hue's install.sh installed its .vsix,
# and the marketplace ids installed since (install-extensions.sh)
cat >"$t/bin/code" <<EOF
#!/bin/sh
case "\$1" in
  --list-extensions) [ -f "$t/hue-installed" ] && echo rui.hue; cat "$t/extensions" 2>/dev/null; exit 0 ;;
  --install-extension) case "\$2" in *.vsix) touch "$t/hue-installed" ;; *) echo "\$2" >>"$t/extensions" ;; esac ;;
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
want=$(grep -vE '^[[:space:]]*(#|$)' "$repo/.config/vscode/extensions.txt" | sort)
[ "$(sort "$t/extensions" 2>/dev/null)" = "$want" ] && pass "VS Code extensions installed" || fail "VS Code extensions: got $(cat "$t/extensions" 2>/dev/null)"
for f in settings.json keybindings.json; do
  vs="$H/Library/Application Support/Code/User/$f"
  [ "$(readlink "$vs")" = "$H/.config/vscode/$f" ] && pass "VS Code $f linked" || fail "VS Code $f not linked"
done
lg="$H/Library/Application Support/lazygit/config.yml"
[ "$(readlink "$lg")" = "$H/.config/lazygit/config.yml" ] && pass "lazygit config linked" || fail "lazygit config not linked"

echo "== second run: nothing to do"
out=$(sync); rc=$?
[ $rc -eq 0 ] && [ -z "$out" ] && pass "no changes" || fail "rc=$rc, output: $out"

echo "== the clone has a branch or uncommitted changes: said on stderr"
grep -qE 'installing branch|uncommitted' "$t/stderr" && fail "warned on a clean main: $(cat "$t/stderr")" || pass "no warning on a clean clone"
echo scratch >"$t/dotfiles/scratch.txt"
sync >/dev/null
grep -q 'has uncommitted changes' "$t/stderr" && pass "warns about uncommitted changes" || fail "no warning: $(cat "$t/stderr")"
rm "$t/dotfiles/scratch.txt"
g -C "$t/dotfiles" switch -q -c topic --track "$(g -C "$t/dotfiles" rev-parse --abbrev-ref '@{u}')" # pull needs an upstream
sync >/dev/null
grep -q "installing branch 'topic'" "$t/stderr" && pass "warns about a branch" || fail "no branch warning: $(cat "$t/stderr")"
g -C "$t/dotfiles" switch -q -
g -C "$t/dotfiles" branch -q -D topic

echo "== Hue changed without a version bump: said on stderr"
hue_version=$(jq -r .version "$repo/.config/vscode/extensions/hue/package.json")
echo "an-older-sha $hue_version" >"$H/.cache/hue.sha"
out=$(sync)
[[ $out == "hue: installed $hue_version"* ]] && pass "reinstalled" || fail "not reinstalled: $out"
grep -q "hue: changed but still $hue_version" "$t/stderr" && pass "asks for a version bump" || fail "no bump warning: $(cat "$t/stderr")"
echo "an-older-sha" >"$H/.cache/hue.sha" # a stamp from before it held the version
sync >/dev/null
grep -q 'bump' "$t/stderr" && fail "warned with an old stamp" || pass "no warning with an old stamp"

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

echo "== dotfiles-diff: what a sync would change, changing nothing"
diff_() {
  HOME=$H PATH="$t/bin:$PATH" DOTFILES_DIR=$t/dotfiles \
    zsh -f -c 'source "$DOTFILES_DIR/.config/zsh/dotfiles.zsh" && dotfiles-diff "$@"' _ "$@" 2>&1
}
state() { (cd "$H" && find . -type f -exec cksum {} + | sort); }
out=$(diff_)
[[ $out == *"This machine matches it."* ]] && pass "matches right after a sync" || fail "after a sync: $out"
echo "# local edit" >>"$H/.zshrc"
rm "$H/.config/sketchybar/colors.sh"
mkdir -p "$H/.config/cmux" && echo old >"$H/.config/cmux/settings.json"
before=$(state)
out=$(diff_)
[[ $out == *"changed  ~/.zshrc"* ]] && pass "lists a changed file" || fail "no changed ~/.zshrc: $out"
[[ $out == *"new      ~/.config/sketchybar/colors.sh"* ]] && pass "lists a missing file as new" || fail "no new colors.sh: $out"
[[ $out == *"removed  ~/.config/cmux/settings.json"* ]] && pass "lists an obsolete file" || fail "no removed cmux: $out"
out=$(diff_ -p)
[[ $out == *"+++ "*"/.zshrc"* && $out == *"-# local edit"* ]] && pass "-p shows the diff (- is this machine)" || fail "-p: $out"
[ "$(state)" = "$before" ] && pass "changes nothing in HOME" || fail "dotfiles-diff changed HOME"
# Upstream moves ahead: compared with what the sync would pull, not the clone
g clone -q "$t/origin.git" "$t/other"
echo "# from upstream" >>"$t/other/.config/git/ignore"
g -C "$t/other" commit -qam upstream && g -C "$t/other" push -q
out=$(diff_)
[[ $out == *"1 commit(s) ahead of the clone"* && $out == *"changed  ~/.config/git/ignore"* ]] &&
  pass "compares with the upstream the sync pulls" || fail "upstream: $out"
g -C "$t/dotfiles" pull -q --ff-only # back in step for the rest
echo "# not pushed" >>"$t/dotfiles/.config/git/ignore"
g -C "$t/dotfiles" commit -qam "not pushed"
out=$(diff_)
[[ $out == *"Compared with $(g -C "$t/dotfiles" branch --show-current) "* && $out == *"changed  ~/.config/git/ignore"* ]] &&
  pass "compares with HEAD when the clone is ahead" || fail "ahead: $out"

echo "== a broken build fails the sync"
echo 'not go' >>"$t/dotfiles/.config/dotfiles/local-files/main.go"
if sync >/dev/null; then fail "dotfiles-sync passed with a broken build"; else pass "dotfiles-sync fails"; fi

[ $failed -eq 0 ] && echo "PASS" || echo "FAILED"
exit $failed

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const heredocCommit = "git commit -m \"$(cat <<'EOF'\nfeat(x): add the thing\n\nbody line\nEOF\n)\""

func TestCheck(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		deny string // a part of the reason; "" = allowed
	}{
		// allowed
		{"plain commit", `git commit -m "feat(x): add the thing"`, ""},
		{"heredoc commit", heredocCommit, ""},
		{"two -m", `git commit -m "fix: a" -m "the body, with accents: não"`, ""},
		{"amend no edit", `git commit --amend --no-edit`, ""},
		{"no message (editor)", `git commit`, ""},
		{"not a commit", `git log --oneline -n 3`, ""},
		{"push", `git push origin feat/x`, ""},
		{"commit text in an echo", `echo "git commit --no-verify"`, ""},
		{"trailer text in a grep", `grep -r "Co-Authored-By" .`, ""},
		{"commit-tree is not commit", `git commit-tree HEAD^{tree} -m x`, ""},
		{"gh pr view", `gh pr view 12`, ""},
		{"pr create", `gh pr create --title "feat: add the thing" --body "text"`, ""},
		{"pr create heredoc body", "gh pr create --title \"fix: x\" --body \"$(cat <<'EOF'\n## Summary\n- one\nEOF\n)\"", ""},
		{"empty command", ``, ""},

		// commits
		{"co-author in -m", `git commit -m "feat: x" -m "Co-Authored-By: Claude <noreply@anthropic.com>"`, "attribution"},
		{"co-author in heredoc", "git commit -m \"$(cat <<'EOF'\nfeat: x\n\nCo-Authored-By: Someone <a@b.c>\nEOF\n)\"", "attribution"},
		{"generated with", "git commit -m \"$(cat <<'EOF'\nfeat: x\n\n🤖 Generated with [Claude Code](https://claude.com/claude-code)\nEOF\n)\"", "attribution"},
		{"--no-verify", `git commit --no-verify -m "feat: x"`, "--no-verify"},
		{"-n", `git commit -n -m "feat: x"`, "--no-verify"},
		{"-nm", `git commit -nm "feat: x"`, "--no-verify"},
		{"-am is fine", `git commit -am "feat: x"`, ""},
		{"-m value with n is fine", `git commit -m "nothing -n here"`, ""},
		{"chained", `git add . && git commit --no-verify -m "feat: x"`, "--no-verify"},
		{"git -C", `git -C ../other commit -n -m "feat: x"`, "--no-verify"},
		{"git -c", `git -c user.name=x commit --no-verify -m "x"`, "--no-verify"},
		{"rtk", `rtk git commit --no-verify -m "feat: x"`, "--no-verify"},
		{"env prefix", `GIT_AUTHOR_NAME=x git commit --no-verify -m "x"`, "--no-verify"},
		{"subshell", `(cd x && git commit --no-verify -m "x")`, "--no-verify"},
		{"after a heredoc", "cat <<'EOF' > notes.txt\nsome text\nEOF\ngit commit --no-verify -m \"x\"", "--no-verify"},
		{"long header", `git commit -m "feat: ` + strings.Repeat("a", 100) + `"`, "characters"},
		{"final full stop", `git commit -m "feat: add the thing."`, "full stop"},
		{"not English", `git commit -m "feat: adiciona a validação"`, "English"},
		{"-F - heredoc", "git commit -F - <<'EOF'\nfeat: adiciona a validação\nEOF", "English"},
		{"-F - heredoc ok", "git commit -F - <<'EOF'\nfeat: add the validation\nEOF", ""},
		{"--message=", `git commit --message="fix: done."`, "full stop"},
		{"heredoc header after a blank line", "git commit -m \"$(cat <<'EOF'\n\nfeat: adição\nEOF\n)\"", "English"},
		{"heredoc with <<-", "git commit -m \"$(cat <<-EOF\n\tfeat: done.\n\tEOF\n)\"", "full stop"},

		// pull requests
		{"pr attribution", `gh pr create --title "feat: x" --body "Co-authored-by: a <b@c.d>"`, "attribution"},
		{"pr generated", "gh pr create --title \"fix: x\" --body \"$(cat <<'EOF'\n🤖 Generated with Claude Code\nEOF\n)\"", "attribution"},
		{"pr title full stop", `gh pr create --title "feat: x." --body "b"`, "full stop"},
		{"pr title English", `gh pr create -t "feat: validação" -b "b"`, "English"},
		{"pr title=", `gh pr create --title="feat: add the thing." --body b`, "full stop"},
		{"pr edit", `gh pr edit 3 --title "fix: x."`, "full stop"},
		{"pr create after a commit", `git commit -m "feat: x" && gh pr create --title "fix: y." --body b`, "full stop"},
		{"pr create without title", `gh pr create --body b`, "--title"},
		{"pr create without body", `gh pr create --title "feat: x"`, "--body"},
		{"pr create --fill", `gh pr create --fill`, ""},
		{"pr create --body-file", `gh pr create --title "feat: x" --body-file notes.md`, ""},
		{"pr create --web", `gh pr create --web`, ""},
		{"pr create empty body", `gh pr create --title "feat: x" --body ""`, "empty"},
		{"pr edit needs no body", `gh pr edit 3 --title "fix: x"`, ""},

		// pushes (the current branch is "feat/x" unless a test says otherwise)
		{"push", `git push -u origin feat/x`, ""},
		{"push main", `git push origin main`, ""},
		{"force-push a feature branch", `git push --force origin feat/x`, ""},
		{"force-with-lease a feature branch", `git push --force-with-lease origin feat/x`, ""},
		{"force-push main", `git push --force origin main`, "main"},
		{"-f main", `git push -f origin main`, "main"},
		{"-fu master", `git push -fu origin master`, "master"},
		{"force-with-lease develop", `git push --force-with-lease=develop origin develop`, "develop"},
		{"+main", `git push origin +main`, "main"},
		{"HEAD:main forced", `git push -f origin HEAD:main`, "main"},
		{"refs/heads/main forced", `git push -f origin feat/x:refs/heads/main`, "main"},
		{"delete main", `git push origin --delete main`, "main"},
		{"delete main by refspec", `git push origin :main`, "main"},
		{"delete a feature branch", `git push origin --delete feat/old`, ""},
		{"bare force-push on a feature branch", `git push --force`, ""},
		{"bare force-push chained", `git add . && git push -f origin`, ""},
		{"git -C push", `git -C ../other push -f origin main`, "main"},
		{"push option value", `git push -o ci.skip origin feat/x`, ""},
		{"force-push after --", `git push -f -- origin main`, "main"},
	}
	currentBranch = func(string) string { return "feat/x" }
	aliases := map[string]string{
		"ci":   "commit -v",
		"ship": "push --force origin main",
		"sk":   "!git commit --no-verify -m",
		"ok":   "status",
	}
	aliasOf = func(_, name string) string { return aliases[name] }
	cases = append(cases, []struct {
		name string
		cmd  string
		deny string
	}{
		// --no-verify however it is spelled, on every command that has hooks
		{"--no-verif", `git commit --no-verif -m "feat: x"`, "--no-verify"},
		{"--no-v", `git commit --no-v -m "feat: x"`, "--no-verify"},
		{"--no- is another option", `git commit --no-edit`, ""},
		{"--no-status is another option", `git commit --no-status -m "x"`, ""},
		{"push --no-verify", `git push --no-verify origin feat/x`, "--no-verify"},
		{"push --no-verif", `git push --no-verif origin feat/x`, "--no-verify"},
		{"merge --no-verify", `git merge --no-verify feat/x`, "--no-verify"},
		{"rebase --no-verify", `git rebase --no-verify main`, "--no-verify"},
		{"push -n is a dry run", `git push -n origin feat/x`, ""},
		{"--no-verify after --", `git commit -m x -- --no-verify`, ""},

		// the hooks themselves
		{"-c core.hooksPath", `git -c core.hooksPath=/dev/null commit -m "feat: x"`, "hooks"},
		{"-c core.hookspath lower case", `git -c core.hookspath=/dev/null commit -m x`, "hooks"},
		{"-c hook.x.enabled", `git -c hook.sops-guard.enabled=false commit -m x`, "hooks"},
		{"-c hook.x.command", `git -c hook.commit-msg-go.command=true commit -m x`, "hooks"},
		{"--config-env", `git --config-env=core.hooksPath=EMPTY commit -m x`, "hooks"},
		{"-c alias", `git -c alias.ci="commit --no-verify" ci -m x`, "alias"},
		{"-c something else", `git -c user.name=x commit -m "feat: x"`, ""},
		{"env GIT_CONFIG_KEY", `GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=core.hooksPath GIT_CONFIG_VALUE_0=/dev/null git commit -m x`, "GIT_CONFIG"},
		{"export GIT_CONFIG_PARAMETERS", `export GIT_CONFIG_PARAMETERS="'hook.sops-guard.enabled=false'"`, "GIT_CONFIG"},
		{"config set hooksPath", `git config core.hooksPath /dev/null`, "hooks"},
		{"config --global hooksPath", `git config --global core.hooksPath /dev/null`, "hooks"},
		{"config hook disabled", `git config hook.sops-guard.enabled false`, "hooks"},
		{"config set (new syntax)", `git config set hook.sops-guard.enabled false`, "hooks"},
		{"config unset hook", `git config --unset hook.sops-guard.enabled`, "hooks"},
		{"config --local hook", `git config --local hook.shellcheck.enabled false`, "hooks"},
		{"config reads", `git config hook.sops-guard.enabled`, ""},
		{"config --get", `git config --get core.hooksPath`, ""},
		{"config --get-regexp", `git config --get-regexp '^hook\.'`, ""},
		{"config --list", `git config --list --show-origin`, ""},
		{"config get (new syntax)", `git config get hook.sops-guard.enabled`, ""},
		{"config another key", `git config user.name "x"`, ""},
		{"config alias is allowed", `git config --global alias.co checkout`, ""},

		// commands inside commands
		{"bash -c", `bash -c 'git commit --no-verify -m "feat: x"'`, "--no-verify"},
		{"sh -c", `sh -c "git push -f origin main"`, "main"},
		{"zsh -lc", `zsh -lc 'git commit -n -m x'`, "--no-verify"},
		{"bash -c with a clean command", `bash -c 'git status'`, ""},
		{"eval", `eval "git commit --no-verify -m x"`, "--no-verify"},
		{"eval unquoted", `eval git commit --no-verify -m x`, "--no-verify"},
		{"$( )", `echo "$(git commit --no-verify -m x)"`, "--no-verify"},
		{"$( ) nested", `echo $(echo $(git push -f origin main))`, "main"},
		{"backticks", "echo `git commit -n -m x`", "--no-verify"},
		{"arithmetic is not a command", `echo $((1 + 2))`, ""},
		{"sudo", `sudo git push -f origin main`, "main"},
		{"env", `env FOO=1 git commit --no-verify -m x`, "--no-verify"},
		{"env -i", `env -i git commit --no-verify -m x`, "--no-verify"},
		{"nohup", `nohup git push -f origin main`, "main"},
		{"backslash", `\git commit --no-verify -m x`, "--no-verify"},
		{"quoted program", `"git" commit --no-verify -m x`, "--no-verify"},
		{"full path", `/usr/bin/git commit --no-verify -m x`, "--no-verify"},
		{"alias to a commit", `git ci -m "feat: validação"`, "English"},
		{"alias to a push", `git ship`, "main"},
		{"shell alias", `git sk "feat: x"`, "--no-verify"},
		{"alias to status", `git ok`, ""},
		{"unknown alias", `git nothing-here`, ""},
	}...)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ask := check(c.cmd, "")
			switch {
			case c.deny == "" && got != "":
				t.Errorf("stopped: %s", got)
			case c.deny != "" && !strings.Contains(got, c.deny):
				t.Errorf("reason %q does not mention %q", got, c.deny)
			case ask:
				t.Errorf("a deny came back as a question: %s", got)
			}
		})
	}
}

func TestBareForcePushOnProtectedBranch(t *testing.T) {
	defer func(f func(string) string) { currentBranch = f }(currentBranch)
	var asked string
	currentBranch = func(dir string) string { asked = dir; return "main" }

	for _, cmd := range []string{`git push --force`, `git push -f origin`, `git push -f origin HEAD`, `git push --force-with-lease origin HEAD:main`} {
		if got, ask := check(cmd, "/repo"); !strings.Contains(got, "main") || ask {
			t.Errorf("%s: got %q (ask=%v)", cmd, got, ask)
		}
	}
	if asked != "/repo" {
		t.Errorf("the branch was looked up in %q, not the session's directory", asked)
	}
	// Not forced: a plain push of main is the user's normal flow
	if got, _ := check(`git push origin HEAD`, "/repo"); got != "" {
		t.Errorf("plain push stopped: %s", got)
	}
}

func TestAsk(t *testing.T) {
	cases := []struct {
		name, cmd string
		ask       bool
	}{
		{"pr merge", `gh pr merge 12 --squash`, true},
		{"pr merge auto", `gh pr merge --auto`, true},
		{"reset hard", `git reset --hard HEAD~1`, true},
		{"reset soft", `git reset --soft HEAD~1`, false},
		{"reset path", `git reset HEAD file`, false},
		{"clean -fd", `git clean -fd`, true},
		{"clean -fdx", `git clean -fdx`, true},
		{"clean --force", `git clean --force`, true},
		{"clean dry run", `git clean -n`, false},
		{"clean -nfd is a dry run", `git clean -nfd`, false},
		{"clean --dry-run", `git clean -f --dry-run`, false},
		{"clean without force", `git clean`, false},
		{"chained", `git fetch && git reset --hard origin/main`, true},
		{"pr view", `gh pr view 12`, false},
		{"merge text in an echo", `echo "gh pr merge"`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ask := check(c.cmd, "")
			if ask != c.ask || (got != "") != c.ask {
				t.Errorf("reason %q, ask=%v; want ask=%v", got, ask, c.ask)
			}
		})
	}

	// A deny in the same command wins over a question
	got, ask := check(`gh pr merge 3 && git commit --no-verify -m "x"`, "")
	if ask || !strings.Contains(got, "--no-verify") {
		t.Errorf("deny should win: %q (ask=%v)", got, ask)
	}
}

func TestRun(t *testing.T) {
	ev := func(cmd string) *bytes.Reader {
		data, _ := json.Marshal(map[string]any{"tool_name": "Bash", "tool_input": map[string]string{"command": cmd}})
		return bytes.NewReader(data)
	}

	var out bytes.Buffer
	run(ev(`git commit --no-verify -m "feat: x"`), &out)
	var resp struct {
		HookSpecificOutput struct {
			HookEventName            string `json:"hookEventName"`
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("not JSON: %q", out.String())
	}
	h := resp.HookSpecificOutput
	if h.HookEventName != "PreToolUse" || h.PermissionDecision != "deny" || h.PermissionDecisionReason == "" {
		t.Errorf("unexpected response: %+v", h)
	}

	out.Reset()
	run(ev(`git commit -m "feat: x"`), &out)
	if out.Len() != 0 {
		t.Errorf("allowed command printed %q", out.String())
	}

	// A question is an "ask", not a deny
	out.Reset()
	run(ev(`gh pr merge 12`), &out)
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil || resp.HookSpecificOutput.PermissionDecision != "ask" {
		t.Errorf("pr merge: %q", out.String())
	}

	// Input that is not an event is allowed, like any other tool
	for _, in := range []string{"", "not json", "{}", `{"tool_input": 3}`} {
		out.Reset()
		run(strings.NewReader(in), &out)
		if out.Len() != 0 {
			t.Errorf("input %q printed %q", in, out.String())
		}
	}
}

func TestExtractHeredocs(t *testing.T) {
	body, docs := extractHeredocs("cat <<'EOF' > f\na; b\nEOF\necho ok <<< here\ngit status")
	if len(docs) != 1 || docs[0] != "a; b" {
		t.Errorf("docs = %q", docs)
	}
	if strings.Contains(body, "a; b") || !strings.Contains(body, "<<@heredoc-0 > f") || !strings.Contains(body, "git status") {
		t.Errorf("body = %q", body)
	}

	// Without a terminator the rest is the body, and nothing loops
	if _, docs := extractHeredocs("cat <<EOF\nnever closed"); len(docs) != 1 || docs[0] != "never closed" {
		t.Errorf("unterminated docs = %q", docs)
	}
	// `<<` that is not a heredoc (a shift, a stray) is left as it is
	if body, docs := extractHeredocs("echo $((1 << 2))"); len(docs) != 0 || body != "echo $((1 << 2))" {
		t.Errorf("shift: body = %q, docs = %q", body, docs)
	}
}

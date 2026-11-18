package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const (
	home = "/Users/me"
	cwd  = "/Users/me/work/repo"
)

func TestToolPaths(t *testing.T) {
	h := hook{home: home, cwd: cwd}
	for _, tt := range []struct {
		name, path string
		deny       string // a part of the reason; "" = allowed
	}{
		// allowed
		{"source file", "main.go", ""},
		{"absolute in the repo", cwd + "/src/app.ts", ""},
		{"env template", ".env.example", ""},
		{"env sample", "config/.env.sample", ""},
		{"the dotfiles claude/ source", cwd + "/claude/settings.json", ""},
		{"claude memory", home + "/.claude/projects/x/memory/note.md", ""},
		{"claude CLAUDE.md", home + "/.claude/CLAUDE.md", ""},
		{"a git file that is not config", ".git/HEAD", ""},
		{"gitconfig-like name elsewhere", cwd + "/.gitconfig-notes", ""},
		{"environment", ".environment", ""},
		{"empty", "", ""},

		// ssh and aws
		{"ssh key", home + "/.ssh/id_ed25519", "ssh"},
		{"ssh config", "~/.ssh/config", "ssh"},
		{"authorized_keys", home + "/.ssh/authorized_keys", "ssh"},
		{"aws credentials", "$HOME/.aws/credentials", "AWS"},
		{"aws config", "${HOME}/.aws/config", "AWS"},

		// secrets by name
		{"env", ".env", "env file"},
		{"env local", "app/.env.local", "env file"},
		{"envrc", ".envrc", "env file"},
		{"pem", "certs/server.pem", "key or certificate"},
		{"key", "server.key", "key or certificate"},
		{"age key", home + "/.config/sops/age/keys.txt", "age key"},
		{"agekey", "my.agekey", "age key"},
		{"ssh key name elsewhere", "backup/id_rsa", "ssh private key"},
		{"case", cwd + "/.ENV", "env file"},

		// git
		{"git hook", ".git/hooks/pre-commit", "git hooks"},
		{"git config", ".git/config", "git hooks"},
		{"git hook in a worktree path", cwd + "/../other/.git/hooks/commit-msg", "git hooks"},
		{"global gitconfig", home + "/.gitconfig", "global git config"},
		{"gitconfig-work", "~/.gitconfig-work", "global git config"},
		{"global git hooks", home + "/git_hooks/go/commit-msg", "global git hooks"},
		{"git ignore config dir", home + "/.config/git/ignore", "global git hooks"},

		// the guardrails
		{"user settings", home + "/.claude/settings.json", "guardrails"},
		{"user settings ~", "~/.claude/settings.json", "guardrails"},
		{"local settings", "~/.claude/settings.local.json", "guardrails"},
		{"a hook", home + "/.claude/hooks/git-guard/main.go", "guardrails"},
		{"project settings", cwd + "/.claude/settings.json", "guardrails"},
		{"project hooks", ".claude/hooks/x.sh", "guardrails"},
		{"prod regex", home + "/.config/kube-guard/prod-regex", "production"},
		{"dotdot", cwd + "/../../.ssh/id_rsa", "ssh"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := h.checkPath(tt.path)
			switch {
			case tt.deny == "" && got != "":
				t.Errorf("%q denied: %s", tt.path, got)
			case tt.deny != "" && !strings.Contains(got, tt.deny):
				t.Errorf("%q: got %q, want a reason with %q", tt.path, got, tt.deny)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	h := hook{home: home, cwd: cwd}
	for _, tt := range []struct {
		name, cmd string
		deny      string
	}{
		// allowed: reads, and writes elsewhere
		{"read ssh config", `cat ~/.ssh/config`, ""},
		{"read settings", `cat ~/.claude/settings.json`, ""},
		{"ls hooks", `ls ~/.claude/hooks`, ""},
		{"grep git hooks", `grep -r x .git/hooks`, ""},
		{"git status", `git status`, ""},
		{"write a file", `echo hi > out.txt`, ""},
		{"redirect stderr", `go test ./... 2>&1`, ""},
		{"redirect to /dev/null", `kubectl get pods >/dev/null 2>&1`, ""},
		{"tee elsewhere", `echo x | tee /tmp/x`, ""},
		{"cp from a protected path (a read)", `cp ~/.ssh/config /tmp/x`, ""},
		{"sed without -i", `sed s/a/b/ ~/.ssh/config`, ""},
		{"rm in the repo", `rm -rf build`, ""},
		{"env example", `cp .env.example .env.example.bak`, ""},
		{"echo text", `echo "rm ~/.ssh/id_rsa"`, ""},
		{"empty", ``, ""},

		// redirections
		{"redirect", `echo x > ~/.ssh/authorized_keys`, "redirects"},
		{"append", `echo x >> ~/.claude/settings.json`, "redirects"},
		{"no space", `echo x>.env`, "redirects"},
		{"quoted target", `echo x > "$HOME/.aws/credentials"`, "redirects"},
		{"clobber", `echo x >| .git/hooks/pre-commit`, "redirects"},
		{"heredoc into a file", "cat > ~/.claude/hooks/x.sh <<'EOF'\nexit 0\nEOF", "redirects"},

		// programs that write
		{"tee", `echo x | tee ~/.gitconfig`, "tee"},
		{"tee -a", `echo x | tee -a ~/.claude/settings.json`, "tee"},
		{"rm", `rm ~/.ssh/id_rsa`, "rm"},
		{"rm -rf the hooks", `rm -rf ~/.claude/hooks`, "rm"},
		{"mv the source", `mv ~/.claude/settings.json /tmp/x`, "mv"},
		{"mv onto", `mv /tmp/x .git/hooks/pre-commit`, "mv"},
		{"cp onto", `cp /tmp/x ~/.claude/settings.json`, "cp"},
		{"cp -t", `cp -t ~/.ssh /tmp/x`, "cp"},
		{"cp --target-directory=", `cp --target-directory=.git/hooks /tmp/x`, "cp"},
		{"ln", `ln -sf /tmp/x ~/.claude/settings.json`, "ln"},
		{"chmod", `chmod 777 ~/.ssh/id_rsa`, "chmod"},
		{"sed -i", `sed -i s/a/b/ ~/.claude/settings.json`, "sed"},
		{"sed -i.bak", `sed -i.bak s/a/b/ .git/config`, "sed"},
		{"sed -ni", `sed -ni 1p ~/.gitconfig`, "sed"},
		{"sed --in-place", `sed --in-place s/a/b/ .env`, "sed"},
		{"perl -pi", `perl -pi -e 's/a/b/' ~/.gitconfig`, "perl"},
		{"dd", `dd if=/dev/zero of=~/.ssh/id_rsa`, "dd"},
		{"touch an env", `touch .env`, "touch"},
		{"truncate", `truncate -s 0 ~/.config/kube-guard/prod-regex`, "truncate"},

		// hidden in the shell
		{"after &&", `cd x && rm ~/.ssh/id_rsa`, "rm"},
		{"pipe to xargs", `ls | xargs rm ~/.ssh/id_rsa`, "rm"},
		{"sudo", `sudo rm ~/.ssh/id_rsa`, "rm"},
		{"rtk", `rtk rm ~/.ssh/id_rsa`, "rm"},
		{"env prefix", `FOO=1 tee ~/.gitconfig`, "tee"},
		{"sh -c", `sh -c "echo x > ~/.claude/settings.json"`, "redirects"},
		{"bash -lc", `bash -lc 'rm ~/.ssh/id_rsa'`, "rm"},
		{"eval", `eval "rm ~/.ssh/id_rsa"`, "rm"},
		{"substitution", `echo $(rm ~/.ssh/id_rsa)`, "rm"},
		{"backticks", "echo `rm ~/.ssh/id_rsa`", "rm"},
		{"full path program", `/bin/rm ~/.ssh/id_rsa`, "rm"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := h.checkCommand(tt.cmd, 0)
			switch {
			case tt.deny == "" && got != "":
				t.Errorf("%q denied: %s", tt.cmd, got)
			case tt.deny != "" && !strings.Contains(got, tt.deny):
				t.Errorf("%q: got %q, want a reason with %q", tt.cmd, got, tt.deny)
			}
		})
	}
}

func decide(t *testing.T, ev map[string]any) map[string]string {
	t.Helper()
	data, _ := json.Marshal(ev)
	var out bytes.Buffer
	run(bytes.NewReader(data), &out, home)
	if out.Len() == 0 {
		return nil
	}
	var resp struct {
		Out map[string]string `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("not JSON: %q", out.String())
	}
	return resp.Out
}

func TestRun(t *testing.T) {
	for _, tt := range []struct {
		name string
		ev   map[string]any
		deny bool
	}{
		{"Write denied", map[string]any{"tool_name": "Write", "cwd": cwd, "tool_input": map[string]string{"file_path": home + "/.ssh/config"}}, true},
		{"Edit denied", map[string]any{"tool_name": "Edit", "cwd": cwd, "tool_input": map[string]string{"file_path": ".env"}}, true},
		{"MultiEdit denied", map[string]any{"tool_name": "MultiEdit", "cwd": cwd, "tool_input": map[string]string{"file_path": ".git/config"}}, true},
		{"NotebookEdit denied", map[string]any{"tool_name": "NotebookEdit", "cwd": cwd, "tool_input": map[string]string{"notebook_path": home + "/.claude/settings.json"}}, true},
		{"Bash denied", map[string]any{"tool_name": "Bash", "cwd": cwd, "tool_input": map[string]string{"command": "rm ~/.ssh/id_rsa"}}, true},
		{"Write allowed", map[string]any{"tool_name": "Write", "cwd": cwd, "tool_input": map[string]string{"file_path": "main.go"}}, false},
		{"Bash allowed", map[string]any{"tool_name": "Bash", "cwd": cwd, "tool_input": map[string]string{"command": "go test ./..."}}, false},
		{"Read is not its business", map[string]any{"tool_name": "Read", "cwd": cwd, "tool_input": map[string]string{"file_path": home + "/.ssh/config"}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := decide(t, tt.ev)
			if tt.deny && got["permissionDecision"] != "deny" {
				t.Errorf("want deny, got %v", got)
			}
			if !tt.deny && got != nil {
				t.Errorf("want nothing, got %v", got)
			}
		})
	}

	var out bytes.Buffer
	run(strings.NewReader("not json"), &out, home)
	if out.Len() != 0 {
		t.Errorf("bad input printed %q, want nothing", out.String())
	}
}

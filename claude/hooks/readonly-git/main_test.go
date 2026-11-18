package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	for _, tt := range []struct {
		name, cmd string
		deny      string // a part of the reason; "" = allowed
	}{
		// allowed: git that only reads
		{"diff", `git diff`, ""},
		{"diff against a base", `git diff main...HEAD`, ""},
		{"diff stat", `git diff --stat origin/main...HEAD`, ""},
		{"diff of a path", `git diff HEAD -- claude/settings.json`, ""},
		{"log", `git log --oneline -5`, ""},
		{"log with a format", `git log --format=%h -3`, ""},
		{"show", `git show HEAD~1`, ""},
		{"status", `git status --short`, ""},
		{"ls-files", `git ls-files claude`, ""},
		{"blame", `git blame -L 1,20 main.go`, ""},
		{"merge-base", `git merge-base main HEAD`, ""},
		{"rev-parse", `git rev-parse --abbrev-ref HEAD`, ""},
		{"rtk in front", `rtk git diff`, ""},
		{"a flag after -- is a path", `git diff -- --output`, ""},
		{"empty", ``, ""},

		// denied: not a read
		{"commit", `git commit -m x`, "not one of"},
		{"push", `git push`, "not one of"},
		{"reset", `git reset --hard`, "not one of"},
		{"checkout", `git checkout x`, "not one of"},
		{"config", `git config user.name x`, "not one of"},
		{"clean", `git clean -fd`, "not one of"},
		{"stash", `git stash`, "not one of"},
		{"other programs", `cat README.md`, "may only run git"},
		{"rm", `rm -rf x`, "may only run git"},
		{"gh", `gh pr diff 3`, "may only run git"},
		{"git alone", `git`, "may only run git"},
		{"env prefix", `FOO=1 git diff`, "may only run git"},
		{"absolute path to git", `/usr/bin/git diff`, "may only run git"},

		// denied: an option that runs a program or moves the repository
		{"-c", `git -c diff.external=./x diff`, "No option before"},
		{"-C", `git -C /other diff`, "No option before"},
		{"--git-dir", `git --git-dir=/x diff`, "No option before"},
		{"--no-pager is still an option before", `git --no-pager diff`, "No option before"},

		// denied: a flag that writes or runs
		{"--output=", `git diff --output=out.patch`, "writes a file"},
		{"--output file", `git log --output out.txt`, "writes a file"},
		{"--ext-diff", `git diff --ext-diff`, "runs a program"},
		{"--textconv", `git show --textconv HEAD:x`, "runs a program"},
		{"-O", `git diff -Otool`, "runs a program"},
		{"--open-files-in-pager", `git grep --open-files-in-pager`, "not one of"},
		{"--output on show", `git show --output=x HEAD`, "writes a file"},

		// denied: shell syntax
		{"semicolon", `git diff; rm x`, "one simple git command"},
		{"and", `git diff && git push`, "one simple git command"},
		{"pipe", `git diff | head`, "one simple git command"},
		{"redirect", `git diff > out.patch`, "one simple git command"},
		{"redirect in", `git diff < x`, "one simple git command"},
		{"substitution", `git diff $(rm x)`, "one simple git command"},
		{"backticks", "git diff `rm x`", "one simple git command"},
		{"newline", "git diff\nrm x", "one simple git command"},
		{"background", `git diff &`, "one simple git command"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := check(tt.cmd)
			switch {
			case tt.deny == "" && got != "":
				t.Errorf("%q denied: %s", tt.cmd, got)
			case tt.deny != "" && !strings.Contains(got, tt.deny):
				t.Errorf("%q: got %q, want a reason with %q", tt.cmd, got, tt.deny)
			}
		})
	}
}

func TestRun(t *testing.T) {
	event := func(cmd string) *bytes.Reader {
		b, _ := json.Marshal(map[string]any{"tool_input": map[string]string{"command": cmd}})
		return bytes.NewReader(b)
	}

	var out bytes.Buffer
	run(event("git commit -m x"), &out)
	var resp struct {
		Out map[string]string `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("not JSON: %q", out.String())
	}
	if resp.Out["permissionDecision"] != "deny" || !strings.Contains(resp.Out["permissionDecisionReason"], "readonly-git") {
		t.Errorf("decision = %v", resp.Out)
	}

	out.Reset()
	run(event("git diff main...HEAD"), &out)
	if out.Len() != 0 {
		t.Errorf("a read printed %q, want nothing", out.String())
	}

	out.Reset()
	run(strings.NewReader("not json"), &out)
	if out.Len() != 0 {
		t.Errorf("bad input printed %q, want nothing", out.String())
	}
}

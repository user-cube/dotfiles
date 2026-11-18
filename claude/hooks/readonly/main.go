// readonly: a PreToolUse hook that narrows one subagent's Bash to commands that
// only read. It is not a global hook: each agent names it in its own header,
// with the profiles it needs.
//
//	command: ~/.claude/hooks/readonly/readonly git helm
//
// A subagent's `tools:` line takes whole tools, not patterns: Bash is all of
// Bash. A profile (profiles.go) lists, per program, the subcommands that read,
// the flags that don't (a flag that writes a file, talks to a cluster, fetches
// from the network or runs a program) and the arguments that may not be passed
// (a remote URL, a path to a secret). Profiles are added together: `git helm`
// allows what either allows.
//
//	allowed   one simple command: a program a profile names, a subcommand it
//	          lists, flags it does not deny. `rtk` in front is allowed (the
//	          global rtk hook rewrites commands).
//	denied    anything else; an option before the subcommand (`git -c`, `helm
//	          --kube-context` can run a program or move to another target);
//	          shell syntax: ; & | < > ` $( ) and a newline.
//
// The global hooks (deny-sensitive-reads, write-guard) still apply on top: this
// narrows, it does not replace them. It exists because a command such as
// `helm template -f secrets.yaml` reads a file the Read tool's hooks never see.
//
// Reads the hook event (JSON) on stdin; prints a deny decision, or nothing to
// allow. Always exits 0. A profile it does not know denies everything: a typo
// in an agent's header must not widen what the agent may run.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
)

const rule = "local rule in ~/.claude/hooks/readonly"

type event struct {
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

func main() { run(os.Stdin, os.Stdout, os.Args[1:]) }

func run(in io.Reader, out io.Writer, profileNames []string) {
	data, _ := io.ReadAll(in)
	var ev event
	if json.Unmarshal(data, &ev) != nil {
		return
	}
	if reason := check(ev.ToolInput.Command, profileNames); reason != "" {
		resp, _ := json.Marshal(map[string]any{"hookSpecificOutput": map[string]string{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": reason,
		}})
		fmt.Fprintln(out, string(resp))
	}
}

// programs merges the profiles named; ok is false (and bad says which) when one is unknown
func programs(profileNames []string) (merged map[string]program, bad string, ok bool) {
	if len(profileNames) == 0 {
		return nil, "none given", false
	}
	merged = map[string]program{}
	for _, name := range profileNames {
		p, known := profiles[name]
		if !known {
			return nil, name, false
		}
		for prog, r := range p {
			merged[prog] = r
		}
	}
	return merged, "", true
}

// check returns why the command is denied, or "" to allow it
func check(cmd string, profileNames []string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	progs, bad, ok := programs(profileNames)
	if !ok {
		return "readonly: profile " + bad + " is not one it knows, so it denies everything: fix the agent's header (" + rule + ")."
	}
	if strings.ContainsAny(cmd, ";&|<>`\n\r") || strings.Contains(cmd, "$(") {
		return "This agent runs one simple command: no ; & | < > backticks, $( ) or new lines (" + rule + ")."
	}
	words := strings.Fields(cmd)
	if len(words) > 0 && words[0] == "rtk" {
		words = words[1:]
	}
	if len(words) == 0 {
		return ""
	}
	r, known := progs[words[0]]
	if !known {
		return "This agent may only run: " + names(progs) + ". " + words[0] + " is not one of them (" + rule + ")."
	}
	return r.check(words)
}

func names(progs map[string]program) string {
	var list []string
	for p := range progs {
		list = append(list, p)
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}

// check applies one program's rule to its command line (words[0] is the program)
func (r program) check(words []string) string {
	prog, rest := words[0], words[1:]
	where := prog
	if r.subs != nil {
		// the options a program may take before its subcommand (terraform -chdir=dir)
		for len(rest) > 0 && startsWithAny(rest[0], r.leading) {
			rest = rest[1:]
		}
		if len(rest) == 0 {
			return prog + " alone: give it one of " + keys(r.subs) + " (" + rule + ")."
		}
		sub := rest[0]
		if strings.HasPrefix(sub, "-") {
			return "No option before the " + prog + " subcommand: -c, -C, --git-dir, --kube-context and the like can run a program or point at another target (" + rule + ")."
		}
		if !r.subs[sub] {
			return prog + " " + sub + " is not one of the " + prog + " commands this agent may run: " + keys(r.subs) + " (" + rule + ")."
		}
		if next, deeper := r.deeper[sub]; deeper {
			if len(rest) < 2 || !next[rest[1]] {
				return prog + " " + sub + " may only be followed by " + keys(next) + " (" + rule + ")."
			}
		}
		where = prog + " " + sub
		// a subcommand that only reads when run in a certain mode (terraform fmt -check)
		if n, has := r.needs[sub]; has && !slices.ContainsFunc(rest[1:], func(w string) bool { return startsWithAny(w, n.anyOf) }) {
			return where + " " + n.why + " (" + rule + ")."
		}
		rest = rest[1:]
	} else if n, has := r.needs[""]; has && !slices.ContainsFunc(rest, func(w string) bool { return startsWithAny(w, n.anyOf) }) {
		// a program with no subcommand that only reads in a certain mode (ansible-playbook --syntax-check)
		return prog + " " + n.why + " (" + rule + ")."
	}
	for i, w := range rest {
		if w == "--" {
			rest = rest[:i]
			break
		}
	}
	for _, w := range rest {
		for _, f := range r.flags {
			if strings.HasPrefix(w, f.prefix) {
				return where + " " + w + " " + f.why + ", which this agent may not do (" + rule + ")."
			}
		}
	}
	if r.arg != nil {
		for _, w := range words[1:] {
			if why := r.arg(w); why != "" {
				return where + " is given " + w + ", " + why + " (" + rule + ")."
			}
		}
	}
	return ""
}

func startsWithAny(w string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(w, p) {
			return true
		}
	}
	return false
}

func keys(m map[string]bool) string {
	var list []string
	for k := range m {
		list = append(list, k)
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}

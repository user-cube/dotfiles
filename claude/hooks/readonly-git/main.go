// readonly-git: a PreToolUse hook for one subagent's Bash (security-reviewer,
// in its own frontmatter), not a global one.
//
// A subagent's `tools:` line takes whole tools, not patterns: Bash is all of
// Bash. This is what narrows it to git commands that only read, so a reviewer
// can look at the diff and nothing else:
//
//	allowed   git diff | log | show | status | ls-files | ls-tree | blame |
//	          rev-parse | merge-base | shortlog | describe, with their flags
//	denied    anything else, and in these: a global option before the
//	          subcommand (-c, -C, --git-dir ... can run a program or move the
//	          repository), --output (writes a file), --ext-diff, --textconv and
//	          -O / --open-files-in-pager (run a program)
//	denied    any shell syntax: ; & | > < ` $( ) and a newline. One simple command.
//
// `rtk` in front is allowed (the global rtk hook rewrites git commands). The
// global hooks (deny-sensitive-reads, write-guard) still apply on top of this:
// it narrows, it does not replace them.
//
// Reads the hook event (JSON) on stdin; prints a deny decision, or nothing to
// allow. Always exits 0: input it can't parse is allowed, like any other tool,
// and the matcher in the agent's header is what scopes it to Bash.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const rule = "local rule in ~/.claude/hooks/readonly-git"

type event struct {
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

var subcommands = set("diff log show status ls-files ls-tree blame rev-parse merge-base shortlog describe")

// Flags that write a file or run a program, by their start: --output=x and
// --output x, --ext-diff, --textconv, -O<pager>, --open-files-in-pager
var denied = []string{"--output", "--ext-diff", "--textconv", "-O", "--open-files-in-pager", "--exec"}

func set(words string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(words) {
		m[w] = true
	}
	return m
}

func main() { run(os.Stdin, os.Stdout) }

func run(in io.Reader, out io.Writer) {
	data, _ := io.ReadAll(in)
	var ev event
	if json.Unmarshal(data, &ev) != nil {
		return
	}
	if reason := check(ev.ToolInput.Command); reason != "" {
		resp, _ := json.Marshal(map[string]any{"hookSpecificOutput": map[string]string{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": reason,
		}})
		fmt.Fprintln(out, string(resp))
	}
}

// check returns why the command is denied, or "" to allow it
func check(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	if strings.ContainsAny(cmd, ";&|<>`\n\r") || strings.Contains(cmd, "$(") {
		return "This agent runs one simple git command: no ; & | < > backticks, $( ) or new lines (" + rule + ")."
	}
	words := strings.Fields(cmd)
	if len(words) > 0 && words[0] == "rtk" {
		words = words[1:]
	}
	if len(words) < 2 || words[0] != "git" {
		return "This agent may only run git commands that read: diff, log, show, status, ls-files, ls-tree, blame, rev-parse, merge-base, shortlog, describe (" + rule + ")."
	}
	if !subcommands[words[1]] {
		if strings.HasPrefix(words[1], "-") {
			return "No option before the git subcommand: -c, -C and --git-dir can run a program or point at another repository (" + rule + ")."
		}
		return "git " + words[1] + " is not one of the commands that only read: diff, log, show, status, ls-files, ls-tree, blame, rev-parse, merge-base, shortlog, describe (" + rule + ")."
	}
	for _, w := range words[2:] {
		if w == "--" {
			break
		}
		for _, d := range denied {
			if strings.HasPrefix(w, d) {
				return "git " + words[1] + " " + w + " writes a file or runs a program, which this agent may not do (" + rule + ")."
			}
		}
	}
	return ""
}

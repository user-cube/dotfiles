package main

// The Claude Code side of kube-guard: a PreToolUse hook (matcher: Bash) that
// asks the user before a kubectl / helm command Claude runs changes a
// production cluster. Same decision as the zsh wrappers (parse, production,
// contextOf), which Claude's shell never goes through.
//
//	kube-guard claude-hook     reads the hook event (JSON) on stdin; prints an
//	                           "ask" decision, or nothing to allow. Exit 0 always.
//
// It looks inside sh -c, eval, $(...), backticks and sudo / env / rtk. LIMIT:
// a command built at runtime ($CMD, a script) is not seen: a safety net, not
// a sandbox. KUBE_GUARD=0 is not honoured here: it is the user's switch.

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const hookRule = "local rule in ~/.config/kube-guard"

const hookMaxDepth = 4

var (
	hookWrappers = []string{"rtk", "command", "env", "sudo", "nohup", "time", "exec", "xargs"}
	hookShells   = []string{"sh", "bash", "zsh", "dash", "ksh"}
	hookEnvAssn  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
)

type hookEvent struct {
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

// claudeHook reads the event from in and writes the decision to out
func claudeHook(in io.Reader, out io.Writer) int {
	data, _ := io.ReadAll(in)
	var ev hookEvent
	if json.Unmarshal(data, &ev) != nil {
		return 0
	}
	if reason := hookCheck(ev.ToolInput.Command, 0); reason != "" {
		resp, _ := json.Marshal(map[string]any{"hookSpecificOutput": map[string]string{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "ask",
			"permissionDecisionReason": reason,
		}})
		fmt.Fprintln(out, string(resp))
	}
	return 0
}

// hookCheck returns why the user must decide, or "" to let the command through
func hookCheck(cmd string, depth int) string {
	if cmd == "" || depth > hookMaxDepth {
		return ""
	}
	for _, s := range hookSubstitutions(cmd) {
		if r := hookCheck(s, depth+1); r != "" {
			return r
		}
	}
	for _, seg := range hookSegments(hookTokenize(cmd)) {
		args := hookStrip(seg)
		if len(args) == 0 {
			continue
		}
		switch prog := filepath.Base(args[0]); {
		case prog == "kubectl" || prog == "kubecolor" || prog == "helm":
			tool := prog
			if tool == "kubecolor" {
				tool = "kubectl"
			}
			if r := hookKube(tool, args[1:]); r != "" {
				return r
			}
		case prog == "eval":
			if r := hookCheck(strings.Join(args[1:], " "), depth+1); r != "" {
				return r
			}
		case slices.Contains(hookShells, prog):
			// bash -c "...", also bash -lc "..."
			for i, a := range args[1:] {
				if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.ContainsRune(a, 'c') {
					if i+2 < len(args) {
						if r := hookCheck(args[i+2], depth+1); r != "" {
							return r
						}
					}
					break
				}
			}
		}
	}
	return ""
}

// hookKube: one kubectl / helm command; "" when it only reads or the context
// is not production
func hookKube(tool string, args []string) string {
	l := parse(tool, args, isPlugin)
	if !l.changes(tool) {
		return ""
	}
	ctx := contextOf(l)
	prod, err := production(ctx)
	if err != nil {
		return fmt.Sprintf("kube-guard could not tell if the context is production (%v); %s %s (%s).", err, tool, strings.Join(args, " "), hookRule)
	}
	if !prod {
		return ""
	}
	return fmt.Sprintf("%s is production: %s %s changes it. The user decides (%s).", ctx, tool, strings.Join(args, " "), hookRule)
}

// hookStrip drops what comes before the program: VAR=x, rtk, sudo, env ...
func hookStrip(seg []string) []string {
	wrapped := false
	for len(seg) > 0 {
		switch a := seg[0]; {
		case hookEnvAssn.MatchString(a):
		case slices.Contains(hookWrappers, a):
			wrapped = true
		case wrapped && strings.HasPrefix(a, "-"): // env -i, sudo -E
		default:
			return seg
		}
		seg = seg[1:]
	}
	return seg
}

// hookSubstitutions: the commands inside $(...) and `...`, not $((arithmetic))
func hookSubstitutions(s string) []string {
	var subs []string
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '$' && strings.HasPrefix(s[i:], "$(") && !strings.HasPrefix(s[i:], "$(("):
			depth, j := 1, i+2
			for ; j < len(s) && depth > 0; j++ {
				switch s[j] {
				case '(':
					depth++
				case ')':
					depth--
				}
			}
			if depth == 0 {
				subs = append(subs, s[i+2:j-1])
				i = j - 1
			}
		case s[i] == '`':
			if j := strings.IndexByte(s[i+1:], '`'); j >= 0 {
				subs = append(subs, s[i+1:i+1+j])
				i += j + 1
			}
		}
	}
	return subs
}

type hookToken struct {
	s  string
	op bool // a shell operator: && || ; | & newline ( )
}

// hookTokenize splits a command the way a shell would: quotes, backslashes and
// operators, no expansion. A heredoc body is tokenized like any text: a line
// of it that starts with kubectl is checked too (a false ask, never a miss).
func hookTokenize(s string) []hookToken {
	var toks []hookToken
	var cur strings.Builder
	inWord := false
	flush := func() {
		if inWord {
			toks = append(toks, hookToken{s: cur.String()})
			cur.Reset()
			inWord = false
		}
	}
	op := func(o string) {
		flush()
		toks = append(toks, hookToken{s: o, op: true})
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s):
			i++
			if s[i] != '\n' { // backslash-newline continues the line
				cur.WriteByte(s[i])
				inWord = true
			}
		case c == '\'':
			inWord = true
			if j := strings.IndexByte(s[i+1:], '\''); j < 0 {
				cur.WriteString(s[i+1:])
				i = len(s)
			} else {
				cur.WriteString(s[i+1 : i+1+j])
				i += j + 1
			}
		case c == '"':
			inWord = true
			for i++; i < len(s) && s[i] != '"'; i++ {
				if s[i] == '\\' && i+1 < len(s) && strings.IndexByte("\"\\$`", s[i+1]) >= 0 {
					i++
				}
				cur.WriteByte(s[i])
			}
		case c == ' ' || c == '\t':
			flush()
		case c == '\n' || c == ';' || c == '(' || c == ')':
			op(string(c))
		case c == '&' || c == '|':
			if i+1 < len(s) && s[i+1] == c {
				op(string(c) + string(c))
				i++
			} else {
				op(string(c))
			}
		default:
			cur.WriteByte(c)
			inWord = true
		}
	}
	flush()
	return toks
}

// hookSegments: the commands of a line, split at the operators
func hookSegments(toks []hookToken) [][]string {
	var segs [][]string
	var cur []string
	for _, t := range toks {
		if t.op {
			if len(cur) > 0 {
				segs = append(segs, cur)
				cur = nil
			}
			continue
		}
		cur = append(cur, t.s)
	}
	if len(cur) > 0 {
		segs = append(segs, cur)
	}
	return segs
}

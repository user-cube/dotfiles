// write-guard: Claude Code PreToolUse hook (matchers: Edit, Write, MultiEdit,
// NotebookEdit, Bash).
//
// deny-sensitive-reads stops Claude reading what is private; this stops it
// changing what keeps it safe. Denied, for a write (an edit, a redirection,
// tee, sed -i, rm, mv, cp ... onto it):
//
//	~/.ssh, ~/.aws                  keys, credentials, client config
//	.env files, *.pem, *.key, age   (.env.example and other templates are fine)
//	.git/hooks, .git/config         core.hooksPath there turns every git hook off
//	.claude/settings*.json, hooks/  the guardrails themselves (~/.claude and any project's)
//	~/.gitconfig*, ~/.config/git/, ~/git_hooks/   the global git hooks and their switches
//	~/.config/kube-guard/prod-regex   what kube-guard calls production
//
// Reading any of them is not this hook's business. The dotfiles clone's own
// claude/ directory is not ~/.claude: Claude edits the source there, and a
// sync (the user's step) is what puts it in place.
//
// LIMIT: for Bash the paths are taken from the command text. A script or an
// interpreter (python -c, node -e) that writes a path built at runtime is not
// seen. A safety net, not a sandbox.
//
// Reads the hook event (JSON) on stdin; prints a deny decision, or nothing to
// allow. Always exits 0: input it can't parse is allowed, like any other tool.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const rule = "local rule in ~/.claude/hooks/write-guard"

type event struct {
	ToolName  string `json:"tool_name"`
	Cwd       string `json:"cwd"`
	ToolInput struct {
		Command      string `json:"command"`
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
	} `json:"tool_input"`
}

type hook struct{ home, cwd string }

func main() {
	home, _ := os.UserHomeDir()
	run(os.Stdin, os.Stdout, home)
}

func run(in io.Reader, out io.Writer, home string) {
	data, _ := io.ReadAll(in)
	var ev event
	if json.Unmarshal(data, &ev) != nil {
		return
	}
	h := hook{home: home, cwd: ev.Cwd}
	if reason := h.check(ev); reason != "" {
		resp, _ := json.Marshal(map[string]any{"hookSpecificOutput": map[string]string{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": reason,
		}})
		fmt.Fprintln(out, string(resp))
	}
}

// check returns why the tool call is denied, or "" to allow it.
func (h hook) check(ev event) string {
	switch ev.ToolName {
	case "Edit", "Write", "MultiEdit":
		return h.checkPath(ev.ToolInput.FilePath)
	case "NotebookEdit":
		return h.checkPath(ev.ToolInput.NotebookPath)
	case "Bash":
		return h.checkCommand(ev.ToolInput.Command, 0)
	}
	return ""
}

func (h hook) checkPath(path string) string {
	if path == "" {
		return ""
	}
	if why := h.protected(path); why != "" {
		return fmt.Sprintf("%s is protected (%s; %s). Ask the user to change it themselves.", path, why, rule)
	}
	return ""
}

// protected says why a path must not be written, or "" when it may be
func (h hook) protected(path string) string {
	p := strings.ToLower(filepath.Clean(h.abs(h.expand(path))))
	home := strings.ToLower(filepath.Clean(h.home))
	segs := strings.Split(p, "/")
	base := segs[len(segs)-1]
	under := func(dir string) bool { return p == dir || strings.HasPrefix(p, dir+"/") }

	if slices.Contains(segs, ".ssh") {
		return "ssh keys and client config"
	}
	if slices.Contains(segs, ".aws") {
		return "AWS credentials and config"
	}
	// .git/hooks/..., .git/config: a hooksPath or a hook there ends every guard
	if i := slices.Index(segs, ".git"); i >= 0 && i+1 < len(segs) &&
		(segs[i+1] == "hooks" || (segs[i+1] == "config" && i+2 == len(segs))) {
		return "git hooks and config: they can turn the hooks off"
	}
	// the guardrails: ~/.claude and any project's .claude
	if i := slices.Index(segs, ".claude"); i >= 0 && i+1 < len(segs) &&
		(segs[i+1] == "hooks" || segs[i+1] == "settings.json" || segs[i+1] == "settings.local.json") {
		return "Claude Code settings and hooks: the guardrails themselves"
	}
	if home != "" && home != "." {
		switch {
		case filepath.Dir(p) == home && strings.HasPrefix(base, ".gitconfig"):
			return "the global git config: it points at the hooks"
		case under(home + "/git_hooks"), under(home + "/.config/git"):
			return "the global git hooks and their config"
		case p == home+"/.config/kube-guard/prod-regex":
			return "what kube-guard calls production"
		}
	}
	switch {
	case isEnvFile(base):
		return "an env file"
	case strings.HasSuffix(base, ".pem"), strings.HasSuffix(base, ".key"):
		return "a key or certificate file"
	case sshKeyName.MatchString(base):
		return "an ssh private key"
	case ageKeyName.MatchString(base), strings.HasSuffix(p, "sops/age/keys.txt"):
		return "an age key"
	}
	return ""
}

var (
	sshKeyName = regexp.MustCompile(`^id_(rsa|dsa|ecdsa|ed25519)(_sk)?$`)
	ageKeyName = regexp.MustCompile(`^age-key[^/]*\.txt$|\.agekey$`)
	envAssign  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	// a redirection's target: > f, >> f, >| f, >"f" (not 2>&1, &>)
	redirect = regexp.MustCompile(`>[>|]?[[:space:]]*["']?([^[:space:];&|<>()'"]+)`)
)

// Suffixes of env files that are templates, not secrets: .env.example
var envTemplate = map[string]bool{"example": true, "sample": true, "template": true, "dist": true}

// .env, .envrc, .env.local, .env-prod: env files; .env.example and other templates are not
func isEnvFile(base string) bool {
	rest, ok := strings.CutPrefix(base, ".env")
	switch {
	case !ok:
		return false
	case rest == "" || rest == "rc":
		return true
	case strings.ContainsAny(rest[:1], ".-_"):
		// .env.example, .env.example.bak, .env.local.example: a template part anywhere
		for _, part := range strings.FieldsFunc(rest, func(r rune) bool { return r == '.' || r == '-' || r == '_' }) {
			if envTemplate[part] {
				return false
			}
		}
		return true
	}
	return false // .environment
}

// expand turns ~, $HOME and ${HOME} at the start of a path into the home directory
func (h hook) expand(p string) string {
	for _, pre := range []string{"~", "${HOME}", "$HOME"} {
		if p == pre || strings.HasPrefix(p, pre+"/") {
			return h.home + p[len(pre):]
		}
	}
	return p
}

func (h hook) abs(p string) string {
	if filepath.IsAbs(p) || h.cwd == "" {
		return p
	}
	return filepath.Join(h.cwd, p)
}

// How deep a command inside a command (sh -c, $(...)) is followed
const maxDepth = 4

// Programs that run what they are given: stripped, so `sudo rm x` is an rm
var wrappers = []string{"rtk", "command", "env", "sudo", "nohup", "time", "exec", "xargs"}

var shells = []string{"sh", "bash", "zsh", "dash", "ksh"}

// Programs that change every path they are given
var writesAll = set("tee rm mv unlink shred truncate touch mkdir rmdir chmod chown chgrp")

// Programs that change only their destination: the last argument, or -t DIR
var writesLast = set("cp rsync install ln")

func set(words string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(words) {
		m[w] = true
	}
	return m
}

func (h hook) checkCommand(cmd string, depth int) string {
	if cmd == "" || depth > maxDepth {
		return ""
	}
	for _, m := range redirect.FindAllStringSubmatch(cmd, -1) {
		if why := h.protected(m[1]); why != "" {
			return fmt.Sprintf("Command redirects into %s, which is protected (%s; %s). Ask the user to change it themselves.", m[1], why, rule)
		}
	}
	for _, s := range substitutions(cmd) {
		if r := h.checkCommand(s, depth+1); r != "" {
			return r
		}
	}
	for _, seg := range segments(tokenize(cmd)) {
		args := stripPrefix(seg)
		if len(args) == 0 {
			continue
		}
		prog := filepath.Base(args[0])
		switch {
		case prog == "eval":
			if r := h.checkCommand(strings.Join(args[1:], " "), depth+1); r != "" {
				return r
			}
		case slices.Contains(shells, prog):
			// bash -c "...", also bash -lc "..."
			for i, a := range args[1:] {
				if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.ContainsRune(a, 'c') {
					if i+2 < len(args) {
						if r := h.checkCommand(args[i+2], depth+1); r != "" {
							return r
						}
					}
					break
				}
			}
		default:
			for _, target := range h.written(prog, args[1:]) {
				if why := h.protected(target); why != "" {
					return fmt.Sprintf("Command writes to %s with %s, which is protected (%s; %s). Ask the user to change it themselves.", target, prog, why, rule)
				}
			}
		}
	}
	return ""
}

// written: the paths prog changes, from its arguments
func (h hook) written(prog string, args []string) []string {
	var paths []string // every non-flag argument
	var target string  // -t DIR / --target-directory=DIR
	inPlace := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-t" || a == "--target-directory":
			if i+1 < len(args) {
				target = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--target-directory="):
			target = strings.TrimPrefix(a, "--target-directory=")
		case strings.HasPrefix(a, "of="): // dd of=file
			paths = append(paths, strings.TrimPrefix(a, "of="))
		case a == "--in-place" || strings.HasPrefix(a, "--in-place="):
			inPlace = true
		case strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.ContainsRune(a, 'i'):
			inPlace = true
		case !strings.HasPrefix(a, "-"):
			paths = append(paths, a)
		}
	}
	switch {
	case prog == "dd":
		return paths
	case (prog == "sed" || prog == "perl" || prog == "ruby") && inPlace:
		return paths
	case writesAll[prog]:
		return append(paths, target)
	case writesLast[prog]:
		if target != "" {
			return []string{target}
		}
		if len(paths) > 0 {
			return paths[len(paths)-1:]
		}
	}
	return nil
}

// substitutions: the commands inside $(...) and `...`, not $((arithmetic))
func substitutions(s string) []string {
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

// stripPrefix drops what comes before the program: VAR=x, rtk, sudo, env ...
func stripPrefix(seg []string) []string {
	wrapped := false
	for len(seg) > 0 {
		switch a := seg[0]; {
		case envAssign.MatchString(a):
		case slices.Contains(wrappers, a):
			wrapped = true
		case wrapped && strings.HasPrefix(a, "-"): // env -i, sudo -E
		default:
			return seg
		}
		seg = seg[1:]
	}
	return seg
}

type token struct {
	s  string
	op bool // a shell operator: && || ; | & newline ( )
}

// tokenize splits a command the way a shell would: quotes, backslashes and
// operators, no expansion. A heredoc body is tokenized like any text, which
// can only make a false deny, never a miss.
func tokenize(s string) []token {
	var toks []token
	var cur strings.Builder
	inWord := false
	flush := func() {
		if inWord {
			toks = append(toks, token{s: cur.String()})
			cur.Reset()
			inWord = false
		}
	}
	op := func(o string) {
		flush()
		toks = append(toks, token{s: o, op: true})
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

// segments: the commands of a line, split at the operators
func segments(toks []token) [][]string {
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

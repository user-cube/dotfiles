// git-guard: Claude Code PreToolUse hook (matcher: Bash).
//
// Checks the commits and pull requests Claude writes, before they exist:
//
//	git commit      no attribution (Co-Authored-By, "Generated with Claude"),
//	                no --no-verify / -n (it skips commit-msg and sops-guard),
//	                a header that is short, English and has no final full stop
//	gh pr create    the same attribution rule and the same title rule; a title
//	gh pr edit      and a body are required (create), or gh stops to ask
//	git push        no force-push to, and no deleting of, main/master/develop
//
// The user is asked (not denied) for: gh pr merge, git reset --hard and
// git clean -f.
//
// The header rules apply to what the message says: the commit-msg git hook
// still adds the type and the issue key afterwards, so the header is checked
// as written, with room to spare (maxHeader).
//
// LIMIT: the message is read from the command text (-m, --message, a heredoc,
// -F - with a heredoc). A message written to a file (-F file, --body-file) or
// built at runtime ($(...), variables) is not seen; the git hook is the
// check that always runs. A safety net, not a sandbox.
//
// Reads the hook event (JSON) on stdin; prints a deny decision, or nothing to
// allow. Always exits 0: input it can't parse is allowed, like any other tool.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The longest header or title as written: the type and the issue key that
// commit-msg adds come on top (feat: ABC-123 ), as does git's own 72 limit
const maxHeader = 100

const rule = "local rule in ~/.claude/hooks/git-guard"

var (
	// Any Co-Authored-By trailer (the user does not want one), and the usual
	// ways a tool signs its work
	attribution = regexp.MustCompile(`(?i)co-authored-by:|generated (with|by) .{0,40}claude|noreply@anthropic\.com|🤖 generated`)
	// Letters with a diacritic: the sign of a message that is not in English
	accented = regexp.MustCompile(`[À-ÖØ-öø-ÿ]`)
	// A heredoc taken out of the command: <<@heredoc-N
	heredocRef = regexp.MustCompile(`<<@heredoc-(\d+)`)
	envAssign  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
)

type event struct {
	Cwd       string `json:"cwd"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

func main() {
	run(os.Stdin, os.Stdout)
}

func run(in io.Reader, out io.Writer) {
	data, _ := io.ReadAll(in)
	var ev event
	if json.Unmarshal(data, &ev) != nil {
		return
	}
	if reason, ask := check(ev.ToolInput.Command, ev.Cwd); reason != "" {
		decision := "deny"
		if ask {
			decision = "ask"
		}
		resp, _ := json.Marshal(map[string]any{"hookSpecificOutput": map[string]string{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       decision,
			"permissionDecisionReason": reason,
		}})
		fmt.Fprintln(out, string(resp))
	}
}

// The branches nobody force-pushes to or deletes
var protected = map[string]bool{"main": true, "master": true, "develop": true}

// currentBranch is the branch checked out in dir; a variable so the tests can
// stand in for git
var currentBranch = func(dir string) string {
	args := []string{"symbolic-ref", "--short", "HEAD"}
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}
	out, _ := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(out))
}

// How deep a command inside a command (sh -c, $(...), an alias) is followed
const maxDepth = 4

// The subcommands whose hooks --no-verify skips
var hookSubs = []string{"commit", "push", "merge", "rebase", "am"}

// Programs that run what they are given: stripped, so `sudo git commit` is a commit
var wrappers = []string{"rtk", "command", "env", "sudo", "nohup", "time", "exec"}

var shells = []string{"sh", "bash", "zsh", "dash", "ksh"}

// GIT_CONFIG_KEY_0=core.hooksPath, GIT_CONFIG_PARAMETERS='hook.x.enabled'=...
var envHooks = regexp.MustCompile(`(?i)^GIT_CONFIG\w*=.*(hookspath|hook\.)`)

// aliasOf is the git alias called name in dir, "" when there is none; a
// variable so the tests can stand in for git
var aliasOf = func(dir, name string) string {
	args := []string{"config", "--get", "alias." + name}
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}
	out, _ := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(out))
}

// check returns why the command is stopped, or "" to let it through. ask says
// the user decides (the reason is shown to them) instead of a flat deny; a
// deny in any part of the command wins over an ask.
func check(cmd, cwd string) (reason string, ask bool) {
	return checkDepth(cmd, cwd, 0)
}

func checkDepth(cmd, cwd string, depth int) (reason string, ask bool) {
	if cmd == "" || depth > maxDepth {
		return "", false
	}
	// note records a result; true means a deny, which ends the search
	note := func(r string, a bool) bool {
		switch {
		case r == "":
		case !a:
			reason, ask = r, false
			return true
		case reason == "":
			reason, ask = r, true
		}
		return false
	}
	// what the command runs inside: sh -c "...", eval, $(...), `...`, an alias
	inner := func(s string) bool { return note(checkDepth(s, cwd, depth+1)) }

	body, docs := extractHeredocs(cmd)
	for _, s := range substitutions(body) {
		if inner(s) {
			return reason, ask
		}
	}
	for _, seg := range segments(tokenize(body)) {
		for _, tok := range seg {
			if envHooks.MatchString(tok) {
				return "A GIT_CONFIG_* variable that changes core.hooksPath or hook.* turns the git hooks off (" + rule + ").", false
			}
		}
		args := stripPrefix(seg)
		if len(args) == 0 {
			continue
		}
		var deny, question string
		switch prog := filepath.Base(args[0]); {
		case prog == "git":
			deny, question = checkGit(args[1:], cwd, docs, cmd, func(s string) (string, bool) { return checkDepth(s, cwd, depth+1) })
		case prog == "gh":
			if len(args) >= 3 && args[1] == "pr" {
				switch args[2] {
				case "create", "edit":
					deny = checkPR(cmd, args[2], args[3:], docs)
				case "merge":
					question = "gh pr merge changes the base branch for everyone; it is the user's call (" + rule + ")."
				}
			}
		case prog == "eval":
			if inner(strings.Join(args[1:], " ")) {
				return reason, ask
			}
		case slices.Contains(shells, prog):
			// bash -c "...", also bash -lc "..."
			for i, a := range args[1:] {
				if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.ContainsRune(a, 'c') {
					if i+2 < len(args) && inner(args[i+2]) {
						return reason, ask
					}
					break
				}
			}
		}
		if deny != "" {
			return deny, false
		}
		note(question, true)
	}
	return reason, ask
}

// checkGit: one `git ...` command (args come after the word git)
func checkGit(args []string, cwd string, docs []string, cmd string, run func(string) (string, bool)) (deny, question string) {
	cfg, sub, rest := gitParse(args)
	for _, kv := range cfg {
		if why := hookConfig(kv); why != "" {
			return "git -c " + kv + ": " + why + " (" + rule + ").", ""
		}
	}
	// An alias can hide a commit or a --no-verify: follow it (git does not let
	// an alias take the name of a command it has)
	for n := 0; sub != "" && n < maxDepth && !slices.Contains(handled, sub); n++ {
		alias := aliasOf(cwd, sub)
		if alias == "" {
			break
		}
		if shell, ok := strings.CutPrefix(alias, "!"); ok {
			quoted := make([]string, len(rest))
			for i, r := range rest {
				quoted[i] = "'" + strings.ReplaceAll(r, "'", `'\''`) + "'"
			}
			r, isAsk := run(shell + " " + strings.Join(quoted, " "))
			if isAsk {
				return "", r
			}
			return r, ""
		}
		f := strings.Fields(alias)
		if len(f) == 0 {
			break
		}
		_, sub, rest = gitParse(append(f, rest...))
	}
	if slices.Contains(hookSubs, sub) {
		options, _, _ := strings.Cut(strings.Join(rest, "\x00"), "\x00--\x00") // up to a bare --
		if slices.ContainsFunc(strings.Split(options, "\x00"), isNoVerify) {
			return noVerify(sub), ""
		}
	}
	switch sub {
	case "commit":
		deny = checkCommit(cmd, rest, docs)
	case "push":
		deny = checkPush(rest, cwd)
	case "config":
		deny = checkConfig(rest)
	case "reset":
		if slices.Contains(rest, "--hard") {
			question = "git reset --hard throws away uncommitted work and can't be undone (" + rule + ")."
		}
	case "clean":
		if cleans(rest) {
			question = "git clean deletes untracked files for good (" + rule + ")."
		}
	}
	return deny, question
}

// The subcommands that are not looked up as aliases
var handled = []string{"commit", "push", "merge", "rebase", "am", "reset", "clean", "config"}

// isNoVerify: --no-verify, or the shortest start of it git still takes
// (--no-v ... --no-verif): git accepts any unambiguous beginning of an option
func isNoVerify(a string) bool {
	return len(a) >= len("--no-v") && strings.HasPrefix("--no-verify", a)
}

// hookConfig says why a config key given to git must not be set, or ""
func hookConfig(kv string) string {
	key := strings.ToLower(strings.SplitN(kv, "=", 2)[0])
	switch {
	case key == "core.hookspath" || strings.HasPrefix(key, "hook."):
		return "changes which git hooks run"
	case strings.HasPrefix(key, "alias."):
		return "an alias set on the command line can hide what runs"
	}
	return ""
}

// checkConfig: `git config` that changes core.hooksPath or hook.*; reading them is fine
func checkConfig(args []string) string {
	var pos []string
	write, read := false, false
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--unset", "--unset-all", "--add", "--replace-all", "--edit", "-e", "--rename-section", "--remove-section":
			write = true
		case "--get", "--get-all", "--get-regexp", "--list", "-l", "--get-urlmatch":
			read = true
		case "-f", "--file", "--blob", "--type", "--default":
			i++ // takes a value
		default:
			if !strings.HasPrefix(a, "-") {
				pos = append(pos, a)
			}
		}
	}
	if len(pos) > 0 { // git config set|unset|get|list ...
		switch pos[0] {
		case "set", "unset", "edit", "rename-section", "remove-section":
			write, pos = true, pos[1:]
		case "get", "list":
			read, pos = true, pos[1:]
		}
	}
	if !read && len(pos) >= 2 {
		write = true // git config key value
	}
	if !write || len(pos) == 0 {
		return ""
	}
	key := strings.ToLower(pos[0])
	if key == "core.hookspath" || strings.HasPrefix(key, "hook.") {
		return "git config " + pos[0] + " changes which git hooks run (" + rule + "). Ask the user to do it themselves."
	}
	return ""
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

// cleans: git clean that really deletes (-f / --force, and not a dry run)
func cleans(args []string) bool {
	force := false
	for _, a := range args {
		switch {
		case a == "--dry-run":
			return false
		case a == "--force":
			force = true
		case strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--"):
			if strings.ContainsRune(a, 'n') {
				return false
			}
			force = force || strings.ContainsRune(a, 'f')
		}
	}
	return force
}

// checkPush: no force-push to, and no deleting of, a protected branch
func checkPush(args []string, cwd string) string {
	force, del := false, false
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			pos = append(pos, args[i+1:]...)
			i = len(args)
		case a == "--force" || a == "--force-with-lease" || a == "--force-if-includes",
			strings.HasPrefix(a, "--force-with-lease="), strings.HasPrefix(a, "--force-if-includes="):
			force = true
		case a == "--delete":
			del = true
		case a == "--repo" || a == "--push-option" || a == "--receive-pack" || a == "--exec":
			i++ // takes a value
		case strings.HasPrefix(a, "--"):
		case strings.HasPrefix(a, "-") && len(a) > 1:
			force = force || strings.ContainsRune(a, 'f')
			del = del || strings.ContainsRune(a, 'd')
			if a == "-o" {
				i++
			}
		default:
			pos = append(pos, a)
		}
	}
	var refspecs []string
	if len(pos) > 1 { // the first is the remote
		refspecs = pos[1:]
	}
	stop := func(branch string) string {
		return "Force-pushing to or deleting " + branch + " is blocked (" + rule + "). Work on a branch and open a pull request."
	}
	for _, rs := range refspecs {
		plus := strings.HasPrefix(rs, "+")
		dst, deleting := strings.TrimPrefix(rs, "+"), false
		if i := strings.LastIndex(dst, ":"); i >= 0 {
			deleting = i == 0 // :branch deletes it
			dst = dst[i+1:]
		}
		dst = strings.TrimPrefix(dst, "refs/heads/")
		if dst == "HEAD" {
			dst = currentBranch(cwd)
		}
		if protected[dst] && (force || plus || del || deleting) {
			return stop(dst)
		}
	}
	if force && len(refspecs) == 0 { // no branch named: the current one
		if b := currentBranch(cwd); protected[b] {
			return stop(b)
		}
	}
	return ""
}

func checkCommit(cmd string, args []string, docs []string) string {
	if attribution.MatchString(cmd) {
		return "The commit carries an attribution (Co-Authored-By or 'Generated with Claude'); the user does not want one (" + rule + "). Remove it and commit again."
	}
	var msgs []string
	stdin := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() string {
			if i+1 < len(args) {
				i++
				return args[i]
			}
			return ""
		}
		switch {
		case a == "--":
			i = len(args)
		case a == "--message":
			msgs = append(msgs, next())
		case strings.HasPrefix(a, "--message="):
			msgs = append(msgs, strings.TrimPrefix(a, "--message="))
		case a == "--file":
			stdin = stdin || next() == "-"
		case a == "--file=-":
			stdin = true
		case strings.HasPrefix(a, "--"):
			// another long option: nothing to check
		case strings.HasPrefix(a, "-") && len(a) > 1:
			// short options, maybe together (-nm "msg", -am"msg"): letters up
			// to the first one that takes a value, which is the rest of the
			// word or the next one
		cluster:
			for j := 1; j < len(a); j++ {
				switch c := a[j]; c {
				case 'n':
					return noVerify("commit")
				case 'm', 'F', 'C', 'c', 't':
					v := a[j+1:]
					if v == "" {
						v = next()
					}
					if c == 'm' {
						msgs = append(msgs, v)
					} else if c == 'F' && v == "-" {
						stdin = true
					}
					break cluster
				}
			}
		}
	}
	if len(msgs) == 0 && stdin {
		// git commit -F - <<'EOF' ... EOF: the message is the heredoc
		for _, a := range args {
			if m := heredocRef.FindString(a); m != "" {
				msgs = append(msgs, resolve(a, docs))
			}
		}
	}
	if len(msgs) == 0 {
		return ""
	}
	return checkTitle("commit header", firstLine(resolve(msgs[0], docs)))
}

func noVerify(sub string) string {
	return "git " + sub + " --no-verify skips the git hooks (commit-msg, sops-guard, ...) (" + rule + "). Run it without; if a hook blocks, fix what it says."
}

func checkPR(cmd, sub string, args []string, docs []string) string {
	if attribution.MatchString(cmd) {
		return "The pull request carries an attribution (Co-Authored-By or 'Generated with Claude'); the user does not want one (" + rule + "). Remove it and try again."
	}
	title, hasTitle := flagValue(args, "--title", "-t")
	body, hasBody := flagValue(args, "--body", "-b")
	_, hasBodyFile := flagValue(args, "--body-file", "-F")
	fill := hasFlag(args, "--fill", "--fill-first", "--fill-verbose", "--web", "-w")

	// Without these gh asks questions, and nobody is there to answer
	if sub == "create" {
		switch {
		case !hasTitle && !fill:
			return "gh pr create needs --title (or --fill): without it gh stops to ask (" + rule + ")."
		case !hasBody && !hasBodyFile && !fill:
			return "gh pr create needs --body or --body-file (or --fill): a pull request without a description is not opened (" + rule + ")."
		}
	}
	if hasBody && strings.TrimSpace(resolve(body, docs)) == "" {
		return "The pull request body is empty (" + rule + "). Say what changed and why."
	}
	if hasTitle {
		return checkTitle("pull request title", firstLine(resolve(title, docs)))
	}
	return ""
}

// flagValue: the value of the first of names in args (--title x, --title=x, -t x)
func flagValue(args []string, names ...string) (string, bool) {
	for i, a := range args {
		for _, n := range names {
			if a == n && i+1 < len(args) {
				return args[i+1], true
			}
			if strings.HasPrefix(n, "--") && strings.HasPrefix(a, n+"=") {
				return strings.TrimPrefix(a, n+"="), true
			}
		}
	}
	return "", false
}

func hasFlag(args []string, names ...string) bool {
	for _, a := range args {
		if slices.Contains(names, a) {
			return true
		}
	}
	return false
}

// checkTitle: a commit header or PR title, as written
func checkTitle(what, title string) string {
	switch n := utf8.RuneCountInString(title); {
	case n > maxHeader:
		return "The " + what + " is " + strconv.Itoa(n) + " characters; keep it under " + strconv.Itoa(maxHeader) + " and put the detail in the body (" + rule + ")."
	case strings.HasSuffix(title, "."):
		return "The " + what + " ends with a full stop; drop it (" + rule + ")."
	case accented.MatchString(title):
		return "The " + what + " is not in English (letters with accents); commits and PRs are written in English (" + rule + ")."
	}
	return ""
}

// resolve: the heredoc a value points at ($(cat <<'EOF' ... EOF)), else the value
func resolve(v string, docs []string) string {
	if m := heredocRef.FindStringSubmatch(v); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n < len(docs) {
			return docs[n]
		}
	}
	return v
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

// gitParse splits `git [global options] sub ...`: the config set with -c /
// --config-env (key=value), the subcommand and what follows it
func gitParse(args []string) (cfg []string, sub string, rest []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			return cfg, a, args[i+1:]
		}
		switch {
		case (a == "-c" || a == "--config-env") && i+1 < len(args):
			cfg = append(cfg, args[i+1])
			i++
		case strings.HasPrefix(a, "--config-env="):
			cfg = append(cfg, strings.TrimPrefix(a, "--config-env="))
		case a == "-C" || a == "--git-dir" || a == "--work-tree" || a == "--namespace" || a == "--exec-path":
			i++ // takes a value
		}
	}
	return cfg, "", nil
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

// extractHeredocs takes the bodies out of a command, in order: each marker
// becomes <<@heredoc-N, so a message in `$(cat <<'EOF' ... EOF)` can't hide a
// `;` or a `git commit` from the tokenizer, and stays reachable through docs.
func extractHeredocs(s string) (string, []string) {
	var docs []string
	var out strings.Builder
	for {
		i := strings.Index(s, "<<")
		if i < 0 {
			break
		}
		j := i + 2
		if j < len(s) && s[j] == '<' { // <<< here-string
			out.WriteString(s[:j+1])
			s = s[j+1:]
			continue
		}
		if j < len(s) && s[j] == '-' {
			j++
		}
		for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
			j++
		}
		quote := byte(0)
		if j < len(s) && (s[j] == '\'' || s[j] == '"') {
			quote = s[j]
			j++
		}
		k := j
		for k < len(s) && (s[k] == '_' || s[k] >= '0' && s[k] <= '9' || s[k] >= 'a' && s[k] <= 'z' || s[k] >= 'A' && s[k] <= 'Z') {
			k++
		}
		word := s[j:k]
		if quote != 0 && k < len(s) && s[k] == quote {
			k++
		}
		nl := strings.IndexByte(s[k:], '\n')
		if word == "" || nl < 0 { // not a heredoc, or no body follows
			out.WriteString(s[:i+2])
			s = s[i+2:]
			continue
		}
		// the body runs to the line that is the word alone
		bodyStart := k + nl + 1
		pos, end, found := bodyStart, len(s), false
		for pos <= len(s) {
			e := strings.IndexByte(s[pos:], '\n')
			if e < 0 {
				e = len(s) - pos
			}
			if strings.TrimSpace(s[pos:pos+e]) == word {
				end, found = pos+e, true
				break
			}
			pos += e + 1
		}
		body := s[bodyStart:min(pos, len(s))]
		if !found {
			body = s[bodyStart:]
		}
		docs = append(docs, strings.TrimSuffix(body, "\n"))
		out.WriteString(s[:i])
		out.WriteString("<<@heredoc-")
		out.WriteString(strconv.Itoa(len(docs) - 1))
		out.WriteString(s[k : k+nl]) // the rest of the marker's line
		s = s[end:]
		if !found {
			s = ""
		}
	}
	out.WriteString(s)
	return out.String(), docs
}

type token struct {
	s  string
	op bool // a shell operator: && || ; | & newline ( )
}

// tokenize splits a command the way a shell would: quotes, backslashes and
// operators, no expansion
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

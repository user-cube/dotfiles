// deny-sensitive-reads: Claude Code PreToolUse hook (matchers: Bash, Read).
//
// Bash: the Read(...) deny rules in settings.json do not cover Bash: a plain
// `cat file` still goes through. This hook takes the paths a command names
// and blocks it when one of them is private (see private): only what really
// is (keys, credentials, env files), so ~/.ssh/config, ~/.aws/config,
// .env.example or a docker-compose file stay readable. Read: the same paths.
//
// secrets/ and credentials/ directories: files there may be read only when
// they ARE encrypted, judged by content with `sops filestatus` (which also
// rejects a hand-written fake `sops:` block), not by file name. They are
// guarded here rather than with Read(...) deny rules because a permission
// rule can't say "except encrypted files".
//
// LIMIT: for Bash the paths are taken from the command text. Paths built at
// runtime (variables, cd then a relative path, base64) are not resolved; when
// a secrets/ path can't be resolved to a file, the name rule applies
// (.enc. / .encrypted. / .sops. in the command). A safety net, not a sandbox.
//
// Reads the hook event (JSON) on stdin; prints a deny decision, or nothing to
// allow. Always exits 0: input it can't parse is allowed, like any other tool.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type event struct {
	ToolName  string `json:"tool_name"`
	Cwd       string `json:"cwd"`
	ToolInput struct {
		Command  string `json:"command"`
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

// (?m): ^ and $ match per line, as in grep, which the shell version used
var (
	secretDirs    = regexp.MustCompile(`(?im)(^|[^a-z0-9_-])(secrets|credentials)/`)
	encryptedName = regexp.MustCompile(`\.(enc|encrypted|sops)\.(ya?ml|json|hcl|env|ini)`)

	// The words of a command that can be paths: split at blanks, quotes and
	// backquotes, redirections, `=` (--file=x) and `:` (scp host:path)
	pathToken = regexp.MustCompile("[^[:space:]\"'`<>|;&()=:,]+")
	// An age secret key pasted or grepped for in the command itself
	ageSecret = regexp.MustCompile(`AGE-SECRET-KEY`)
	// ssh private keys by their default names, wherever they are (cd ~/.ssh && cat id_rsa)
	sshKeyName = regexp.MustCompile(`^id_(rsa|dsa|ecdsa|ed25519)(_sk)?$`)
	ageKeyName = regexp.MustCompile(`^age-key[^/]*\.txt$|\.agekey$`)

	// Paths in a command that go through secrets/ or credentials/ ...
	secretPath = regexp.MustCompile(`(?i)[^[:space:]"'=<>|;&()]*(secrets|credentials)/[^[:space:]"'<>|;&()]*`)
	// ... as a whole path segment: not external-secrets/
	secretSegment = regexp.MustCompile(`(?im)(^|[/~])(secrets|credentials)/`)

	sopsMeta = regexp.MustCompile(`(?m)^sops:|"sops"[[:space:]]*:`)
)

// hook decides; encrypted is sopsEncrypted except in the tests.
type hook struct {
	home, cwd string
	encrypted func(path string) bool
}

func main() {
	home, _ := os.UserHomeDir()
	run(os.Stdin, os.Stdout, home, sopsEncrypted)
}

func run(in io.Reader, out io.Writer, home string, encrypted func(string) bool) {
	data, _ := io.ReadAll(in)
	var ev event
	if json.Unmarshal(data, &ev) != nil {
		return
	}
	h := hook{home: home, cwd: ev.Cwd, encrypted: encrypted}
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
	if ev.ToolName == "Read" {
		return h.checkRead(ev.ToolInput.FilePath)
	}
	return h.checkCommand(ev.ToolInput.Command)
}

func (h hook) checkRead(path string) string {
	if path == "" {
		return ""
	}
	if why := h.private(path); why != "" {
		return fmt.Sprintf("%s is private (%s; local rule in ~/.claude/hooks/deny-sensitive-reads). Ask the user to do it themselves.", path, why)
	}
	if !secretDirs.MatchString(path) {
		return ""
	}
	// Nothing to leak from a missing file: let Read report "not found"
	if _, err := os.Stat(h.abs(path)); err != nil {
		return ""
	}
	if !h.isEncryptedFile(h.abs(path)) {
		return "File under secrets/ or credentials/ is not SOPS-encrypted (sops filestatus; local rule in ~/.claude/hooks/deny-sensitive-reads)."
	}
	return ""
}

func (h hook) checkCommand(cmd string) string {
	if cmd == "" {
		return ""
	}
	if ageSecret.MatchString(cmd) {
		return "Command holds or looks for an age secret key (local rule in ~/.claude/hooks/deny-sensitive-reads). Ask the user to do it themselves."
	}
	for _, token := range pathToken.FindAllString(cmd, -1) {
		if why := h.private(token); why != "" {
			return fmt.Sprintf("Command touches %s, which is private (%s; local rule in ~/.claude/hooks/deny-sensitive-reads). Ask the user to do it themselves.", token, why)
		}
	}
	// secrets/ and credentials/: every path the command names must resolve to
	// files that are all encrypted; a path that resolves to no file falls back
	// to the name rule.
	for _, token := range secretPath.FindAllString(cmd, -1) {
		if !secretSegment.MatchString(token) {
			continue
		}
		files := h.files(token)
		for _, f := range files {
			if !h.isEncryptedFile(f) {
				return fmt.Sprintf("Command touches %s, which is not SOPS-encrypted (sops filestatus; local rule).", h.shown(token, f))
			}
		}
		if len(files) == 0 && !encryptedName.MatchString(token) {
			return fmt.Sprintf("Command touches secrets/ or credentials/ (%s) and it can't be checked as an encrypted file (local rule).", token)
		}
	}
	return ""
}

// The files in ~/.ssh that hold no secret: client config, host keys, public
// keys, ControlMaster sockets (cm-%C, .config/ssh/controlmaster.conf)
var sshPublic = map[string]bool{"config": true, "known_hosts": true, "known_hosts.old": true, "authorized_keys": true}

// Suffixes of env files that are templates, not secrets: .env.example
var envTemplate = map[string]bool{"example": true, "sample": true, "template": true, "dist": true}

// private says why a path is private, or "" when it is not. Only what holds
// secrets: a whole ~/.ssh or ~/.aws (cp -r, tar) or a glob in them counts,
// since it takes the keys along.
func (h hook) private(path string) string {
	// the macOS file system ignores case; a full stop ends a sentence, not a path
	p := strings.ToLower(path)
	if t := strings.TrimRight(p, "."); t != "" && !strings.HasSuffix(t, "/") {
		p = t
	}
	segs := strings.Split(p, "/")
	base := segs[len(segs)-1]

	if i := slices.Index(segs, ".ssh"); i >= 0 {
		rest := strings.Join(segs[i+1:], "/")
		public := !strings.ContainsAny(rest, "*?[") && (sshPublic[rest] || strings.HasPrefix(rest, "config.d/") ||
			strings.HasPrefix(rest, "cm-") || (strings.HasSuffix(rest, ".pub") && !strings.Contains(rest, "/")))
		if !public {
			return "an ssh private key, or ~/.ssh as a whole"
		}
	}
	if i := slices.Index(segs, ".aws"); i >= 0 && strings.Join(segs[i+1:], "/") != "config" {
		return "AWS credentials or cached tokens, or ~/.aws as a whole"
	}
	switch {
	case sshKeyName.MatchString(base):
		return "an ssh private key"
	case ageKeyName.MatchString(base), strings.HasSuffix(p, "sops/age/keys.txt"):
		return "an age key"
	case isEnvFile(base):
		return "an env file"
	case strings.HasSuffix(base, ".pem"):
		return "a PEM key or certificate"
	// each.key / local.key are HCL or Python, not files: a .key counts when it
	// looks like a path or is a file here
	case strings.HasSuffix(base, ".key") && (strings.ContainsAny(path, "/~") || h.isFile(path)):
		return "a key file"
	case base == "env.php", strings.HasSuffix(p, "config/database.yml"):
		return "application credentials"
	}
	return ""
}

// .env, .envrc, .env.local, .env-prod: env files; .env.example and other templates are not
func isEnvFile(base string) bool {
	rest, ok := strings.CutPrefix(base, ".env")
	switch {
	case !ok:
		return false
	case rest == "" || rest == "rc":
		return true
	case strings.ContainsAny(rest[:1], ".-_"):
		return !envTemplate[rest[1:]]
	}
	return false // .environment
}

func (h hook) isFile(path string) bool {
	st, err := os.Stat(h.abs(path))
	return err == nil && st.Mode().IsRegular()
}

// files: the regular files a command-line path names, ~ and globs expanded
// (github*.encrypted.yaml); relative paths from the session's directory.
func (h hook) files(token string) []string {
	p := token
	if strings.HasPrefix(p, "~") {
		p = h.home + p[1:]
	}
	matches, _ := filepath.Glob(h.abs(p))
	var files []string
	for _, m := range matches {
		if st, err := os.Stat(m); err == nil && st.Mode().IsRegular() {
			files = append(files, m)
		}
	}
	return files
}

// shown: file f as the command wrote it, relative when the token was
func (h hook) shown(token, f string) string {
	if filepath.IsAbs(token) || strings.HasPrefix(token, "~") || h.cwd == "" {
		return f
	}
	if rel, err := filepath.Rel(h.cwd, f); err == nil {
		return rel
	}
	return f
}

func (h hook) abs(p string) string {
	if filepath.IsAbs(p) || h.cwd == "" {
		return p
	}
	return filepath.Join(h.cwd, p)
}

func (h hook) isEncryptedFile(path string) bool {
	if st, err := os.Stat(path); err != nil || !st.Mode().IsRegular() {
		return false
	}
	return h.encrypted(path)
}

// sopsEncrypted is true only when sops itself reports the file as encrypted. A
// cleartext file returns false; a fake or broken sops block makes sops error
// out: both deny. Without sops: the metadata block and at least one ENC[ value.
func sopsEncrypted(path string) bool {
	if _, err := exec.LookPath("sops"); err != nil {
		data, err := os.ReadFile(path)
		return err == nil && sopsMeta.Match(data) && bytes.Contains(data, []byte("ENC["))
	}
	out, err := exec.Command("sops", "filestatus", path).Output()
	var status struct {
		Encrypted bool `json:"encrypted"`
	}
	return err == nil && json.Unmarshal(out, &status) == nil && status.Encrypted
}

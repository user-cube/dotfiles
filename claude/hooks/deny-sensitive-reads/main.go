// deny-sensitive-reads: Claude Code PreToolUse hook (matchers: Bash, Read).
//
// Bash: the Read(...) deny rules in settings.json do not cover Bash: a plain
// `cat file` still goes through. This hook inspects the command line and
// blocks it when it touches sensitive paths.
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

	// Cleartext credentials: blocked unconditionally.
	// `.key` must look like a path, not an expression: `each.key` and `local.key`
	// are ordinary HCL/Python and must not trip this.
	credentials = regexp.MustCompile(`(?im)age-key|\.agekey|AGE-SECRET-KEY|\.pem([^a-z]|$)|[/~][a-z0-9_.-]*\.key([^a-z]|$)|` +
		`(^|[^a-z0-9_-])\.aws/|(^|[^a-z0-9_-])\.ssh/|(^|[^a-z0-9_-])\.env|(^|/)env\.php|config/database\.yml`)
	compose = regexp.MustCompile(`(?i)docker-compose[^ \n]*\.ya?ml`)

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
	if path == "" || !secretDirs.MatchString(path) {
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
	if credentials.MatchString(cmd) {
		return "Command touches a cleartext credential file (local rule in ~/.claude/hooks/deny-sensitive-reads). Ask the user to do it themselves."
	}
	if compose.MatchString(cmd) {
		return "Command touches a docker-compose file (local rule). Ask the user to do it themselves."
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

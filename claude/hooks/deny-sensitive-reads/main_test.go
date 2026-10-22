package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	cleartext = "password: hunter2\n"
	encrypted = "password: ENC[AES256_GCM,data:abc,iv:def,tag:ghi,type:str]\nsops:\n    version: 3.9.0\n"
)

// tree: a session directory with secrets/ and credentials/, and a home with ~/secrets/
func tree(t *testing.T) (cwd, home string) {
	t.Helper()
	root := t.TempDir()
	cwd, home = filepath.Join(root, "repo"), filepath.Join(root, "home")
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(cwd, "secrets/plain.yaml"), cleartext)
	write(filepath.Join(cwd, "secrets/app.enc.yaml"), encrypted)
	write(filepath.Join(cwd, "credentials/db.sops.json"), encrypted)
	write(filepath.Join(cwd, "charts/external-secrets/values.yaml"), cleartext)
	write(filepath.Join(home, "secrets/token.txt"), cleartext)
	return cwd, home
}

// fakeEncrypted stands in for sops in the table: encrypted = has an ENC[ value
func fakeEncrypted(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && bytes.Contains(data, []byte("ENC["))
}

func TestCommand(t *testing.T) {
	cwd, home := tree(t)
	h := hook{home: home, cwd: cwd, encrypted: fakeEncrypted}
	const (
		cred     = "cleartext credential"
		composeF = "docker-compose"
		notEnc   = "not SOPS-encrypted"
		unknown  = "can't be checked"
	)
	cases := []struct {
		cmd, deny string // deny: part of the reason, "" = allowed
	}{
		{"", ""},
		{"ls -la", ""},
		{"git status", ""},

		// cleartext credentials, whatever the file holds
		{"cat ~/.ssh/id_ed25519", cred},
		{"cat ~/.aws/credentials", cred},
		{"cat .env", cred},
		{"cat .envrc", cred}, // .env* as in the Read deny rules
		{"openssl x509 -in cert.pem -noout", cred},
		{"cat ~/keys/server.key", cred},
		{"cat age-key.txt", cred},
		{"grep AGE-SECRET-KEY keys.txt", cred},
		{"cat wp/env.php", cred},
		{"cat config/database.yml", cred},
		{"ls\nenv.php", cred}, // ^ per line, like grep
		{"echo each.key local.key", ""},
		{"cat keypair.pemfile", ""},
		// The text alone trips it, also in a comment or a heredoc: a safety net
		// over the command line, it can't tell a mention from a read
		{"python3 - <<'EOF'\n# ~/.ssh/config stays per machine\nEOF", cred},

		{"docker compose -f docker-compose.prod.yml up", composeF},
		{"docker compose -f compose.yaml up", ""},

		// secrets/ and credentials/: by content
		{"cat secrets/plain.yaml", notEnc},
		{"cat secrets/app.enc.yaml", ""},
		{"sops -d credentials/db.sops.json", ""},
		{"cat secrets/app*.yaml", ""},
		{"cat secrets/*.yaml", notEnc},
		{"cat " + filepath.Join(cwd, "secrets/plain.yaml"), notEnc},
		{"cat ~/secrets/token.txt", notEnc},
		{`cat "secrets/plain.yaml"`, notEnc},
		{"x=secrets/plain.yaml", notEnc},
		// no such file: the name rule
		{"cat secrets/missing.yaml", unknown},
		{"cat secrets/missing.sops.json", ""},
		{`cat "$DIR/secrets/app.enc.yaml"`, ""},
		{`cat "$DIR/secrets/plain.yaml"`, unknown},
		{"ls secrets/", unknown}, // a directory is not a file sops can vouch for
		// a whole path segment only
		{"cat charts/external-secrets/values.yaml", ""},
		{"kubectl get externalsecrets/app", ""},
		{"CREDENTIALS/plain.yaml", unknown}, // case-insensitive; no such file here
	}
	for _, c := range cases {
		got := h.checkCommand(c.cmd)
		if c.deny == "" && got != "" {
			t.Errorf("%q: denied (%s), want allowed", c.cmd, got)
		}
		if c.deny != "" && !strings.Contains(got, c.deny) {
			t.Errorf("%q: got %q, want a deny with %q", c.cmd, got, c.deny)
		}
	}
}

func TestRead(t *testing.T) {
	cwd, home := tree(t)
	h := hook{home: home, cwd: cwd, encrypted: fakeEncrypted}
	cases := []struct {
		path string
		deny bool
	}{
		{"", false},
		{filepath.Join(cwd, "secrets/plain.yaml"), true},
		{"secrets/plain.yaml", true}, // relative: from the session's directory
		{filepath.Join(cwd, "secrets/app.enc.yaml"), false},
		{filepath.Join(cwd, "secrets/missing.yaml"), false}, // Read reports "not found"
		{filepath.Join(cwd, "secrets"), false},              // no trailing slash: not under secrets/
		{filepath.Join(cwd, "charts/external-secrets/values.yaml"), false},
		// Cleartext credentials are the Read(...) deny rules' job, not this hook's
		{filepath.Join(home, ".ssh/id_ed25519"), false},
	}
	for _, c := range cases {
		if got := h.checkRead(c.path) != ""; got != c.deny {
			t.Errorf("%q: denied = %v, want %v", c.path, got, c.deny)
		}
	}
}

func TestRun(t *testing.T) {
	cwd, home := tree(t)
	call := func(in string) string {
		var out bytes.Buffer
		run(strings.NewReader(in), &out, home, fakeEncrypted)
		return out.String()
	}

	out := call(`{"tool_name":"Bash","cwd":"` + cwd + `","tool_input":{"command":"cat secrets/plain.yaml"}}`)
	var resp struct {
		HookSpecificOutput struct {
			HookEventName, PermissionDecision, PermissionDecisionReason string
		}
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("deny is not JSON: %q", out)
	}
	if r := resp.HookSpecificOutput; r.HookEventName != "PreToolUse" || r.PermissionDecision != "deny" || r.PermissionDecisionReason == "" {
		t.Errorf("unexpected deny: %+v", r)
	}

	for _, in := range []string{
		`{"tool_name":"Bash","cwd":"` + cwd + `","tool_input":{"command":"cat secrets/app.enc.yaml"}}`,
		`{"tool_name":"Read","cwd":"` + cwd + `","tool_input":{"file_path":"README.md"}}`,
		`{"tool_name":"Edit","tool_input":{"file_path":"x"}}`,
		`not json`,
		``,
	} {
		if out := call(in); out != "" {
			t.Errorf("%s: got %q, want no output (allowed)", in, out)
		}
	}
}

// Without sops on PATH: the metadata block and an ENC[ value are both needed
func TestEncryptedWithoutSops(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	for name, c := range map[string]struct {
		content string
		want    bool
	}{
		"enc.yaml":   {encrypted, true},
		"plain.yaml": {cleartext, false},
		"meta-only":  {"sops:\n    version: 3.9.0\n", false},
		"enc-only":   {"password: ENC[x]\n", false},
		"json":       {`{"password": "ENC[x]", "sops": {"version": "3.9.0"}}`, true},
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(c.content), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := sopsEncrypted(path); got != c.want {
			t.Errorf("%s: encrypted = %v, want %v", name, got, c.want)
		}
	}
}

// With sops: a hand-written block that only looks encrypted is rejected
func TestEncryptedWithSops(t *testing.T) {
	if _, err := exec.LookPath("sops"); err != nil {
		t.Skip("sops not installed")
	}
	dir := t.TempDir()
	for name, content := range map[string]string{"fake.yaml": encrypted, "plain.yaml": cleartext} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if sopsEncrypted(path) {
			t.Errorf("%s: sops reports it as encrypted", name)
		}
	}
}

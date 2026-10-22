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
	write(filepath.Join(cwd, "server.key"), cleartext)
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
		ssh     = "ssh private key"
		aws     = "AWS credentials"
		env     = "an env file"
		pem     = "PEM"
		key     = "a key file"
		age     = "age"
		app     = "application credentials"
		notEnc  = "not SOPS-encrypted"
		unknown = "can't be checked"
	)
	cases := []struct {
		cmd, deny string // deny: part of the reason, "" = allowed
	}{
		{"", ""},
		{"ls -la", ""},
		{"git status", ""},

		// ~/.ssh: keys, the whole directory and globs are private; config,
		// host keys, public keys and ControlMaster sockets are not
		{"cat ~/.ssh/id_ed25519", ssh},
		{"cat ~/.ssh/github_work", ssh}, // a key under its own name
		{"cat $HOME/.ssh/ID_RSA", ssh},
		{"scp host:~/.ssh/id_rsa .", ssh},
		{"ssh -i ~/.ssh/deploy host", ssh},
		{"ssh -o IdentityFile=~/.ssh/deploy host", ssh},
		{"tar czf keys.tgz ~/.ssh", ssh},
		{"cp -r ~/.ssh/ /tmp/x", ssh},
		{"cat ~/.ssh/*", ssh},
		{"cat ~/.ssh/*.pub", ssh}, // a glob: can't tell what it expands to
		{"cd ~/.ssh && cat id_ed25519", ssh},
		{"cat ~/.ssh/config", ""},
		{"cat ~/.ssh/config.d/work", ""},
		{"ssh-keygen -F github.com -f ~/.ssh/known_hosts", ""},
		{"cat ~/.ssh/id_ed25519.pub", ""},
		{"cat ~/.ssh/authorized_keys", ""},
		{"ls ~/.ssh/cm-0123abcd", ""},
		{"grep ControlPath ~/.config/ssh/controlmaster.conf", ""},
		// a mention in a comment or a heredoc: only private paths count
		{"python3 - <<'EOF'\n# ~/.ssh/config stays per machine\nEOF", ""},
		{"echo 'see `~/.ssh/config` and ~/.aws/config.'", ""},
		{"echo `cat ~/.ssh/id_rsa`", ssh},
		{"cat ~/.ssh/id_rsa.", ssh},

		// ~/.aws: only config is not a secret
		{"cat ~/.aws/credentials", aws},
		{"ls ~/.aws/sso/cache/", aws},
		{"cat ~/.aws/cli/cache/x.json", aws},
		{"zip -r aws.zip ~/.aws", aws},
		{"cat ~/.aws/config", ""},
		{"aws configure list-profiles", ""},

		// env files; templates are not
		{"cat .env", env},
		{"cat app/.env.local", env},
		{"cat .env.production", env},
		{"cat .env-prod", env},
		{"cat .envrc", env},
		{"cat .env.example", ""},
		{"cp .env.sample .env.template", ""},
		{"cat .environment", ""},
		{"direnv allow", ""},

		// keys and app credentials, whatever they hold
		{"openssl x509 -in cert.pem -noout", pem},
		{"cat ~/keys/server.key", key},
		{"cat server.key", key}, // a file here
		{"echo each.key local.key", ""},
		{"cat keypair.pemfile", ""},
		{"cat age-key.txt", age},
		{"cat backup.agekey", age},
		{`cat "$HOME/Library/Application Support/sops/age/keys.txt"`, age},
		{"grep AGE-SECRET-KEY keys.txt", age},
		{"age-keygen -o /tmp/test-id.txt", ""}, // the command, not a key file
		{"cat wp/env.php", app},
		{"cat config/database.yml", app},
		{"ls\nenv.php", app},

		// not private: code and config that only mention credentials
		{"docker compose -f docker-compose.prod.yml up", ""},
		{"cat docker-compose.yml", ""},
		{"cat charts/app/values.yaml", ""},

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
		// the same private paths as for Bash
		{filepath.Join(home, ".ssh/id_ed25519"), true},
		{filepath.Join(home, ".ssh/github_work"), true},
		{filepath.Join(home, ".ssh/config"), false},
		{filepath.Join(home, ".ssh/id_ed25519.pub"), false},
		{filepath.Join(home, ".aws/credentials"), true},
		{filepath.Join(home, ".aws/config"), false},
		{filepath.Join(cwd, ".env"), true},
		{filepath.Join(cwd, ".env.example"), false},
		{filepath.Join(cwd, "docker-compose.yml"), false},
		{filepath.Join(cwd, "server.key"), true},
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

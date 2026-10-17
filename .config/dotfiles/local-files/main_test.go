package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// memVault is a vault in memory: notes by name, attachments by item.
type memVault struct {
	ids   map[string]string            // note name -> id
	files map[string]map[string][]byte // id -> file name -> content
}

func newMemVault() *memVault {
	return &memVault{ids: map[string]string{}, files: map[string]map[string][]byte{}}
}

func (m *memVault) Sync() error                          { return nil }
func (m *memVault) FindItem(name string) (string, error) { return m.ids[name], nil }

func (m *memVault) ListNames(prefix string) ([]string, error) {
	var names []string
	for n := range m.ids {
		if strings.HasPrefix(n, prefix) {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	return names, nil
}

func (m *memVault) CreateNote(name, _ string) (string, error) {
	id := "id-" + name
	m.ids[name] = id
	m.files[id] = map[string][]byte{}
	return id, nil
}

func (m *memVault) ReplaceAttachment(id, fileName, path string) error {
	data, err := os.ReadFile(path)
	m.files[id][fileName] = data
	return err
}

func (m *memVault) DownloadAttachment(id, fileName, dest string) error {
	return os.WriteFile(dest, m.files[id][fileName], 0o600)
}

func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// claudeDoc reads ~/.claude.json with every value compacted, to compare
// against one-line JSON (the file is written indented)
func claudeDoc(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(read(t, path))); err != nil {
		t.Fatal(err)
	}
	doc := map[string]json.RawMessage{}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

// A backup from machine A, restored on machine B that already has some of it
func TestBackupThenRestore(t *testing.T) {
	v := newMemVault()
	homeA, homeB := t.TempDir(), t.TempDir()
	write(t, filepath.Join(homeA, ".kube/config"), "kube-A", 0o600)
	write(t, filepath.Join(homeA, ".terraformrc"), "tf-A", 0o644)
	write(t, filepath.Join(homeA, claudeJSON),
		`{"oauthAccount":{"a":1},"mcpServers":{"grafana":{"env":{"T":"A"}},"bookstack":{"env":{"T":"A"}}}}`, 0o600)

	var out bytes.Buffer
	if err := backup(v, homeA, "A", &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"created Bitwarden item: Local files (A)", "  ~/.kube/config\n", "  ~/.terraformrc\n",
		"  ~/.claude.json  (MCP servers only: bookstack, grafana)"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("backup output lacks %q:\n%s", want, out.String())
		}
	}

	// B: its own .terraformrc, its own grafana, its own login and projects
	write(t, filepath.Join(homeB, ".terraformrc"), "tf-B", 0o644)
	write(t, filepath.Join(homeB, claudeJSON),
		`{"oauthAccount":{"b":2},"projects":{"/p":{}},"mcpServers":{"grafana":{"env":{"T":"B"}}}}`, 0o600)

	out.Reset()
	if err := restore(v, homeB, "A", false, &out); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(homeB, ".kube/config")); got != "kube-A" {
		t.Errorf(".kube/config = %q, want kube-A", got)
	}
	if fi, _ := os.Stat(filepath.Join(homeB, ".kube/config")); fi.Mode().Perm() != 0o600 {
		t.Errorf(".kube/config mode = %v, want 0600", fi.Mode().Perm())
	}
	if got := read(t, filepath.Join(homeB, ".terraformrc")); got != "tf-B" {
		t.Errorf(".terraformrc = %q, want B's own (kept)", got)
	}
	doc := claudeDoc(t, filepath.Join(homeB, claudeJSON))
	if string(doc["oauthAccount"]) != `{"b":2}` || string(doc["projects"]) != `{"/p":{}}` {
		t.Errorf("login / projects changed: %s %s", doc["oauthAccount"], doc["projects"])
	}
	var servers mcpServers
	_ = json.Unmarshal(doc["mcpServers"], &servers)
	if string(servers["grafana"]) != `{"env":{"T":"B"}}` || string(servers["bookstack"]) != `{"env":{"T":"A"}}` {
		t.Errorf("mcpServers = %v, want B's grafana and A's bookstack", servers)
	}
	if fi, _ := os.Stat(filepath.Join(homeB, claudeJSON)); fi.Mode().Perm() != 0o600 {
		t.Errorf(".claude.json mode = %v, want 0600", fi.Mode().Perm())
	}
	for _, want := range []string{"  ~/.terraformrc  (exists, kept; --force to overwrite)",
		"  ~/.claude.json  (MCP servers: bookstack, grafana)", "    already here, kept: grafana (--force to overwrite)"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("restore output lacks %q:\n%s", want, out.String())
		}
	}

	// --force: A's copies win
	out.Reset()
	if err := restore(v, homeB, "A", true, &out); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(homeB, ".terraformrc")); got != "tf-A" {
		t.Errorf("--force: .terraformrc = %q, want tf-A", got)
	}
	_ = json.Unmarshal(claudeDoc(t, filepath.Join(homeB, claudeJSON))["mcpServers"], &servers)
	if string(servers["grafana"]) != `{"env":{"T":"A"}}` {
		t.Errorf("--force: grafana = %s, want A's", servers["grafana"])
	}

	// Again without --force: everything is there, ~/.claude.json isn't rewritten
	out.Reset()
	if err := restore(v, homeB, "A", false, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "  ~/.claude.json  (MCP servers bookstack, grafana: exists, kept; --force to overwrite)") {
		t.Errorf("second restore output:\n%s", out.String())
	}
}

// A new Mac has no ~/.claude.json yet
func TestRestoreCreatesClaudeJSON(t *testing.T) {
	v := newMemVault()
	homeA, homeB := t.TempDir(), t.TempDir()
	write(t, filepath.Join(homeA, ".zshrc.local"), "x", 0o644)
	write(t, filepath.Join(homeA, claudeJSON), `{"mcpServers":{"k8s":{"command":"npx"}}}`, 0o600)
	if err := backup(v, homeA, "A", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := restore(v, homeB, "A", false, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var servers mcpServers
	_ = json.Unmarshal(claudeDoc(t, filepath.Join(homeB, claudeJSON))["mcpServers"], &servers)
	if string(servers["k8s"]) != `{"command":"npx"}` {
		t.Errorf("mcpServers = %v", servers)
	}
}

func TestBackupWithoutMCPServers(t *testing.T) {
	v := newMemVault()
	home := t.TempDir()
	write(t, filepath.Join(home, ".zshrc.local"), "x", 0o644)
	var out bytes.Buffer
	if err := backup(v, home, "A", &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "MCP") {
		t.Errorf("no ~/.claude.json, yet the output mentions MCP:\n%s", out.String())
	}
}

func TestBackupNothingToSave(t *testing.T) {
	if err := backup(newMemVault(), t.TempDir(), "A", &bytes.Buffer{}); err == nil {
		t.Fatal("want an error when none of the files exist")
	}
}

func TestRestoreUnknownMachine(t *testing.T) {
	v := newMemVault()
	v.CreateNote("Local files (Old-Mac)", "")
	err := restore(v, t.TempDir(), "Nope", false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "machines with one: Old-Mac") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadArchiveRejectsUnsafePaths(t *testing.T) {
	for _, name := range []string{"../evil", "/etc/evil", "a/../../evil"} {
		t.Run(name, func(t *testing.T) {
			// writeArchive only takes files under home: build the bad one by hand
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			tw := tar.NewWriter(gz)
			if err := addFile(tw, name, 0o644, time.Now(), []byte("x")); err != nil {
				t.Fatal(err)
			}
			tw.Close()
			gz.Close()
			path := filepath.Join(t.TempDir(), "a.tar.gz")
			write(t, path, buf.String(), 0o600)
			if _, err := readArchive(path); err == nil {
				t.Fatalf("%q: want an error", name)
			}
		})
	}
}

// Backups made by the old local-files.sh (macOS `tar czf -C ~`) still restore
func TestReadArchiveFromSystemTar(t *testing.T) {
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("no tar")
	}
	home := t.TempDir()
	write(t, filepath.Join(home, ".kube/config"), "kube", 0o600)
	write(t, filepath.Join(home, ".zshrc.local"), "zsh", 0o644)
	path := filepath.Join(t.TempDir(), "old.tar.gz")
	if out, err := exec.Command("tar", "czf", path, "-C", home, ".zshrc.local", ".kube/config").CombinedOutput(); err != nil {
		t.Fatalf("tar: %v: %s", err, out)
	}
	entries, err := readArchive(path)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range entries {
		got[e.name] = string(e.data)
	}
	if len(got) != 2 || got[".kube/config"] != "kube" || got[".zshrc.local"] != "zsh" {
		t.Errorf("entries = %v", got)
	}
}

func TestMergeServers(t *testing.T) {
	a, b := json.RawMessage(`"a"`), json.RawMessage(`"b"`)
	tests := []struct {
		name       string
		here       mcpServers
		saved      mcpServers
		force      bool
		wantGrafan string
		wantKept   []string
	}{
		{"new server", nil, mcpServers{"grafana": a}, false, `"a"`, nil},
		{"existing kept", mcpServers{"grafana": b}, mcpServers{"grafana": a}, false, `"b"`, []string{"grafana"}},
		{"existing forced", mcpServers{"grafana": b}, mcpServers{"grafana": a}, true, `"a"`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, kept := mergeServers(tt.here, tt.saved, tt.force)
			if string(merged["grafana"]) != tt.wantGrafan || !slices.Equal(kept, tt.wantKept) {
				t.Errorf("got %s, kept %v; want %s, kept %v", merged["grafana"], kept, tt.wantGrafan, tt.wantKept)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		cmd       string
		args      []string
		machine   string
		force     bool
		wantError bool
	}{
		{"restore", nil, "", false, false},
		{"restore", []string{"Old-Mac"}, "Old-Mac", false, false},
		{"restore", []string{"--force"}, "", true, false},
		{"restore", []string{"Old-Mac", "--force"}, "Old-Mac", true, false},
		{"restore", []string{"--force", "Old-Mac"}, "Old-Mac", true, false},
		{"restore", []string{""}, "", false, false}, // the playbook's empty machine
		{"restore", []string{"a", "b"}, "", false, true},
		{"restore", []string{"--forse"}, "", false, true},
		{"backup", []string{"x"}, "", false, true},
		{"backup", nil, "", false, false},
	}
	for _, tt := range tests {
		machine, force, err := parseArgs(tt.cmd, tt.args)
		if (err != nil) != tt.wantError || machine != tt.machine || force != tt.force {
			t.Errorf("parseArgs(%s %q) = %q, %v, %v", tt.cmd, tt.args, machine, force, err)
		}
	}
}

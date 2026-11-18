package main

import (
	"bytes"
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestChangedFiles(t *testing.T) {
	status := strings.Join([]string{
		" M main.go",
		"?? new/file.sh",
		"A  added.json",
		"R  renamed.go", "old.go", // a rename: the old path follows
		" D gone.go",
		"MM both.zsh",
		"",
	}, "\x00")
	got := changedFiles(status)
	want := []string{"added.json", "both.zsh", "main.go", "new/file.sh", "renamed.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("changedFiles = %v, want %v", got, want)
	}
	if got := changedFiles(""); len(got) != 0 {
		t.Errorf("empty status gave %v", got)
	}
}

func TestGoModules(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"a/go.mod", "a/sub/x.go", "b/c/go.mod", "b/c/d/y.go", "loose/z.go"} {
		os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755)
		os.WriteFile(filepath.Join(root, p), nil, 0o644)
	}
	got := goModules(root, []string{"a/sub/x.go", "a/other.go", "b/c/d/y.go", "loose/z.go", "readme.md", "b/c/go.sum"})
	want := []string{"a", "b/c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("goModules = %v, want %v", got, want)
	}
}

func TestTail(t *testing.T) {
	var lines []string
	for range 100 {
		lines = append(lines, "line")
	}
	lines[99] = "the verdict"
	got := tail(strings.Join(lines, "\n") + "\n")
	if !strings.HasPrefix(got, "[...]\n") || !strings.HasSuffix(got, "the verdict") || strings.Count(got, "\n") != maxLines {
		t.Errorf("tail = %q", got)
	}
	if got := tail("short\n"); got != "short" {
		t.Errorf("tail(short) = %q", got)
	}
}

// --- against real git, go and bash ---

func need(t *testing.T, tools ...string) {
	t.Helper()
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip("needs " + tool)
		}
	}
}

// repo is a git repository with a clean Go module in mod/ and a hook that has
// no shellcheck and its own cache directory
func repo(t *testing.T) (string, hook) {
	t.Helper()
	need(t, "git", "go", "bash")
	// the user's global git config has hooks of its own (gitleaks, sops-guard)
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	root := t.TempDir()
	write(t, root, "mod/go.mod", "module m\n\ngo 1.22\n")
	write(t, root, "mod/m.go", "package m\n\nfunc F() int { return 1 }\n")
	write(t, root, "mod/m_test.go", "package m\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) {\n\tif F() != 1 {\n\t\tt.Fatal(\"no\")\n\t}\n}\n")
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	root, _ = filepath.EvalSymlinks(root)
	h := hook{
		run: execRunner,
		has: func(tool string) bool {
			// the tests that want shellcheck, zsh or gitleaks say so; the rest do not depend on what is installed
			return tool != "shellcheck" && tool != "zsh" && tool != "gitleaks" && func() bool { _, err := exec.LookPath(tool); return err == nil }()
		},
		cacheDir: filepath.Join(t.TempDir(), "cache"),
	}
	return root, h
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	p := filepath.Join(root, name)
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCleanTreeLetsClaudeStop(t *testing.T) {
	root, h := repo(t)
	if got := h.check(context.Background(), root); got != "" {
		t.Errorf("clean tree: %q", got)
	}
}

func TestGoChangeThatPasses(t *testing.T) {
	root, h := repo(t)
	write(t, root, "mod/m.go", "package m\n\nfunc F() int { return 1 }\n\nfunc G() int { return 2 }\n")
	if got := h.check(context.Background(), root); got != "" {
		t.Errorf("a passing change: %q", got)
	}
}

func TestGoTestFailureBlocks(t *testing.T) {
	root, h := repo(t)
	write(t, root, "mod/m.go", "package m\n\nfunc F() int { return 2 }\n")
	got := h.check(context.Background(), root)
	if !strings.Contains(got, "go test in mod") || !strings.Contains(got, "no") {
		t.Errorf("want a go test failure in mod, got %q", got)
	}
}

func TestGoVetFailureBlocks(t *testing.T) {
	root, h := repo(t)
	write(t, root, "mod/m.go", "package m\n\nimport \"fmt\"\n\nfunc F() int { fmt.Printf(\"%d\", \"x\"); return 1 }\n")
	got := h.check(context.Background(), root)
	if !strings.Contains(got, "go vet in mod") {
		t.Errorf("want a go vet failure in mod, got %q", got)
	}
}

func TestSameStateIsCheckedOnce(t *testing.T) {
	root, h := repo(t)
	write(t, root, "mod/m.go", "package m\n\nfunc F() int { return 2 }\n")
	if h.check(context.Background(), root) == "" {
		t.Fatal("the first check should fail")
	}
	if got := h.check(context.Background(), root); got != "" {
		t.Errorf("the same state again: %q", got)
	}
	write(t, root, "mod/m.go", "package m\n\nfunc F() int { return 3 }\n")
	if h.check(context.Background(), root) == "" {
		t.Error("a changed state is checked again")
	}
}

func TestShellSyntaxErrorBlocks(t *testing.T) {
	root, h := repo(t)
	write(t, root, "bin/x.sh", "#!/bin/sh\nif true; then\n  echo hi\n")
	if got := h.check(context.Background(), root); !strings.Contains(got, "bash bin/x.sh") {
		t.Errorf("want a bash -n failure, got %q", got)
	}
	write(t, root, "bin/x.sh", "#!/bin/sh\necho hi\n")
	if got := h.check(context.Background(), root); got != "" {
		t.Errorf("a valid script: %q", got)
	}
}

func TestJSON(t *testing.T) {
	root, h := repo(t)
	write(t, root, "claude/settings.json", `{"a": 1,}`)
	if got := h.check(context.Background(), root); !strings.Contains(got, "JSON claude/settings.json") {
		t.Errorf("want a JSON failure, got %q", got)
	}
	// JSONC under vscode/ has comments: not checked
	root, h = repo(t)
	write(t, root, ".config/vscode/settings.json", "{\n  // a comment\n  \"a\": 1,\n}\n")
	if got := h.check(context.Background(), root); got != "" {
		t.Errorf("vscode JSONC: %q", got)
	}
}

func TestNotARepository(t *testing.T) {
	need(t, "git")
	_, h := repo(t)
	if got := h.check(context.Background(), t.TempDir()); got != "" {
		t.Errorf("outside a repository: %q", got)
	}
	if got := h.check(context.Background(), ""); got != "" {
		t.Errorf("no cwd: %q", got)
	}
}

func TestOutOfTimeIsNeverAFailure(t *testing.T) {
	root, h := repo(t)
	write(t, root, "mod/m.go", "package m\n\nfunc F() int { return 2 }\n")
	ctx, cancel := context.WithCancel(context.Background())
	real := h.run
	h.run = func(c context.Context, dir, name string, args ...string) (string, error) {
		if name == "go" {
			cancel() // the budget runs out while go runs
			return "signal: killed", context.Canceled
		}
		return real(c, dir, name, args...)
	}
	if got := h.check(ctx, root); got != "" {
		t.Errorf("a timeout was reported as a failure: %q", got)
	}
}

func TestReportIsCut(t *testing.T) {
	root, h := repo(t)
	for i := range 80 {
		write(t, root, "bin/s"+strconv.Itoa(i)+".sh", "#!/bin/sh\nif true; then\n")
	}
	got := h.check(context.Background(), root)
	if len(got) > maxReport+len("\n[cut]") || !strings.HasSuffix(got, "[cut]") {
		t.Errorf("report of %d bytes not cut: ...%q", len(got), got[max(0, len(got)-20):])
	}
}

func event_(cwd string, active bool) *bytes.Reader {
	b, _ := json.Marshal(map[string]any{"cwd": cwd, "stop_hook_active": active})
	return bytes.NewReader(b)
}

func TestRun(t *testing.T) {
	root, h := repo(t)
	write(t, root, "mod/m.go", "package m\n\nfunc F() int { return 2 }\n")

	// Claude was already sent back once: it may stop
	var out bytes.Buffer
	run(event_(root, true), &out, h)
	if out.Len() != 0 {
		t.Errorf("stop_hook_active: %q", out.String())
	}

	// the user's switch
	t.Setenv("STOP_CHECK", "0")
	run(event_(root, false), &out, h)
	if out.Len() != 0 {
		t.Errorf("STOP_CHECK=0: %q", out.String())
	}
	t.Setenv("STOP_CHECK", "")

	run(strings.NewReader("not json"), &out, h)
	if out.Len() != 0 {
		t.Errorf("bad input: %q", out.String())
	}

	run(event_(root, false), &out, h)
	var resp map[string]string
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("not JSON: %q", out.String())
	}
	if resp["decision"] != "block" || !strings.Contains(resp["reason"], "go test in mod") {
		t.Errorf("decision = %v", resp)
	}
}

// --- gitleaks ---

// fakeScanner stands in for gitleaks: it records the arguments and answers with report
func fakeScanner(report string, seen *[]string) runner {
	real := execRunner
	return func(ctx context.Context, dir, name string, args ...string) (string, error) {
		if name != "gitleaks" {
			return real(ctx, dir, name, args...)
		}
		*seen = args
		return report, nil
	}
}

func withGitleaks(h hook) hook {
	have := h.has
	h.has = func(tool string) bool { return tool == "gitleaks" || have(tool) }
	return h
}

func TestSecretsAreReportedByPlaceNeverByValue(t *testing.T) {
	root, h := repo(t)
	write(t, root, "app/config.txt", "x\n")
	var seen []string
	// the real report is redacted; a field that was not would still not be printed
	h.run = fakeScanner(`[{"File":"`+root+`/app/config.txt","StartLine":3,"RuleID":"github-pat","Description":"GitHub Personal Access Token","Secret":"super-secret-value","Match":"token = super-secret-value"}]`, &seen)
	got := withGitleaks(h).check(context.Background(), root)
	if !strings.Contains(got, "app/config.txt:3  GitHub Personal Access Token (github-pat)") {
		t.Errorf("want the place and the rule, got %q", got)
	}
	if strings.Contains(got, "super-secret-value") {
		t.Errorf("the secret is in the report: %q", got)
	}
	if !strings.Contains(got, "rotated") {
		t.Errorf("it should say a real secret has to be rotated: %q", got)
	}
	if !strings.Contains(strings.Join(seen, " "), "--redact") || !strings.Contains(strings.Join(seen, " "), root+"/app/config.txt") {
		t.Errorf("gitleaks should be redacted and given the changed file: %v", seen)
	}
}

func TestSecretsUseTheRepositoryConfig(t *testing.T) {
	root, h := repo(t)
	write(t, root, ".gitleaks.toml", "title = \"x\"\n")
	write(t, root, "a.txt", "x\n")
	var seen []string
	h.run = fakeScanner(`[]`, &seen)
	withGitleaks(h).check(context.Background(), root)
	joined := strings.Join(seen, " ")
	if !strings.Contains(joined, "-c "+root+"/.gitleaks.toml") {
		t.Errorf("the repository's config should be passed: %v", seen)
	}
}

func TestSecretsClean(t *testing.T) {
	root, h := repo(t)
	write(t, root, "a.txt", "x\n")
	var seen []string
	h.run = fakeScanner(`[]`, &seen)
	if got := withGitleaks(h).check(context.Background(), root); got != "" {
		t.Errorf("a clean scan: %q", got)
	}
}

func TestSecretsScannerThatFailsIsNotAFailure(t *testing.T) {
	root, h := repo(t)
	write(t, root, "a.txt", "x\n")
	for _, report := range []string{"", "gitleaks: bad config", "[not json"} {
		var seen []string
		h.run = fakeScanner(report, &seen)
		h.cacheDir = filepath.Join(t.TempDir(), "cache")
		if got := withGitleaks(h).check(context.Background(), root); got != "" {
			t.Errorf("report %q was reported as a failure: %q", report, got)
		}
	}
}

func TestSecretsWithoutGitleaks(t *testing.T) {
	root, h := repo(t)
	write(t, root, "a.txt", "x\n")
	var seen []string
	h.run = fakeScanner(`[{"File":"a.txt","StartLine":1,"RuleID":"r","Description":"d"}]`, &seen)
	if got := h.check(context.Background(), root); got != "" || seen != nil {
		t.Errorf("not installed: got %q, gitleaks called with %v", got, seen)
	}
}

func TestSecretsOnlyTheChangedFilesThatExist(t *testing.T) {
	root, h := repo(t)
	var seen []string
	h.run = fakeScanner(`[]`, &seen)
	// only a deleted file changed: nothing to scan, so gitleaks is not run
	os.Remove(filepath.Join(root, "mod/m_test.go"))
	withGitleaks(h).check(context.Background(), root)
	if seen != nil {
		t.Errorf("a deleted file was scanned: %v", seen)
	}
}

// the real gitleaks, with a token made up here (never written in the source,
// which gitleaks scans in CI) that looks like a GitHub one
func TestRealGitleaks(t *testing.T) {
	need(t, "gitleaks")
	root, h := repo(t)
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	r := rand.New(rand.NewSource(7))
	token := "ghp_"
	for range 36 {
		token += string(letters[r.Intn(len(letters))])
	}
	write(t, root, "deploy/env.txt", "token = \""+token+"\"\n")
	h = withGitleaks(h)
	h.has = func(tool string) bool { _, err := exec.LookPath(tool); return err == nil && tool == "gitleaks" }
	got := h.check(context.Background(), root)
	if !strings.Contains(got, "deploy/env.txt:1") || !strings.Contains(got, "github-pat") {
		t.Errorf("want the place and the rule, got %q", got)
	}
	if strings.Contains(got, token) || strings.Contains(got, token[:12]) {
		t.Errorf("the token is in the report: %q", got)
	}
}

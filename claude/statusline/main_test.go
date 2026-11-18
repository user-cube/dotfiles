package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// fake stands in for git and kube-guard: the key is the command and its
// arguments, a missing key is a failing tool
func fake(answers map[string]string) runner {
	return func(_ context.Context, _, name string, args ...string) (string, error) {
		if out, ok := answers[name+" "+strings.Join(args, " ")]; ok {
			return out, nil
		}
		return "", errors.New("no such answer")
	}
}

const session = `{
  "cwd": "/Users/me/work/dotfiles",
  "model": {"display_name": "Opus"},
  "workspace": {"current_dir": "/Users/me/work/dotfiles"},
  "cost": {"total_cost_usd": 1.234},
  "context_window": {"used_percentage": 42.4},
  "rate_limits": {"five_hour": {"used_percentage": 18}},
  "pr": {"number": 7}
}`

func plain(run runner) deps { return deps{run: run, kubeBin: "kube-guard"} }

func TestRender(t *testing.T) {
	branch := map[string]string{
		"git branch --show-current":                   "main\n",
		"git status --porcelain --untracked-files=no": "",
		"kube-guard context":                          "dev-eu\napi\nother\n",
	}
	dirty := map[string]string{
		"git branch --show-current":                   "feat/x\n",
		"git status --porcelain --untracked-files=no": " M main.go\n",
	}
	prod := map[string]string{"kube-guard context": "prod-eu\n\nprod\n"}
	merge := func(ms ...map[string]string) map[string]string {
		all := map[string]string{}
		for _, m := range ms {
			for k, v := range m {
				all[k] = v
			}
		}
		return all
	}

	for _, tt := range []struct {
		name, json string
		answers    map[string]string
		want       string
	}{
		{"everything", session, branch,
			"dotfiles  main  ⎈ dev-eu/api │ Opus · ctx 42% · $1.23 · 5h 18% · PR #7"},
		{"uncommitted changes", session, merge(dirty),
			"dotfiles  feat/x* │ Opus · ctx 42% · $1.23 · 5h 18% · PR #7"},
		{"production, no namespace", session, merge(branch, prod),
			"dotfiles  main  ⎈ prod-eu PROD │ Opus · ctx 42% · $1.23 · 5h 18% · PR #7"},
		{"detached", session, map[string]string{
			"git branch --show-current": "\n", "git rev-parse --short HEAD": "abc1234\n"},
			"dotfiles  abc1234 │ Opus · ctx 42% · $1.23 · 5h 18% · PR #7"},
		{"no repository, no kube", session, nil,
			"dotfiles │ Opus · ctx 42% · $1.23 · 5h 18% · PR #7"},
		{"early in the session", `{"cwd":"/a/b","model":{"display_name":"Opus"},"context_window":{"used_percentage":null},"cost":{"total_cost_usd":0}}`, nil,
			"b │ Opus"},
		{"workspace wins over cwd", `{"cwd":"/a/b","workspace":{"current_dir":"/a/c"}}`, nil, "c"},
		{"no directory", `{"model":{"display_name":"Opus"}}`, nil, "Opus"},
		{"nothing", `{}`, nil, ""},
		{"bad JSON", `not json`, nil, ""},
		{"kube-guard prints nothing: no context", session, map[string]string{"kube-guard context": ""},
			"dotfiles │ Opus · ctx 42% · $1.23 · 5h 18% · PR #7"},
		{"rate limit without a value", `{"cwd":"/a/b","rate_limits":{"five_hour":{}}}`, nil, "b"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := render([]byte(tt.json), plain(fake(tt.answers))); got != tt.want {
				t.Errorf("\n got %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestColors(t *testing.T) {
	d := deps{run: fake(map[string]string{"kube-guard context": "prod-eu\n\nprod\n"}), kubeBin: "kube-guard", colorOn: true}
	got := render([]byte(`{"cwd":"/a/b","context_window":{"used_percentage":95},"rate_limits":{"five_hour":{"used_percentage":75}}}`), d)
	for _, want := range []string{
		"\x1b[1;31m⎈ prod-eu PROD\x1b[0m", // production: bold red
		"\x1b[31mctx 95%\x1b[0m",          // nearly full: red
		"\x1b[33m5h 75%\x1b[0m",           // filling up: yellow
		"\x1b[1;34mb\x1b[0m",              // the directory
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if got := render([]byte(`{"cwd":"/a/b"}`), plain(fake(nil))); strings.Contains(got, "\x1b") {
		t.Errorf("colors off still painted: %q", got)
	}
}

func TestLevel(t *testing.T) {
	for p, want := range map[float64]string{0: "32", 69.9: "32", 70: "33", 89.9: "33", 90: "31", 100: "31"} {
		if got := level(p); got != want {
			t.Errorf("level(%v) = %s, want %s", p, got, want)
		}
	}
}

// --- against the real git and a real process ---

func TestRealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("needs git")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	os.WriteFile(dir+"/a.txt", []byte("1"), 0o644)
	git("add", ".")
	git("commit", "-q", "-m", "init")

	d := deps{run: execRunner, kubeBin: "/nonexistent/kube-guard"}
	json := `{"workspace":{"current_dir":"` + dir + `"}}`
	if got := render([]byte(json), d); !strings.HasSuffix(got, "main") {
		t.Errorf("clean: %q", got)
	}
	os.WriteFile(dir+"/a.txt", []byte("2"), 0o644)
	if got := render([]byte(json), d); !strings.HasSuffix(got, "main*") {
		t.Errorf("modified: %q", got)
	}
	// a file nobody tracks does not make it dirty
	git("checkout", "-q", "--", "a.txt")
	os.WriteFile(dir+"/new.txt", []byte("x"), 0o644)
	if got := render([]byte(json), d); !strings.HasSuffix(got, "main") {
		t.Errorf("untracked: %q", got)
	}
	// not a repository: just the directory
	other := t.TempDir()
	if got := render([]byte(`{"cwd":"`+other+`"}`), d); got != strings.TrimLeft(other[strings.LastIndex(other, "/"):], "/") {
		t.Errorf("outside a repository: %q", got)
	}
}

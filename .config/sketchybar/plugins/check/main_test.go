package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// memBar records the sketchybar calls
type memBar struct{ calls [][]string }

func (m *memBar) run(args ...string) error {
	m.calls = append(m.calls, args)
	return nil
}

var testColors = map[string]string{"FG": "0xfff8f8f2", "DIM": "0xff75715e", "GREEN": "0xffa6e22e", "YELLOW": "0xffe6db74", "RED": "0xfff92672"}

// configDir: colors.sh (the real one) and, when checks != "", a checks file
func configDir(t *testing.T, checks string) string {
	t.Helper()
	dir := t.TempDir()
	colors, err := os.ReadFile("../../colors.sh")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "colors.sh"), colors, 0o644); err != nil {
		t.Fatal(err)
	}
	if checks != "" {
		if err := os.WriteFile(filepath.Join(dir, "checks"), []byte(checks), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestParseChecks(t *testing.T) {
	example, err := os.ReadFile("../../checks.example")
	if err != nil {
		t.Fatal(err)
	}
	got := parseChecks(string(example) + `
	  # indented comment | true
   | no name
no-pipe
my vpn |   nc -z host 443 | grep -q x
trailing |true   `)
	want := []check{
		{"rancher", "kube rancher-prod"},
		{"eks", "kube eks-dev"},
		{"vpn", "nc -z -G 2 10.0.0.1 443"},
		{"no-pipe", ""},
		{"myvpn", "nc -z host 443 | grep -q x"},
		{"trailing", "true   "},
	}
	if !slices.Equal(got, want) {
		t.Errorf("parseChecks:\n got %q\nwant %q", got, want)
	}
}

func TestCommand(t *testing.T) {
	got := command(context.Background(), "kube arn:aws:eks:eu-west-1:1:cluster/prod").Args
	want := []string{"kubectl", "--context", "arn:aws:eks:eu-west-1:1:cluster/prod", "get", "--raw", "/api", "--request-timeout=5s"}
	if !slices.Equal(got, want) {
		t.Errorf("kube: %q", got)
	}
	if got := command(context.Background(), "nc -z host 443").Args; !slices.Equal(got, []string{"bash", "-c", "nc -z host 443"}) {
		t.Errorf("shell: %q", got)
	}
	if got := command(context.Background(), "kubectl version").Args[0]; got != "bash" {
		t.Errorf("only `kube ` is the shorthand, got %q", got)
	}
}

func TestReason(t *testing.T) {
	cases := []struct {
		r    result
		want string
	}{
		{result{code: 137, timedOut: true}, "timeout"},
		{result{code: 1, stderr: "error: You must be logged in to the server (Unauthorized)"}, "expired"},
		{result{code: 1, stderr: "the server has asked for the client to provide credentials (401)"}, "expired"},
		{result{code: 1, stderr: "failed to refresh token: oauth2"}, "expired"},
		{result{code: 1, stderr: "Error from server (Forbidden): forbidden"}, "forbidden"},
		{result{code: 1, stderr: "dial tcp 10.0.0.1:443: connect: connection refused"}, "refused"},
		{result{code: 1, stderr: `error: context "nope" does not exist`}, "no context"},
		{result{code: 1, stderr: "dial tcp: lookup api.example.com: no such host"}, "dns"},
		{result{code: 6, stderr: "curl: (6) Could not resolve host: x"}, "dns"},
		{result{code: 1, stderr: "Unable to connect: i/o timeout"}, "timeout"},
		{result{code: 1, stderr: "connection timed\nout"}, "timeout"}, // lines joined with spaces
		{result{code: 3}, "exit 3"},
		{result{code: 1, stderr: "FORBIDDEN"}, "exit 1"}, // case-sensitive, as before
	}
	for _, c := range cases {
		if got := reason(c.r); got != c.want {
			t.Errorf("%+v: got %q, want %q", c.r, got, c.want)
		}
	}
}

func TestRunAll(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "open"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := withPath(os.Environ(), bin+":"+os.Getenv("PATH"))
	checks := []check{
		{"ok", "true"},
		{"exit", "exit 3"},
		{"stderr", "echo Unauthorized >&2; exit 1"},
		{"stdout", "echo noise"},
		{"slow", "sleep 30"},
		{"orphan", "sleep 30 & sleep 30"},         // its own process group: killed too
		{"stdin", "read -r x"},                    // no stdin: EOF, not a hang
		{"browser", "open https://login.example"}, // the fake open
	}
	start := time.Now()
	results := runAll(checks, 300*time.Millisecond, env)
	// In parallel, and nothing waits on the 30 s sleeps
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("runAll took %v", d)
	}
	want := []struct {
		code     int
		timedOut bool
	}{{0, false}, {3, false}, {1, false}, {0, false}, {137, true}, {137, true}, {1, false}, {1, false}}
	for i, r := range results {
		if r.index != i || r.name != checks[i].name {
			t.Errorf("result %d is %q (index %d)", i, r.name, r.index)
		}
		if r.code != want[i].code || r.timedOut != want[i].timedOut {
			t.Errorf("%s: code %d timedOut %v, want %d %v", r.name, r.code, r.timedOut, want[i].code, want[i].timedOut)
		}
	}
	if !strings.Contains(results[2].stderr, "Unauthorized") {
		t.Errorf("stderr not kept: %q", results[2].stderr)
	}
}

func TestPopup(t *testing.T) {
	results := []result{
		{check: check{"arn:aws:eks:eu-west-1:1:cluster/prod", ""}, index: 0, code: 0},
		{check: check{"dev", ""}, index: 1, code: 1, stderr: "Unauthorized"},
		{check: check{"rancher", ""}, index: 2, code: 0},
		{check: check{"vpn", ""}, index: 3, code: 137, timedOut: true},
	}
	args := popup("checks", results, testColors, time.Date(2026, 10, 22, 9, 5, 0, 0, time.UTC))
	joined := strings.Join(args, " ")

	// Failures first, then in order
	var order []string
	for i, a := range args {
		if a == "--add" {
			order = append(order, args[i+2])
		}
	}
	if want := []string{"checks.item.header", "checks.item.1", "checks.item.3", "checks.item.0", "checks.item.2"}; !slices.Equal(order, want) {
		t.Errorf("order %q, want %q", order, want)
	}
	for _, want := range []string{
		"label=09:05",
		"icon.width=96", // rancher: 7 * 8 + 40
		"icon=" + okIcon + "  prod",
		"icon=" + failIcon + "  dev icon.color=" + testColors["RED"],
		"label=expired",
		"label=timeout",
		"popup.checks",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("popup has no %q:\n%s", want, joined)
		}
	}
}

func TestRun(t *testing.T) {
	dir := configDir(t, "ok | true\nbad | exit 2\n# off | false\n")
	colors, _ := readColors(filepath.Join(dir, "colors.sh"))
	b := &memBar{}
	if err := run(b, "checks", "routine", dir, time.Second, time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(b.calls) != 4 {
		t.Fatalf("calls: %q", b.calls)
	}
	if got := b.calls[0]; !slices.Equal(got, []string{"--set", "checks", "drawing=on", "label.color=" + colors["YELLOW"]}) {
		t.Errorf("first call %q", got)
	}
	if got := b.calls[1]; got[0] != "--remove" {
		t.Errorf("second call %q", got)
	}
	if got := b.calls[3]; !slices.Equal(got, []string{"--set", "checks", "label=1/2", "label.color=" + colors["RED"]}) {
		t.Errorf("label %q", got)
	}

	b = &memBar{}
	if err := run(b, "checks", "", configDir(t, "ok | true\n"), time.Second, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := b.calls[len(b.calls)-1]; !slices.Equal(got, []string{"--set", "checks", "label=1/1", "label.color=" + colors["GREEN"]}) {
		t.Errorf("all passed: %q", got)
	}
}

func TestRunClicks(t *testing.T) {
	for sender, want := range map[string]string{"mouse.clicked": "popup.drawing=toggle", "mouse.exited.global": "popup.drawing=off"} {
		b := &memBar{}
		// No colors.sh needed: a click never runs the checks
		if err := run(b, "checks", sender, t.TempDir(), time.Second, time.Now()); err != nil {
			t.Fatal(err)
		}
		if len(b.calls) != 1 || !slices.Equal(b.calls[0], []string{"--set", "checks", want}) {
			t.Errorf("%s: %q", sender, b.calls)
		}
	}
}

// Nothing to check (only comments, and no kubectl on PATH): hidden
func TestRunNothing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	b := &memBar{}
	if err := run(b, "checks", "", configDir(t, "# rancher | kube rancher-prod\n"), time.Second, time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(b.calls) != 1 || !slices.Equal(b.calls[0], []string{"--set", "checks", "drawing=off"}) {
		t.Errorf("calls: %q", b.calls)
	}
}

func TestReadColors(t *testing.T) {
	colors, err := readColors("../../colors.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"FG", "DIM", "GREEN", "YELLOW", "RED"} {
		if !strings.HasPrefix(colors[name], "0x") || len(colors[name]) != 10 {
			t.Errorf("%s = %q", name, colors[name])
		}
	}
}

// check: the SketchyBar `checks` item (sketchybarrc), connectivity checks.
//
// One check per line of $CONFIG_DIR/checks (`name | command`, per machine, not
// in dotfiles) or, without that file, every kube context in the kubeconfig.
// Runs them all in parallel (no stdin, 10 s timeout each); the label shows
// passed/total, red when any fails, hidden when there is nothing to check.
// Click toggles a popup with each check, failures first, with a short reason.
// `kube <context>` is shorthand for "the context's credentials are accepted":
// /api needs an authenticated user, unlike /healthz or /version.
//
// SketchyBar runs it as the item's script, with NAME, SENDER and CONFIG_DIR
// in the environment. Built by dotfiles-sync and the playbook.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"
)

const (
	timeout  = 10 * time.Second
	okIcon   = "" // nf-fa-check
	failIcon = "" // nf-fa-times
)

type check struct{ name, cmd string }

type result struct {
	check
	index    int // item checks.item.<index>: context names can hold / and : (EKS ARNs)
	code     int
	timedOut bool
	stderr   string
}

// bar is what the item needs from SketchyBar; sketchybarCLI is the real one,
// the tests record the calls.
type bar interface {
	run(args ...string) error
}

type sketchybarCLI struct{}

func (sketchybarCLI) run(args ...string) error { return exec.Command("sketchybar", args...).Run() }

func main() {
	// SketchyBar's PATH lacks Homebrew; /usr/local/bin: the official AWS CLI
	// installer puts `aws` there. Kept where they are when already on it.
	path := os.Getenv("PATH")
	for _, dir := range []string{"/usr/local/bin", "/opt/homebrew/bin"} {
		if !slices.Contains(filepath.SplitList(path), dir) {
			path = dir + ":" + path
		}
	}
	os.Setenv("PATH", path)
	err := run(sketchybarCLI{}, os.Getenv("NAME"), os.Getenv("SENDER"), os.Getenv("CONFIG_DIR"), timeout, time.Now())
	if err != nil {
		fmt.Fprintln(os.Stderr, "check:", err)
		os.Exit(1)
	}
}

func run(b bar, name, sender, dir string, limit time.Duration, now time.Time) error {
	switch sender {
	case "mouse.clicked":
		return b.run("--set", name, "popup.drawing=toggle")
	case "mouse.exited.global":
		return b.run("--set", name, "popup.drawing=off")
	}
	colors, err := readColors(filepath.Join(dir, "colors.sh"))
	if err != nil {
		return err
	}

	var checks []check
	if data, err := os.ReadFile(filepath.Join(dir, "checks")); err == nil {
		checks = parseChecks(string(data))
	}
	if len(checks) == 0 {
		checks = kubeChecks()
	}
	if len(checks) == 0 {
		return b.run("--set", name, "drawing=off")
	}

	// Exec credential plugins (kubelogin) open a browser when the login expired; a
	// fake `open` first in PATH keeps the bar from popping one every few minutes.
	bin, err := os.MkdirTemp("", "sketchybar-check")
	if err != nil {
		return err
	}
	defer os.RemoveAll(bin)
	if err := os.WriteFile(filepath.Join(bin, "open"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		return err
	}
	env := withPath(os.Environ(), bin+":"+os.Getenv("PATH"))

	if err := b.run("--set", name, "drawing=on", "label.color="+colors["YELLOW"]); err != nil {
		return err
	}
	results := runAll(checks, limit, env)

	_ = b.run("--remove", `/checks\.item\..*/`) // none yet on the first run
	if err := b.run(popup(name, results, colors, now)...); err != nil {
		return err
	}
	passed, color := 0, colors["GREEN"]
	for _, r := range results {
		if r.code == 0 {
			passed++
		} else {
			color = colors["RED"]
		}
	}
	return b.run("--set", name, fmt.Sprintf("label=%d/%d", passed, len(results)), "label.color="+color)
}

// parseChecks reads `name | command` lines: blanks in the name are dropped,
// # comments and empty names skipped, the command keeps any further |.
func parseChecks(text string) []check {
	var checks []check
	for _, line := range strings.Split(text, "\n") {
		name, cmd, _ := strings.Cut(line, "|")
		name = strings.Map(func(r rune) rune {
			if r == ' ' || r == '\t' {
				return -1
			}
			return r
		}, name)
		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}
		checks = append(checks, check{name, strings.TrimLeft(cmd, " \t")})
	}
	return checks
}

// kubeChecks: `kube <context>` for every context in the kubeconfig
func kubeChecks() []check {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return nil
	}
	out, err := exec.Command("kubectl", "config", "get-contexts", "-o", "name").Output()
	if err != nil {
		return nil
	}
	var lines []string
	for _, ctx := range strings.Fields(string(out)) {
		lines = append(lines, ctx+" | kube "+ctx)
	}
	return parseChecks(strings.Join(lines, "\n"))
}

func command(ctx context.Context, cmd string) *exec.Cmd {
	if kubeCtx, ok := strings.CutPrefix(cmd, "kube "); ok {
		return exec.CommandContext(ctx, "kubectl", "--context", kubeCtx, "get", "--raw", "/api", "--request-timeout=5s")
	}
	return exec.CommandContext(ctx, "bash", "-c", cmd)
}

// runAll runs every check at once, each in its own process group so that a
// timeout also stops what it started; results in the checks' order.
func runAll(checks []check, limit time.Duration, env []string) []result {
	results := make([]result, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), limit)
			defer cancel()
			cmd := command(ctx, c.cmd)
			var stderr bytes.Buffer
			cmd.Env, cmd.Stderr = env, &stderr // stdin and stdout: /dev/null
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
			cmd.WaitDelay = time.Second // a leftover child holding stderr open
			err := cmd.Run()
			results[i] = result{check: c, index: i, code: exitCode(err), timedOut: ctx.Err() != nil, stderr: stderr.String()}
		}()
	}
	wg.Wait()
	return results
}

// exitCode as a shell reports it: 128 + the signal for a killed process
func exitCode(err error) int {
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
		return exit.ExitCode()
	default:
		return 127 // not started (no bash, no kubectl)
	}
}

// Short reasons for a failed check, matched in this order in its stderr
var reasons = []struct {
	label string
	words []string
}{
	{"expired", []string{"Unauthorized", "401", "expired", "token"}},
	{"forbidden", []string{"Forbidden", "403"}},
	{"refused", []string{"refused"}},
	{"no context", []string{"does not exist"}},
	{"dns", []string{"no such host", "Could not resolve"}},
	{"timeout", []string{"timed out", "Timeout", "timeout"}},
}

func reason(r result) string {
	if r.timedOut {
		return "timeout"
	}
	stderr := strings.ReplaceAll(r.stderr, "\n", " ")
	for _, rule := range reasons {
		for _, w := range rule.words {
			if strings.Contains(stderr, w) {
				return rule.label
			}
		}
	}
	return fmt.Sprintf("exit %d", r.code)
}

// popup: the sketchybar arguments for the header and one row per check,
// failures first; EKS ARNs show only the cluster name.
func popup(name string, results []result, colors map[string]string, now time.Time) []string {
	sorted := slices.Clone(results)
	slices.SortStableFunc(sorted, func(a, b result) int {
		return boolInt(a.code == 0) - boolInt(b.code == 0)
	})
	width := 0
	for _, r := range results {
		width = max(width, utf8.RuneCountInString(short(r.name)))
	}
	nameWidth := fmt.Sprintf("icon.width=%d", width*8+40) // Hack Nerd Font is monospaced

	args := []string{"--add", "item", "checks.item.header", "popup." + name,
		"--set", "checks.item.header", "icon=Checks", "icon.font=SF Pro:Bold:13.0", "icon.color=" + colors["FG"],
		nameWidth, "label=" + now.Format("15:04"), "label.color=" + colors["DIM"],
		"label.font=SF Pro:Regular:12.0", "icon.padding_left=12", "label.padding_right=14",
		"background.height=28"}
	for _, r := range sorted {
		icon, color, label := okIcon, colors["GREEN"], "ok"
		if r.code != 0 {
			icon, color, label = failIcon, colors["RED"], reason(r)
		}
		item := fmt.Sprintf("checks.item.%d", r.index)
		args = append(args, "--add", "item", item, "popup."+name,
			"--set", item, "icon="+icon+"  "+short(r.name), "icon.color="+color,
			"icon.font=Hack Nerd Font:Bold:13.0", nameWidth,
			"label="+label, "label.color="+colors["DIM"],
			"label.font=SF Pro:Regular:12.0", "icon.padding_left=12", "label.padding_right=14")
	}
	return args
}

func short(name string) string { return name[strings.LastIndex(name, "/")+1:] }

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// readColors: the palette from colors.sh (`export NAME=0xAARRGGBB`), shared
// with the shell plugins
func readColors(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	colors := map[string]string{}
	for _, m := range colorLine.FindAllStringSubmatch(string(data), -1) {
		colors[m[1]] = m[2]
	}
	return colors, nil
}

var colorLine = regexp.MustCompile(`(?m)^(?:export\s+)?([A-Z_]+)=(0x[0-9a-fA-F]{8})\s*$`)

func withPath(env []string, path string) []string {
	out := slices.DeleteFunc(slices.Clone(env), func(kv string) bool { return strings.HasPrefix(kv, "PATH=") })
	return append(out, "PATH="+path)
}

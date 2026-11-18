// statusline: the Claude Code status line (settings.json "statusLine").
//
// One line, what the session needs to know at a glance:
//
//	dotfiles  main*  ⎈ prod-eu/api PROD │ Opus · ctx 42% · $1.23 · 5h 18% · PR #7
//	dir       branch  kube context        model  context  cost    rate limit   open PR
//	          (* = uncommitted changes)   (PROD in red: kube-guard's production regex)
//
// Claude Code runs it with the session as JSON on stdin and shows its first
// line. The Kubernetes context comes from kube-guard (`kube-guard context`),
// so "production" means the same here as in the shell and in the hook that
// asks before Claude changes it. Anything that is not there (no repository, no
// kube context, no cost yet, a field Claude Code did not send) is left out, and
// a tool that takes longer than a moment is left out too: the line must never
// hold the session up. Colors unless NO_COLOR is set.
//
// Always exits 0.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// How long git or kube-guard may take each; Claude Code debounces and cancels
// a run that is overtaken, but a slow tool is still not worth waiting for
const toolTimeout = 400 * time.Millisecond

type input struct {
	Cwd   string `json:"cwd"`
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
	Cost struct {
		TotalCostUSD float64 `json:"total_cost_usd"`
	} `json:"cost"`
	ContextWindow struct {
		UsedPercentage *float64 `json:"used_percentage"` // null early in the session
	} `json:"context_window"`
	RateLimits struct {
		FiveHour *struct {
			UsedPercentage *float64 `json:"used_percentage"`
		} `json:"five_hour"`
	} `json:"rate_limits"`
	PR *struct {
		Number int `json:"number"`
	} `json:"pr"`
}

// runner runs a command in dir and returns its output; a variable so the tests
// can stand in for git and kube-guard
type runner func(ctx context.Context, dir, name string, args ...string) (string, error)

func execRunner(ctx context.Context, dir, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}

type deps struct {
	run     runner
	kubeBin string // kube-guard
	colorOn bool
}

func main() {
	home, _ := os.UserHomeDir()
	kube := os.Getenv("KUBE_GUARD_BIN")
	if kube == "" {
		kube = filepath.Join(home, ".config/kube-guard/kube-guard")
	}
	d := deps{run: execRunner, kubeBin: kube, colorOn: os.Getenv("NO_COLOR") == ""}
	data, _ := io.ReadAll(os.Stdin)
	fmt.Println(render(data, d))
}

// render builds the line from the session JSON; "" when it cannot be read
func render(data []byte, d deps) string {
	var in input
	if json.Unmarshal(data, &in) != nil {
		return ""
	}
	dir := in.Workspace.CurrentDir
	if dir == "" {
		dir = in.Cwd
	}
	ctx := context.Background()

	var left, right []string
	if dir != "" {
		left = append(left, d.paint("1;34", filepath.Base(dir)))
		if g := d.git(ctx, dir); g != "" {
			left = append(left, g)
		}
	}
	if k := d.kube(ctx); k != "" {
		left = append(left, k)
	}

	if m := in.Model.DisplayName; m != "" {
		right = append(right, m)
	}
	if p := in.ContextWindow.UsedPercentage; p != nil {
		right = append(right, d.paint(level(*p), fmt.Sprintf("ctx %.0f%%", *p)))
	}
	if c := in.Cost.TotalCostUSD; c > 0 {
		right = append(right, fmt.Sprintf("$%.2f", c))
	}
	if f := in.RateLimits.FiveHour; f != nil && f.UsedPercentage != nil {
		right = append(right, d.paint(level(*f.UsedPercentage), fmt.Sprintf("5h %.0f%%", *f.UsedPercentage)))
	}
	if in.PR != nil && in.PR.Number > 0 {
		right = append(right, fmt.Sprintf("PR #%d", in.PR.Number))
	}

	sep := d.paint("2", " │ ")
	dot := d.paint("2", " · ")
	switch {
	case len(left) > 0 && len(right) > 0:
		return strings.Join(left, "  ") + sep + strings.Join(right, dot)
	case len(left) > 0:
		return strings.Join(left, "  ")
	}
	return strings.Join(right, dot)
}

// git: the branch of dir, with a * when files are modified or staged (untracked
// files are not looked for: slower, and noise); "" outside a repository
func (d deps) git(ctx context.Context, dir string) string {
	out, err := d.run(ctx, dir, "git", "branch", "--show-current")
	branch := strings.TrimSpace(out)
	if err != nil {
		return ""
	}
	if branch == "" { // detached
		sha, err := d.run(ctx, dir, "git", "rev-parse", "--short", "HEAD")
		if err != nil || strings.TrimSpace(sha) == "" {
			return ""
		}
		branch = strings.TrimSpace(sha)
	}
	text := d.paint("35", branch)
	if st, err := d.run(ctx, dir, "git", "status", "--porcelain", "--untracked-files=no"); err == nil && strings.TrimSpace(st) != "" {
		text += d.paint("33", "*")
	}
	return text
}

// kube: ⎈ context/namespace, and PROD in red on a production context, from
// `kube-guard context` (context, namespace and prod|other, one per line);
// "" when there is no context or no kube-guard
func (d deps) kube(ctx context.Context) string {
	out, err := d.run(ctx, "", d.kubeBin, "context")
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 3 || lines[0] == "" {
		return ""
	}
	name := lines[0]
	if lines[1] != "" {
		name += "/" + lines[1]
	}
	if lines[2] == "prod" {
		return d.paint("1;31", "⎈ "+name+" PROD")
	}
	return d.paint("36", "⎈ "+name)
}

// level is the color of a percentage that fills up: green, yellow from 70, red from 90
func level(p float64) string {
	switch {
	case p >= 90:
		return "31"
	case p >= 70:
		return "33"
	}
	return "32"
}

func (d deps) paint(code, s string) string {
	if !d.colorOn {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

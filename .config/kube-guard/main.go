// kube-guard: asks before a kubectl / helm command that changes a production
// cluster, and has k9s and sofka open read-only there. The zsh functions in
// .config/zsh/kube-guard.zsh call it before running the real command:
//
//	kube-guard check kubectl|helm ARGS...  exit 0: run it; else don't (answered no)
//	kube-guard tui k9s|sofka ARGS...       exit 0: open it read-only; 1: as asked
//	kube-guard context                     CONTEXT, NAMESPACE and prod|other on three
//	                                       lines, for the SketchyBar kube item (nothing: no context)
//
// Production: the context (--context / --kube-context, else the current one,
// of --kubeconfig when given) matches KUBE_PROD_REGEX, default prod|prd like
// the SketchyBar kube item, in any case. Reads (get, describe, logs, diff,
// rollout status, helm template...) pass without asking. KUBE_GUARD=0 skips it.
//
// The question is asked on the terminal (/dev/tty): stdin may be the manifest
// (kubectl apply -f -). Colors on a terminal, none with NO_COLOR.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/mattn/go-isatty"
)

const defaultProdRegex = "prod|prd"

var (
	red    = color.New(color.FgRed, color.Bold)
	yellow = color.New(color.FgYellow, color.Bold)
	faint  = color.New(color.Faint)
)

// setup turns colors on when stderr is a terminal and NO_COLOR is unset
// (fatih/color decides from stdout, which may be a pipe: kubectl get | grep)
func setup() {
	color.NoColor = !isatty.IsTerminal(os.Stderr.Fd()) || os.Getenv("NO_COLOR") != ""
}

// production says whether ctx matches KUBE_PROD_REGEX (any case)
func production(ctx string) (bool, error) {
	expr := os.Getenv("KUBE_PROD_REGEX")
	if expr == "" {
		expr = defaultProdRegex
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return false, fmt.Errorf("KUBE_PROD_REGEX %q: %v", expr, err)
	}
	return ctx != "" && re.MatchString(strings.ToLower(ctx)), nil
}

// contextOf is the context the line runs on
func contextOf(l line) string {
	if l.context != "" {
		return l.context
	}
	home, _ := os.UserHomeDir()
	return currentContext(l.kubeconfig, os.Getenv("KUBECONFIG"), home)
}

func isPlugin(word string) bool {
	_, err := exec.LookPath("kubectl-" + strings.ReplaceAll(word, "-", "_"))
	return err == nil
}

// confirm asks y/N on the terminal; anything but y (or no terminal) is no
func confirm(label string) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	defer tty.Close()
	p := promptui.Prompt{Label: label, IsConfirm: true, Stdin: tty, Stdout: nopCloser{tty}}
	if color.NoColor { // promptui colors its own templates
		plain := "{{ . }}? [y/N] "
		p.Templates = &promptui.PromptTemplates{Prompt: plain, Confirm: plain, Valid: plain, Invalid: plain, Success: "{{ . }}: "}
	}
	_, err = p.Run()
	return err == nil
}

// nopCloser: promptui closes its Stdout when done, the tty is closed above
type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

// check: exit 0 when the command may run
func check(tool string, args []string, ask func(string) bool, out io.Writer) int {
	if os.Getenv("KUBE_GUARD") == "0" {
		return 0
	}
	l := parse(tool, args, isPlugin)
	if !l.changes(tool) {
		return 0
	}
	ctx := contextOf(l)
	prod, err := production(ctx)
	if err != nil {
		fmt.Fprintln(out, red.Sprint("kube-guard:"), err)
		return 2
	}
	if !prod {
		return 0
	}
	label := fmt.Sprintf("kube-guard: %s is production. %s %s", red.Sprint(ctx), tool, strings.Join(args, " "))
	if ask(label) {
		return 0
	}
	fmt.Fprintln(out, yellow.Sprint("kube-guard:"), "not run")
	return 1
}

// tui: exit 0 when the TUI should open read-only (said on out), 1 when as asked
func tui(tool string, args []string, out io.Writer) int {
	if os.Getenv("KUBE_GUARD") == "0" || slices.Contains(args, "--write") || slices.Contains(args, "--readonly") {
		return 1
	}
	ctx := contextOf(parse(tool, args, nil))
	prod, err := production(ctx)
	if err != nil {
		fmt.Fprintln(out, red.Sprint("kube-guard:"), err, faint.Sprint("(read-only)"))
		return 0
	}
	if !prod {
		return 1
	}
	fmt.Fprintf(out, "%s %s is production, %s read-only %s\n", yellow.Sprint("kube-guard:"),
		red.Sprint(ctx), tool, faint.Sprintf("(%s --write to change things)", tool))
	return 0
}

// kubeContext prints the current context, its namespace and whether it is
// production, one per line (a shell reads an empty namespace line as it is,
// where it would merge two tabs); nothing when there is no current context. Faster
// than kubectl config current-context + view --minify, which the bar ran every 5 s
func kubeContext(out io.Writer) int {
	home, _ := os.UserHomeDir()
	env := os.Getenv("KUBECONFIG")
	ctx := currentContext("", env, home)
	if ctx == "" {
		return 0
	}
	prod, err := production(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kube-guard:", err)
		return 2
	}
	kind := "other"
	if prod {
		kind = "prod"
	}
	fmt.Fprintf(out, "%s\n%s\n%s\n", ctx, namespace(ctx, "", env, home), kind)
	return 0
}

func run(args []string) int {
	setup()
	if len(args) == 1 && args[0] == "context" {
		return kubeContext(os.Stdout)
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kube-guard check kubectl|helm ARGS... | kube-guard tui TOOL ARGS... | kube-guard context")
		return 2
	}
	mode, tool, rest := args[0], args[1], args[2:]
	switch {
	case mode == "check" && (tool == "kubectl" || tool == "helm"):
		return check(tool, rest, confirm, os.Stderr)
	case mode == "tui":
		return tui(tool, rest, os.Stderr)
	}
	fmt.Fprintln(os.Stderr, errors.New("kube-guard: unknown mode or tool: "+mode+" "+tool))
	return 2
}

func main() { os.Exit(run(os.Args[1:])) }

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
// of --kubeconfig when given) matches the production regex, in any case:
// KUBE_PROD_REGEX, else the first line of ~/.config/kube-guard/prod-regex (per
// machine, never committed: what launchd and SketchyBar see too, which never
// read ~/.zshrc.local), else prod|prd. Reads (get, describe, logs, diff,
// rollout status, helm template...) pass without asking. KUBE_GUARD=0 skips it.
//
// The question (y/N, then Enter) is asked on the terminal (/dev/tty): stdin may be the manifest
// (kubectl apply -f -). Colors on a terminal, none with NO_COLOR.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

const defaultProdRegex = "prod|prd"

// prodRegexFile is the per-machine production regex, under the home directory
const prodRegexFile = ".config/kube-guard/prod-regex"

var (
	blue   = color.New(color.FgBlue, color.Bold)
	red    = color.New(color.FgRed, color.Bold)
	yellow = color.New(color.FgYellow, color.Bold)
	faint  = color.New(color.Faint)
)

// setup turns colors on when stderr is a terminal and NO_COLOR is unset
// (fatih/color decides from stdout, which may be a pipe: kubectl get | grep)
func setup() {
	color.NoColor = !isatty.IsTerminal(os.Stderr.Fd()) || os.Getenv("NO_COLOR") != ""
}

// prodRegex is the production regex and where it comes from: KUBE_PROD_REGEX,
// else the first line of ~/.config/kube-guard/prod-regex that is neither blank
// nor a # comment, else the default
func prodRegex() (expr, from string) {
	if expr := os.Getenv("KUBE_PROD_REGEX"); expr != "" {
		return expr, "KUBE_PROD_REGEX"
	}
	home, _ := os.UserHomeDir()
	if data, err := os.ReadFile(filepath.Join(home, prodRegexFile)); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
				return line, "~/" + prodRegexFile
			}
		}
	}
	return defaultProdRegex, "the default"
}

// production says whether ctx matches the production regex (any case)
func production(ctx string) (bool, error) {
	expr, from := prodRegex()
	re, err := regexp.Compile(expr)
	if err != nil {
		return false, fmt.Errorf("%s %q: %v", from, expr, err)
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

// confirm asks y/N on the terminal; anything but y / yes (or no terminal) is no
func confirm(label string) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	defer tty.Close()
	return ask(tty, tty, label)
}

// ask writes the question to out and reads one line of answer from in
//
//	? kube-guard: prod-eu is production. kubectl delete pod x? [y/N]
func ask(in io.Reader, out io.Writer, label string) bool {
	fmt.Fprintf(out, "%s %s? %s ", blue.Sprint("?"), label, faint.Sprint("[y/N]"))
	answer, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	}
	return false
}

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

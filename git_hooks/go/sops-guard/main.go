// sops-guard – global pre-commit hook (hook.sops-guard in .gitconfig-hooks).
//
// Stops the commit of a file sops should have encrypted but that is staged in
// cleartext. Such a file is one named *.sops.* / *.enc.* (secrets.sops.yaml,
// app.enc.env...) or one a path_regex of a staged .sops.yaml matches: the one
// at the root or any in a folder above the file, each matched against the
// path relative to its own folder. The regexes are Go's, as sops reads them
// (RE2: \d, (?i)...), not POSIX ERE. It is encrypted when it carries sops's MAC
// (mac: ENC[...], "mac": "ENC[...", sops_mac=ENC[...). Everything is read as
// staged. Needs no sops. Skip once with `git commit --no-verify`.
//
// Colors and emojis on a terminal; plain text elsewhere (lazygit, VS Code, CI)
// and with NO_COLOR set.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

const configName = ".sops.yaml"

var (
	// a creation rule's path_regex line, list item or not
	pathRegexLine = regexp.MustCompile(`^[\s-]*path_regex:\s*(.*)$`)
	// sops's MAC in YAML, JSON, dotenv and INI
	macRe = regexp.MustCompile(`mac"?\s*[:=]\s*"?ENC\[`)
)

// rule is a path_regex of the .sops.yaml in dir ("." for the repo's root)
type rule struct {
	dir string
	re  *regexp.Regexp
}

// finding is a staged file that should be encrypted and is not, and why
type finding struct {
	path, reason string
}

// The styles of the output. fatih/color decides from stdout; the hook writes
// to stderr, so setup decides from that
var (
	red    = color.New(color.FgRed, color.Bold)
	yellow = color.New(color.FgYellow)
	cyan   = color.New(color.FgCyan)
	faint  = color.New(color.Faint)
	emoji  = false
)

// setup turns colors and emojis on when stderr is a terminal and NO_COLOR is unset
func setup() {
	tty := isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())
	color.NoColor = !tty || os.Getenv("NO_COLOR") != ""
	emoji = !color.NoColor
}

// icon is the emoji and a space on a terminal, else nothing
func icon(e string) string {
	if emoji {
		return e + " "
	}
	return ""
}

// warnf prints a warning that doesn't stop the commit
func warnf(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "%s%s %s\n", icon("⚠️ "), yellow.Sprint("sops-guard:"), fmt.Sprintf(format, a...))
}

// git runs git and returns its stdout
func git(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %v: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// staged reads a file's staged content
func staged(p string) ([]byte, error) { return git("show", ":"+p) }

// splitZ splits git's -z output
func splitZ(out []byte) []string {
	var paths []string
	for _, p := range bytes.Split(out, []byte{0}) {
		if len(p) > 0 {
			paths = append(paths, string(p))
		}
	}
	return paths
}

// pathRegexes returns the path_regex values of a .sops.yaml, unquoted as YAML
// does. Only the creation rules' path_regex is needed, so the file is read line
// by line rather than with a YAML library
func pathRegexes(config string) []string {
	var values []string
	for _, line := range strings.Split(config, "\n") {
		m := pathRegexLine.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		if v := yamlScalar(m[1]); v != "" {
			values = append(values, v)
		}
	}
	return values
}

// yamlScalar unquotes a one-line YAML scalar and drops a trailing comment.
// 'single': ” is a quote. "double": \\ and \" (and \/, \t); any other escape
// is kept as written, so "a\.yaml" stays a\.yaml. plain: up to " #"
func yamlScalar(s string) string {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "'"):
		var b strings.Builder
		for i := 1; i < len(s); i++ {
			if s[i] == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					b.WriteByte('\'')
					i++
					continue
				}
				return b.String()
			}
			b.WriteByte(s[i])
		}
		return b.String() // unterminated: what there is
	case strings.HasPrefix(s, `"`):
		var b strings.Builder
		for i := 1; i < len(s); i++ {
			switch c := s[i]; {
			case c == '"':
				return b.String()
			case c == '\\' && i+1 < len(s):
				i++
				switch s[i] {
				case '\\', '"', '/':
					b.WriteByte(s[i])
				case 't':
					b.WriteByte('\t')
				default:
					b.WriteByte('\\')
					b.WriteByte(s[i])
				}
			default:
				b.WriteByte(c)
			}
		}
		return b.String()
	default:
		if i := strings.Index(s, " #"); i >= 0 {
			s = s[:i]
		}
		if strings.HasPrefix(s, "#") {
			return ""
		}
		return strings.TrimSpace(s)
	}
}

// loadRules reads the path_regex of every staged .sops.yaml. A regex Go can't
// compile is reported and skipped: sops itself would refuse that config
func loadRules(configs []string, read func(string) ([]byte, error), warn io.Writer) []rule {
	var rules []rule
	for _, c := range configs {
		data, err := read(c)
		if err != nil {
			warnf(warn, "%s: %v", c, err)
			continue
		}
		for _, v := range pathRegexes(string(data)) {
			re, err := regexp.Compile(v)
			if err != nil {
				warnf(warn, "%s: path_regex %q: %v (skipped)", c, v, err)
				continue
			}
			rules = append(rules, rule{dir: path.Dir(c), re: re})
		}
	}
	return rules
}

// whyEncrypt says why p should be encrypted ("" when it needn't be): named
// *.sops.* / *.enc.*, or matched by the path_regex of a .sops.yaml in one of
// its folders, relative to that folder
func whyEncrypt(p string, rules []rule) string {
	base := path.Base(p)
	for _, n := range []string{".sops.", ".enc."} {
		if strings.Contains(base, n) {
			return "named *" + n + "*"
		}
	}
	for _, r := range rules {
		rel := p
		if r.dir != "." {
			var ok bool
			if rel, ok = strings.CutPrefix(p, r.dir+"/"); !ok {
				continue // a .sops.yaml in another folder
			}
		}
		if r.re.MatchString(rel) {
			return fmt.Sprintf("path_regex %s of %s", r.re, path.Join(r.dir, configName))
		}
	}
	return ""
}

func encrypted(content []byte) bool { return macRe.Match(content) }

// check returns the staged files that should be encrypted and are not
func check(stagedFiles, configs []string, read func(string) ([]byte, error), warn io.Writer) ([]finding, error) {
	rules := loadRules(configs, read, warn)
	var bad []finding
	for _, f := range stagedFiles {
		if path.Base(f) == configName {
			continue
		}
		reason := whyEncrypt(f, rules)
		if reason == "" {
			continue
		}
		content, err := read(f)
		if err != nil {
			return nil, err
		}
		if !encrypted(content) {
			bad = append(bad, finding{f, reason})
		}
	}
	return bad, nil
}

// report prints the findings:
//
//	🔒 sops-guard: 2 files sops should encrypt staged in cleartext
//	   ✗ k8s/db.yaml   path_regex k8s/.*\.yaml$ of .sops.yaml
//	   ✗ app.sops.env  named *.sops.*
//	💡 sops encrypt -i FILE, then git add FILE · skip once: git commit --no-verify
func report(w io.Writer, bad []finding) {
	files := "files"
	if len(bad) == 1 {
		files = "file"
	}
	fmt.Fprintf(w, "%s%s %d %s sops should encrypt staged in cleartext\n",
		icon("🔒"), red.Sprint("sops-guard:"), len(bad), files)
	width := 0
	for _, f := range bad {
		width = max(width, len(f.path))
	}
	for _, f := range bad {
		fmt.Fprintf(w, "   %s %-*s  %s\n", red.Sprint("✗"), width, f.path, faint.Sprint(f.reason))
	}
	fmt.Fprintf(w, "%s%s, then %s %s %s\n", icon("💡"),
		cyan.Sprint("sops encrypt -i FILE"), cyan.Sprint("git add FILE"),
		faint.Sprint("· skip once:"), cyan.Sprint("git commit --no-verify"))
}

// fail prints an error of the hook itself
func fail(err error) int {
	fmt.Fprintf(os.Stderr, "%s%s %v\n", icon("❌"), red.Sprint("sops-guard:"), err)
	return 1
}

func run() int {
	setup()
	out, err := git("diff", "--cached", "--name-only", "-z", "--diff-filter=ACMR")
	if err != nil {
		return fail(err)
	}
	stagedFiles := splitZ(out)
	if len(stagedFiles) == 0 {
		return 0
	}
	// every .sops.yaml in the index: one not changed in this commit counts too
	out, err = git("ls-files", "-z", "--cached", "--", configName, "*/"+configName)
	if err != nil {
		return fail(err)
	}
	bad, err := check(stagedFiles, splitZ(out), staged, os.Stderr)
	if err != nil {
		return fail(err)
	}
	if len(bad) == 0 {
		return 0
	}
	report(os.Stderr, bad)
	return 1
}

func main() { os.Exit(run()) }

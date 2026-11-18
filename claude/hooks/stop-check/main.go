// stop-check: Claude Code Stop hook.
//
// The PreToolUse guards stop what Claude must not do; this checks what it
// leaves behind. When Claude is about to say it is done, the files changed in
// the working tree (git status, untracked included) are checked, and if a
// check fails Claude is told why and carries on instead of stopping:
//
//	*.go            go vet ./... and go test ./... in the module of each changed file
//	*.sh            shellcheck, or bash -n when there is no shellcheck
//	*.zsh           zsh -n
//	*.json          valid JSON (not under vscode/: those are JSONC)
//
// It does not loop: when Claude has already been sent back once
// (stop_hook_active) it lets it stop. The same state is not checked twice: a
// fingerprint of the changed files (path, size, time) is kept per repository,
// so a turn that only answers a question costs nothing and a failure is
// reported once, not on every turn. A check that outlives the time budget is
// skipped, never reported as a failure.
//
// STOP_CHECK=0 switches it off (the user's switch, in the environment Claude
// Code was started from). Outside a git repository it does nothing.
//
// Reads the hook event (JSON) on stdin; prints a block decision, or nothing to
// let Claude stop. Always exits 0.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// How long all the checks may take together; the hook's own timeout in
// settings.json is a little longer
const budget = 100 * time.Second

// The longest report kept per failed check, and in all
const (
	maxLines  = 25
	maxReport = 4000
)

type event struct {
	Cwd            string `json:"cwd"`
	StopHookActive bool   `json:"stop_hook_active"`
}

// runner runs a command in dir and returns its combined output; a variable so
// the tests can stand in for the tools
type runner func(ctx context.Context, dir, name string, args ...string) (string, error)

func execRunner(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

type hook struct {
	run      runner
	has      func(tool string) bool // is the tool installed
	cacheDir string
}

func main() {
	cache, _ := os.UserCacheDir()
	h := hook{
		run:      execRunner,
		has:      func(tool string) bool { _, err := exec.LookPath(tool); return err == nil },
		cacheDir: filepath.Join(cache, "stop-check"),
	}
	run(os.Stdin, os.Stdout, h)
}

func run(in io.Reader, out io.Writer, h hook) {
	data, _ := io.ReadAll(in)
	var ev event
	if json.Unmarshal(data, &ev) != nil || ev.StopHookActive || os.Getenv("STOP_CHECK") == "0" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	if report := h.check(ctx, ev.Cwd); report != "" {
		resp, _ := json.Marshal(map[string]string{"decision": "block", "reason": report})
		fmt.Fprintln(out, string(resp))
	}
}

// check returns what to tell Claude, or "" to let it stop
func (h hook) check(ctx context.Context, cwd string) string {
	if cwd == "" {
		return ""
	}
	top, err := h.run(ctx, cwd, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(top)
	status, err := h.run(ctx, root, "git", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return ""
	}
	files := changedFiles(status)
	if len(files) == 0 {
		return ""
	}
	if !h.fresh(root, files) {
		return "" // this exact state was checked already
	}

	var failures []string
	add := func(title, output string) {
		failures = append(failures, "### "+title+"\n"+tail(output))
	}
	for _, dir := range goModules(root, files) {
		if !h.has("go") {
			break
		}
		if out, err := h.run(ctx, filepath.Join(root, dir), "go", "vet", "./..."); err != nil && ctx.Err() == nil {
			add("go vet in "+shown(dir), out)
		} else if out, err := h.run(ctx, filepath.Join(root, dir), "go", "test", "./..."); err != nil && ctx.Err() == nil {
			add("go test in "+shown(dir), out)
		}
	}
	for _, f := range files {
		switch ext := filepath.Ext(f); {
		case ext == ".sh":
			tool, args := "bash", []string{"-n", f}
			if h.has("shellcheck") {
				tool, args = "shellcheck", []string{f}
			}
			if out, err := h.run(ctx, root, tool, args...); err != nil && ctx.Err() == nil {
				add(tool+" "+f, out)
			}
		case ext == ".zsh" && h.has("zsh"):
			if out, err := h.run(ctx, root, "zsh", "-n", f); err != nil && ctx.Err() == nil {
				add("zsh -n "+f, out)
			}
		case ext == ".json" && !strings.Contains(f, "vscode/"):
			if err := validJSON(filepath.Join(root, f)); err != nil {
				add("JSON "+f, err.Error())
			}
		}
	}
	if ctx.Err() != nil || len(failures) == 0 {
		return "" // out of time: never a failure
	}
	report := "The checks on the files you changed failed. Fix them before you stop (STOP_CHECK=0 in the environment switches this hook off; the user's call, not yours):\n\n" +
		strings.Join(failures, "\n\n")
	if len(report) > maxReport {
		report = report[:maxReport] + "\n[cut]"
	}
	return report
}

// changedFiles reads `git status --porcelain=v1 -z`: "XY path\0", and a rename
// or copy adds the old path as one more entry. Deleted files are not checked.
func changedFiles(status string) []string {
	var files []string
	entries := strings.Split(status, "\x00")
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if len(e) < 4 {
			continue
		}
		x, y, path := e[0], e[1], e[3:]
		if x == 'R' || x == 'C' || y == 'R' || y == 'C' {
			i++ // the old path
		}
		if x == 'D' || y == 'D' {
			continue
		}
		files = append(files, path)
	}
	sort.Strings(files)
	return files
}

// goModules: the directories (relative to root) of the modules the changed Go
// files belong to, each once
func goModules(root string, files []string) []string {
	seen := map[string]bool{}
	var mods []string
	for _, f := range files {
		base := filepath.Base(f)
		if filepath.Ext(f) != ".go" && base != "go.mod" && base != "go.sum" {
			continue
		}
		dir := filepath.Dir(f)
		for {
			if _, err := os.Stat(filepath.Join(root, dir, "go.mod")); err == nil {
				if !seen[dir] {
					seen[dir] = true
					mods = append(mods, dir)
				}
				break
			}
			if dir == "." || dir == "/" {
				break
			}
			dir = filepath.Dir(dir)
		}
	}
	sort.Strings(mods)
	return mods
}

func shown(dir string) string {
	if dir == "." {
		return "the repository root"
	}
	return dir
}

func validJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // gone since the status: nothing to check
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return errors.New(err.Error())
	}
	return nil
}

// tail keeps the last lines of a tool's output: where its verdict is
func tail(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > maxLines {
		lines = append([]string{"[...]"}, lines[len(lines)-maxLines:]...)
	}
	return strings.Join(lines, "\n")
}

// fresh says whether this state of the changed files was not checked before,
// and records it either way. Without a cache directory it is always fresh.
func (h hook) fresh(root string, files []string) bool {
	sum := sha256.New()
	for _, f := range files {
		st, err := os.Stat(filepath.Join(root, f))
		if err != nil {
			continue
		}
		fmt.Fprintf(sum, "%s\x00%d\x00%d\x00", f, st.Size(), st.ModTime().UnixNano())
	}
	state := fmt.Sprintf("%x", sum.Sum(nil))
	if h.cacheDir == "" {
		return true
	}
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(root)))[:16]
	path := filepath.Join(h.cacheDir, key)
	if last, err := os.ReadFile(path); err == nil && string(last) == state {
		return false
	}
	if os.MkdirAll(h.cacheDir, 0o700) == nil {
		_ = os.WriteFile(path, []byte(state), 0o600)
	}
	return true
}

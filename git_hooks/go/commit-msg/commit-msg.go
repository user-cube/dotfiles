// commit-msg – Git hook
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	validPrefixes = "feat|fix|chore|docs|style|refactor|perf|test|build|ci"
	// "fix(api)!: rest" → type, scope, bang, rest
	prefixedRe = regexp.MustCompile(`(?i)^(` + validPrefixes + `)(\([^)]*\))?(!)?:\s*(.*)$`)
	// "fix rest" → "fix: rest"
	bareRe  = regexp.MustCompile(`(?i)^(` + validPrefixes + `)\s+(.*)$`)
	issueRe = regexp.MustCompile(`(?i)[A-Z]+-\d+`)
)

// Messages git writes itself (or autosquash relies on): left as they are
var passthroughPrefixes = []string{"fixup!", "squash!", "amend!", "merge ", "revert \""}

// The per-repo settings (YAML): ~/.config/git/commit-msg.yaml, or the
// ~/.gitconfig-hook it was before, while that one is not moved yet
var settingsPaths = []string{".config/git/commit-msg.yaml", ".gitconfig-hook"}

// settingsFile is the first of settingsPaths under home that exists, or ""
func settingsFile(home string) string {
	for _, p := range settingsPaths {
		if _, err := os.Stat(filepath.Join(home, p)); err == nil {
			return filepath.Join(home, p)
		}
	}
	return ""
}

// loadDefaultAction returns the default_action for the given repo path from
// the settings file. Falls back to "feat" if the repo is not listed.
func loadDefaultAction(repoPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "feat"
	}
	file := settingsFile(home)
	if file == "" {
		return "feat"
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return "feat"
	}
	return actionFor(string(data), repoPath)
}

// actionFor reads the default_action of repoPath from the settings' YAML
func actionFor(data, repoPath string) string {
	var currentPath string
	for _, line := range strings.Split(data, "\n") {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, "- path:"); ok {
			currentPath = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(trimmed, "path:"); ok {
			currentPath = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(trimmed, "default_action:"); ok {
			if currentPath == repoPath {
				return strings.TrimSpace(after)
			}
		}
	}
	return "feat"
}

func git(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(out)), err
}

// gitPathExists reports whether a file inside $GIT_DIR exists (works in worktrees)
func gitPathExists(name string) bool {
	p, err := git("rev-parse", "--git-path", name)
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// currentBranch returns the checked-out branch, or the branch being rebased
// (HEAD is detached during `rebase -i` reword/squash). ok is false when there
// is no branch at all (plain detached HEAD).
func currentBranch() (branch string, ok bool) {
	if b, err := git("symbolic-ref", "--short", "HEAD"); err == nil {
		return b, true
	}
	for _, name := range []string{"rebase-merge/head-name", "rebase-apply/head-name"} {
		p, err := git("rev-parse", "--git-path", name)
		if err != nil {
			continue
		}
		if data, err := os.ReadFile(p); err == nil {
			return strings.TrimPrefix(strings.TrimSpace(string(data)), "refs/heads/"), true
		}
	}
	return "", false
}

// issueKey extracts ABC-123 from the last path element of the branch
// (features/PROJECT-1234-foo → PROJECT-1234), or falls back to that element.
func issueKey(branch string) string {
	lastSegment := branch[strings.LastIndex(branch, "/")+1:]
	if key := issueRe.FindString(lastSegment); key != "" {
		return strings.ToUpper(key)
	}
	return lastSegment
}

// tagged reports whether s already starts with the issue key (amend, reword)
func tagged(s, key string) bool {
	s, key = strings.ToUpper(s), strings.ToUpper(key)
	return s == key || strings.HasPrefix(s, key+" ")
}

// formatHeader rewrites the first line of the commit message:
//
//	fix something        → fix: ISSUE something
//	something            → <defaultAction>: ISSUE something
//	no-prefix: something → ISSUE something
//	no-track: something  → something
//
// Headers that already carry the key, and fixup!/squash!/Merge/Revert
// messages, are returned unchanged.
func formatHeader(header, key, defaultAction string) string {
	lowered := strings.ToLower(header)
	if rest, ok := strings.CutPrefix(lowered, "no-track:"); ok {
		return strings.TrimSpace(header[len(header)-len(rest):])
	}
	if rest, ok := strings.CutPrefix(lowered, "no-prefix:"); ok {
		rest = strings.TrimSpace(header[len(header)-len(rest):])
		if tagged(rest, key) {
			return rest
		}
		return key + " " + rest
	}
	for _, p := range passthroughPrefixes {
		if strings.HasPrefix(lowered, p) {
			return header
		}
	}
	if defaultAction == "disabled" || tagged(header, key) {
		return header
	}

	if m := prefixedRe.FindStringSubmatch(header); m != nil {
		if tagged(m[4], key) {
			return header
		}
		return fmt.Sprintf("%s%s%s: %s %s", m[1], m[2], m[3], key, m[4])
	}
	if m := bareRe.FindStringSubmatch(header); m != nil {
		return fmt.Sprintf("%s: %s %s", strings.ToLower(m[1]), key, m[2])
	}
	return fmt.Sprintf("%s: %s %s", defaultAction, key, header)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Commit message file not found.")
		os.Exit(1)
	}
	commitMsgFile := os.Args[1]

	// Merges, conflicted cherry-picks and reverts keep the message git wrote
	for _, name := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD"} {
		if gitPathExists(name) {
			return
		}
	}

	branch, ok := currentBranch()
	if !ok {
		return // detached HEAD outside a rebase: nothing to take the key from
	}

	// Block commits on protected branches
	disallowed := map[string]bool{"main": true, "master": true, "develop": true}
	if disallowed[strings.ToLower(branch)] {
		fmt.Println("Branch not allowed. The disallowed branches are: main, master, develop")
		os.Exit(1)
	}

	repoPath, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		fmt.Println("Error: Unable to determine the repository path.")
		os.Exit(1)
	}
	defaultAction := loadDefaultAction(repoPath)

	data, err := os.ReadFile(commitMsgFile)
	if err != nil {
		fmt.Println("Error: Unable to read the commit message file.")
		os.Exit(1)
	}

	// Only the header changes; the body and git's comments are kept as they are
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		header := strings.TrimSpace(line)
		if header == "" || strings.HasPrefix(header, "#") {
			continue
		}
		lines[i] = formatHeader(header, issueKey(branch), defaultAction)
		break
	}
	// No header: the message stays empty and git aborts the commit as usual

	if err := os.WriteFile(commitMsgFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		fmt.Println("Error: Unable to write to the commit message file.")
		os.Exit(1)
	}
}

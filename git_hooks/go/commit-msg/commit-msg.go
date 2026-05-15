// commit-msg – Git hook
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// loadDefaultAction reads ~/.gitconfig-hook and returns the default_action for
// the given repo path. Falls back to "feat" if the repo is not listed.
func loadDefaultAction(repoPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "feat"
	}
	data, err := os.ReadFile(filepath.Join(home, ".gitconfig-hook"))
	if err != nil {
		return "feat"
	}

	var currentPath string
	for _, line := range strings.Split(string(data), "\n") {
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Commit message file not found.")
		os.Exit(1)
	}
	commitMsgFile := os.Args[1]

	//---------------------------------------------------------------------------
	// 1.  Current branch
	//---------------------------------------------------------------------------
	branchCmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	branchBytes, err := branchCmd.Output()
	if err != nil {
		fmt.Println("Error: Unable to determine the branch name.")
		os.Exit(1)
	}
	branchFull := strings.TrimSpace(string(branchBytes))

	// Block commits on protected branches
	disallowed := map[string]bool{"main": true, "master": true, "develop": true}
	if disallowed[strings.ToLower(branchFull)] {
		fmt.Println("Branch not allowed. The disallowed branches are: main, master, develop")
		os.Exit(1)
	}

	// Last path element, e.g. features/PROJECT-1234-foo → PROJECT-1234-foo
	lastSegment := branchFull[strings.LastIndex(branchFull, "/")+1:]

	// Extract ABC-123 numeric key (case-insensitive)
	issueRe := regexp.MustCompile(`(?i)[A-Z]+-\d+`)
	issueKey := strings.ToUpper(issueRe.FindString(lastSegment))
	if issueKey == "" {
		issueKey = lastSegment // fall back if no match
	}

	//---------------------------------------------------------------------------
	// 1b. Load default action from ~/.gitconfig-hook
	//---------------------------------------------------------------------------
	repoPathCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	repoPathBytes, err := repoPathCmd.Output()
	if err != nil {
		fmt.Println("Error: Unable to determine the repository path.")
		os.Exit(1)
	}
	repoPath := strings.TrimSpace(string(repoPathBytes))
	defaultAction := loadDefaultAction(repoPath)

	//---------------------------------------------------------------------------
	// 2.  Read the existing commit message
	//---------------------------------------------------------------------------
	file, err := os.Open(commitMsgFile)
	if err != nil {
		fmt.Println("Error: Unable to read the commit message file.")
		os.Exit(1)
	}
	defer file.Close()

	var messageLines, commentLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || (len(messageLines) > 0 && line == "") {
			commentLines = append(commentLines, line)
		} else {
			messageLines = append(messageLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Error: Failed to read commit message.")
		os.Exit(1)
	}
	if len(messageLines) == 0 {
		messageLines = []string{""}
	}
	originalMsg := strings.TrimSpace(messageLines[0])

	//---------------------------------------------------------------------------
	// 3.  Detect mode keywords
	//      no-prefix: → issue key only, no semantic prefix  → ISSUE something
	//      no-track:  → bypass everything                   → something
	//---------------------------------------------------------------------------
	noPrefix := false
	noTrack := false
	lowered := strings.ToLower(originalMsg)
	switch {
	case strings.HasPrefix(lowered, "no-prefix:"):
		noPrefix = true
		originalMsg = strings.TrimSpace(originalMsg[len("no-prefix:"):])
	case strings.HasPrefix(lowered, "no-track:"):
		noTrack = true
		originalMsg = strings.TrimSpace(originalMsg[len("no-track:"):])
	default:
		// Apply repo-level default_action if no explicit keyword
		switch defaultAction {
		case "disabled":
			noTrack = true
		}
	}

	//---------------------------------------------------------------------------
	// 4.  Add / normalise Conventional-Commit prefix (normal mode only)
	//---------------------------------------------------------------------------
	msgToUse := originalMsg
	if !noPrefix && !noTrack {
		validPrefixes := []string{
			"feat", "fix", "chore", "docs", "style",
			"refactor", "perf", "test", "build", "ci",
		}
		loweredMsg := strings.ToLower(originalMsg)
		valid := false
		for _, p := range validPrefixes {
			if strings.HasPrefix(loweredMsg, p+":") || strings.HasPrefix(loweredMsg, p+"(") {
				valid = true
				break
			} else if strings.HasPrefix(loweredMsg, p+" ") {
				rest := strings.TrimSpace(originalMsg[len(p):])
				msgToUse = fmt.Sprintf("%s: %s", p, rest)
				valid = true
				break
			}
		}
		if !valid {
			msgToUse = fmt.Sprintf("%s: %s", defaultAction, originalMsg)
		}
	}

	//---------------------------------------------------------------------------
	// 5.  Assemble final header
	//---------------------------------------------------------------------------
	var finalHeader string
	switch {
	case noTrack:
		finalHeader = msgToUse
	case noPrefix:
		finalHeader = fmt.Sprintf("%s %s", issueKey, msgToUse)
	default:
		// msgToUse is "prefix: rest" → "prefix: ISSUE rest"
		parts := strings.SplitN(msgToUse, ": ", 2)
		if len(parts) == 2 {
			finalHeader = fmt.Sprintf("%s: %s %s", parts[0], issueKey, parts[1])
		} else {
			finalHeader = fmt.Sprintf("%s %s", issueKey, msgToUse)
		}
	}

	//---------------------------------------------------------------------------
	// 6.  Write back to the commit message file
	//---------------------------------------------------------------------------
	allLines := append([]string{finalHeader}, commentLines...)
	output := strings.Join(allLines, "\n") + "\n"
	if err := os.WriteFile(commitMsgFile, []byte(output), 0644); err != nil {
		fmt.Println("Error: Unable to write to the commit message file.")
		os.Exit(1)
	}
}

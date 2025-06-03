// commit-msg – Git hook
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

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

	// Last path element, e.g. features/PE-1234-foo → PE-1234-foo
	lastSegment := branchFull[strings.LastIndex(branchFull, "/")+1:]

	// Extract ABC-123 numeric key (case-insensitive)
	issueRe := regexp.MustCompile(`(?i)[A-Z]+-\d+`)
	issueKey := strings.ToUpper(issueRe.FindString(lastSegment))
	if issueKey == "" {
		issueKey = lastSegment // fall back if no match
	}

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
	// 3.  Detect “disabled” to bypass Conventional-Commit prefix
	//---------------------------------------------------------------------------
	disableSemantic := false
	lowered := strings.ToLower(originalMsg)
	switch {
	case strings.HasPrefix(lowered, "disabled:"):
		disableSemantic = true
		originalMsg = strings.TrimSpace(originalMsg[len("disabled:"):])
	case strings.HasPrefix(lowered, "disabled "):
		disableSemantic = true
		originalMsg = strings.TrimSpace(originalMsg[len("disabled "):])
	}

	//---------------------------------------------------------------------------
	// 4.  Add / normalise Conventional-Commit prefix (unless disabled)
	//---------------------------------------------------------------------------
	msgToUse := originalMsg
	if !disableSemantic {
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
				// “fix something” → “fix: something”
				rest := strings.TrimSpace(originalMsg[len(p):])
				msgToUse = fmt.Sprintf("%s: %s", p, rest)
				valid = true
				break
			}
		}
		if !valid {
			msgToUse = "feat: " + originalMsg
		}
	}

	//---------------------------------------------------------------------------
	// 5.  Assemble final header
	//---------------------------------------------------------------------------
	var finalHeader string
	if disableSemantic {
		finalHeader = fmt.Sprintf("%s: %s", issueKey, msgToUse)
	} else {
		finalHeader = fmt.Sprintf("%s %s", issueKey, msgToUse)
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

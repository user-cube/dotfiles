package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Commit message file not found.")
		os.Exit(1)
	}

	commitMsgFile := os.Args[1]

	// Get branch name
	branchNameCmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	branchNameBytes, err := branchNameCmd.Output()
	if err != nil {
		fmt.Println("Error: Unable to determine the branch name.")
		os.Exit(1)
	}
	branchName := strings.TrimSpace(string(branchNameBytes))
	branchParts := strings.Split(branchName, "/")
	branchName = branchParts[len(branchParts)-1]

	// Disallowed branches
	disallowedBranches := map[string]bool{
		"main":    true,
		"master":  true,
		"develop": true,
	}
	if disallowedBranches[branchName] {
		msg := "Branch not allowed. The disallowed branches are:"
		for branch := range disallowedBranches {
			msg += " " + branch + ","
		}
		fmt.Println(msg[:len(msg)-1])
		os.Exit(1)
	}

	// Read commit message and split lines
	file, err := os.Open(commitMsgFile)
	if err != nil {
		fmt.Println("Error: Unable to read the commit message file.")
		os.Exit(1)
	}
	defer file.Close()

	var messageLines []string
	var commentLines []string
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

	// Define valid prefixes
	validPrefixes := []string{
		"feat", "fix", "chore", "docs", "style", "refactor", "perf", "test", "build", "ci",
	}

	// Normalize or add prefix
	msgToUse := originalMsg
	loweredMsg := strings.ToLower(originalMsg)
	valid := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(loweredMsg, prefix+":") || strings.HasPrefix(loweredMsg, prefix+"(") {
			valid = true
			break
		} else if strings.HasPrefix(loweredMsg, prefix+" ") {
			// Normalize it: "chore add X" → "chore: add X"
			rest := strings.TrimSpace(originalMsg[len(prefix):])
			msgToUse = fmt.Sprintf("%s: %s", prefix, rest)
			valid = true
			break
		}
	}
	if !valid {
		// Fallback if no valid prefix
		msgToUse = "feat: " + originalMsg
	}

	// Final commit message
	finalMsg := fmt.Sprintf("%s %s", branchName, msgToUse)

	// Reattach comment lines
	allLines := []string{finalMsg}
	allLines = append(allLines, commentLines...)
	output := strings.Join(allLines, "\n") + "\n"

	// Write back to file
	err = os.WriteFile(commitMsgFile, []byte(output), 0644)
	if err != nil {
		fmt.Println("Error: Unable to write to the commit message file.")
		os.Exit(1)
	}
}

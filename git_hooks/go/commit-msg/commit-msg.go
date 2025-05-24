package main

import (
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

	// Get the current branch name
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

	// Read commit message
	commitMsgBytes, err := os.ReadFile(commitMsgFile)
	if err != nil {
		fmt.Println("Error: Unable to read the commit message file.")
		os.Exit(1)
	}
	originalMsg := strings.TrimSpace(string(commitMsgBytes))

	// Define valid prefixes
	validPrefixes := []string{
		"feat", "fix", "chore", "docs", "style", "refactor", "perf", "test", "build", "ci",
	}

	// Check if original message starts with a valid prefix
	loweredMsg := strings.ToLower(originalMsg)
	valid := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(loweredMsg, prefix+":") || strings.HasPrefix(loweredMsg, prefix+"(") {
			valid = true
			break
		}
	}

	// Modify the message if not valid
	var finalMsg string
	if valid {
		finalMsg = originalMsg
	} else {
		finalMsg = fmt.Sprintf("%s feat: %s", branchName, originalMsg)
	}

	// Write back to the file
	err = os.WriteFile(commitMsgFile, []byte(finalMsg), 0644)
	if err != nil {
		fmt.Println("Error: Unable to write to the commit message file.")
		os.Exit(1)
	}
}

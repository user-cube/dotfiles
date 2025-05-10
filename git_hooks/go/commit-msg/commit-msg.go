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
	branchNameCmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	branchNameBytes, err := branchNameCmd.Output()
	if err != nil {
		fmt.Println("Error: Unable to determine the branch name.")
		os.Exit(1)
	}
	branchName := strings.TrimSpace(string(branchNameBytes))

	// Extract the last part of the branch name if it contains "/"
	branchParts := strings.Split(branchName, "/")
	branchName = branchParts[len(branchParts)-1]

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
		// Trim the last comma and print
		fmt.Println(msg[:len(msg)-1])
		os.Exit(1)
	}

	commitMsgBytes, err := os.ReadFile(commitMsgFile)
	if err != nil {
		fmt.Println("Error: Unable to read the commit message file.")
		os.Exit(1)
	}
	commitMsg := fmt.Sprintf("%s - %s", branchName, string(commitMsgBytes))
	err = os.WriteFile(commitMsgFile, []byte(commitMsg), 0644)
	if err != nil {
		fmt.Println("Error: Unable to write to the commit message file.")
		os.Exit(1)
	}
}

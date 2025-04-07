package git

import (
	"os/exec"
)

// GitDiff contains information about staged changes
type GitDiff struct {
	StagedFiles     []string
	Diff            string
	Branch          string
	JiraID          string
	JiraDescription string
}

// GetGitDiff retrieves information about staged changes
func GetGitDiff(jiraID, jiraDesc string) (GitDiff, error) {
	// This is a stub function to be implemented later
	return GitDiff{
		JiraID:          jiraID,
		JiraDescription: jiraDesc,
	}, nil
}

// CommitWithMessage commits staged changes with the provided message
func CommitWithMessage(message string) error {
	// This is a stub function to be implemented later
	cmd := exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}

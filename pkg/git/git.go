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
	SystemPrompt    string // Prompt for LLM system context
	UserPrompt      string // Template for user prompt
	
	// Enhanced context fields
	ProjectContext  string            // Brief context about the project
	FileContents    map[string]string // Full content of modified files
	FileTypes       map[string]string // File type information
	CommitHistory   map[string][]string // Recent commit history for changed files
	FileSummaries   map[string]string // Summarized information about each file
	RelatedFiles    []string         // Related files that might provide context
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

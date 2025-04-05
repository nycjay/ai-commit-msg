package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetDiff returns the Git diff for staged changes
func GetDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--staged")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting git diff: %w", err)
	}

	return string(output), nil
}

// CommitWithMessage commits the staged changes with the given message
func CommitWithMessage(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error committing changes: %w", err)
	}

	return nil
}

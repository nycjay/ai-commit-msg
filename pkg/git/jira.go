package git

import (
	"fmt"
	"regexp"
)

// JiraPrefixes contains the list of all known Jira project prefixes
// used by the organization. These are used to extract Jira IDs from
// branch names and commit messages.
var JiraPrefixes = []string{
	"GTN",
	"GTBUG",
	"TOOLS",
	"TASK",
}

// IsJiraPrefix checks if the given string is a known Jira prefix
func IsJiraPrefix(prefix string) bool {
	for _, knownPrefix := range JiraPrefixes {
		if prefix == knownPrefix {
			return true
		}
	}
	return false
}

// ExtractJiraIDFromBranchName attempts to extract a Jira ID from a branch name
// It supports formats like:
// - feature/GTBUG-123-description
// - bugfix/GTN-456-fix-memory-leak
// - TOOLS-789_update_config
// - TASK-101-improve-docs
func extractJiraIDFromBranchName(branchName string) string {
	// Create patterns based on the defined JiraPrefixes
	var patterns []string
	
	// Add specific patterns for known prefixes
	for _, prefix := range JiraPrefixes {
		patterns = append(patterns, prefix+"-\\d+")
	}
	
	// Add a generic pattern for any uppercase followed by dash and numbers
	patterns = append(patterns, "[A-Z]+-\\d+")
	
	// Try each pattern
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(branchName)
		if len(matches) > 0 {
			fmt.Printf("Extracted Jira ID from branch name: %s\n", matches[0])
			return matches[0]
		}
	}

	return ""
}

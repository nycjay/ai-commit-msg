package git

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

// ExtractJiraID tries to extract a Jira ID from a string (usually a branch name)
// It supports formats like:
// - feature/GTBUG-123-description
// - bugfix/GTN-456-fix-memory-leak
// - TOOLS-789_update_config
// - TASK-101-improve-docs
func ExtractJiraID(input string) string {
	// This is a placeholder - the actual implementation would be in the main.go file
	// We're just defining the hardcoded prefixes and helper functions here
	return ""
}

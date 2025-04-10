package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnhancedGitDiff extends GitDiff with additional context information
type EnhancedGitDiff struct {
	GitDiff
	FileContents   map[string]string            // Full content of modified files
	FileTypes      map[string]string            // File type information
	CommitHistory  map[string][]string          // Recent commit history for each changed file
	FileSummaries  map[string]string            // Summarized information about each file
	FileStructure  map[string]map[string]string // Key functions/classes in each file
	RelatedFiles   []string                     // Related files that might provide context
	ProjectContext string                       // Brief description of the project context
}

// GetEnhancedGitDiff retrieves detailed information about staged changes
func GetEnhancedGitDiff(jiraID, jiraDesc string, contextLines int) (EnhancedGitDiff, error) {
	// Initialize the enhanced diff
	enhancedDiff := EnhancedGitDiff{
		GitDiff: GitDiff{
			JiraID:          jiraID,
			JiraDescription: jiraDesc,
		},
		FileContents:  make(map[string]string),
		FileTypes:     make(map[string]string),
		CommitHistory: make(map[string][]string),
		FileSummaries: make(map[string]string),
		FileStructure: make(map[string]map[string]string),
	}

	// Check if we're in a git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return enhancedDiff, fmt.Errorf("not in a git repository")
	}

	// Get list of staged files
	cmd = exec.Command("git", "diff", "--name-only", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return enhancedDiff, fmt.Errorf("error getting staged files: %v", err)
	}

	// Process the list of staged files
	stagingOutput := strings.TrimSpace(string(output))
	if stagingOutput == "" {
		return enhancedDiff, fmt.Errorf("no staged changes found")
	}
	
	enhancedDiff.StagedFiles = strings.Split(stagingOutput, "\n")
	fmt.Printf("Found %d staged files\n", len(enhancedDiff.StagedFiles))
	for _, file := range enhancedDiff.StagedFiles {
		fmt.Printf("  %s\n", file)
	}

	// Get the standard diff with context
	cmd = exec.Command("git", "diff", "--cached", fmt.Sprintf("--unified=%d", contextLines))
	output, err = cmd.Output()
	if err != nil {
		return enhancedDiff, fmt.Errorf("error getting diff: %v", err)
	}
	enhancedDiff.Diff = string(output)

	// Get branch information
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err = cmd.Output()
	if err != nil {
		fmt.Printf("Warning: Failed to get branch name: %v\n", err)
	} else {
		enhancedDiff.Branch = strings.TrimSpace(string(output))
		fmt.Printf("Branch: %s\n", enhancedDiff.Branch)

		// Try to extract Jira ID from branch name if not provided
		if enhancedDiff.JiraID == "" && enhancedDiff.Branch != "" {
			enhancedDiff.JiraID = extractJiraIDFromBranchName(enhancedDiff.Branch)
		}
	}

	// Enhanced context: file contents and types
	for _, file := range enhancedDiff.StagedFiles {
		if file == "" {
			continue
		}

		// Get file type based on extension
		ext := filepath.Ext(file)
		enhancedDiff.FileTypes[file] = getFileType(ext)

		// Get full file content for important context
		if isBinaryFile(file) || isLargeFile(file) {
			// Skip binary or very large files
			enhancedDiff.FileContents[file] = "[Binary or large file, content not included]"
		} else {
			// Get the staged version of the file
			cmd = exec.Command("git", "show", fmt.Sprintf(":%s", file))
			output, err := cmd.Output()
			if err == nil {
				enhancedDiff.FileContents[file] = string(output)
			}
		}

		// Get commit history for the file (last 3 commits)
		cmd = exec.Command("git", "log", "-n", "3", "--pretty=format:%h %s", "--", file)
		output, err = cmd.Output()
		if err == nil && len(output) > 0 {
			history := strings.Split(strings.TrimSpace(string(output)), "\n")
			enhancedDiff.CommitHistory[file] = history
		}

		// Extract function/class definitions for code files
		if isCodeFile(file) {
			enhancedDiff.FileStructure[file] = extractImportantStructures(file, enhancedDiff.FileContents[file])
		}

		// Generate a brief summary of the file
		enhancedDiff.FileSummaries[file] = generateFileSummary(file, enhancedDiff.FileContents[file])
	}

	// Find related files that might provide context
	enhancedDiff.RelatedFiles = findRelatedFiles(enhancedDiff.StagedFiles)

	// Add project context
	enhancedDiff.ProjectContext = getProjectContext()

	return enhancedDiff, nil
}

// isBinaryFile checks if a file is binary based on git attributes
func isBinaryFile(file string) bool {
	cmd := exec.Command("git", "check-attr", "binary", "--", file)
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), "binary: set") {
		return true
	}
	return false
}

// isLargeFile checks if a file is too large to include in full
func isLargeFile(file string) bool {
	// Get file size
	cmd := exec.Command("git", "ls-files", "-s", "--", file)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Very crude size estimation - would be better to parse the actual size
	// For now, just check if the output contains a large file marker
	return len(output) > 0 && strings.Contains(string(output), "100755") // Executable files are often binary
}

// getFileType returns a description of the file type based on extension
func getFileType(extension string) string {
	switch strings.ToLower(extension) {
	case ".go":
		return "Go source code"
	case ".js":
		return "JavaScript source code"
	case ".ts":
		return "TypeScript source code"
	case ".py":
		return "Python source code"
	case ".java":
		return "Java source code"
	case ".c", ".cpp", ".h", ".hpp":
		return "C/C++ source code"
	case ".md", ".markdown":
		return "Markdown documentation"
	case ".json":
		return "JSON data file"
	case ".yaml", ".yml":
		return "YAML configuration file"
	case ".toml":
		return "TOML configuration file"
	case ".html", ".htm":
		return "HTML file"
	case ".css":
		return "CSS stylesheet"
	case ".sql":
		return "SQL database script"
	case ".xml":
		return "XML file"
	case ".sh", ".bash":
		return "Shell script"
	case ".bat", ".cmd":
		return "Windows batch file"
	case ".txt":
		return "Plain text file"
	case ".gitignore":
		return "Git ignore file"
	default:
		return fmt.Sprintf("File with %s extension", extension)
	}
}

// isCodeFile checks if a file is a source code file
func isCodeFile(file string) bool {
	ext := filepath.Ext(file)
	codeExtensions := map[string]bool{
		".go":   true,
		".js":   true,
		".ts":   true,
		".py":   true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".hpp":  true,
	}
	return codeExtensions[strings.ToLower(ext)]
}

// extractImportantStructures extracts important functions, classes, or imports from a file
func extractImportantStructures(file string, content string) map[string]string {
	structures := make(map[string]string)
	
	// This is a very basic implementation - in a real solution, you would 
	// want to use language-specific parsers to extract structures properly
	
	if content == "" {
		return structures
	}
	
	lines := strings.Split(content, "\n")
	
	// Look for import statements, function declarations, class definitions, etc.
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Simple pattern matching for common code structures
		if strings.HasPrefix(trimmedLine, "import ") || 
		   strings.HasPrefix(trimmedLine, "package ") ||
		   strings.HasPrefix(trimmedLine, "func ") ||
		   strings.HasPrefix(trimmedLine, "type ") ||
		   strings.HasPrefix(trimmedLine, "class ") ||
		   strings.HasPrefix(trimmedLine, "def ") {
			
			// Extract a few lines for context
			startLine := max(0, i-1)
			endLine := min(len(lines), i+3)
			
			codeBlock := strings.Join(lines[startLine:endLine], "\n")
			
			// Use the first few words as a key
			words := strings.Fields(trimmedLine)
			if len(words) >= 2 {
				key := strings.Join(words[0:2], " ")
				structures[key] = codeBlock
			}
		}
	}
	
	return structures
}

// generateFileSummary creates a brief summary of a file's purpose
func generateFileSummary(file string, content string) string {
	// Examine file name, extensions, and content to generate a meaningful summary
	// This is a simplified implementation; in practice, you might use more sophisticated techniques
	
	fileName := filepath.Base(file)
	ext := filepath.Ext(file)
	dir := filepath.Dir(file)
	
	summary := fmt.Sprintf("File: %s in directory: %s", fileName, dir)
	
	// Look for file headers or package declarations
	if content != "" {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if i > 10 { // Only look at the first few lines
				break
			}
			
			trimmedLine := strings.TrimSpace(line)
			
			// Look for comments that might describe the file
			if strings.HasPrefix(trimmedLine, "//") || 
			   strings.HasPrefix(trimmedLine, "#") || 
			   strings.HasPrefix(trimmedLine, "/*") {
				if len(trimmedLine) > 3 {
					summary = fmt.Sprintf("%s\nDescription: %s", summary, trimmedLine)
					break
				}
			}
			
			// Look for package declarations in Go files
			if ext == ".go" && strings.HasPrefix(trimmedLine, "package ") {
				summary = fmt.Sprintf("%s\nGo package: %s", summary, strings.TrimPrefix(trimmedLine, "package "))
			}
		}
	}
	
	return summary
}

// findRelatedFiles finds files that may be related to the changed files
func findRelatedFiles(stagedFiles []string) []string {
	var relatedFiles []string
	
	// Look for imports or references to other files
	for _, file := range stagedFiles {
		// Get directory and suggest other files in the same directory
		dir := filepath.Dir(file)
		if dir != "." {
			cmd := exec.Command("git", "ls-files", dir)
			output, err := cmd.Output()
			if err == nil {
				dirFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
				for _, dirFile := range dirFiles {
					if dirFile != file && !contains(stagedFiles, dirFile) && !contains(relatedFiles, dirFile) {
						relatedFiles = append(relatedFiles, dirFile)
					}
				}
			}
		}
	}
	
	// Limit the number of related files to avoid overloading the context
	if len(relatedFiles) > 5 {
		relatedFiles = relatedFiles[:5]
	}
	
	return relatedFiles
}

// getProjectContext provides a brief context about the project
func getProjectContext() string {
	var context strings.Builder
	
	// Get the repository name
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err == nil {
		repoPath := strings.TrimSpace(string(output))
		repoName := filepath.Base(repoPath)
		context.WriteString(fmt.Sprintf("Repository: %s\n", repoName))
	}
	
	// Add branch information
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err = cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		context.WriteString(fmt.Sprintf("Branch: %s\n", branch))
	}
	
	// Look for README to extract project description
	cmd = exec.Command("git", "ls-files", "*README*")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		readmeFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(readmeFiles) > 0 {
			readmeFile := readmeFiles[0]
			cmd = exec.Command("git", "show", fmt.Sprintf(":%s", readmeFile))
			readme, err := cmd.Output()
			if err == nil {
				// Extract the first paragraph from the README
				readmeContent := string(readme)
				lines := strings.Split(readmeContent, "\n")
				var description strings.Builder
				for i, line := range lines {
					if i > 10 { // Limit to first 10 lines
						break
					}
					trimmedLine := strings.TrimSpace(line)
					if trimmedLine != "" {
						description.WriteString(trimmedLine)
						description.WriteString(" ")
					}
				}
				if description.Len() > 0 {
					context.WriteString(fmt.Sprintf("Project description: %s\n", description.String()))
				}
			}
		}
	}
	
	return context.String()
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

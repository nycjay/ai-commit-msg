package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	anthropicAPI    = "https://api.anthropic.com/v1/messages"
	keychainService = "ai-commit-msg"
	keychainAccount = "anthropic-api-key"
	version         = "0.1.0"
)

type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Response struct {
	Content []Content `json:"content"`
}

type Content struct {
	Text string `json:"text"`
}

type GitDiff struct {
	StagedFiles []string
	Diff        string
	Branch      string
	JiraID      string // New field for Jira ID
}

var verbose bool
var executableDir string

// logVerbose prints a message only if verbose mode is enabled
func logVerbose(format string, args ...interface{}) {
	if verbose {
		fmt.Printf(format+"\n", args...)
	}
}

func printHelp() {
	fmt.Printf("AI Commit Message Generator v%s\n\n", version)
	fmt.Println("A tool that uses Claude AI to generate high-quality commit messages for your staged changes.")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("  ai-commit-msg [OPTIONS]")
	fmt.Println("")
	fmt.Println("OPTIONS:")
	fmt.Println("  -k, --key             Anthropic API key (can also be set with ANTHROPIC_API_KEY environment variable")
	fmt.Println("                        or stored in the Mac keychain)")
	fmt.Println("  -j, --jira            Jira issue ID (e.g., GTBUG-123 or GTN-456) to include in the commit message")
	fmt.Println("  -s, --store-key       Store the provided API key in Mac keychain for future use")
	fmt.Println("  -a, --auto            Automatically commit using the generated message without confirmation")
	fmt.Println("  -v, --verbose         Enable verbose output for debugging")
	fmt.Println("  -h, --help            Display this help information")
	fmt.Println("")
	fmt.Println("SETUP GUIDE:")
	fmt.Println("  First-time users:")
	fmt.Println("    1. Simply run 'ai-commit-msg' without any options")
	fmt.Println("    2. When prompted, enter your Anthropic API key (input will be hidden)")
	fmt.Println("    3. Choose to save it securely in your Mac keychain")
	fmt.Println("")
	fmt.Println("  For subsequent uses, just run 'ai-commit-msg' - your key will be loaded automatically")
	fmt.Println("")
	fmt.Println("KEYCHAIN STORAGE:")
	fmt.Println("  The API key is stored in the Mac keychain with these identifiers:")
	fmt.Println("  - Service name: " + keychainService)
	fmt.Println("  - Account name: " + keychainAccount)
	fmt.Println("  You can view or delete this entry using Keychain Access app or the 'security' command.")
	fmt.Println("")
	fmt.Println("JIRA INTEGRATION:")
	fmt.Println("  The tool will include Jira issue IDs in commit messages:")
	fmt.Println("  - Specify an ID directly: ai-commit-msg --jira GTBUG-123")
	fmt.Println("  - If no ID is provided, the tool will suggest a placeholder")
	fmt.Println("")
	fmt.Println("ALTERNATIVE SETUP METHODS:")
	fmt.Println("  1. Mac Keychain (recommended):")
	fmt.Println("     ai-commit-msg --store-key --key your-api-key-here")
	fmt.Println("  2. Environment Variable:")
	fmt.Println("     export ANTHROPIC_API_KEY=your-api-key-here")
	fmt.Println("")
	fmt.Println("CUSTOMIZATION:")
	fmt.Println("  The prompts used for generating commit messages can be customized by editing:")
	fmt.Println("  - prompts/system_prompt.txt - Guidelines for the AI")
	fmt.Println("  - prompts/user_prompt.txt - Template for the request")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Generate a commit message (will prompt for API key if not found):")
	fmt.Println("  ai-commit-msg")
	fmt.Println("")
	fmt.Println("  # Generate with a specific Jira issue ID:")
	fmt.Println("  ai-commit-msg -j GTBUG-123")
	fmt.Println("")
	fmt.Println("  # Generate and automatically commit:")
	fmt.Println("  ai-commit-msg -a")
	fmt.Println("")
	fmt.Println("  # Generate with verbose output:")
	fmt.Println("  ai-commit-msg -v")
	fmt.Println("")
}

// readPasswordFromTerminal reads a password from the terminal without echoing it
func readPasswordFromTerminal(prompt string) (string, error) {
	fmt.Print(prompt)
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Add a newline after the user presses Enter
	if err != nil {
		return "", err
	}
	return string(passwordBytes), nil
}

// findExecutableDir determines the directory containing the executable
func findExecutableDir() (string, error) {
	// First try using os.Executable()
	execPath, err := os.Executable()
	if err == nil {
		return filepath.Dir(execPath), nil
	}

	// Fallback to current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return cwd, nil
}

// readPromptFile reads a prompt file from the prompts directory
func readPromptFile(filename string) (string, error) {
	// First check if the file exists relative to the executable directory
	promptsDir := filepath.Join(executableDir, "prompts")
	promptPath := filepath.Join(promptsDir, filename)

	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		// If not found, look in the project directory structure
		// This helps during development
		projectDir := filepath.Join(executableDir, "..")
		promptPath = filepath.Join(projectDir, "prompts", filename)

		// Check a few more possible locations
		if _, err := os.Stat(promptPath); os.IsNotExist(err) {
			// Try the current directory
			promptPath = filepath.Join("prompts", filename)

			// If still not found, return the default prompt
			if _, err := os.Stat(promptPath); os.IsNotExist(err) {
				if filename == "system_prompt.txt" {
					return getDefaultSystemPrompt(), nil
				} else if filename == "user_prompt.txt" {
					return getDefaultUserPrompt(), nil
				}
				return "", fmt.Errorf("prompt file %s not found", filename)
			}
		}
	}

	logVerbose("Reading prompt file from: %s", promptPath)
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// getDefaultSystemPrompt returns the default system prompt
func getDefaultSystemPrompt() string {
	return `You are an expert developer creating high-quality git commit messages for a professional development team that uses Jira.
Follow these precise guidelines:

1. Use the imperative mood ("Add feature" not "Added feature")
2. First line must be under 50 characters and summarize the change
3. ALWAYS include the Jira issue ID in the first line, using one of these formats:
   - For bugs: "fix(component): GTBUG-123 Fix description"
   - For features: "feat(component): GTN-123 Add description"
4. After the summary line, include a blank line followed by a more detailed description
5. In the detailed description, explain WHY the change was made, not just WHAT was changed
6. For complex changes, use bullet points to list individual modifications

The commit message should follow this conventional commits format:
<type>(<optional scope>): <Jira ID> <short description>

<detailed description>

Where type is one of:
- feat: A new feature
- fix: A bug fix
- docs: Documentation changes
- style: Formatting, missing semicolons, etc; no code change
- refactor: Code change that neither fixes a bug nor adds a feature
- perf: Code change that improves performance
- test: Adding or updating tests
- chore: Changes to build process or auxiliary tools

If no Jira ID is obvious from the changes, use your best judgment to determine if it's a bug fix (GTBUG-XXX) or a feature (GTN-XXX) and use a placeholder ID like GTBUG-??? or GTN-???, noting that the developer should replace it with the correct ID.`
}

// getDefaultUserPrompt returns the default user prompt
func getDefaultUserPrompt() string {
	return `I need a commit message for the following changes on branch '%s'.

Files changed:
%s

Diff:
%s

Jira ID: %s

Please provide a suitable commit message that follows our team's guidelines exactly:
1. First line must include the Jira issue ID specified above
2. Use the format: "<type>(<component>): <Jira ID> <description>"
3. If no Jira ID is provided, use a placeholder (GTBUG-??? or GTN-???)
4. Analyze whether the change is a bugfix (GTBUG) or a feature (GTN) based on the nature of the changes
5. Make sure the first line is under 50 characters
6. Add a detailed description after a blank line

The component should reflect the primary area of code being changed. For a database change, use "db"; for UI changes, use "ui", etc.`
}

func parseArgs() (string, string, bool, bool, bool, []string) {
	// Variables to store the extracted values
	var apiKey, jiraID string
	var storeKey, autoCommit, helpFlag bool
	var unknownFlags []string

	// Known flags
	knownSingleFlags := map[string]bool{
		"-v": true, "--verbose": true,
		"-a": true, "--auto": true,
		"-s": true, "--store-key": true,
		"-h": true, "--help": true,
	}

	knownParamFlags := map[string]bool{
		"-k": true, "--key": true,
		"-j": true, "--jira": true,
	}

	// First, check for help flag (simple case, just check for -h or --help)
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			helpFlag = true
			return "", "", false, false, true, unknownFlags
		}
	}

	// Process all other args
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		// Check for flags that don't require values
		if knownSingleFlags[arg] {
			// Set appropriate flag
			switch arg {
			case "-v", "--verbose":
				verbose = true
			case "-a", "--auto":
				autoCommit = true
			case "-s", "--store-key":
				storeKey = true
			}
			continue
		}

		// Check for flags that require values
		if knownParamFlags[arg] && i+1 < len(os.Args) {
			// Set appropriate value
			switch arg {
			case "-k", "--key":
				apiKey = os.Args[i+1]
			case "-j", "--jira":
				jiraID = os.Args[i+1]
			}
			i++ // Skip the next argument since we've used it
			continue
		}

		// Check for combined forms like -vah (verbose+auto+help)
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 2 {
			// For combined flags like -vah, process each character
			validCombined := true
			for _, char := range arg[1:] {
				flagChar := fmt.Sprintf("-%c", char)
				if knownSingleFlags[flagChar] {
					// Set appropriate flag
					switch flagChar {
					case "-v":
						verbose = true
					case "-a":
						autoCommit = true
					case "-s":
						storeKey = true
					case "-h":
						helpFlag = true
					}
				} else if knownParamFlags[flagChar] {
					// This is a flag that needs a parameter, which isn't valid in combined form
					validCombined = false
					break
				} else {
					validCombined = false
					break
				}
			}

			if validCombined {
				continue
			}
		}

		// If we get here, it's an unknown flag or parameter
		if strings.HasPrefix(arg, "-") {
			unknownFlags = append(unknownFlags, arg)
		}
	}

	return apiKey, jiraID, storeKey, autoCommit, helpFlag, unknownFlags
}

func main() {
	// Get the executable directory
	var err error
	executableDir, err = findExecutableDir()
	if err != nil {
		fmt.Printf("Error finding executable directory: %v\n", err)
		os.Exit(1)
	}

	// Parse command line arguments directly
	apiKey, jiraID, storeKey, autoCommit, helpFlag, unknownFlags := parseArgs()

	// Handle unknown flags
	if len(unknownFlags) > 0 {
		fmt.Println("Error: Unknown flag(s) detected:")
		for _, flag := range unknownFlags {
			fmt.Printf("  %s\n", flag)
		}
		fmt.Println("Use -h or --help to see available options")
		fmt.Println()
		// Exit the program with an error code
		os.Exit(1)
	}

	// If help flag is provided, show help and exit
	if helpFlag {
		printHelp()
		os.Exit(0)
	}

	logVerbose("Starting AI Commit Message Generator v%s", version)
	logVerbose("Executable directory: %s", executableDir)
	logVerbose("Keychain configuration: Service='%s', Account='%s'", keychainService, keychainAccount)

	if jiraID != "" {
		logVerbose("Using provided Jira ID: %s", jiraID)
	}

	// Handle storing the key in keychain if requested
	if storeKey && apiKey != "" {
		logVerbose("Storing API key in keychain...")
		if err := storeAPIKeyInKeychain(apiKey); err != nil {
			fmt.Printf("Error storing API key in keychain: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("API key stored successfully in keychain.")
		if !autoCommit {
			// Exit if we're just storing the key and not committing
			os.Exit(0)
		}
	}

	// Get API key from various sources
	if apiKey == "" {
		logVerbose("No API key provided via --key flag, checking environment...")
		// Try environment variable
		apiKey = os.Getenv("ANTHROPIC_API_KEY")

		// If still empty, try Mac keychain
		if apiKey == "" {
			logVerbose("No API key found in environment, checking Mac keychain (service='%s', account='%s')...",
				keychainService, keychainAccount)
			var err error
			apiKey, err = getAPIKeyFromKeychain()
			if err != nil {
				logVerbose("Error retrieving API key from keychain: %v", err)
				// Continue without exiting, as we'll check for empty key below
			} else if apiKey != "" {
				logVerbose("API key successfully retrieved from keychain")
			}
		} else {
			logVerbose("API key found in environment variable")
		}

		// If still empty, prompt the user
		if apiKey == "" {
			fmt.Println("\nNo API key found. You'll need an Anthropic API key to use this tool.")
			fmt.Println("You can get one from: https://console.anthropic.com/")
			fmt.Println("")

			// Read password without echoing it
			enteredKey, err := readPasswordFromTerminal("Please enter your Anthropic API key: ")
			if err != nil {
				fmt.Printf("Error reading API key: %v\n", err)
				os.Exit(1)
			}

			apiKey = strings.TrimSpace(enteredKey)

			if apiKey == "" {
				fmt.Println("No API key provided. Exiting.")
				os.Exit(1)
			}

			fmt.Print("Would you like to store this API key securely in your Mac keychain for future use? (y/n): ")
			var saveResponse string
			fmt.Scanln(&saveResponse)
			saveResponse = strings.TrimSpace(strings.ToLower(saveResponse))

			if saveResponse == "y" || saveResponse == "yes" {
				if err := storeAPIKeyInKeychain(apiKey); err != nil {
					fmt.Printf("Error storing API key in keychain: %v\n", err)
				} else {
					fmt.Println("API key stored successfully in keychain. You won't need to enter it again.")
				}
			}
		}
	} else {
		logVerbose("API key provided via --key flag")
	}

	// Get git diff information
	logVerbose("Getting git diff information...")
	diffInfo, err := getGitDiff(jiraID)
	if err != nil {
		fmt.Printf("Error getting git diff: %v\n", err)
		os.Exit(1)
	}

	if len(diffInfo.StagedFiles) == 0 || (len(diffInfo.StagedFiles) == 1 && diffInfo.StagedFiles[0] == "") {
		fmt.Println("No staged changes found. Stage your changes using 'git add'.")
		os.Exit(1)
	}

	logVerbose("Found %d staged files in branch '%s'", len(diffInfo.StagedFiles), diffInfo.Branch)
	if verbose {
		for i, file := range diffInfo.StagedFiles {
			fmt.Printf("  %d: %s\n", i+1, file)
		}
	}

	// Generate commit message
	fmt.Println("Generating commit message with Claude AI...")
	startTime := time.Now()
	message, err := generateCommitMessage(apiKey, diffInfo)
	if err != nil {
		fmt.Printf("Error generating commit message: %v\n", err)
		os.Exit(1)
	}
	logVerbose("Commit message generated in %.2f seconds", time.Since(startTime).Seconds())

	// Display the suggested commit message
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Suggested commit message:")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println(message)
	fmt.Println(strings.Repeat("=", 50))

	// Handle the commit
	if autoCommit {
		logVerbose("Auto-commit enabled, committing changes...")
		err = commitWithMessage(message)
		if err != nil {
			fmt.Printf("Error committing changes: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Print("Use this message? (y)es/(e)dit/(n)o: ")
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(response)

		if response == "y" || response == "yes" {
			logVerbose("User selected 'yes', committing changes...")
			err = commitWithMessage(message)
			if err != nil {
				fmt.Printf("Error committing changes: %v\n", err)
				os.Exit(1)
			}
		} else if response == "e" || response == "edit" {
			logVerbose("User selected 'edit', opening editor...")
			editedMessage, err := editMessage(message)
			if err != nil {
				fmt.Printf("Error editing message: %v\n", err)
				os.Exit(1)
			}
			if editedMessage != "" {
				logVerbose("User provided edited message, committing changes...")
				err = commitWithMessage(editedMessage)
				if err != nil {
					fmt.Printf("Error committing changes: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Println("Commit aborted.")
			}
		} else {
			fmt.Println("Commit aborted.")
		}
	}
}

// getAPIKeyFromKeychain retrieves the API key from Mac keychain
func getAPIKeyFromKeychain() (string, error) {
	logVerbose("Executing keychain command to retrieve API key...")
	cmd := exec.Command("security", "find-generic-password", "-s", keychainService, "-a", keychainAccount, "-w")
	output, err := cmd.Output()
	if err != nil {
		// Don't return the error details as they might contain sensitive info or be verbose
		return "", fmt.Errorf("failed to retrieve API key from keychain")
	}
	return strings.TrimSpace(string(output)), nil
}

// storeAPIKeyInKeychain stores the API key in Mac keychain
func storeAPIKeyInKeychain(apiKey string) error {
	// First, try to delete any existing entry
	logVerbose("Deleting any existing keychain entry...")
	deleteCmd := exec.Command("security", "delete-generic-password", "-s", keychainService, "-a", keychainAccount)
	// Ignore errors from delete as the entry might not exist
	_ = deleteCmd.Run()

	// Add the new password
	logVerbose("Adding new keychain entry (service='%s', account='%s')...", keychainService, keychainAccount)
	addCmd := exec.Command("security", "add-generic-password", "-s", keychainService, "-a", keychainAccount, "-w", apiKey)
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to store API key in keychain")
	}
	return nil
}

func getGitDiff(jiraID string) (GitDiff, error) {
	var diffInfo GitDiff
	diffInfo.JiraID = jiraID

	// Check if we're in a git repository
	logVerbose("Checking if current directory is a git repository...")
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return diffInfo, fmt.Errorf("not in a git repository")
	}

	// Get list of staged files
	logVerbose("Getting list of staged files...")
	cmd = exec.Command("git", "diff", "--name-only", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return diffInfo, err
	}
	diffInfo.StagedFiles = strings.Split(strings.TrimSpace(string(output)), "\n")

	// Get the diff details
	logVerbose("Getting diff details...")
	cmd = exec.Command("git", "diff", "--cached")
	output, err = cmd.Output()
	if err != nil {
		return diffInfo, err
	}
	diffInfo.Diff = string(output)
	logVerbose("Diff length: %d bytes", len(diffInfo.Diff))

	// Get current branch name
	logVerbose("Getting current branch name...")
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err = cmd.Output()
	if err != nil {
		return diffInfo, err
	}
	diffInfo.Branch = strings.TrimSpace(string(output))

	// Try to extract Jira ID from branch name if not provided
	if diffInfo.JiraID == "" {
		// Common branch naming patterns like feature/GTBUG-123-description or bugfix/GTN-456-description
		logVerbose("Trying to extract Jira ID from branch name: %s", diffInfo.Branch)

		// Look for GTBUG-XXX pattern
		if idx := strings.Index(strings.ToUpper(diffInfo.Branch), "GTBUG-"); idx >= 0 {
			start := idx
			end := start + 6 // "GTBUG-" length

			// Find the end of the number part
			for end < len(diffInfo.Branch) && (diffInfo.Branch[end] >= '0' && diffInfo.Branch[end] <= '9') {
				end++
			}

			if end > start+6 { // Make sure we found at least one digit
				diffInfo.JiraID = diffInfo.Branch[start:end]
				logVerbose("Extracted Jira ID from branch name: %s", diffInfo.JiraID)
			}
		} else if idx := strings.Index(strings.ToUpper(diffInfo.Branch), "GTN-"); idx >= 0 {
			// Look for GTN-XXX pattern
			start := idx
			end := start + 4 // "GTN-" length

			// Find the end of the number part
			for end < len(diffInfo.Branch) && (diffInfo.Branch[end] >= '0' && diffInfo.Branch[end] <= '9') {
				end++
			}

			if end > start+4 { // Make sure we found at least one digit
				diffInfo.JiraID = diffInfo.Branch[start:end]
				logVerbose("Extracted Jira ID from branch name: %s", diffInfo.JiraID)
			}
		}
	}

	return diffInfo, nil
}

func generateCommitMessage(apiKey string, diffInfo GitDiff) (string, error) {
	// Read system prompt from file
	systemPrompt, err := readPromptFile("system_prompt.txt")
	if err != nil {
		logVerbose("Error reading system prompt: %v, using default", err)
		systemPrompt = getDefaultSystemPrompt()
	}

	// Read user prompt template from file
	userPromptTemplate, err := readPromptFile("user_prompt.txt")
	if err != nil {
		logVerbose("Error reading user prompt: %v, using default", err)
		userPromptTemplate = getDefaultUserPrompt()
	}

	// Format the user prompt with the diff information
	userPrompt := fmt.Sprintf(
		userPromptTemplate,
		diffInfo.Branch,
		strings.Join(diffInfo.StagedFiles, "\n"),
		diffInfo.Diff,
		diffInfo.JiraID, // Add the Jira ID to the prompt
	)

	logVerbose("Building Claude API request...")
	request := Request{
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 1000,
		System:    systemPrompt,
		Messages: []Message{
			{Role: "user", Content: userPrompt},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	logVerbose("Sending request to Claude API...")
	req, err := http.NewRequest("POST", anthropicAPI, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	logVerbose("Parsing Claude API response...")
	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return response.Content[0].Text, nil
}

func commitWithMessage(message string) error {
	logVerbose("Executing git commit command...")
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		fmt.Println("Successfully committed with message.")
	}
	return err
}

func editMessage(message string) (string, error) {
	// Create a temporary file with the message
	logVerbose("Creating temporary file for message editing...")
	tempFile, err := os.CreateTemp("", "commit-msg-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(message); err != nil {
		return "", err
	}
	tempFile.Close()

	// Get the editor from environment or default to vim
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	logVerbose("Using editor: %s", editor)

	// Open the editor
	logVerbose("Opening editor with temporary file: %s", tempFile.Name())
	cmd := exec.Command(editor, tempFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Read the edited content
	logVerbose("Reading edited content from temporary file...")
	editedContent, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(editedContent)), nil
}

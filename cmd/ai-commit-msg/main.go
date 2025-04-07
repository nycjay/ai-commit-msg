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

	"github.com/nycjay/ai-commit-msg/pkg/key"
	"golang.org/x/term"
)

const (
	anthropicAPI = "https://api.anthropic.com/v1/messages"
	version      = "0.1.0"
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
	StagedFiles     []string
	Diff            string
	Branch          string
	JiraID          string // New field for Jira ID
	JiraDescription string // New field for Jira description
}

var verbose bool
var executableDir string
var keyManager *key.KeyManager

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
	fmt.Println("                        or stored in your system credential manager)")
	fmt.Println("  -j, --jira            Jira issue ID (e.g., GTBUG-123 or GTN-456) to include in the commit message")
	fmt.Println("  -d, --jira-desc       Jira issue description to provide additional context for the commit message")
	fmt.Println("  -s, --store-key       Store the provided API key in your system credential manager for future use")
	fmt.Println("  -a, --auto            Automatically commit using the generated message without confirmation")
	fmt.Println("  -v, --verbose         Enable verbose output for debugging")
	fmt.Println("  -h, --help            Display this help information")
	fmt.Println("")
	fmt.Println("SETUP & API KEY:")
	fmt.Println("  First-time users:")
	fmt.Println("    1. Run 'ai-commit-msg' without any options")
	fmt.Println("    2. When prompted, enter your Anthropic API key (input will be hidden)")
	fmt.Println("    3. Choose to save it securely in your system credential manager")
	fmt.Println("")
	fmt.Println("  Alternative setup methods:")
	fmt.Println("    - Store directly in credential manager: ai-commit-msg --store-key --key YOUR-API-KEY")
	fmt.Println("    - Environment variable:                 export ANTHROPIC_API_KEY=YOUR-API-KEY")
	fmt.Println("")

	// Initialize keyManager if it's nil
	if keyManager == nil {
		keyManager = key.NewKeyManager(verbose)
	}

	fmt.Printf("  The API key is stored in the %s with:\n", keyManager.GetCredentialStoreName())
	fmt.Println("    - Service name: " + key.KeychainService)
	fmt.Println("    - Account name: " + key.KeychainAccount)
	fmt.Println("")
	fmt.Println("JIRA INTEGRATION:")
	fmt.Println("  The tool will include Jira issue IDs in commit messages:")
	fmt.Println("  - Specify an ID directly: ai-commit-msg --jira GTBUG-123")
	fmt.Println("  - Add a description: ai-commit-msg --jira GTBUG-123 --jira-desc \"Fix memory leak issue\"")
	fmt.Println("  - If no ID is provided, the tool will try to extract it from the branch name or suggest a placeholder")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Generate a commit message (will prompt for API key if not found):")
	fmt.Println("  ai-commit-msg")
	fmt.Println("")
	fmt.Println("  # Generate with a specific Jira issue ID:")
	fmt.Println("  ai-commit-msg -j GTBUG-123")
	fmt.Println("")
	fmt.Println("  # Generate with a Jira issue ID and description:")
	fmt.Println("  ai-commit-msg -j GTBUG-123 -d \"Fix memory leak in authentication process\"")
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

	logVerbose("Reading prompt file from: %s", promptPath)
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func parseArgs() (string, string, string, bool, bool, bool, []string) {
	// Variables to store the extracted values
	var apiKey, jiraID, jiraDesc string
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
		"-d": true, "--jira-desc": true,
	}

	// First, check for help flag (simple case, just check for -h or --help)
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			helpFlag = true
			return "", "", "", false, false, true, unknownFlags
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
			case "-d", "--jira-desc":
				jiraDesc = os.Args[i+1]
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

	return apiKey, jiraID, jiraDesc, storeKey, autoCommit, helpFlag, unknownFlags
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
	apiKey, jiraID, jiraDesc, storeKey, autoCommit, helpFlag, unknownFlags := parseArgs()

	// Initialize key manager
	keyManager = key.NewKeyManager(verbose)

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
	logVerbose("Platform: %s, Credential store: %s", keyManager.GetPlatform(), keyManager.GetCredentialStoreName())

	if jiraID != "" {
		logVerbose("Using provided Jira ID: %s", jiraID)
		if jiraDesc != "" {
			logVerbose("Using provided Jira description: %s", jiraDesc)
		}
	}

	// Handle storing the key in credential store if requested
	if storeKey && apiKey != "" {
		if !keyManager.CredentialStoreAvailable() {
			fmt.Printf("Error: No credential store available for your platform (%s).\n", keyManager.GetPlatform())
			fmt.Println("Please use the environment variable instead: export ANTHROPIC_API_KEY=your-api-key")
			os.Exit(1)
		}
		
		logVerbose("Storing API key in %s...", keyManager.GetCredentialStoreName())
		if err := keyManager.StoreInKeychain(apiKey); err != nil {
			fmt.Printf("Error storing API key: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("API key stored successfully in %s.\n", keyManager.GetCredentialStoreName())
		if !autoCommit {
			// Exit if we're just storing the key and not committing
			os.Exit(0)
		}
	}

	// Get API key from various sources
	if apiKey == "" {
		logVerbose("No API key provided via --key flag, checking environment...")
		// Try to get the key using our key manager
		apiKey, err = keyManager.GetKey("")
		if err != nil {
			logVerbose("Error getting API key: %v", err)
			
			// If still empty, prompt the user
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

			// Basic validation
			if !keyManager.ValidateKey(apiKey) {
				fmt.Println("Warning: The provided API key doesn't match the expected format.")
				fmt.Println("Claude API keys typically start with 'sk_ant_' and are at least 20 characters long.")
				fmt.Print("Continue anyway? (y/n): ")
				var continueResponse string
				fmt.Scanln(&continueResponse)
				if strings.ToLower(continueResponse) != "y" && strings.ToLower(continueResponse) != "yes" {
					fmt.Println("Exiting.")
					os.Exit(1)
				}
			}

			// Ask if the user wants to store the key
			if keyManager.CredentialStoreAvailable() {
				fmt.Printf("Would you like to store this API key securely in your %s for future use? (y/n): ", 
					keyManager.GetCredentialStoreName())
				var saveResponse string
				fmt.Scanln(&saveResponse)
				saveResponse = strings.TrimSpace(strings.ToLower(saveResponse))

				if saveResponse == "y" || saveResponse == "yes" {
					if err := keyManager.StoreInKeychain(apiKey); err != nil {
						fmt.Printf("Error storing API key: %v\n", err)
					} else {
						fmt.Printf("API key stored successfully in %s. You won't need to enter it again.\n", 
							keyManager.GetCredentialStoreName())
					}
				}
			} else {
				fmt.Println("Note: No secure credential store is available on your platform.")
				fmt.Println("To avoid entering your API key each time, set the ANTHROPIC_API_KEY environment variable.")
			}
		}
	} else {
		logVerbose("API key provided via --key flag")
	}

	// Get git diff information
	logVerbose("Getting git diff information...")
	diffInfo, err := getGitDiff(jiraID, jiraDesc)
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

func getGitDiff(jiraID string, jiraDesc string) (GitDiff, error) {
	var diffInfo GitDiff
	diffInfo.JiraID = jiraID
	diffInfo.JiraDescription = jiraDesc

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
		return "", fmt.Errorf("error reading system prompt: %v", err)
	}

	// Read user prompt template from file
	userPromptTemplate, err := readPromptFile("user_prompt.txt")
	if err != nil {
		return "", fmt.Errorf("error reading user prompt template: %v", err)
	}

	// Format the user prompt with the diff information
	userPrompt := fmt.Sprintf(
		userPromptTemplate,
		diffInfo.Branch,
		strings.Join(diffInfo.StagedFiles, "\n"),
		diffInfo.Diff,
		diffInfo.JiraID,
		diffInfo.JiraDescription,
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
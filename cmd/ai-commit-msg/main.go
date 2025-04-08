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

	"github.com/nycjay/ai-commit-msg/pkg/config"
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
	JiraID          string // Field for Jira ID
	JiraDescription string // Field for Jira description
}

// Global variables
var executableDir string
var cfg *config.Config

// log prints a message only if the current verbosity level is >= the required level
func log(level config.VerbosityLevel, format string, args ...interface{}) {
	if cfg.GetVerbosity() >= level {
		fmt.Printf(format+"\n", args...)
	}
}

// Legacy logVerbose function for backward compatibility
// logs at Verbose level (2)
func logVerbose(format string, args ...interface{}) {
	log(config.Verbose, format, args...)
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
	fmt.Println("  -v                    Enable verbose output (level 1)")
	fmt.Println("  -vv                   Enable more verbose output (level 2)")
	fmt.Println("  -vvv                  Enable debug output (level 3)")
	fmt.Println("  -c, --context N       Number of context lines to include in the diff (default: 3)")
	fmt.Println("  -cc                   Include more context lines (10)")
	fmt.Println("  -ccc                  Include maximum context (entire file)")
	fmt.Println("  -m, --model MODEL     Specify Claude model to use (default: claude-3-haiku-20240307)")
	fmt.Println("  --remember            Remember command-line options in config for future use")
	fmt.Println("  -h, --help            Display this help information")
	fmt.Println("")
	fmt.Println("CONFIGURATION:")
	fmt.Println("  The tool stores configuration in:")
	
	// Get the actual config directory
	configDir, err := cfg.GetConfigDirectory()
	if err != nil {
		// If we can't get the exact path, show a generic example
		fmt.Println("  ~/.config/ai-commit-msg/config.toml (or $XDG_CONFIG_HOME/ai-commit-msg/config.toml)")
	} else {
		fmt.Printf("  %s/config.toml\n", configDir)
	}
	fmt.Println("  You can also use environment variables with the AI_COMMIT_ prefix:")
	fmt.Println("  - AI_COMMIT_VERBOSITY=2     # Set verbosity level")
	fmt.Println("  - AI_COMMIT_CONTEXT_LINES=5 # Set context lines")
	fmt.Println("  - AI_COMMIT_MODEL_NAME=...  # Set Claude model")
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

	keyManager := cfg.GetKeyManager()
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
	fmt.Println("  # Generate with additional context lines:")
	fmt.Println("  ai-commit-msg -cc")
	fmt.Println("")
	fmt.Println("  # Generate with a specific number of context lines:")
	fmt.Println("  ai-commit-msg --context 8")
	fmt.Println("")
	fmt.Println("  # Generate and automatically commit:")
	fmt.Println("  ai-commit-msg -a")
	fmt.Println("")
	fmt.Println("  # Generate with different levels of verbosity:")
	fmt.Println("  ai-commit-msg -v     # Basic verbose output")
	fmt.Println("  ai-commit-msg -vv    # More detailed output")
	fmt.Println("  ai-commit-msg -vvv   # Debug level output")
	fmt.Println("")
	fmt.Println("  # Remember settings for future use:")
	fmt.Println("  ai-commit-msg -cc --remember")
	fmt.Println("")
	fmt.Println("  # Use a different Claude model:")
	fmt.Println("  ai-commit-msg --model claude-3-opus-20240229")
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

func parseArgs() (bool, []string) {
	// Variables to store the extracted values
	var unknownFlags []string

	// First, check for help flag (simple case, just check for -h or --help)
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			return true, unknownFlags
		}
	}

	// Use the config package to parse arguments
	unknownFlags, err := cfg.ParseCommandLineArgs(os.Args[1:])
	if err != nil {
		fmt.Printf("Error parsing command line arguments: %v\n", err)
	}

	return false, unknownFlags
}

func main() {
	// Get the executable directory
	var err error
	executableDir, err = findExecutableDir()
	if err != nil {
		fmt.Printf("Error finding executable directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize config
	cfg = config.GetInstance()
	if err := cfg.LoadConfig(); err != nil {
		fmt.Printf("Warning: Error loading config: %v\n", err)
	}

	// Parse command line arguments
	isHelp, unknownFlags := parseArgs()

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
	if isHelp {
		printHelp()
		os.Exit(0)
	}

	// Save config if remember flag is enabled
	if cfg.IsRememberFlagsEnabled() {
		if err := cfg.SaveConfig(); err != nil {
			fmt.Printf("Warning: Error saving config: %v\n", err)
		}
	}

	log(config.Normal, "Starting AI Commit Message Generator v%s", version)
	log(config.Verbose, "Executable directory: %s", executableDir)
	log(config.Verbose, "Platform: %s, Credential store: %s", 
		cfg.GetKeyManager().GetPlatform(), 
		cfg.GetKeyManager().GetCredentialStoreName())
	log(config.MoreVerbose, "Verbosity level: %d", cfg.GetVerbosity())
	log(config.Verbose, "Context lines: %d", cfg.GetContextLines())

	// Check Jira details
	jiraID := cfg.GetJiraID()
	jiraDesc := cfg.GetJiraDesc()
	if jiraID != "" {
		logVerbose("Using provided Jira ID: %s", jiraID)
		if jiraDesc != "" {
			logVerbose("Using provided Jira description: %s", jiraDesc)
		}
	}

	// Get the keyManager for easier access
	keyManager := cfg.GetKeyManager()

	// Handle storing the key in credential store if requested
	apiKey := cfg.GetAPIKey()
	if cfg.IsStoreKeyEnabled() && apiKey != "" {
		if !keyManager.CredentialStoreAvailable() {
			fmt.Printf("Error: No credential store available for your platform (%s).\n", keyManager.GetPlatform())
			fmt.Println("Please use the environment variable instead: export ANTHROPIC_API_KEY=your-api-key")
			os.Exit(1)
		}
		
		logVerbose("Storing API key in %s...", keyManager.GetCredentialStoreName())
		if err := cfg.StoreAPIKey(apiKey); err != nil {
			fmt.Printf("Error storing API key: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("API key stored successfully in %s.\n", keyManager.GetCredentialStoreName())
		if !cfg.GetAutoCommit() {
			// Exit if we're just storing the key and not committing
			os.Exit(0)
		}
	}

	// Get API key from various sources if not already set
	if apiKey == "" {
		logVerbose("No API key provided via --key flag, checking environment...")
		
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
			fmt.Println("Anthropic API keys typically start with 'sk-ant-', 'sk-', or similar prefixes")
			fmt.Println("and are at least 20 characters long.")
			fmt.Print("Continue anyway? (y/n): ")
			var continueResponse string
			fmt.Scanln(&continueResponse)
			if strings.ToLower(continueResponse) != "y" && strings.ToLower(continueResponse) != "yes" {
				fmt.Println("Exiting.")
				os.Exit(1)
			}
		}

		// Update the API key in config
		cfg.SetAPIKey(apiKey)

		// Ask if the user wants to store the key
		if keyManager.CredentialStoreAvailable() {
			fmt.Printf("Would you like to store this API key securely in your %s for future use? (y/n): ", 
				keyManager.GetCredentialStoreName())
			var saveResponse string
			fmt.Scanln(&saveResponse)
			saveResponse = strings.TrimSpace(strings.ToLower(saveResponse))

			if saveResponse == "y" || saveResponse == "yes" {
				if err := cfg.StoreAPIKey(apiKey); err != nil {
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
	} else {
		logVerbose("API key provided via config, environment or command line")
	}

	// Get git diff information
	logVerbose("Getting git diff information with context lines: %d", cfg.GetContextLines())
	diffInfo, err := getGitDiff(cfg.GetJiraID(), cfg.GetJiraDesc(), cfg.GetContextLines())
	if err != nil {
		fmt.Printf("Error getting git diff: %v\n", err)
		os.Exit(1)
	}

	if len(diffInfo.StagedFiles) == 0 || (len(diffInfo.StagedFiles) == 1 && diffInfo.StagedFiles[0] == "") {
		fmt.Println("No staged changes found. Stage your changes using 'git add'.")
		os.Exit(1)
	}

	logVerbose("Found %d staged files in branch '%s'", len(diffInfo.StagedFiles), diffInfo.Branch)
	if cfg.GetVerbosity() >= config.MoreVerbose {
		for i, file := range diffInfo.StagedFiles {
			fmt.Printf("  %d: %s\n", i+1, file)
		}
	}

	// Generate commit message
	fmt.Println("Generating commit message with Claude AI...")
	startTime := time.Now()
	message, err := generateCommitMessage(cfg.GetAPIKey(), cfg.GetModelName(), diffInfo)
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
	if cfg.GetAutoCommit() {
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

func getGitDiff(jiraID string, jiraDesc string, contextLines int) (GitDiff, error) {
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
	log(config.Verbose, "Getting list of staged files...")
	cmd = exec.Command("git", "diff", "--name-only", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return diffInfo, err
	}
	diffInfo.StagedFiles = strings.Split(strings.TrimSpace(string(output)), "\n")
	
	// Log file details at MoreVerbose level with clear formatting
	if cfg.GetVerbosity() >= config.MoreVerbose {
		log(config.MoreVerbose, "===== STAGED FILES (%d) =====", len(diffInfo.StagedFiles))
		for i, file := range diffInfo.StagedFiles {
			log(config.MoreVerbose, "File #%d: %s", i+1, file)
			
			// Get file stats
			statCmd := exec.Command("git", "diff", "--cached", "--stat", file)
			statOutput, statErr := statCmd.Output()
			if statErr == nil {
				log(config.MoreVerbose, "  Changes: %s", strings.TrimSpace(string(statOutput)))
			}
			
			// For debug level, show more detailed file info
			if cfg.GetVerbosity() >= config.Debug {
				// Get file type
				typeCmd := exec.Command("git", "check-attr", "diff", "--", file)
				typeOutput, typeErr := typeCmd.Output()
				if typeErr == nil {
					log(config.Debug, "  Attributes: %s", strings.TrimSpace(string(typeOutput)))
				}
				
				// Get file size
				sizeCmd := exec.Command("git", "ls-files", "-s", file)
				sizeOutput, sizeErr := sizeCmd.Output()
				if sizeErr == nil {
					log(config.Debug, "  Details: %s", strings.TrimSpace(string(sizeOutput)))
				}
			}
		}
		log(config.MoreVerbose, "=============================")
	}

	// Get the diff details with context
	logVerbose("Getting diff details with context lines: %d...", contextLines)
	var args []string
	args = append(args, "diff", "--cached")
	
	// Handle different context levels
	if contextLines >= 0 {
		// Use the specified number of context lines
		args = append(args, fmt.Sprintf("--unified=%d", contextLines))
	} else {
		// For maximum context (-1), we need to get full file content for each changed file
		// We'll have to handle this specially
		logVerbose("Using maximum context (full file content)...")
		if len(diffInfo.StagedFiles) > 0 && diffInfo.StagedFiles[0] != "" {
			var fullDiff strings.Builder
			
			// Add a standard diff header
			fmt.Fprintf(&fullDiff, "# Showing full files for maximum context\n\n")
			
			for _, file := range diffInfo.StagedFiles {
				// Get the full content of each staged file
				fileCmd := exec.Command("git", "show", fmt.Sprintf(":%s", file))
				fileOutput, err := fileCmd.Output()
				if err == nil {
					fmt.Fprintf(&fullDiff, "=== %s ===\n", file)
					fullDiff.Write(fileOutput)
					fmt.Fprintf(&fullDiff, "\n\n")
				}
			}
			
			// Also include the standard diff for clarity on what actually changed
			diffCmd := exec.Command("git", "diff", "--cached")
			diffOutput, err := diffCmd.Output()
			if err == nil {
				fmt.Fprintf(&fullDiff, "=== CHANGES ===\n")
				fullDiff.Write(diffOutput)
			}
			
			diffInfo.Diff = fullDiff.String()
			log(config.MoreVerbose, "Full context diff length: %d bytes", len(diffInfo.Diff))
			
			// Get the branch info and return
			diffInfo = getBranchInfo(diffInfo)
			return diffInfo, nil
		}
	}
	
	// Standard diff with specified context
	cmd = exec.Command("git", args...)
	output, err = cmd.Output()
	if err != nil {
		return diffInfo, err
	}
	diffInfo.Diff = string(output)
	log(config.MoreVerbose, "Diff length: %d bytes", len(diffInfo.Diff))

	// Get the branch info and return
	return getBranchInfo(diffInfo), nil
}

// getBranchInfo gets the branch information and returns the updated diffInfo
func getBranchInfo(diffInfo GitDiff) GitDiff {
	// Get current branch name
	logVerbose("Getting current branch name...")
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err == nil {
		diffInfo.Branch = strings.TrimSpace(string(output))
	}

	// Try to extract Jira ID from branch name if not provided
	if diffInfo.JiraID == "" && diffInfo.Branch != "" {
		// Common branch naming patterns like feature/GTBUG-123-description or bugfix/GTN-456-description
		log(config.Verbose, "Trying to extract Jira ID from branch name: %s", diffInfo.Branch)
		
		// Get more branch info at MoreVerbose level with clear formatting
		if cfg.GetVerbosity() >= config.MoreVerbose {
			log(config.MoreVerbose, "===== BRANCH INFO =====")
			log(config.MoreVerbose, "Name: %s", diffInfo.Branch)
			
			// Get branch creation date
			dateCmd := exec.Command("git", "show", "-s", "--format=%ci", diffInfo.Branch)
			dateOutput, dateErr := dateCmd.Output()
			if dateErr == nil {
				log(config.MoreVerbose, "Created: %s", strings.TrimSpace(string(dateOutput)))
			}
			
			// Get branch tracking info
			trackCmd := exec.Command("git", "for-each-ref", "--format='%(upstream:short)'", "refs/heads/"+diffInfo.Branch)
			trackOutput, trackErr := trackCmd.Output()
			if trackErr == nil && len(trackOutput) > 0 {
				log(config.MoreVerbose, "Tracks: %s", strings.TrimSpace(string(trackOutput)))
			}
			
			// For debug level, show more detailed branch info
			if cfg.GetVerbosity() >= config.Debug {
				// Get last commit info
				commitCmd := exec.Command("git", "log", "-1", "--pretty=%h %s", diffInfo.Branch)
				commitOutput, commitErr := commitCmd.Output()
				if commitErr == nil {
					log(config.Debug, "Last commit: %s", strings.TrimSpace(string(commitOutput)))
				}
				
				// Get commit count
				countCmd := exec.Command("git", "rev-list", "--count", diffInfo.Branch)
				countOutput, countErr := countCmd.Output()
				if countErr == nil {
					log(config.Debug, "Commit count: %s", strings.TrimSpace(string(countOutput)))
				}
			}
			
			log(config.MoreVerbose, "======================")
		}

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
				log(config.MoreVerbose, "Extracted Jira ID from branch name: %s", diffInfo.JiraID)
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
				log(config.MoreVerbose, "Extracted Jira ID from branch name: %s", diffInfo.JiraID)
			}
		}
	}

	return diffInfo
}

func generateCommitMessage(apiKey string, modelName string, diffInfo GitDiff) (string, error) {
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

	log(config.Verbose, "Building Claude API request...")
	log(config.Debug, "System prompt length: %d bytes", len(systemPrompt))
	log(config.Debug, "User prompt length: %d bytes", len(userPrompt))
	
	// Print full prompts at debug level with clear formatting
	log(config.Debug, "===== SYSTEM PROMPT START =====")
	log(config.Debug, "%s", systemPrompt)
	log(config.Debug, "===== SYSTEM PROMPT END =====\n")
	
	log(config.Debug, "===== USER PROMPT START =====")
	log(config.Debug, "%s", userPrompt)
	log(config.Debug, "===== USER PROMPT END =====\n")
	request := Request{
		Model:     modelName,
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

	log(config.Verbose, "Sending request to Claude API...")
	requestStartTime := time.Now()
	req, err := http.NewRequest("POST", anthropicAPI, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	requestDuration := time.Since(requestStartTime)
	log(config.MoreVerbose, "API request took %.2f seconds", requestDuration.Seconds())
	
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Log HTTP response details at debug level with clear formatting
	log(config.Debug, "===== HTTP RESPONSE DETAILS =====")
	log(config.Debug, "Status: %s", resp.Status)
	log(config.Debug, "Headers:")
	
	// Print headers in a more readable format
	for key, values := range resp.Header {
		for _, value := range values {
			log(config.Debug, "  %s: %s", key, value)
		}
	}
	log(config.Debug, "==================================")
	
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorMsg := string(bodyBytes)
		log(config.Debug, "API error response body: %s", errorMsg)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, errorMsg)
	}

	logVerbose("Parsing Claude API response...")
	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	// Format API response for better readability
	log(config.Debug, "===== API RESPONSE START =====")
	formattedJSON, err := json.MarshalIndent(response, "", "  ")
	if err == nil {
		log(config.Debug, "%s", string(formattedJSON))
	} else {
		log(config.Debug, "%+v", response)
	}
	log(config.Debug, "===== API RESPONSE END =====\n")
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
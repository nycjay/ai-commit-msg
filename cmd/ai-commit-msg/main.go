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
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/nycjay/ai-commit-msg/pkg/ai"
	"github.com/nycjay/ai-commit-msg/pkg/config"
	"github.com/nycjay/ai-commit-msg/pkg/git"
	"github.com/nycjay/ai-commit-msg/pkg/key"
	"golang.org/x/term"
)

var version string

func init() {
	// If version is not set during build, use a default
	if version == "" {
		version = "0.1.0"
	}
}

const (
	anthropicAPI = "https://api.anthropic.com/v1/messages"
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
	fmt.Println("A tool that uses AI to generate high-quality commit messages for your staged changes.")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("  ai-commit-msg [OPTIONS]")
	fmt.Println("  ai-commit-msg init-prompts")
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
	fmt.Println("  -p, --provider NAME   Specify LLM provider to use (anthropic, openai, gemini) (default: anthropic)")
	fmt.Println("  -m, --model MODEL     Specify model to use (provider-specific)")

	fmt.Println("  --system-prompt PATH  Specify a custom system prompt file path")
	fmt.Println("  --user-prompt PATH    Specify a custom user prompt file path")
	fmt.Println("  --remember            Remember command-line options in config for future use")
	fmt.Println("  -h, --help            Display this help information")
	fmt.Println("  --version             Display version information")
	fmt.Println("")
	fmt.Println("SUBCOMMANDS:")
	fmt.Println("  init-prompts           Initialize custom prompt files in your config directory")
	fmt.Println("  show-config            Display the current configuration")
	fmt.Println("  list-providers        List all supported AI providers")
	fmt.Println("  list-models           List available models for providers")
	fmt.Println("")
	fmt.Println("SUBCOMMAND EXAMPLES:")
	fmt.Println("  # List all providers")
	fmt.Println("  ai-commit-msg list-providers")
	fmt.Println("")
	fmt.Println("  # List models for all providers")
	fmt.Println("  ai-commit-msg list-models")
	fmt.Println("")
	fmt.Println("  # List models for a specific provider")
	fmt.Println("  ai-commit-msg list-models anthropic")
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
	fmt.Println("  - AI_COMMIT_SYSTEM_PROMPT_PATH=... # Custom system prompt file path")
	fmt.Println("  - AI_COMMIT_USER_PROMPT_PATH=...   # Custom user prompt file path")
	fmt.Println("")
	fmt.Println("CUSTOM PROMPTS:")
	promptDir, err := cfg.GetPromptDirectory()
	if err == nil {
		fmt.Printf("  Custom prompt files can be placed in: %s/\n", promptDir)
	} else {
		fmt.Println("  Custom prompt files can be placed in: ~/.config/ai-commit-msg/prompts/")
	}
	fmt.Println("  System prompt file: system_prompt.txt")
	fmt.Println("  User prompt file: user_prompt.txt")
	fmt.Println("  You can also specify custom prompt file paths in the config file or command line")
	fmt.Println("")
	
	fmt.Println("SETUP & API KEYS:")
	fmt.Println("  First-time users:")
	fmt.Println("    1. Run 'ai-commit-msg' without any options")
	fmt.Println("    2. When prompted, enter your API key for the selected provider (input will be hidden)")
	fmt.Println("    3. Choose to save it securely in your system credential manager")
	fmt.Println("")
	fmt.Println("  Multiple providers:")
	fmt.Println("    - Anthropic (Claude): ai-commit-msg --provider anthropic --store-key --key YOUR-ANTHROPIC-KEY")
	fmt.Println("    - OpenAI (GPT):       ai-commit-msg --provider openai --store-key --key YOUR-OPENAI-KEY")
	fmt.Println("    - Gemini:             ai-commit-msg --provider gemini --store-key --key YOUR-GEMINI-KEY")
	fmt.Println("")
	fmt.Println("  Environment variables:")
	fmt.Println("    - Anthropic: export ANTHROPIC_API_KEY=YOUR-ANTHROPIC-KEY")
	fmt.Println("    - OpenAI:    export OPENAI_API_KEY=YOUR-OPENAI-KEY")
	fmt.Println("    - Gemini:    export GEMINI_API_KEY=YOUR-GEMINI-KEY")
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
	fmt.Println("  # Use a different provider:")
	fmt.Println("  ai-commit-msg --provider openai     # Use OpenAI models")
	fmt.Println("  ai-commit-msg --provider gemini     # Use Google's Gemini models")
	fmt.Println("  ai-commit-msg --provider anthropic  # Use Anthropic's Claude models (default)")
	fmt.Println("")
	fmt.Println("  # Use different models:")
	fmt.Println("  ai-commit-msg --provider anthropic --model claude-3-opus-20240229")
	fmt.Println("  ai-commit-msg --provider openai --model gpt-4")
	fmt.Println("  ai-commit-msg --provider gemini --model gemini-1.5-pro")
	fmt.Println("")

	fmt.Println("  # Initialize custom prompt files in your config directory:")
	fmt.Println("  ai-commit-msg init-prompts")
	fmt.Println("")
	fmt.Println("  # Use custom prompt files:")
	fmt.Println("  ai-commit-msg --system-prompt /path/to/system_prompt.txt --user-prompt /path/to/user_prompt.txt")
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

// readPromptFile reads a prompt file from the appropriate location
func readPromptFile(filename string) (string, error) {
	var customPath string

	// Check if we have a custom prompt path for this file
	if filename == "system_prompt.txt" && cfg.GetSystemPromptPath() != "" {
		customPath = cfg.GetSystemPromptPath()
		logVerbose("Using custom system prompt path: %s", customPath)
	} else if filename == "user_prompt.txt" && cfg.GetUserPromptPath() != "" {
		customPath = cfg.GetUserPromptPath()
		logVerbose("Using custom user prompt path: %s", customPath)
	}

	// If we have a custom path, try to read it
	if customPath != "" {
		logVerbose("Reading prompt file from custom path: %s", customPath)
		content, err := os.ReadFile(customPath)
		if err == nil {
			return string(content), nil
		}
		logVerbose("Failed to read custom prompt file: %v, falling back to default", err)
	}

	// Try reading from the user's config directory
	promptDir, err := cfg.GetPromptDirectory()
	if err == nil {
		promptPath := filepath.Join(promptDir, filename)
		logVerbose("Checking for prompt file in config directory: %s", promptPath)
		
		if _, err := os.Stat(promptPath); err == nil {
			logVerbose("Reading prompt file from config directory: %s", promptPath)
			content, err := os.ReadFile(promptPath)
			if err == nil {
				return string(content), nil
			}
			logVerbose("Failed to read prompt file from config directory: %v, falling back to default", err)
		}
	}

	// Fall back to the executable directory
	promptsDir := filepath.Join(executableDir, "prompts")
	promptPath := filepath.Join(promptsDir, filename)

	logVerbose("Reading prompt file from executable directory: %s", promptPath)
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file from any location: %v", err)
	}

	return string(content), nil
}

func parseArgs() (bool, bool, bool, bool, bool, bool, []string) {
	// Variables to store the extracted values
	var unknownFlags []string
	var isInitPrompts bool
	var isShowConfig bool
	var isListProviders bool
	var isListModels bool
	var isVersion bool

	// First, check for version flag
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			isVersion = true
			// Skip further parsing for version command
			return false, false, false, false, false, true, unknownFlags
		}
	}

	// First, check for help flag (simple case, just check for -h or --help)
	// Also check for various subcommands
	for i, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			return true, false, false, false, false, false, unknownFlags
		} else if arg == "init-prompts" {
			isInitPrompts = true
		} else if arg == "show-config" {
			isShowConfig = true
		} else if arg == "list-providers" {
			isListProviders = true
		} else if arg == "list-models" {
			// Check if there's a provider specified
			if i+2 < len(os.Args) {
				cfg.SetListModelProvider(os.Args[i+2])
			}
			isListModels = true
		}
	}

	// Use the config package to parse arguments
	unknownFlags, err := cfg.ParseCommandLineArgs(os.Args[1:])
	if err != nil {
		fmt.Printf("Error parsing command line arguments: %v\n", err)
	}

	return false, isInitPrompts, isShowConfig, isListProviders, isListModels, isVersion, unknownFlags
}

// printProviderInfo prints details about available providers and models
func printProviderInfo(listProvidersOnly bool, specificProvider string) {
	fmt.Println("AI Commit Message Generator - Provider Information")
	fmt.Println(strings.Repeat("=", 50))

	// Create providers with their full details
	providerDetails := map[string]string{
		"anthropic": "Anthropic Claude",
		"openai":    "OpenAI",
		"gemini":    "Google Gemini",
	}

	if listProvidersOnly || specificProvider == "" {
		fmt.Println("Supported Providers:")
		for provider, description := range providerDetails {
			fmt.Printf("  - %s (%s)\n", provider, description)
		}
	}

	// If not just listing providers, show models
	if !listProvidersOnly {
		// Use AI package to get providers
		providers := map[string]ai.Provider{
			"anthropic": &ai.AnthropicProvider{},
			"openai":    &ai.OpenAIProvider{},
			"gemini":    &ai.GeminiProvider{},
		}

		fmt.Println("\nAvailable Models:")
		
		// If a specific provider is given, only show its models
		if specificProvider != "" {
			provider, ok := providers[specificProvider]
			if !ok {
				fmt.Printf("Error: Provider '%s' not found.\n", specificProvider)
				return
			}

			availableModels := provider.GetAvailableModels()
			defaultModel := provider.GetDefaultModel()

			fmt.Printf("  %s Models:\n", strings.Title(specificProvider))
			for _, model := range availableModels {
				if model == defaultModel {
					fmt.Printf("    - %s (default)\n", model)
				} else {
					fmt.Printf("    - %s\n", model)
				}
			}
		} else {
			// Show models for all providers
			for provider, providerObj := range providers {
				availableModels := providerObj.GetAvailableModels()
				defaultModel := providerObj.GetDefaultModel()
				
				fmt.Printf("  %s Models:\n", strings.Title(provider))
				for _, model := range availableModels {
					if model == defaultModel {
						fmt.Printf("    - %s (default)\n", model)
					} else {
						fmt.Printf("    - %s\n", model)
					}
				}
				fmt.Println()
			}
		}
	}

	fmt.Println("Notes:")
	fmt.Println("  - Run 'ai-commit-msg list-models <provider>' to see models for a specific provider")
	fmt.Println("  - An API key is required to use models from a provider")
}

// printConfigDetails prints the current configuration details
func printConfigDetails() {
	fmt.Println("AI Commit Message Generator - Configuration")
	fmt.Println(strings.Repeat("=", 50))
	
	// Config Directory
	configDir, err := cfg.GetConfigDirectory()
	if err == nil {
		fmt.Printf("Config Directory: %s\n", configDir)
	}

	// Verbosity
	verbosityNames := map[config.VerbosityLevel]string{
		config.Silent:       "Silent",
		config.Normal:       "Normal",
		config.Verbose:      "Verbose",
		config.MoreVerbose:  "More Verbose",
		config.Debug:        "Debug",
	}
	fmt.Printf("Verbosity: %s (Level %d)\n", verbosityNames[cfg.GetVerbosity()], cfg.GetVerbosity())

	// Context Lines
	fmt.Printf("Context Lines: %d\n", cfg.GetContextLines())

	// Provider Configuration
	fmt.Printf("Current Provider: %s\n", cfg.GetProvider())
	
	// Current Provider and Model
	currentProvider := cfg.GetProvider()
	currentModel := cfg.GetModelName()
	fmt.Printf("Current Provider: %s\n", currentProvider)
	fmt.Printf("Current Model: %s\n", currentModel)

	// System and User Prompt Paths
	systemPromptPath := cfg.GetSystemPromptPath()
	userPromptPath := cfg.GetUserPromptPath()
	
	fmt.Println("\nCustom Prompt Paths:")
	if systemPromptPath != "" {
		fmt.Printf("  System Prompt: %s\n", systemPromptPath)
	} else {
		fmt.Println("  System Prompt: Default")
	}
	
	if userPromptPath != "" {
		fmt.Printf("  User Prompt: %s\n", userPromptPath)
	} else {
		fmt.Println("  User Prompt: Default")
	}

	// Flags
	fmt.Println("\nFlags:")
	fmt.Printf("  Remember Flags: %v\n", cfg.IsRememberFlagsEnabled())
	
	// Prompt Directory
	promptDir, err := cfg.GetPromptDirectory()
	if err == nil {
		fmt.Printf("\nPrompt Directory: %s\n", promptDir)
	}

	// Keychain Information
	keyManager := cfg.GetKeyManager()
	fmt.Println("\nCredential Store:")
	fmt.Printf("  Platform: %s\n", keyManager.GetPlatform())
	fmt.Printf("  Credential Store: %s\n", keyManager.GetCredentialStoreName())
	fmt.Println("\nNote: Sensitive API keys are not displayed for security reasons.")
}

// ensurePromptDirectoryExists makes sure the prompt directory exists
func ensurePromptDirectoryExists() error {
	promptDir, err := cfg.GetPromptDirectory()
	if err != nil {
		return err
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		return fmt.Errorf("failed to create prompt directory: %v", err)
	}

	return nil
}

// copyPromptFile copies a prompt file from the executable directory to the user's prompt directory
// if it doesn't already exist in the user's prompt directory
func copyPromptFile(filename string, provider string) error {
	// Check if the prompt directory exists
	promptDir, err := cfg.GetPromptDirectory()
	if err != nil {
		return err
	}

	// Create provider-specific prompt directory
	providerDir := filepath.Join(promptDir, provider)
	if err := os.MkdirAll(providerDir, 0755); err != nil {
		return fmt.Errorf("failed to create provider prompt directory: %v", err)
	}

	// Check if the file already exists in the provider's prompt directory
	destPath := filepath.Join(providerDir, filename)
	if _, err := os.Stat(destPath); err == nil {
		// File already exists, no need to copy
		return nil
	}

	// Try reading the file from the provider's directory in executable directory first
	srcPath := filepath.Join(executableDir, "prompts", provider, filename)
	content, err := os.ReadFile(srcPath)
	
	// If not found in provider's directory, fall back to the root prompts directory
	if err != nil {
		srcPath = filepath.Join(executableDir, "prompts", filename)
		content, err = os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read source prompt file: %v", err)
		}
	}

	// Write the file to the provider's prompt directory
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write destination prompt file: %v", err)
	}

	logVerbose("Copied %s to %s", srcPath, destPath)
	return nil
}

func main() {
	// Early version check before any other initialization
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			fmt.Printf("AI Commit Message Generator version %s\n", version)
			os.Exit(0)
		}
	}

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
	isHelp, isInitPrompts, isShowConfig, isListProviders, isListModels, _, unknownFlags := parseArgs()

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

	// Help flag is now handled further down in the code, after config initialization
	
	// Handle init-prompts subcommand
	if isInitPrompts {
		fmt.Println("Initializing prompt files in user config directory...")
		
		// Ensure prompt directory exists
		if err := ensurePromptDirectoryExists(); err != nil {
			fmt.Printf("Error: Failed to create prompt directory: %v\n", err)
			os.Exit(1)
		}
		
		// Get base prompt directory
		promptDir, _ := cfg.GetPromptDirectory()
		fmt.Printf("Prompt directory: %s\n", promptDir)
		
		// Define the providers to initialize
		providers := []string{"anthropic", "openai", "gemini"}
		
		// Initialize prompts for each provider
		for _, provider := range providers {
			fmt.Printf("\nInitializing prompts for %s provider:\n", provider)
			
			// Create provider directory
			providerDir := filepath.Join(promptDir, provider)
			if err := os.MkdirAll(providerDir, 0755); err != nil {
				fmt.Printf("Error: Failed to create directory for %s: %v\n", provider, err)
				continue
			}
			
			// Copy system prompt
			if err := copyPromptFile("system_prompt.txt", provider); err != nil {
				fmt.Printf("Error copying system prompt for %s: %v\n", provider, err)
			} else {
				fmt.Printf("Successfully copied system_prompt.txt for %s\n", provider)
			}
			
			// Copy user prompt
			if err := copyPromptFile("user_prompt.txt", provider); err != nil {
				fmt.Printf("Error copying user prompt for %s: %v\n", provider, err)
			} else {
				fmt.Printf("Successfully copied user_prompt.txt for %s\n", provider)
			}
		}
		
		fmt.Println("\nInitialization complete. You can now edit these files to customize your prompts for each provider.")
		fmt.Println("Provider-specific prompt directories:")
		for _, provider := range providers {
			fmt.Printf("  - %s: %s/%s/\n", provider, promptDir, provider)
		}
		os.Exit(0)
	}

	// Handle show-config subcommand
	if isShowConfig {
		// Load config to ensure it's up-to-date before showing
		if err := cfg.LoadConfig(); err != nil {
			fmt.Printf("Warning: Error loading config: %v\n", err)
		}
		
		// Print configuration details
		printConfigDetails()
		os.Exit(0)
	}

	// Handle list-providers subcommand
	if isListProviders {
		// Print provider information (just providers)
		printProviderInfo(true, "")
		os.Exit(0)
	}

	// Handle list-models subcommand
	if isListModels {
		// Get the specific provider (if set)
		specificProvider := cfg.GetListModelProvider()
		
		// Print model information for the specified provider (or all providers)
		printProviderInfo(false, specificProvider)
		os.Exit(0)
	}

	// Version handling has been moved to an earlier stage in main()

	// Set the executable directory in config for prompt loading
	cfg.SetExecutableDir(executableDir)
	
	// Handle help, list-providers and list-models right away - these don't need git or staged changes
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
	
	// Ensure the prompt directory exists
	if err := ensurePromptDirectoryExists(); err != nil {
		logVerbose("Warning: Failed to ensure prompt directory exists: %v", err)
	}

	log(config.Normal, "Starting AI Commit Message Generator v%s", version)
	log(config.Verbose, "Executable directory: %s", executableDir)
	log(config.Verbose, "Platform: %s, Credential store: %s", 
		cfg.GetKeyManager().GetPlatform(), 
		cfg.GetKeyManager().GetCredentialStoreName())
	log(config.MoreVerbose, "Verbosity level: %d", cfg.GetVerbosity())
	log(config.Verbose, "Context lines: %d", cfg.GetContextLines())
	log(config.Verbose, "Using provider: %s", cfg.GetProvider())

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
		
		// First-time setup
		fmt.Println("\n===== Welcome to AI Commit Message Generator =====")
		fmt.Println("It looks like this is the first time you're using this tool.")
		
		// Provider selection
		providers := []string{"Anthropic", "OpenAI", "Gemini"}
		fmt.Println("\nLet's start by choosing your default AI provider:")
		for i, provider := range providers {
			fmt.Printf("%d. %s\n", i+1, provider)
		}
		
		var providerChoice int
		for {
			fmt.Print("\nEnter the number of your preferred provider (1-3): ")
			_, err := fmt.Scanf("%d", &providerChoice)
			if err != nil || providerChoice < 1 || providerChoice > len(providers) {
				fmt.Println("Invalid choice. Please enter a number between 1 and 3.")
				continue
			}
			break
		}
		
		selectedProvider := strings.ToLower(providers[providerChoice-1])
		cfg.SetProvider(selectedProvider)
		
		// Provider-specific API key guidance
		var apiKeyUrl string
		switch selectedProvider {
		case "anthropic":
			apiKeyUrl = "https://console.anthropic.com/"
		case "openai":
			apiKeyUrl = "https://platform.openai.com/account/api-keys"
		case "gemini":
			apiKeyUrl = "https://makersuite.google.com/app/apikey"
		}
		
		fmt.Printf("\nYou've selected %s as your default provider.\n", providers[providerChoice-1])
		fmt.Println("")
		fmt.Printf("To use this tool, you'll need an API key from %s.\n", providers[providerChoice-1])
		fmt.Println("Getting an API key is quick and easy:")
		fmt.Printf("1. Visit: %s\n", apiKeyUrl)
		fmt.Println("2. Sign up or log in to your account")
		fmt.Println("3. Navigate to the API keys section")
		fmt.Println("4. Create a new API key")
		fmt.Println("")
		
		// Read password without echoing it
		enteredKey, err := readPasswordFromTerminal("Paste your API key here: ")
		if err != nil {
			fmt.Printf("Error reading API key: %v\n", err)
			os.Exit(1)
		}

		apiKey = strings.TrimSpace(enteredKey)
		if apiKey == "" {
			fmt.Println("\nNo API key provided. The tool cannot proceed without an API key.")
			fmt.Printf("If you're having trouble, visit %s to obtain a key.\n", apiKeyUrl)
			os.Exit(1)
		}

		// Provider-specific key validation
		var validationPattern string
		var validationDescription string
		switch selectedProvider {
		case "anthropic":
			validationPattern = `^sk-ant-|^sk-`
			validationDescription = "Anthropic API keys typically start with 'sk-ant-' or 'sk-'"
		case "openai":
			validationPattern = `^sk-`
			validationDescription = "OpenAI API keys typically start with 'sk-'"
		case "gemini":
			validationPattern = `.+`  // Gemini keys have less strict format requirements
			validationDescription = "Gemini API keys"
		}

		matched, _ := regexp.MatchString(validationPattern, apiKey)
		if !matched {
			fmt.Printf("\nWarning: The provided %s API key format looks incorrect.\n", 
				strings.Title(selectedProvider))
			fmt.Println(validationDescription)
			fmt.Println("- Should be at least 20 characters long")
			fmt.Println("")
			fmt.Print("Are you sure you want to continue with this key? (y/n): ")
			var continueResponse string
			fmt.Scanln(&continueResponse)
			if strings.ToLower(continueResponse) != "y" && strings.ToLower(continueResponse) != "yes" {
				fmt.Println("API key setup cancelled. Exiting.")
				os.Exit(1)
			}
		}

		// Update the API key in config
		cfg.SetProviderKey(selectedProvider, apiKey)

		// Ask if the user wants to store the key
		if keyManager.CredentialStoreAvailable() {
			fmt.Println("\n🔐 API Key Storage")
			fmt.Println("You can securely store your API key in your system's credential manager.")
			fmt.Println("This allows you to use the tool without re-entering the key each time.")
			fmt.Println("")
			fmt.Printf("Would you like to store your %s API key in %s? (y/n): ", 
				strings.Title(selectedProvider), keyManager.GetCredentialStoreName())
			
			var storeKeyResponse string
			fmt.Scanln(&storeKeyResponse)
			storeKeyResponse = strings.TrimSpace(strings.ToLower(storeKeyResponse))

			if storeKeyResponse == "y" || storeKeyResponse == "yes" {
				if err := cfg.StoreProviderAPIKey(selectedProvider, apiKey); err != nil {
					fmt.Printf("Error storing API key: %v\n", err)
					fmt.Println("\nYou'll need to enter the API key manually each time you use the tool.")
				} else {
					fmt.Println("\n✅ Success!")
					fmt.Printf("API key stored securely in %s.\n", keyManager.GetCredentialStoreName())
					fmt.Println("You won't need to enter it again on this machine.")
				}
			} else {
				fmt.Println("\n⚠️  API key will not be stored.")
				fmt.Println("You'll need to enter the API key manually each time you use the tool.")
				
				envVarName := strings.ToUpper(selectedProvider) + "_API_KEY"
				fmt.Printf("\nTip: You can also set the %s environment variable to avoid manual entry.\n", envVarName)
				fmt.Printf("Example: export %s=your_%s_api_key_here\n", envVarName, selectedProvider)
			}
		} else {
			fmt.Println("\n⚠️  No secure credential store available on your platform.")
			
			envVarName := strings.ToUpper(selectedProvider) + "_API_KEY"
			fmt.Printf("Recommended alternative: Set the %s environment variable.\n", envVarName)
			fmt.Printf("Example: export %s=your_%s_api_key_here\n", envVarName, selectedProvider)
		}

		// Save the configuration to persist the provider selection
		if err := cfg.SaveConfig(); err != nil {
			fmt.Printf("Warning: Could not save configuration: %v\n", err)
		}
	} else {
		logVerbose("API key provided via config, environment or command line")
	}

	// Skip git operations for commands that don't need them
	requiresGit := !isListProviders && !isListModels && !isHelp && !isInitPrompts
	
	// Only proceed with git operations if we need them
	var diffInfo GitDiff
	if requiresGit {
		// Get git diff information
		logVerbose("Getting git diff information with context lines: %d", cfg.GetContextLines())
		var err error
		diffInfo, err = getGitDiff(cfg.GetJiraID(), cfg.GetJiraDesc(), cfg.GetContextLines())
		if err != nil {
			// Only show error for git-related issues if not storing keys
			// This allows "--store-key" to work outside of a git repository
			if !cfg.IsStoreKeyEnabled() {
				fmt.Printf("Error getting git diff: %v\n", err)
				os.Exit(1)
			}
		}

		// Only check for staged files if we're actually generating a commit message
		// and not just storing an API key
		if !cfg.IsStoreKeyEnabled() && 
		   (len(diffInfo.StagedFiles) == 0 || (len(diffInfo.StagedFiles) == 1 && diffInfo.StagedFiles[0] == "")) {
			fmt.Println("No staged changes found. Stage your changes using 'git add'.")
			os.Exit(1)
		}
		
		// Generate commit message
		providerName := cfg.GetProvider()
		fmt.Printf("Generating commit message with %s...\n", strings.Title(providerName))
		startTime := time.Now()
		
		// Use the multi-provider implementation if a provider is specified
		var message string
		
		if providerName != "" && providerName != "anthropic" {
			// Convert GitDiff to git.GitDiff for the multi-provider implementation
			gitDiffInfo := git.GitDiff{
				StagedFiles:     diffInfo.StagedFiles,
				Diff:            diffInfo.Diff,
				Branch:          diffInfo.Branch,
				JiraID:          diffInfo.JiraID,
				JiraDescription: diffInfo.JiraDescription,
			}
			
			// Read prompts for the multi-provider implementation
			systemPrompt, promptErr := readPromptFile("system_prompt.txt")
			if promptErr != nil {
				fmt.Printf("Error reading system prompt: %v\n", promptErr)
				os.Exit(1)
			}
			
			userPrompt, promptErr := readPromptFile("user_prompt.txt")
			if promptErr != nil {
				fmt.Printf("Error reading user prompt: %v\n", promptErr)
				os.Exit(1)
			}
			
			gitDiffInfo.SystemPrompt = systemPrompt
			gitDiffInfo.UserPrompt = userPrompt
			
			// Use the new multi-provider implementation
			message, err = generateCommitMessageMultiProvider(gitDiffInfo)
		} else {
			// Use the original implementation for backward compatibility
			message, err = generateCommitMessage(cfg.GetAPIKey(), cfg.GetModelName(), diffInfo)
		}
		
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
				os.Exit(0)  // Exit the program after aborting
				os.Exit(0)  // Exit the program after aborting
			}
		}
	}

	logVerbose("Found %d staged files in branch '%s'", len(diffInfo.StagedFiles), diffInfo.Branch)
	if cfg.GetVerbosity() >= config.MoreVerbose {
		for i, file := range diffInfo.StagedFiles {
			fmt.Printf("  %d: %s\n", i+1, file)
		}
	}

	// We've already generated and handled the commit message above, 
	// so we shouldn't reach this point. This redundant code has been removed
	// to prevent the double commit message generation issue.
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

// generateCommitMessageMultiProvider generates a commit message using the specified provider
func generateCommitMessageMultiProvider(diffInfo git.GitDiff) (string, error) {
	// Get provider name from config
	providerName := cfg.GetProvider()
	if providerName == "" {
		providerName = string(ai.ProviderAnthropic) // Default to Anthropic
	}
	
	// Create provider using factory
	provider, err := ai.NewProvider(providerName)
	if err != nil {
		return "", fmt.Errorf("failed to create provider: %v", err)
	}
	
	// Get API key for the provider
	apiKey := cfg.GetProviderAPIKey(providerName)
	if apiKey == "" {
		return "", fmt.Errorf("no API key found for provider: %s", providerName)
	}
	
	// Get model name from config, or use default
	modelName := cfg.GetModelName()
	if modelName == "" {
		modelName = provider.GetDefaultModel()
	}
	
	log(config.Verbose, "Using provider: %s with model: %s", providerName, modelName)
	
	// Generate commit message using the provider
	return provider.GenerateCommitMessage(apiKey, modelName, diffInfo)
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

	// Get the editor from environment variables or use system defaults
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	
	// Even if an editor is specified, verify it exists
	editorExists := false
	if editor != "" {
		_, err := exec.LookPath(editor)
		editorExists = (err == nil)
		if !editorExists {
			logVerbose("Warning: Specified editor '%s' not found in PATH", editor)
		}
	}
	
	// If no editor is set or the specified one doesn't exist, find a suitable editor
	if editor == "" || !editorExists {
		// Check for common editors
		possibleEditors := []string{"nano", "vim", "vi", "pico", "emacs", "code", "notepad", "edit", "open -e"}
		for _, e := range possibleEditors {
			// For simple commands (not paths with arguments)
			if !strings.Contains(e, " ") {
				// Look for the editor in PATH
				_, err := exec.LookPath(e)
				if err == nil {
					editor = e
					logVerbose("Found editor: %s", editor)
					editorExists = true
					break
				}
			} else {
				// For complex commands like "open -e" on macOS
				parts := strings.Split(e, " ")
				_, err := exec.LookPath(parts[0])
				if err == nil {
					editor = e
					logVerbose("Found editor: %s", editor)
					editorExists = true
					break
				}
			}
		}
		
		// If we still don't have an editor, default to a simple one that's likely to exist
		if !editorExists {
			if runtime.GOOS == "windows" {
				editor = "notepad"
			} else if runtime.GOOS == "darwin" {
				// On macOS, we can use the built-in TextEdit via 'open -e'
				editor = "open -e"
			} else {
				editor = "vi" // Almost guaranteed to exist on Unix-like systems
			}
			logVerbose("Using default editor for platform: %s", editor)
		}
	}
	
	logVerbose("Using editor: %s", editor)

	// Open the editor - handle commands with arguments
	logVerbose("Opening editor with temporary file: %s", tempFile.Name())
	
	var cmd *exec.Cmd
	if strings.Contains(editor, " ") {
		// For commands with arguments like "open -e"
		parts := strings.Split(editor, " ")
		args := append(parts[1:], tempFile.Name())
		cmd = exec.Command(parts[0], args...)
	} else {
		// For simple commands like "vim"
		cmd = exec.Command(editor, tempFile.Name())
	}
	
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error running editor '%s': %v", editor, err)
	}

	// Read the edited content
	logVerbose("Reading edited content from temporary file...")
	editedContent, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(editedContent)), nil
}
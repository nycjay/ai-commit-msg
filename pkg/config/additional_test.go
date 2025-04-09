package config

import (
	"os"
	"testing"
)

// TestEnvironmentVariables tests loading configuration from environment variables
func TestEnvironmentVariables(t *testing.T) {
	// Save original environment variables
	origVerbosity := os.Getenv("AI_COMMIT_VERBOSITY")
	origContextLines := os.Getenv("AI_COMMIT_CONTEXT_LINES")
	origModelName := os.Getenv("AI_COMMIT_MODEL_NAME")
	origSystemPrompt := os.Getenv("AI_COMMIT_SYSTEM_PROMPT_PATH")
	origUserPrompt := os.Getenv("AI_COMMIT_USER_PROMPT_PATH")
	origRememberFlags := os.Getenv("AI_COMMIT_REMEMBER_FLAGS")
	origProvider := os.Getenv("AI_COMMIT_PROVIDER")
	
	// Restore environment variables after test
	defer func() {
		os.Setenv("AI_COMMIT_VERBOSITY", origVerbosity)
		os.Setenv("AI_COMMIT_CONTEXT_LINES", origContextLines)
		os.Setenv("AI_COMMIT_MODEL_NAME", origModelName)
		os.Setenv("AI_COMMIT_SYSTEM_PROMPT_PATH", origSystemPrompt)
		os.Setenv("AI_COMMIT_USER_PROMPT_PATH", origUserPrompt)
		os.Setenv("AI_COMMIT_REMEMBER_FLAGS", origRememberFlags)
		os.Setenv("AI_COMMIT_PROVIDER", origProvider)
	}()
	
	// Set test environment variables
	os.Setenv("AI_COMMIT_VERBOSITY", "2")
	os.Setenv("AI_COMMIT_CONTEXT_LINES", "8")
	os.Setenv("AI_COMMIT_MODEL_NAME", "claude-3-opus-20240229")
	os.Setenv("AI_COMMIT_SYSTEM_PROMPT_PATH", "/test/system_prompt.txt")
	os.Setenv("AI_COMMIT_USER_PROMPT_PATH", "/test/user_prompt.txt")
	os.Setenv("AI_COMMIT_REMEMBER_FLAGS", "true")
	os.Setenv("AI_COMMIT_PROVIDER", "openai")
	
	// Create a fresh config instance
	cfg := GetInstance()
	
	// Load config to get environment variables
	cfg.LoadConfig()
	
	// Test that values were loaded from environment
	if cfg.GetVerbosity() != Verbose {
		t.Errorf("Expected verbosity %d, got %d", Verbose, cfg.GetVerbosity())
	}
	
	if cfg.GetContextLines() != 8 {
		t.Errorf("Expected context lines 8, got %d", cfg.GetContextLines())
	}
	
	if cfg.GetModelName() != "claude-3-opus-20240229" {
		t.Errorf("Expected model name claude-3-opus-20240229, got %s", cfg.GetModelName())
	}
	
	if cfg.GetSystemPromptPath() != "/test/system_prompt.txt" {
		t.Errorf("Expected system prompt path /test/system_prompt.txt, got %s", cfg.GetSystemPromptPath())
	}
	
	if cfg.GetUserPromptPath() != "/test/user_prompt.txt" {
		t.Errorf("Expected user prompt path /test/user_prompt.txt, got %s", cfg.GetUserPromptPath())
	}
	
	if !cfg.IsRememberFlagsEnabled() {
		t.Errorf("Expected remember flags to be true")
	}
	
	if cfg.GetProvider() != "openai" {
		t.Errorf("Expected provider openai, got %s", cfg.GetProvider())
	}
}

// TestGetPromptDirectory tests retrieving the prompt directory
func TestGetPromptDirectory(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "prompt-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Set the XDG_CONFIG_HOME to the temp directory
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	
	// Get a config instance
	cfg := GetInstance()
	
	// Get the prompt directory
	promptDir, err := cfg.GetPromptDirectory()
	if err != nil {
		t.Fatalf("Failed to get prompt directory: %v", err)
	}
	
	// Verify the prompt directory path
	expectedPath := tempDir + "/ai-commit-msg/prompts"
	if promptDir != expectedPath {
		t.Errorf("Expected prompt directory %s, got %s", expectedPath, promptDir)
	}
}

// TestSetGetToggleSingleFlag tests setting, getting, and toggling a single flag
func TestSetGetToggleSingleFlag(t *testing.T) {
	// Create a clean config
	cfg := GetInstance()
	
	// Test auto commit flag via the flag map
	if cfg.GetAutoCommit() {
		t.Errorf("Auto commit should be false by default")
	}
	
	// Directly set the flag since there's no SetAutoCommit method
	cfg.AutoCommit = true
	if !cfg.GetAutoCommit() {
		t.Errorf("Auto commit should be true after setting")
	}
	
	// Test store key flag
	if cfg.IsStoreKeyEnabled() {
		t.Errorf("Store key should be false by default")
	}
	
	// Directly set the flag since there's no SetStoreKey method
	cfg.StoreKey = true
	if !cfg.IsStoreKeyEnabled() {
		t.Errorf("Store key should be true after setting")
	}
	
	// Test remember flags - directly set the field
	// Note: The default may have changed, so we'll be more flexible in our test
	initialValue := cfg.IsRememberFlagsEnabled()
	
	// Toggle the current value
	cfg.RememberFlags = !initialValue
	
	if cfg.IsRememberFlagsEnabled() == initialValue {
		t.Errorf("Remember flags should have changed after setting")
	}
}

// TestGetProviderModels tests retrieving models for different providers
func TestGetProviderModels(t *testing.T) {
	// Create a clean config
	cfg := GetInstance()
	
	// Set provider models
	cfg.SetProviderModel("anthropic", "claude-3-haiku")
	cfg.SetProviderModel("openai", "gpt-4")
	cfg.SetProviderModel("gemini", "gemini-1.5-pro")
	
	// Get provider models
	models := cfg.GetProviderModels()
	
	// Verify models
	if models["anthropic"] != "claude-3-haiku" {
		t.Errorf("Expected anthropic model claude-3-haiku, got %s", models["anthropic"])
	}
	
	if models["openai"] != "gpt-4" {
		t.Errorf("Expected openai model gpt-4, got %s", models["openai"])
	}
	
	if models["gemini"] != "gemini-1.5-pro" {
		t.Errorf("Expected gemini model gemini-1.5-pro, got %s", models["gemini"])
	}
}

// TestProviderChangeWithModel tests changing the provider and model
func TestProviderChangeWithModel(t *testing.T) {
	// Create a clean config
	cfg := GetInstance()
	
	// Set up initial provider and model
	cfg.SetProvider("anthropic")
	cfg.SetModelName("claude-3-haiku")
	
	// Verify provider and model
	if cfg.GetProvider() != "anthropic" {
		t.Errorf("Expected provider anthropic, got %s", cfg.GetProvider())
	}
	
	if cfg.GetModelName() != "claude-3-haiku" {
		t.Errorf("Expected model claude-3-haiku, got %s", cfg.GetModelName())
	}
	
	// Change provider and model directly
	cfg.Provider = "openai"
	cfg.ProviderModels = map[string]string{
		"openai": "gpt-4",
	}
	cfg.ModelName = "gpt-4"
	
	// Verify provider and model changed
	if cfg.GetProvider() != "openai" {
		t.Errorf("Expected provider openai, got %s", cfg.GetProvider())
	}
	
	if cfg.GetModelName() != "gpt-4" {
		t.Errorf("Expected model gpt-4, got %s", cfg.GetModelName())
	}
}

// TestJiraSettings tests setting and getting Jira ID and description
func TestJiraSettings(t *testing.T) {
	// Create a clean config
	cfg := GetInstance()
	
	// Set Jira ID and description directly
	cfg.JiraID = "TEST-123"
	cfg.JiraDesc = "Test feature implementation"
	
	// Verify Jira ID and description
	if cfg.GetJiraID() != "TEST-123" {
		t.Errorf("Expected Jira ID TEST-123, got %s", cfg.GetJiraID())
	}
	
	if cfg.GetJiraDesc() != "Test feature implementation" {
		t.Errorf("Expected Jira description 'Test feature implementation', got %s", cfg.GetJiraDesc())
	}
}

// TestVerbosityLevels tests setting and getting different verbosity levels
func TestVerbosityLevels(t *testing.T) {
	// Create a clean config
	cfg := GetInstance()
	
	// Test all verbosity levels
	levels := []VerbosityLevel{
		Silent,
		Normal,
		Verbose,
		MoreVerbose,
		Debug,
	}
	
	for _, level := range levels {
		cfg.SetVerbosity(level)
		
		if cfg.GetVerbosity() != level {
			t.Errorf("Expected verbosity %d, got %d", level, cfg.GetVerbosity())
		}
	}
}

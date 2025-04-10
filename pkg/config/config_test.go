package config

import (
	"os"
	"path/filepath"
	"testing"
	
	"github.com/spf13/viper"
	"github.com/nycjay/ai-commit-msg/pkg/key"
)

func TestConfigDefaults(t *testing.T) {
	// Create a clean config instance
	cfg := &Config{
		v:          viper.New(),
		keyManager: key.NewKeyManager(false),
	}
	
	// Set defaults
	cfg.setDefaults()
	
	// Unmarshal the config to get the values into the struct fields
	cfg.v.Unmarshal(cfg)

	// Check defaults
	if cfg.Verbosity != Silent {
		t.Errorf("Default verbosity should be Silent, got %v", cfg.Verbosity)
	}

	if cfg.ContextLines != 3 {
		t.Errorf("Default context lines should be 3, got %v", cfg.ContextLines)
	}

	if cfg.ModelName != "claude-3-haiku-20240307" {
		t.Errorf("Default model name should be claude-3-haiku-20240307, got %v", cfg.ModelName)
	}

	if cfg.RememberFlags != false {
		t.Errorf("Default remember flags should be false, got %v", cfg.RememberFlags)
	}
}

func TestConfigParseArgs(t *testing.T) {
	// Get config instance
	cfg := GetInstance()

	// Test command line args parsing
	args := []string{
		"program", 
		"-v", 
		"--jira", "GTBUG-123", 
		"--context", "5", 
		"--model", "claude-3-opus-20240229",
	}

	unknownFlags, err := cfg.ParseCommandLineArgs(args[1:])
	if err != nil {
		t.Errorf("Error parsing args: %v", err)
	}

	if len(unknownFlags) > 0 {
		t.Errorf("Should not have unknown flags, got %v", unknownFlags)
	}

	if cfg.GetVerbosity() != Verbose {
		t.Errorf("Verbosity should be Verbose, got %v", cfg.GetVerbosity())
	}

	if cfg.GetJiraID() != "GTBUG-123" {
		t.Errorf("JiraID should be GTBUG-123, got %v", cfg.GetJiraID())
	}

	if cfg.GetContextLines() != 5 {
		t.Errorf("Context lines should be 5, got %v", cfg.GetContextLines())
	}

	if cfg.GetModelName() != "claude-3-opus-20240229" {
		t.Errorf("Model name should be claude-3-opus-20240229, got %v", cfg.GetModelName())
	}
	
	// Test -ccc flag enables enhanced context
	args = []string{"program", "-ccc"}
	cfg.ParseCommandLineArgs(args[1:])
	
	if cfg.GetContextLines() != -1 {
		t.Errorf("-ccc should set context lines to -1, got %v", cfg.GetContextLines())
	}
	
	if !cfg.IsEnhancedContextEnabled() {
		t.Errorf("-ccc should enable enhanced context")
	}

	// Test combined flags
	args = []string{"program", "-vas"}
	unknownFlags, err = cfg.ParseCommandLineArgs(args[1:])
	if err != nil {
		t.Errorf("Error parsing args: %v", err)
	}

	if len(unknownFlags) > 0 {
		t.Errorf("Should not have unknown flags, got %v", unknownFlags)
	}

	if cfg.GetVerbosity() != Verbose {
		t.Errorf("Verbosity should be Verbose, got %v", cfg.GetVerbosity())
	}

	if !cfg.GetAutoCommit() {
		t.Errorf("AutoCommit should be true")
	}

	if !cfg.IsStoreKeyEnabled() {
		t.Errorf("StoreKey should be true")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Could not create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set XDG_CONFIG_HOME to the temp directory to control where config is saved
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	// Get config instance with a clean state
	cfg := &Config{
		v:          viper.New(),
		keyManager: key.NewKeyManager(false),
	}
	cfg.setDefaults()

	// Modify some settings
	cfg.SetVerbosity(Verbose)
	cfg.SetContextLines(8)
	cfg.SetModelName("claude-3-opus-20240229")
	cfg.SetRememberFlags(true)

	// Save the config
	if err := cfg.SaveConfig(); err != nil {
		t.Fatalf("Error saving config: %v", err)
	}

	// Verify config file was created
	configDir, err := cfg.getConfigDirectory()
	if err != nil {
		t.Fatalf("Error getting config directory: %v", err)
	}
	configFile := filepath.Join(configDir, ConfigFileName+".toml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", configFile)
	}

	// Create a new config instance
	newCfg := &Config{
		v:          viper.New(),
		keyManager: key.NewKeyManager(false),
	}
	newCfg.setDefaults()

	// Load the config
	if err := newCfg.LoadConfig(); err != nil {
		t.Fatalf("Error loading config: %v", err)
	}

	// Verify loaded values match
	if newCfg.GetVerbosity() != Verbose {
		t.Errorf("Loaded verbosity should be Verbose, got %v", newCfg.GetVerbosity())
	}

	if newCfg.GetContextLines() != 8 {
		t.Errorf("Loaded context lines should be 8, got %v", newCfg.GetContextLines())
	}

	if newCfg.GetModelName() != "claude-3-opus-20240229" {
		t.Errorf("Loaded model name should be claude-3-opus-20240229, got %v", newCfg.GetModelName())
	}

	if !newCfg.IsRememberFlagsEnabled() {
		t.Errorf("Loaded remember flags should be true")
	}
}

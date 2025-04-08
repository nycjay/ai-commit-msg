package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	"github.com/nycjay/ai-commit-msg/pkg/key"
)

const (
	// ConfigFileName is the name of the config file without extension
	ConfigFileName = "config"

	// ConfigDirName is the directory name within XDG_CONFIG_HOME or ~/.config
	ConfigDirName = "ai-commit-msg"

	// EnvPrefix is the prefix for environment variables
	EnvPrefix = "AI_COMMIT"
)

// VerbosityLevel represents the level of verbosity for logging
type VerbosityLevel int

const (
	// Silent means no logs will be printed
	Silent VerbosityLevel = iota
	// Normal shows basic operation logs
	Normal
	// Verbose shows detailed operation logs
	Verbose
	// MoreVerbose shows even more detailed logs including intermediate steps
	MoreVerbose
	// Debug shows all possible logs including debug information
	Debug
)

// Config holds all the configuration for the application
type Config struct {
	// Configuration values stored in Viper
	Verbosity         VerbosityLevel `mapstructure:"verbosity"`
	ContextLines      int            `mapstructure:"context_lines"`
	RememberFlags     bool           `mapstructure:"remember_flags"`
	ModelName         string         `mapstructure:"model_name"`
	SystemPromptPath  string         `mapstructure:"system_prompt_path"`
	UserPromptPath    string         `mapstructure:"user_prompt_path"`

	// Runtime-only values (not saved to config)
	APIKey        string `mapstructure:"-"` // Sensitive, stored in keychain
	JiraID        string `mapstructure:"-"` // Command-line only
	JiraDesc      string `mapstructure:"-"` // Command-line only
	AutoCommit    bool   `mapstructure:"-"` // Command-line only
	StoreKey      bool   `mapstructure:"-"` // Command-line only

	// KeyManager for handling API keys
	keyManager *key.KeyManager
	// Viper instance
	v *viper.Viper
	// Mutex for thread safety
	mu sync.RWMutex
}

var (
	// instance is the singleton instance of Config
	instance *Config
	// once ensures the singleton is initialized only once
	once sync.Once
)

// GetInstance returns the singleton instance of Config
func GetInstance() *Config {
	once.Do(func() {
		instance = &Config{
			v:          viper.New(),
			keyManager: key.NewKeyManager(false), // Initially not verbose
		}
		// Set default values
		instance.setDefaults()
	})
	return instance
}

// LoadConfig loads the configuration from the config file, environment variables, and flags
func (c *Config) LoadConfig() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.v.SetConfigName(ConfigFileName)
	c.v.SetConfigType("toml") // Using TOML format for better readability

	// Add config file search paths
	configDir, err := c.getConfigDirectory()
	if err == nil {
		c.v.AddConfigPath(configDir)
	}

	// Also look in current directory for config
	c.v.AddConfigPath(".")

	// Set up environment variables
	c.v.SetEnvPrefix(EnvPrefix)
	c.v.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	c.v.SetEnvKeyReplacer(replacer)

	// Try to read the config file
	if err := c.v.ReadInConfig(); err != nil {
		// It's okay if the config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal the config into the struct
	if err := c.v.Unmarshal(c); err != nil {
		return fmt.Errorf("unable to decode config: %w", err)
	}

	// Update the key manager's verbosity
	c.keyManager.SetVerbose(c.Verbosity >= Verbose)

	// Try to get API key from keychain or environment
	apiKey, _ := c.keyManager.GetKey("")
	if apiKey != "" {
		c.APIKey = apiKey
	}

	return nil
}

// SaveConfig saves the current configuration to the config file
func (c *Config) SaveConfig() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Only remember flags if enabled
	if !c.RememberFlags {
		// Load the config again to get the original values
		if err := c.v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return fmt.Errorf("error reading config file: %w", err)
			}
		}
	}

	// Update Viper with current values for persistent settings only
	// We only persist settings that make sense to reuse across multiple commits
	// Transaction-specific settings are not persisted
	c.v.Set("verbosity", c.Verbosity)
	c.v.Set("context_lines", c.ContextLines)
	c.v.Set("remember_flags", c.RememberFlags)
	c.v.Set("model_name", c.ModelName)
	c.v.Set("system_prompt_path", c.SystemPromptPath)
	c.v.Set("user_prompt_path", c.UserPromptPath)
	
	// Note: We no longer persist JiraPrefix as it will be hardcoded in the application
	
	// Note: We intentionally don't persist these transaction-specific parameters:
	// - APIKey (sensitive data, stored in keychain instead)
	// - JiraID (specific to a single commit)
	// - JiraDesc (specific to a single commit)
	// - AutoCommit (potentially dangerous to always auto-commit)
	// - StoreKey (one-time operation)

	// Create config directory if it doesn't exist
	configDir, err := c.getConfigDirectory()
	if err != nil {
		return fmt.Errorf("could not determine config directory: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save the config file
	configFile := filepath.Join(configDir, ConfigFileName+".toml")
	return c.v.WriteConfigAs(configFile)
}

// setDefaults sets the default values for the configuration
func (c *Config) setDefaults() {
	c.v.SetDefault("verbosity", Silent)
	c.v.SetDefault("context_lines", 3)
	c.v.SetDefault("remember_flags", false)
	c.v.SetDefault("model_name", "claude-3-haiku-20240307")
	c.v.SetDefault("system_prompt_path", "")
	c.v.SetDefault("user_prompt_path", "")
	
	// Note: JiraPrefix is no longer configurable, it's hardcoded in pkg/git/jira.go
}

// getConfigDirectory returns the directory where the config file should be located
func (c *Config) getConfigDirectory() (string, error) {
	return c.GetConfigDirectory()
}

// GetPromptDirectory returns the directory where custom prompt files should be located
func (c *Config) GetPromptDirectory() (string, error) {
	configDir, err := c.GetConfigDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "prompts"), nil
}

// GetConfigDirectory returns the directory where the config file should be located
func (c *Config) GetConfigDirectory() (string, error) {
	// Check XDG_CONFIG_HOME first, applicable to all platforms
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, ConfigDirName), nil
	}

	// Detect operating system for platform-specific fallbacks
	switch runtime.GOOS {
	case "windows":
		return c.getWindowsConfigDirectory()
	case "darwin":
		return c.getMacOSConfigDirectory()
	case "linux":
		return c.getLinuxConfigDirectory()
	default:
		return c.getLinuxConfigDirectory()
	}
}

// getWindowsConfigDirectory returns the config directory for Windows
func (c *Config) getWindowsConfigDirectory() (string, error) {
	// Check environment variables in order of preference
	appData := os.Getenv("APPDATA")
	if appData != "" {
		return filepath.Join(appData, ConfigDirName), nil
	}

	// Fallback to user profile directory
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	// Try multiple Windows-specific paths
	paths := []string{
		filepath.Join(home, ".config", ConfigDirName),
		filepath.Join(home, "AppData", "Local", ConfigDirName),
		filepath.Join(home, "Application Data", ConfigDirName),
	}

	for _, path := range paths {
		// Check if the directory exists or can be created
		if err := os.MkdirAll(path, 0755); err == nil {
			return path, nil
		}
	}

	// If all else fails, use a default in .config
	return filepath.Join(home, ".config", ConfigDirName), nil
}

// getMacOSConfigDirectory returns the config directory for macOS
func (c *Config) getMacOSConfigDirectory() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	// Try macOS-specific paths
	paths := []string{
		filepath.Join(home, ".config", ConfigDirName),
		filepath.Join(home, "Library", "Application Support", ConfigDirName),
	}

	for _, path := range paths {
		// Check if the directory exists or can be created
		if err := os.MkdirAll(path, 0755); err == nil {
			return path, nil
		}
	}

	// If all else fails, use .config
	return filepath.Join(home, ".config", ConfigDirName), nil
}

// getLinuxConfigDirectory returns the config directory for Linux
func (c *Config) getLinuxConfigDirectory() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", ConfigDirName), nil
}

// GetVerbosity returns the current verbosity level
func (c *Config) GetVerbosity() VerbosityLevel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Verbosity
}

// SetVerbosity sets the verbosity level
func (c *Config) SetVerbosity(level VerbosityLevel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Verbosity = level
	c.keyManager.SetVerbose(level >= Verbose)
}

// GetContextLines returns the number of context lines for diff
func (c *Config) GetContextLines() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ContextLines
}

// SetContextLines sets the number of context lines for diff
func (c *Config) SetContextLines(lines int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ContextLines = lines
}

// GetAPIKey returns the API key
func (c *Config) GetAPIKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.APIKey
}

// SetAPIKey sets the API key
func (c *Config) SetAPIKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.APIKey = key
}

// Note: JiraPrefix methods have been removed as Jira prefixes are now hardcoded in pkg/git/jira.go

// GetModelName returns the Claude model name
func (c *Config) GetModelName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ModelName
}

// SetModelName sets the Claude model name
func (c *Config) SetModelName(model string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ModelName = model
}

// GetSystemPromptPath returns the custom system prompt path
func (c *Config) GetSystemPromptPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SystemPromptPath
}

// SetSystemPromptPath sets the custom system prompt path
func (c *Config) SetSystemPromptPath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SystemPromptPath = path
}

// GetUserPromptPath returns the custom user prompt path
func (c *Config) GetUserPromptPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.UserPromptPath
}

// SetUserPromptPath sets the custom user prompt path
func (c *Config) SetUserPromptPath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UserPromptPath = path
}

// StoreAPIKey stores the API key in the keychain
func (c *Config) StoreAPIKey(key string) error {
	return c.keyManager.StoreInKeychain(key)
}

// ParseCommandLineArgs parses command line arguments into the config
func (c *Config) ParseCommandLineArgs(args []string) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Reset runtime values
	c.JiraID = ""
	c.JiraDesc = ""
	c.AutoCommit = false
	c.StoreKey = false

	// Known flags
	knownSingleFlags := map[string]bool{
		"-v": true, "--verbose": true,
		"-vv": true,
		"-vvv": true,
		"-a": true, "--auto": true,
		"-s": true, "--store-key": true,
		"-h": true, "--help": true,
		"-cc": true, // Medium context level
		"-ccc": true, // Maximum context level
		"--remember": true, // Remember settings for future use
	}

	knownParamFlags := map[string]bool{
		"-k": true, "--key": true,
		"-j": true, "--jira": true,
		"-d": true, "--jira-desc": true,
		"-c": true, "--context": true,
		"-m": true, "--model": true,
		"--system-prompt": true,
		"--user-prompt": true,
	}

	// Collect any unknown flags
	var unknownFlags []string

	// Process all args
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check for flags that don't require values
		if knownSingleFlags[arg] {
			// Set appropriate flag
			switch arg {
			case "-v", "--verbose":
				c.Verbosity = Verbose
			case "-vv":
				c.Verbosity = MoreVerbose
			case "-vvv":
				c.Verbosity = Debug
			case "-a", "--auto":
				c.AutoCommit = true
			case "-s", "--store-key":
				c.StoreKey = true
			case "-cc":
				c.ContextLines = 10 // Medium context
			case "-ccc":
				c.ContextLines = -1 // Signal for maximum context
			case "--remember":
				c.RememberFlags = true
			}
			continue
		}

		// Check for flags that require values
		if knownParamFlags[arg] && i+1 < len(args) {
			// Set appropriate value
			switch arg {
			case "-k", "--key":
				c.APIKey = args[i+1]
			case "-j", "--jira":
				c.JiraID = args[i+1]
			case "-d", "--jira-desc":
				c.JiraDesc = args[i+1]
			case "-c", "--context":
				// Try to parse context lines as an integer
				fmt.Sscanf(args[i+1], "%d", &c.ContextLines)
			case "-m", "--model":
				c.ModelName = args[i+1]
			case "--system-prompt":
				c.SystemPromptPath = args[i+1]
			case "--user-prompt":
				c.UserPromptPath = args[i+1]
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
						// In combined flags, treat -v as basic verbose
						if c.Verbosity < Verbose {
							c.Verbosity = Verbose
						}
					case "-a":
						c.AutoCommit = true
					case "-s":
						c.StoreKey = true
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

	return unknownFlags, nil
}

// GetKeyManager returns the key manager instance
func (c *Config) GetKeyManager() *key.KeyManager {
	return c.keyManager
}

// IsRememberFlagsEnabled returns whether flag values should be remembered
func (c *Config) IsRememberFlagsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.RememberFlags
}

// SetRememberFlags sets whether flag values should be remembered
func (c *Config) SetRememberFlags(remember bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RememberFlags = remember
}

// GetAutoCommit returns whether auto-commit is enabled
func (c *Config) GetAutoCommit() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AutoCommit
}

// GetJiraID returns the Jira ID
func (c *Config) GetJiraID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.JiraID
}

// GetJiraDesc returns the Jira description
func (c *Config) GetJiraDesc() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.JiraDesc
}

// IsStoreKeyEnabled returns whether storing the key is enabled
func (c *Config) IsStoreKeyEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.StoreKey
}

# Configuration Package

This package provides configuration management for the AI Commit Message Generator. It is built using the [Viper](https://github.com/spf13/viper) library and provides a unified interface for accessing configuration from multiple sources.

## Features

- **XDG Support**: Respects the XDG Base Directory Specification
- **Multiple Configuration Sources**:
  - Configuration files (TOML)
  - Environment variables
  - Command-line arguments
- **Persistent Configuration**: Can save user preferences
- **Sensitive Data Handling**: Uses system keychain for API keys
- **Smart Persistence**: Only persists settings that make sense to reuse
- **Thread-Safe**: Can be accessed from multiple goroutines
- **Default Values**: Sensible defaults are provided

## Usage

```go
// Get the config instance (singleton)
cfg := config.GetInstance()

// Load configuration from all sources
err := cfg.LoadConfig()
if err != nil {
    // Handle error
}

// Get values
verbosity := cfg.GetVerbosity()
contextLines := cfg.GetContextLines()

// Set values
cfg.SetVerbosity(config.Verbose)
cfg.SetContextLines(5)

// Parse command-line arguments
unknownFlags, err := cfg.ParseCommandLineArgs(os.Args[1:])
if err != nil {
    // Handle error
}

// Save configuration
if cfg.IsRememberFlagsEnabled() {
    err := cfg.SaveConfig()
    if err != nil {
        // Handle error
    }
}
```

## Configuration Files

The configuration file is stored in the following location (in order of precedence):

1. `$XDG_CONFIG_HOME/ai-commit-msg/config.toml` if `$XDG_CONFIG_HOME` is set
2. `~/.config/ai-commit-msg/config.toml` otherwise
3. `./config.toml` in the current directory (fallback)

## Environment Variables

Environment variables are checked with the prefix `AI_COMMIT_`. For example:

- `AI_COMMIT_VERBOSITY=2` to set verbosity level
- `AI_COMMIT_CONTEXT_LINES=5` to set context lines for git diff
- `AI_COMMIT_MODEL_NAME=claude-3-opus-20240229` to specify the Claude model

## API Key Handling

API keys are handled using the key management package, which stores them securely in the system's credential store:

- macOS: Keychain
- Windows: Credential Manager (when implemented)

The API key is never stored in the configuration file.

## Persistent vs Non-Persistent Settings

The configuration system intelligently distinguishes between settings that should be persisted and those that are transaction-specific:

### Persistent Settings 
These are saved to the config file when `RememberFlags` is enabled:
- **Verbosity**: Logging verbosity level
- **ContextLines**: Number of context lines for git diff
- **ModelName**: Claude AI model to use
- **RememberFlags**: Whether to remember settings

### Non-Persistent Settings
These are never saved to the config file, regardless of `RememberFlags` setting:
- **APIKey**: API key is sensitive data and stored in the system keychain instead
- **JiraID**: Specific to a single commit
- **JiraDesc**: Specific to a single commit
- **JiraPrefix**: Hardcoded in the application with common prefixes (GTN, GTBUG, TOOLS, TASK)
- **AutoCommit**: Potentially dangerous to always auto-commit
- **StoreKey**: One-time operation flag

This distinction ensures that users don't accidentally persist commit-specific information or sensitive data.

## Thread Safety

The configuration package is thread-safe, using read-write mutexes to protect against concurrent access.

# AI Commit Message Generator

A tool that uses AI to automatically generate high-quality git commit messages based on your staged changes. Supports multiple LLM providers including Anthropic's Claude, OpenAI's GPT models, and Google's Gemini.

## Features

- 🤖 Uses AI to analyze your code changes and generate meaningful commit messages
- 🧠 Supports multiple LLM providers: Anthropic (Claude), OpenAI (GPT), and Google (Gemini)
- 📝 Follows best practices for commit messages (conventional commits, imperative mood)
- 🚀 No dependencies - single binary distribution
- 🔐 Securely stores API key in your system's credential manager (macOS Keychain or Windows Credential Manager)
- ✏️ Interactive mode allows you to edit messages before committing
- 🔄 Can be integrated directly into your git workflow
- 💻 Cross-platform support for macOS and Windows
- 📊 Configurable context levels for more accurate commit messages

## Installation

### Building from source

```bash
# Clone the repository
git clone https://github.com/nycjay/ai-commit-msg.git
cd ai-commit-msg

# Build and install
./build.sh
```

### Version Management

The project uses a `VERSION` file to track the current version of the tool:

- The version is automatically read from the `VERSION` file during build
- To update the version, edit the `VERSION` file 
- Example version update:
  ```bash
  echo "1.0.0" > VERSION  # Update to version 1.0.0
  ./build.sh             # Build with the new version
  ```

You can also override the version temporarily during build:
```bash
# Build with a specific version without modifying the VERSION file
./build.sh 1.2.3
```

When releasing a new version:
1. Update the `VERSION` file
2. Create a git tag matching the version
3. Build and distribute the new binary

## API Key Setup

You'll need an API key from one of the supported providers (Anthropic, OpenAI, or Gemini). The setup process is simple:

### 1. First-time setup (recommended)

Simply run the tool without any parameters:

```bash
ai-commit-msg
```

The tool will prompt you for your API key (input will be hidden) and offer to store it securely in your system's credential manager.

### 2. Store directly in credential manager

```bash
# For Anthropic (default)
./ai-commit-msg --store-key --key your-anthropic-key-here

# For OpenAI
./ai-commit-msg --provider openai --store-key --key your-openai-key-here

# For Gemini
./ai-commit-msg --provider gemini --store-key --key your-gemini-key-here
```

### 3. Set as environment variable

```bash
# For Anthropic (default)
export ANTHROPIC_API_KEY="your-anthropic-key-here"

# For OpenAI
export OPENAI_API_KEY="your-openai-key-here"

# For Gemini
export GEMINI_API_KEY="your-gemini-key-here"
```

## Usage

### Basic usage

1. Stage your changes with `git add`
2. Run the tool:
   ```bash
   ai-commit-msg
   ```
3. Review the suggested commit message
4. Choose to use it (y), edit it (e), or cancel (n)

### Command-line options

```
--key "your-api-key"    Specify your Anthropic API key
--store-key             Store the provided API key in your system's credential manager
--auto                  Automatically commit using the generated message without confirmation
-v                      Enable verbose output (level 1)
-vv                     Enable more verbose output (level 2)
-vvv                    Enable debug output including full prompts (level 3)
-c, --context N         Number of context lines to include in the diff (default: 3)
-cc                     Include more context lines (10)
-ccc                    Include maximum context (entire file)
-p, --provider NAME     Specify LLM provider to use (anthropic, openai, gemini) (default: anthropic)
-m, --model MODEL       Specify model to use (provider-specific)
--list-providers        List available providers
--list-models           List available models for selected provider
--system-prompt PATH    Specify a custom system prompt file path
--user-prompt PATH      Specify a custom user prompt file path
--remember              Remember command-line options in config for future use
--help                  Display help information

Subcommands:
init-prompts           Initialize custom prompt files in your config directory
show-config           Display the current configuration
list-providers        List all supported AI providers
list-models          List available models (optionally for a specific provider)

Subcommand Details:
- `list-providers`: 
  Shows all supported AI providers for generating commit messages
  - Usage: `ai-commit-msg list-providers`
  
- `list-models`:
  Lists available models for all providers or for a specific provider
  - List models for all providers: `ai-commit-msg list-models`
  - List models for a specific provider: `ai-commit-msg list-models anthropic`
  
- `show-config`:
  Displays the current configuration settings, including:
  - Configuration directory
  - Current provider and model
  - Context lines
  - Verbosity level
  
- `init-prompts`:
  Initializes custom prompt files in your configuration directory
  - Usage: `ai-commit-msg init-prompts`

Examples:
  ai-commit-msg list-providers      # List all available providers
  ai-commit-msg list-models         # List models for all providers
  ai-commit-msg list-models anthropic  # List models only for Anthropic
  ai-commit-msg show-config         # Show current configuration
  ai-commit-msg init-prompts        # Initialize custom prompt files details and settings
```

### Context Control

The tool allows you to control how much surrounding code context is included when analyzing changes:

- Default (`ai-commit-msg`): 3 lines of context (git default)
- Medium context (`ai-commit-msg -cc`): 10 lines of context
- Maximum context (`ai-commit-msg -ccc`): Includes the entire file content
- Custom context (`ai-commit-msg --context 8`): Specify exact number of lines

Including more context can help the AI better understand the purpose and impact of your changes, especially for small modifications to complex code.

### Verbosity Levels

The tool supports multiple verbosity levels to provide more detailed information during operation:

- **Silent (default)**: Only shows essential output and prompts
- **Level 1** (`-v`): Shows basic operation logs, including file counts and timing
- **Level 2** (`-vv`): Shows more detailed logs with intermediate steps, file statistics, and branch information
- **Level 3** (`-vvv`): Shows debug-level information including the full prompts sent to the AI and detailed API responses

Use higher verbosity levels when:
- Troubleshooting issues with the tool
- Understanding exactly what data is being sent to the AI
- Diagnosing problems with API responses 
- Seeing detailed information about your git repository and changes

### Configuration System

The tool uses a flexible configuration system with the following precedence (highest to lowest):

1. Command-line arguments
2. Environment variables
3. Configuration file
4. Default values

Configuration is stored in the following locations, with paths prioritized based on platform conventions:

- **All Platforms**:
  1. `$XDG_CONFIG_HOME/ai-commit-msg/config.toml` (if XDG_CONFIG_HOME is set)
- **macOS**:
  1. `~/.config/ai-commit-msg/config.toml`
  2. `~/Library/Application Support/ai-commit-msg/config.toml`
- **Windows**:
  1. `%APPDATA%\ai-commit-msg\config.toml`
  2. `~/.config/ai-commit-msg/config.toml`
  3. `%USERPROFILE%\.config\ai-commit-msg\config.toml`
- **Fallback**:
  1. `~/.config/ai-commit-msg/config.toml`

The configuration path is dynamically selected for macOS, Windows, Linux, and other Unix-like systems, with `XDG_CONFIG_HOME` taking precedence across all platforms when set, ensuring broad compatibility and flexible configuration.

You can persist command-line options to the configuration file using the `--remember` flag:

```bash
# Remember to always use more context lines and the opus model
ai-commit-msg -cc --model claude-3-opus-20240229 --remember
```

Note that only settings that make sense across multiple commits are persisted:
- **Persisted**: Verbosity level, context lines, model name
- **Not persisted**: Jira issue ID, Jira description, auto-commit flag, store key flag (these are commit-specific or one-time operations)
- **Hardcoded**: Jira prefixes (GTN, GTBUG, TOOLS, TASK)

Environment variables use the prefix `AI_COMMIT_`:

```bash
export AI_COMMIT_VERBOSITY=2       # Set verbosity level
export AI_COMMIT_CONTEXT_LINES=5   # Set context lines
export AI_COMMIT_PROVIDER=openai   # Set provider (anthropic, openai, gemini)
export AI_COMMIT_MODEL_NAME=gpt-4  # Set model name
export AI_COMMIT_SYSTEM_PROMPT_PATH="/path/to/system_prompt.txt"  # Custom system prompt
export AI_COMMIT_USER_PROMPT_PATH="/path/to/user_prompt.txt"      # Custom user prompt
```

The configuration system is designed to be:
- **Non-intrusive**: Sensitive information (like API keys) is never stored in config files
- **Persistent**: Remember your preferences between runs
- **Flexible**: Multiple ways to configure based on your needs

A sample configuration file is provided in [examples/config.toml](examples/config.toml).

### Examples

Generate a commit message with interactive prompt:
```bash
ai-commit-msg
```

Generate with more context lines:
```bash
ai-commit-msg -cc
```

Generate with a specific number of context lines:
```bash
ai-commit-msg --context 8
```

Generate and automatically commit:
```bash
ai-commit-msg -a
```

Generate with different levels of verbosity:
```bash
ai-commit-msg -v     # Basic verbose output
ai-commit-msg -vv    # More detailed output with intermediate steps
ai-commit-msg -vvv   # Debug level output including full prompts
```

Store API key in credential manager:
```bash
ai-commit-msg --store-key --key sk_ant_your_key_here
```

Select a different AI provider:
```bash
ai-commit-msg --provider openai     # Use OpenAI (GPT) models
ai-commit-msg --provider gemini     # Use Google's Gemini models
ai-commit-msg --provider anthropic  # Use Anthropic's Claude models (default)
```

Use different models:
```bash
ai-commit-msg --provider anthropic --model claude-3-opus-20240229  # Use Claude Opus
ai-commit-msg --provider openai --model gpt-4                      # Use GPT-4
ai-commit-msg --provider gemini --model gemini-1.5-pro             # Use Gemini Pro
```

List available providers and models:
```bash
ai-commit-msg --list-providers      # List all supported providers
ai-commit-msg --list-models         # List all available models for each provider
```

Initialize custom prompt files:
```bash
ai-commit-msg init-prompts
```

Use custom prompt files:
```bash
ai-commit-msg --system-prompt /path/to/system_prompt.txt --user-prompt /path/to/user_prompt.txt
```

Remember settings for future use:
```bash
ai-commit-msg -cc --model claude-3-opus-20240229 --remember
```

View current configuration:
```bash
ai-commit-msg show-config    # Display the current configuration details
```

Display version:
```bash
ai-commit-msg --version     # Show the tool's version information
```

### Git alias (optional)

You can create a git alias for easier access:

```bash
git config --global alias.claude '!ai-commit-msg'
```

Then use:
```bash
git claude
```

## Multi-Provider Support

The tool now supports multiple Large Language Model (LLM) providers:

### Supported Providers

- **Anthropic (Claude)**: The original provider, offering Claude models (default)
- **OpenAI**: Support for GPT models, including GPT-4 and GPT-3.5 Turbo
- **Gemini**: Support for Google's Gemini models

### Provider Selection

You can select the provider to use with the `--provider` flag:

```bash
ai-commit-msg --provider anthropic  # Use Anthropic Claude (default)
ai-commit-msg --provider openai     # Use OpenAI GPT
ai-commit-msg --provider gemini     # Use Google Gemini
```

### Provider-Specific Models

Each provider has its own set of available models. You can list them with:

```bash
ai-commit-msg --list-models
```

And select a specific model with:

```bash
ai-commit-msg --provider openai --model gpt-4
```

### API Keys

Each provider requires its own API key. You can use environment variables:

```bash
export ANTHROPIC_API_KEY="your-anthropic-key"
export OPENAI_API_KEY="your-openai-key"
export GEMINI_API_KEY="your-gemini-key"
```

Or store them securely in your system's credential manager:

```bash
ai-commit-msg --provider anthropic --store-key --key your-anthropic-key
ai-commit-msg --provider openai --store-key --key your-openai-key
ai-commit-msg --provider gemini --store-key --key your-gemini-key
```

### Provider-Specific Prompts

You can customize prompts for each provider in the respective directories:

```
~/.config/ai-commit-msg/prompts/
  ├── anthropic/
  │    ├── system_prompt.txt
  │    └── user_prompt.txt
  ├── openai/
  │    ├── system_prompt.txt
  │    └── user_prompt.txt
  └── gemini/
       ├── system_prompt.txt
       └── user_prompt.txt
```

Initialize these directories with:

```bash
ai-commit-msg init-prompts
```

## Cross-Platform Support

The tool automatically detects your operating system and uses the appropriate credential manager:

- **macOS**: Uses the macOS Keychain for secure storage via the `security` command-line tool
- **Windows**: Uses the Windows Credential Manager via the `wincred` package
- **Linux**: No native credential store support, but environment variables still work

On systems without a supported credential manager, the tool will automatically fall back to using environment variables and will provide appropriate guidance.

### Windows Support

The Windows implementation uses the Windows Credential Manager (WCM) to securely store your API key:

- **Target Name**: `ai-commit-msg:anthropic-api-key` (combines service and account name)
- **Persistence**: LocalMachine level, making it available to all users on the machine

You can view and manage stored credentials using the Windows Credential Manager in Control Panel:
1. Open Control Panel
2. Go to User Accounts
3. Click on "Credential Manager"
4. Look under "Generic Credentials" for entries with the name `ai-commit-msg:anthropic-api-key`

## Customizing the Tool

### Customizing Jira Issue Types

The tool comes preconfigured with the following Jira issue type prefixes:
- GTN
- GTBUG  
- TOOLS
- TASK

If your organization uses different Jira issue type prefixes, you can add them by modifying the `pkg/git/jira.go` file:

1. Open `pkg/git/jira.go` in your editor
2. Locate the `JiraPrefixes` slice
3. Add your organization's Jira prefixes to the list
4. Rebuild the tool with `./build.sh`

```go
// Example: Adding custom Jira prefixes
var JiraPrefixes = []string{
    "GTN",
    "GTBUG",
    "TOOLS", 
    "TASK",
    "FEAT",    // Added custom prefix
    "PROJ-A",  // Added custom prefix
    "BUG",     // Added custom prefix
}
```

After adding your custom prefixes, the tool will recognize these in branch names and commit messages and include them properly in generated commit messages.

### Customizing Prompts

The tool uses Claude AI with carefully crafted prompts to generate commit messages. You can customize these prompts to change how the messages are generated:

#### Available prompt files:

Provider-specific prompts are organized in subdirectories:

- `anthropic/system_prompt.txt` - Instructions for Claude about commit message style and formatting
- `anthropic/user_prompt.txt` - Template for Claude with git diff information
- `openai/system_prompt.txt` - Instructions for GPT models
- `openai/user_prompt.txt` - Template for GPT models
- `gemini/system_prompt.txt` - Instructions for Gemini models
- `gemini/user_prompt.txt` - Template for Gemini models

#### Customizing without rebuilding:

You can customize the prompts without rebuilding the tool using one of these methods:

1. **Initialize custom prompt files in your config directory**:
   ```bash
   ai-commit-msg init-prompts
   ```
   This will initialize provider-specific prompt directories in your config directory (`~/.config/ai-commit-msg/prompts/[provider]/` or equivalent). You can then edit these files to customize prompts for each provider.

2. **Specify custom prompt file paths**:
   ```bash
   ai-commit-msg --system-prompt /path/to/system_prompt.txt --user-prompt /path/to/user_prompt.txt
   ```
   This allows you to use prompt files from any location.

3. **Set custom prompt paths in the config file**:
   ```toml
   system_prompt_path = "/path/to/system_prompt.txt"
   user_prompt_path = "/path/to/user_prompt.txt"
   ```

4. **Use environment variables**:
   ```bash
   export AI_COMMIT_SYSTEM_PROMPT_PATH="/path/to/system_prompt.txt"
   export AI_COMMIT_USER_PROMPT_PATH="/path/to/user_prompt.txt"
   ```

The tool follows this order of precedence when looking for prompt files:
1. Custom paths specified via command line flags
2. Custom paths specified in the config file or environment variables
3. Provider-specific files in the user's config directory (`~/.config/ai-commit-msg/prompts/[provider]/`)
4. Default provider files in the tool's installation directory (`prompts/[provider]/`)
5. Legacy default files in the tool's installation directory

#### When to customize:

- To change the commit message style (e.g., different format, more/less detail)
- To adapt messages for team-specific conventions
- To add specific requirements for your project
- To optimize the prompts for your specific workflow

## Development and Testing

### Running Tests

The project includes unit tests to ensure all functionality works correctly. To run the tests:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./pkg/key

# Run tests with code coverage
go test -cover ./...

# Generate code coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Testing Components

The tests are organized by component:

1. **API Key Management Tests** - Testing the handling of API keys from various sources:
   - Environment variables
   - System credential managers (macOS Keychain, Windows Credential Manager)
   - Command-line arguments
   - Interactive input

2. **Platform Detection Tests** - Testing the detection of the operating system and available credential stores

3. **Git Integration Tests** - Testing the git diff extraction and commit functionality

4. **Message Generation Tests** - Testing the Claude API integration and message processing

5. **Argument Parsing Tests** - Testing command-line argument handling

### Mock Testing

Some tests use mock implementations to avoid dependencies on external systems:

- Credential store operations are mocked to avoid modifying the actual system keychain
- Git operations can be mocked to test without a real git repository
- API calls to Claude are mocked to test without requiring a real API key

### Test Organization

Tests follow Go's standard pattern where test files are named `*_test.go` and placed alongside the code they test.

## Project Architecture

The project is organized into several packages:

- **cmd/ai-commit-msg**: Main application entry point
- **pkg/config**: Configuration management using Viper
- **pkg/key**: API key management with cross-platform credential store support
- **pkg/git**: Git operations and diff processing
- **pkg/ai**: Claude AI integration

### Configuration System

The configuration system (in `pkg/config`) uses the [Viper](https://github.com/spf13/viper) library to provide a flexible and powerful configuration experience:

- **Multiple sources**: Configuration can come from files, environment variables, and command-line flags
- **Automatic binding**: Environment variables are automatically mapped to configuration fields
- **Persistence**: Configuration can be saved to disk for future sessions
- **Thread-safety**: All access to configuration is protected by mutexes
- **XDG compliance**: Configuration files respect the XDG Base Directory Specification

### Credential Management

The key management system (in `pkg/key`) provides secure storage of API keys:

- **Platform detection**: Automatically detects the platform and uses the appropriate credential store
- **macOS support**: Uses the macOS Keychain via the `security` command-line tool
- **Windows support**: Uses the Windows Credential Manager via the `wincred` package
- **Environment fallback**: Falls back to environment variables when credential store is unavailable
- **Secure input**: Reads API keys without echoing them to the terminal

## Tips for best results

1. Stage only related changes together for more focused commit messages
2. For large changes, consider breaking them into smaller, logical commits
3. Use the `-cc` or `-ccc` flags when making small but significant changes
4. Use the edit option to refine messages when needed

## License

MIT
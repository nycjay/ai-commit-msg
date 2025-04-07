# AI Commit Message Generator

A tool that uses Claude AI to automatically generate high-quality git commit messages based on your staged changes.

## Features

- 🤖 Uses Claude AI to analyze your code changes and generate meaningful commit messages
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

## API Key Setup

You'll need an Anthropic API key to use this tool. The setup process is simple:

### 1. First-time setup (recommended)

Simply run the tool without any parameters:

```bash
ai-commit-msg
```

The tool will prompt you for your API key (input will be hidden) and offer to store it securely in your system's credential manager.

### 2. Store directly in credential manager

```bash
./ai-commit-msg --store-key --key your-api-key-here
```

### 3. Set as environment variable

```bash
export ANTHROPIC_API_KEY="your-api-key-here"
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
-v                     Enable verbose output (level 1)
-vv                     Enable more verbose output (level 2)
-vvv                    Enable debug output including full prompts (level 3)
-c, --context N         Number of context lines to include in the diff (default: 3)
-cc                     Include more context lines (10)
-ccc                    Include maximum context (entire file)
--help                  Display help information
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

### Git alias (optional)

You can create a git alias for easier access:

```bash
git config --global alias.claude '!ai-commit-msg'
```

Then use:
```bash
git claude
```

## Cross-Platform Support

The tool automatically detects your operating system and uses the appropriate credential manager:

- **macOS**: Uses the macOS Keychain for secure storage
- **Windows**: Uses the Windows Credential Manager for secure storage
- **Linux**: No native credential store support, but environment variables still work

On systems without a supported credential manager, the tool will automatically fall back to using environment variables and will provide appropriate guidance.

## Customizing Prompts

The tool uses Claude AI with carefully crafted prompts to generate commit messages. Developers can customize these prompts to change how the messages are generated:

### Available prompt files:

- `prompts/system_prompt.txt` - Contains instructions for the AI about commit message style and formatting
- `prompts/user_prompt.txt` - Template used to format the request to the AI with your git diff information

### When to customize:

- To change the commit message style (e.g., different format, more/less detail)
- To adapt messages for team-specific conventions
- To add specific requirements for your project

These customizations require rebuilding the tool since the prompts are compiled into the executable.

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

## Tips for best results

1. Stage only related changes together for more focused commit messages
2. For large changes, consider breaking them into smaller, logical commits
3. Use the `-cc` or `-ccc` flags when making small but significant changes
4. Use the edit option to refine messages when needed

## License

MIT
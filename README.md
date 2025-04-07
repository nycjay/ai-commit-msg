# AI Commit Message Generator

A tool that uses Claude AI to automatically generate high-quality git commit messages based on your staged changes.

## Features

- 🤖 Uses Claude AI to analyze your code changes and generate meaningful commit messages
- 📝 Follows best practices for commit messages (conventional commits, imperative mood)
- 🚀 No dependencies - single binary distribution
- 🔐 Securely stores API key in macOS Keychain
- ✏️ Interactive mode allows you to edit messages before committing
- 🔄 Can be integrated directly into your git workflow

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

The tool will prompt you for your API key (input will be hidden) and offer to store it securely in your macOS Keychain.

### 2. Store directly in macOS Keychain

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
--store-key             Store the provided API key in Mac keychain
--auto                  Automatically commit using the generated message without confirmation
--verbose               Enable verbose output for debugging
--help                  Display help information
```

### Examples

Generate a commit message with interactive prompt:
```bash
ai-commit-msg
```

Generate and automatically commit:
```bash
ai-commit-msg --auto
```

Store API key in keychain:
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

## Tips for best results

1. Stage only related changes together for more focused commit messages
2. For large changes, consider breaking them into smaller, logical commits
3. The tool works best when diff sizes are reasonable (under 4000 tokens)
4. Use the edit option to refine messages when needed

## License

MIT
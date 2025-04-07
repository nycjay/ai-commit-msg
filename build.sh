#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Building AI Commit Message Generator...${NC}"

# Ensure we're in the script's directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Clean any previous build
if [ -f "ai-commit-msg" ]; then
  echo "Removing previous build..."
  rm ai-commit-msg
fi

# Build the tool
echo "Compiling with Go..."
go build -o ai-commit-msg ./cmd/ai-commit-msg

# Make executable
chmod +x ai-commit-msg

# Check if build was successful
if [ -f "ai-commit-msg" ]; then
  echo -e "${GREEN}Build successful!${NC}"
  echo ""
  echo "Getting Started:"
  echo "  1. Simply run ./ai-commit-msg in any git repository"
  echo "  2. If it's your first time, you'll be prompted for your API key"
  echo "  3. Your API key will be stored securely in your system's credential manager"
  echo ""
  echo "For more options:"
  echo "  ./ai-commit-msg --help"
  echo ""
else
  echo "Build failed!"
  exit 1
fi

# Get OS type for platform-specific instructions
OS="$(uname)"

# Optionally create a symlink to make it available in PATH
if [ "$OS" = "Darwin" ] || [ "$OS" = "Linux" ]; then
  echo -e "${YELLOW}Would you like to create a symlink in /usr/local/bin? (y/n)${NC}"
  read -r response
  if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    sudo ln -sf "$SCRIPT_DIR/ai-commit-msg" /usr/local/bin/ai-commit-msg
    echo -e "${GREEN}Symlink created! You can now run 'ai-commit-msg' from anywhere.${NC}"
  fi
elif [ "$OS" = "MINGW" ] || [ "$OS" = "MSYS" ] || [ "$OS" = "CYGWIN" ]; then
  echo "Note: On Windows, you may want to add this directory to your PATH to run ai-commit-msg from anywhere."
fi

echo -e "${GREEN}Done!${NC}"
#!/bin/bash

# Path to the ai-commit-msg binary
# Update this path if you've installed it somewhere else
AI_COMMIT_MSG="$HOME/code/go/ai-commit-msg/ai-commit-msg"

# Check if the binary exists
if [ ! -f "$AI_COMMIT_MSG" ]; then
  echo "Error: ai-commit-msg binary not found at $AI_COMMIT_MSG"
  echo "Please update the path in this script or build the tool first."
  exit 1
fi

# Check if we're in a git repository
if ! git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
  echo "Error: Not in a git repository."
  exit 1
fi

# Check if there are staged changes
if [ -z "$(git diff --cached --name-only)" ]; then
  echo "No staged changes found. Stage your changes using 'git add' first."
  exit 1
fi

# Run the tool with all arguments passed to this script
"$AI_COMMIT_MSG" "$@"
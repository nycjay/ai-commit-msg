package ai

import (
	"github.com/nycjay/ai-commit-msg/pkg/git"
)

// GenerateCommitMessage generates a commit message using Claude AI
func GenerateCommitMessage(apiKey string, diffInfo git.GitDiff) (string, error) {
	// This is a stub function to be implemented later
	return "feat: Sample commit message", nil
}

// Request represents a request to the Claude API
type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []Message `json:"messages"`
}

// Message represents a message in the Claude API request
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents a response from the Claude API
type Response struct {
	Content []Content `json:"content"`
}

// Content represents content in the Claude API response
type Content struct {
	Text string `json:"text"`
}

package ai

import (
	"github.com/nycjay/ai-commit-msg/pkg/git"
)

// Provider represents an LLM provider interface
type Provider interface {
	// GenerateCommitMessage generates a commit message using the provider's LLM
	GenerateCommitMessage(apiKey string, modelName string, diffInfo git.GitDiff) (string, error)
	
	// ValidateAPIKey validates the format of the API key
	ValidateAPIKey(key string) bool
	
	// GetName returns the provider name
	GetName() string
	
	// GetDefaultModel returns the default model name
	GetDefaultModel() string
	
	// GetAvailableModels returns the list of available models
	GetAvailableModels() []string
}

// ProviderType enumerates the supported LLM providers
type ProviderType string

const (
	// ProviderAnthropic represents Anthropic's Claude models
	ProviderAnthropic ProviderType = "anthropic"
	
	// ProviderOpenAI represents OpenAI's GPT models
	ProviderOpenAI ProviderType = "openai"
	
	// ProviderGemini represents Google's Gemini models
	ProviderGemini ProviderType = "gemini"
)

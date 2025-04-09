package ai

import (
	"fmt"
	"strings"
)

// NewProvider creates a provider based on the provider name
func NewProvider(providerName string) (Provider, error) {
	if providerName == "" {
		return nil, fmt.Errorf("provider name cannot be empty")
	}
	
	switch strings.ToLower(providerName) {
	case string(ProviderAnthropic), "claude":
		return NewAnthropicProvider(), nil
	case string(ProviderOpenAI), "gpt":
		return NewOpenAIProvider(), nil
	case string(ProviderGemini), "google":
		return NewGeminiProvider(), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}

// GetAllProviders returns all available providers
func GetAllProviders() []Provider {
	return []Provider{
		NewAnthropicProvider(),
		NewOpenAIProvider(),
		NewGeminiProvider(),
	}
}

// GetProviderByName returns a provider by its name
func GetProviderByName(name string) Provider {
	// Return nil for unknown or empty provider names
	if name == "" {
		return nil
	}
	
	provider, err := NewProvider(name)
	if err != nil {
		return nil
	}
	return provider
}

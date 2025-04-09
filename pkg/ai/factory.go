package ai

import (
	"fmt"
	"strings"
)

// NewProvider creates a provider based on the provider name
func NewProvider(providerName string) (Provider, error) {
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
	provider, err := NewProvider(name)
	if err != nil {
		// Default to Anthropic if provider not found
		return NewAnthropicProvider()
	}
	return provider
}

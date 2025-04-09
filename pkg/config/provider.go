package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SetProvider sets the provider
func (c *Config) SetProvider(provider string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Provider = provider
}

// GetProviderModel returns the model for a specific provider
func (c *Config) GetProviderModel(provider string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	provider = strings.ToLower(provider)
	
	// If provider matches the legacy provider (anthropic), use the legacy model setting for backward compatibility
	if provider == "anthropic" && c.ModelName != "" {
		return c.ModelName
	}
	
	// Check provider-specific model map
	if c.ProviderModels != nil {
		if model, ok := c.ProviderModels[provider]; ok && model != "" {
			return model
		}
	}
	
	// Default models if not configured
	switch provider {
	case "openai", "gpt":
		return "gpt-4o"
	case "gemini", "google":
		return "gemini-1.5-pro"
	default:
		return "claude-3-haiku-20240307" // Default to Anthropic model
	}
}

// SetProviderModel sets the model for a specific provider
func (c *Config) SetProviderModel(provider, model string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	provider = strings.ToLower(provider)
	
	// For anthropic, also set the legacy ModelName for backward compatibility
	if provider == "anthropic" {
		c.ModelName = model
	}
	
	// Ensure map is initialized
	if c.ProviderModels == nil {
		c.ProviderModels = make(map[string]string)
	}
	
	// Set the provider-specific model
	c.ProviderModels[provider] = model
}

// GetProviderKey returns the API key for a specific provider
func (c *Config) GetProviderKey(provider string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	provider = strings.ToLower(provider)
	
	// Check provider keys map first
	if c.ProviderKeys != nil {
		if key, ok := c.ProviderKeys[provider]; ok && key != "" {
			return key
		}
	}
	
	// For anthropic, also check the legacy APIKey field for backward compatibility
	if provider == "anthropic" && c.APIKey != "" {
		return c.APIKey
	}
	
	return ""
}

// SetProviderKey sets the API key for a specific provider
func (c *Config) SetProviderKey(provider, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	provider = strings.ToLower(provider)
	
	// Ensure map is initialized
	if c.ProviderKeys == nil {
		c.ProviderKeys = make(map[string]string)
	}
	
	// Set the provider-specific key
	c.ProviderKeys[provider] = key
	
	// For anthropic, also set the legacy APIKey field for backward compatibility
	if provider == "anthropic" {
		c.APIKey = key
	}
}

// StoreProviderAPIKey stores the API key for a specific provider
func (c *Config) StoreProviderAPIKey(provider, key string) error {
	return c.keyManager.StoreProviderKey(provider, key)
}

// GetProviderPromptPath returns the path for provider-specific prompts
func (c *Config) GetProviderPromptPath(provider, promptType string) (string, error) {
	promptDir, err := c.GetPromptDirectory()
	if err != nil {
		return "", err
	}
	
	// Provider-specific prompt directory
	providerDir := filepath.Join(promptDir, strings.ToLower(provider))
	
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(providerDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create provider prompt directory: %v", err)
	}
	
	return filepath.Join(providerDir, promptType), nil
}

// ReadProviderPrompt reads a prompt file for a specific provider
func (c *Config) ReadProviderPrompt(provider, promptFile string) (string, error) {
	provider = strings.ToLower(provider)
	
	// Try custom prompt paths first (these override provider-specific paths)
	var customPath string
	if promptFile == "system_prompt.txt" && c.GetSystemPromptPath() != "" {
		customPath = c.GetSystemPromptPath()
	} else if promptFile == "user_prompt.txt" && c.GetUserPromptPath() != "" {
		customPath = c.GetUserPromptPath()
	}
	
	if customPath != "" {
		content, err := os.ReadFile(customPath)
		if err == nil {
			return string(content), nil
		}
	}
	
	// Try provider-specific path in config directory
	promptDir, err := c.GetPromptDirectory()
	if err == nil {
		providerPromptDir := filepath.Join(promptDir, provider)
		promptPath := filepath.Join(providerPromptDir, promptFile)
		
		if _, err := os.Stat(promptPath); err == nil {
			content, err := os.ReadFile(promptPath)
			if err == nil {
				return string(content), nil
			}
		}
	}
	
	// If not found and provider isn't anthropic, fall back to anthropic prompts
	if provider != "anthropic" {
		anthropicPromptDir := filepath.Join(promptDir, "anthropic")
		promptPath := filepath.Join(anthropicPromptDir, promptFile)
		
		if _, err := os.Stat(promptPath); err == nil {
			content, err := os.ReadFile(promptPath)
			if err == nil {
				return string(content), nil
			}
		}
	}
	
	// Try root prompt directory (legacy location)
	promptPath := filepath.Join(promptDir, promptFile)
	if _, err := os.Stat(promptPath); err == nil {
		content, err := os.ReadFile(promptPath)
		if err == nil {
			return string(content), nil
		}
	}
	
	// Finally, try prompts directory relative to executable
	// Note: This requires executableDir to be accessible
	executableDir := c.GetExecutableDir()
	if executableDir != "" {
		promptsDir := filepath.Join(executableDir, "prompts")
		promptPath := filepath.Join(promptsDir, promptFile)
		
		content, err := os.ReadFile(promptPath)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file from any location: %v", err)
		}
		
		return string(content), nil
	}
	
	return "", fmt.Errorf("could not find prompt file %s for provider %s", promptFile, provider)
}

// SetExecutableDir sets the executable directory for prompt loading
func (c *Config) SetExecutableDir(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.executableDir = dir
}

// GetExecutableDir gets the executable directory
func (c *Config) GetExecutableDir() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.executableDir
}



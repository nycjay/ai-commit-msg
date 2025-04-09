package key

import (
	"fmt"
	"os"
	"strings"
)

// GetProviderKeyInfo returns the account name and environment variable for a provider
func (k *KeyManager) GetProviderKeyInfo(provider string) (string, string) {
	provider = strings.ToLower(provider)
	
	switch provider {
	case "openai", "gpt":
		return OpenAIAccount, OpenAIEnvVar
	case "gemini", "google":
		return GeminiAccount, GeminiEnvVar
	default: // Default to Anthropic
		return KeychainAccount, EnvVarName
	}
}

// GetProviderKey gets the API key for a specific provider using all available methods
func (k *KeyManager) GetProviderKey(provider string, cmdLineKey string) (string, error) {
	account, envVar := k.GetProviderKeyInfo(provider)
	
	// Check if key was provided via command line
	if cmdLineKey != "" {
		k.log("Using API key provided via command line for provider: %s", provider)
		return cmdLineKey, nil
	}

	// Try environment variable
	envKey := os.Getenv(envVar)
	if envKey != "" {
		k.log("Using API key from environment variable: %s", envVar)
		return envKey, nil
	}

	// Try credential store
	k.log("No API key found in environment, checking credential store for account: %s", account)
	credStoreKey, err := k.getFromCredentialStore(account)
	if err != nil {
		k.log("Error retrieving API key from credential store: %v", err)
		return "", err
	}

	if credStoreKey != "" {
		k.log("Using API key from credential store")
		return credStoreKey, nil
	}

	return "", fmt.Errorf("no API key found for provider: %s", provider)
}

// StoreProviderKey stores the API key for a specific provider
func (k *KeyManager) StoreProviderKey(provider string, apiKey string) error {
	account, _ := k.GetProviderKeyInfo(provider)
	k.log("Storing API key for provider: %s with account: %s", provider, account)
	return k.storeInCredentialStore(account, apiKey)
}

// getFromCredentialStore is a wrapper to get from credential store with a specific account
func (k *KeyManager) getFromCredentialStore(account string) (string, error) {
	// Save the original account
	originalAccount := k.keychainAccount
	
	// Set the account for this operation
	k.keychainAccount = account
	
	// Get the key
	key, err := k.getFromKeychainFn()
	
	// Restore the original account
	k.keychainAccount = originalAccount
	
	return key, err
}

// storeInCredentialStore is a wrapper to store in credential store with a specific account
func (k *KeyManager) storeInCredentialStore(account string, key string) error {
	// Save the original account
	originalAccount := k.keychainAccount
	
	// Set the account for this operation
	k.keychainAccount = account
	
	// Store the key
	err := k.storeInKeychainFn(key)
	
	// Restore the original account
	k.keychainAccount = originalAccount
	
	return err
}

// ValidateProviderKey validates an API key for a specific provider
func (k *KeyManager) ValidateProviderKey(provider string, key string) bool {
	provider = strings.ToLower(provider)
	
	// Basic validation that applies to all providers
	if len(key) < 20 {
		return false
	}
	
	switch provider {
	case "openai", "gpt":
		return strings.HasPrefix(key, "sk-")
	case "gemini", "google":
		return len(key) >= 30 // Gemini API keys are typically 39 characters
	default: // Anthropic
		return strings.HasPrefix(key, "sk_ant_") ||
			strings.HasPrefix(key, "sk-ant-") ||
			strings.HasPrefix(key, "sk-") ||
			strings.HasPrefix(key, "sk-o-") ||
			strings.HasPrefix(key, "kp-")
	}
}

package key

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// KeychainService is the service name used in the Mac keychain
	KeychainService = "ai-commit-msg"
	
	// KeychainAccount is the account name used in the Mac keychain
	KeychainAccount = "anthropic-api-key"
	
	// EnvVarName is the environment variable name for the API key
	EnvVarName = "ANTHROPIC_API_KEY"
)

// KeychainGetter defines a function type for getting from keychain
type KeychainGetter func() (string, error)

// KeychainStorer defines a function type for storing in keychain
type KeychainStorer func(string) error

// KeyManager handles API key operations
type KeyManager struct {
	keychainService string
	keychainAccount string
	envVarName      string
	verbose         bool
	
	// Function fields for easier testing
	getFromKeychainFn KeychainGetter
	storeInKeychainFn KeychainStorer
}

// NewKeyManager creates a new KeyManager instance
func NewKeyManager(verbose bool) *KeyManager {
	km := &KeyManager{
		keychainService: KeychainService,
		keychainAccount: KeychainAccount,
		envVarName:      EnvVarName,
		verbose:         verbose,
	}
	
	// Initialize with the default implementations
	km.getFromKeychainFn = km.defaultGetFromKeychain
	km.storeInKeychainFn = km.defaultStoreInKeychain
	
	return km
}

// log prints a message if verbose mode is enabled
func (k *KeyManager) log(format string, args ...interface{}) {
	if k.verbose {
		fmt.Printf(format+"\n", args...)
	}
}

// GetFromEnvironment retrieves the API key from the environment variable
func (k *KeyManager) GetFromEnvironment() string {
	k.log("Checking for API key in environment variable: %s", k.envVarName)
	return os.Getenv(k.envVarName)
}

// GetFromKeychain retrieves the API key from Mac keychain
func (k *KeyManager) GetFromKeychain() (string, error) {
	return k.getFromKeychainFn()
}

// defaultGetFromKeychain is the default implementation of getting from keychain
func (k *KeyManager) defaultGetFromKeychain() (string, error) {
	k.log("Executing keychain command to retrieve API key...")
	cmd := exec.Command("security", "find-generic-password", "-s", k.keychainService, "-a", k.keychainAccount, "-w")
	output, err := cmd.Output()
	if err != nil {
		// Don't return the error details as they might contain sensitive info or be verbose
		return "", fmt.Errorf("failed to retrieve API key from keychain")
	}
	return strings.TrimSpace(string(output)), nil
}

// StoreInKeychain stores the API key in Mac keychain
func (k *KeyManager) StoreInKeychain(apiKey string) error {
	return k.storeInKeychainFn(apiKey)
}

// defaultStoreInKeychain is the default implementation of storing in keychain
func (k *KeyManager) defaultStoreInKeychain(apiKey string) error {
	// First, try to delete any existing entry
	k.log("Deleting any existing keychain entry...")
	deleteCmd := exec.Command("security", "delete-generic-password", "-s", k.keychainService, "-a", k.keychainAccount)
	// Ignore errors from delete as the entry might not exist
	_ = deleteCmd.Run()

	// Add the new password
	k.log("Adding new keychain entry (service='%s', account='%s')...", k.keychainService, k.keychainAccount)
	addCmd := exec.Command("security", "add-generic-password", "-s", k.keychainService, "-a", k.keychainAccount, "-w", apiKey)
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to store API key in keychain")
	}
	return nil
}

// GetKey retrieves the API key following the precedence order:
// 1. command-line arg (if provided)
// 2. environment variable
// 3. keychain
func (k *KeyManager) GetKey(cmdLineKey string) (string, error) {
	// Check if key was provided via command line
	if cmdLineKey != "" {
		k.log("Using API key provided via command line")
		return cmdLineKey, nil
	}

	// Try environment variable
	envKey := k.GetFromEnvironment()
	if envKey != "" {
		k.log("Using API key from environment variable")
		return envKey, nil
	}

	// Try Mac keychain
	k.log("No API key found in environment, checking Mac keychain...")
	keychainKey, err := k.GetFromKeychain()
	if err != nil {
		k.log("Error retrieving API key from keychain: %v", err)
		return "", err
	}

	if keychainKey != "" {
		k.log("Using API key from keychain")
		return keychainKey, nil
	}

	return "", fmt.Errorf("no API key found")
}

// ValidateKey performs basic validation on the API key format
func (k *KeyManager) ValidateKey(apiKey string) bool {
	// Basic validation - Claude API keys typically start with "sk_ant_"
	// and have a minimum length
	return len(apiKey) >= 20 && strings.HasPrefix(apiKey, "sk_ant_")
}

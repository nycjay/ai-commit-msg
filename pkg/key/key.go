package key

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	// KeychainService is the service name used in credential stores
	KeychainService = "ai-commit-msg"
	
	// Provider-specific account names for credential stores
	KeychainAccount = "anthropic-api-key" // Original Anthropic account
	OpenAIAccount   = "openai-api-key"    // OpenAI account
	GeminiAccount   = "gemini-api-key"    // Gemini account
	
	// Provider-specific environment variable names
	EnvVarName      = "ANTHROPIC_API_KEY" // Original Anthropic env var
	OpenAIEnvVar    = "OPENAI_API_KEY"    // OpenAI env var
	GeminiEnvVar    = "GEMINI_API_KEY"    // Gemini env var
)

// Platform represents the OS platform for credential storage
type Platform string

const (
	// PlatformMac represents macOS
	PlatformMac Platform = "darwin"
	
	// PlatformWindows represents Windows
	PlatformWindows Platform = "windows"
	
	// PlatformLinux represents Linux
	PlatformLinux Platform = "linux"
	
	// PlatformUnknown represents an unknown platform
	PlatformUnknown Platform = "unknown"
)

// KeychainGetter defines a function type for getting from credential store
type KeychainGetter func() (string, error)

// KeychainStorer defines a function type for storing in credential store
type KeychainStorer func(string) error

// KeyManager handles API key operations
type KeyManager struct {
	keychainService string
	keychainAccount string
	envVarName      string
	verbose         bool
	platform        Platform
	
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
		platform:        detectPlatform(),
	}
	
	// Initialize with the platform-specific implementations
	// The platformGetFromCredentialStore function will handle platform detection
	km.getFromKeychainFn = km.platformGetFromCredentialStore
	km.storeInKeychainFn = km.platformStoreInCredentialStore
	
	return km
}

// detectPlatform determines the OS platform
func detectPlatform() Platform {
	switch runtime.GOOS {
	case "darwin":
		return PlatformMac
	case "windows":
		return PlatformWindows
	case "linux":
		return PlatformLinux
	default:
		return PlatformUnknown
	}
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

// GetFromKeychain retrieves the API key from credential store
func (k *KeyManager) GetFromKeychain() (string, error) {
	return k.getFromKeychainFn()
}

// platformGetFromCredentialStore retrieves the API key from the platform-specific credential store
func (k *KeyManager) platformGetFromCredentialStore() (string, error) {
	// Simple platform check based on detected platform at runtime
	switch k.platform {
	case PlatformMac:
		if runtime.GOOS == "darwin" {
			// On macOS, use the security command to access the keychain
			k.log("Executing keychain command to retrieve API key...")
			cmd := exec.Command("security", "find-generic-password", "-s", k.keychainService, "-a", k.keychainAccount, "-w")
			output, err := cmd.Output()
			if err != nil {
				// Don't return the error details as they might contain sensitive info or be verbose
				return "", fmt.Errorf("failed to retrieve API key from macOS keychain")
			}
			return strings.TrimSpace(string(output)), nil
		}
		return "", fmt.Errorf("macOS Keychain is only available on macOS")
	case PlatformWindows:
		// On Windows, we can't directly use the Windows Credential Manager here
		// because of build constraints - Windows support is in the Windows-specific file
		return "", fmt.Errorf("Windows Credential Manager is only available on Windows")
	default:
		return "", fmt.Errorf("no credential store available for platform: %s", k.platform)
	}
}

// StoreInKeychain stores the API key in the credential store
func (k *KeyManager) StoreInKeychain(apiKey string) error {
	return k.storeInKeychainFn(apiKey)
}

// platformStoreInCredentialStore stores the API key in the platform-specific credential store
func (k *KeyManager) platformStoreInCredentialStore(apiKey string) error {
	// Simple platform check based on detected platform at runtime
	switch k.platform {
	case PlatformMac:
		if runtime.GOOS == "darwin" {
			// On macOS, use the security command to access the keychain
			
			// First, try to delete any existing entry
			k.log("Deleting any existing keychain entry...")
			deleteCmd := exec.Command("security", "delete-generic-password", "-s", k.keychainService, "-a", k.keychainAccount)
			// Ignore errors from delete as the entry might not exist
			_ = deleteCmd.Run()
			
			// Add the new password
			k.log("Adding new keychain entry (service='%s', account='%s')...", k.keychainService, k.keychainAccount)
			addCmd := exec.Command("security", "add-generic-password", "-s", k.keychainService, "-a", k.keychainAccount, "-w", apiKey)
			if err := addCmd.Run(); err != nil {
				return fmt.Errorf("failed to store API key in macOS keychain")
			}
			return nil
		}
		return fmt.Errorf("macOS Keychain is only available on macOS")
	case PlatformWindows:
		// On Windows, we can't directly use the Windows Credential Manager here
		// because of build constraints - Windows support is in the Windows-specific file
		return fmt.Errorf("Windows Credential Manager is only available on Windows")
	default:
		return fmt.Errorf("no credential store available for platform: %s", k.platform)
	}
}

// The original method is now replaced by the enhanced version with provider support in provider_keys.go

// GetKeyLegacy is the legacy version of GetKey without provider support
// It's kept for backward compatibility, but should not be called directly.
func (k *KeyManager) GetKeyLegacy(cmdLineKey string) (string, error) {
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

	// Try credential store
	k.log("No API key found in environment, checking credential store...")
	credStoreKey, err := k.GetFromKeychain()
	if err != nil {
		k.log("Error retrieving API key from credential store: %v", err)
		return "", err
	}

	if credStoreKey != "" {
		k.log("Using API key from credential store")
		return credStoreKey, nil
	}

	return "", fmt.Errorf("no API key found")
}

// ValidateKey performs basic validation on the API key format
func (k *KeyManager) ValidateKey(apiKey string) bool {
	// Anthropic API keys may have different formats:
	// - Older format: "sk_ant_..." (starts with sk_ant_)
	// - Newer format: "sk-ant-..." (uses hyphens instead of underscores)
	// We check for minimum length and either prefix
	return len(apiKey) >= 20 && (
		strings.HasPrefix(apiKey, "sk_ant_") ||
		strings.HasPrefix(apiKey, "sk-ant-") ||
		strings.HasPrefix(apiKey, "sk-") ||  // Some newer keys may just start with sk-
		strings.HasPrefix(apiKey, "sk-o-") || // Organization keys
		strings.HasPrefix(apiKey, "kp-"))     // Other possible prefix
}

// GetPlatform returns the detected OS platform
func (k *KeyManager) GetPlatform() Platform {
	return k.platform
}

// CredentialStoreAvailable returns whether a credential store is available for the current platform
func (k *KeyManager) CredentialStoreAvailable() bool {
	// Both macOS and Windows now have fully implemented credential stores
	return k.platform == PlatformMac || k.platform == PlatformWindows
}

// GetCredentialStoreName returns a user-friendly name for the current credential store
func (k *KeyManager) GetCredentialStoreName() string {
	switch k.platform {
	case PlatformMac:
		return "macOS Keychain"
	case PlatformWindows:
		return "Windows Credential Manager"
	default:
		return "none"
	}
}

// SetVerbose sets the verbosity of the key manager
func (k *KeyManager) SetVerbose(verbose bool) {
	k.verbose = verbose
}

// This is defined in provider_keys.go - in key.go we just define GetKeyLegacy

// This is used internally by the provider_keys.go implementation
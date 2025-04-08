//go:build ignore
// +build ignore

package key

import (
	"fmt"
)

// This file contains platform-agnostic implementations of the credential store functions
// that are used when the platform-specific files are not built (e.g., during tests).
// These stubs are not used in production builds, but are useful for IDE completion and tests.

// macGetFromKeychain is a generic implementation
func (k *KeyManager) macGetFromKeychain() (string, error) {
	return "", fmt.Errorf("macOS Keychain access not implemented")
}

// macStoreInKeychain is a generic implementation
func (k *KeyManager) macStoreInKeychain(apiKey string) error {
	return fmt.Errorf("macOS Keychain access not implemented")
}

// windowsGetFromCredentialManager is a generic implementation
func (k *KeyManager) windowsGetFromCredentialManager() (string, error) {
	return "", fmt.Errorf("Windows Credential Manager access not implemented")
}

// windowsStoreInCredentialManager is a generic implementation
func (k *KeyManager) windowsStoreInCredentialManager(apiKey string) error {
	return fmt.Errorf("Windows Credential Manager access not implemented")
}

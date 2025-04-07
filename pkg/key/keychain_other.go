//go:build !darwin && !windows
// +build !darwin,!windows

package key

import (
	"fmt"
)

// These are stub implementations for platforms without native credential store support

// macGetFromKeychain is a stub for non-Mac platforms
func (k *KeyManager) macGetFromKeychain() (string, error) {
	return "", fmt.Errorf("macOS keychain is not available on this platform")
}

// macStoreInKeychain is a stub for non-Mac platforms
func (k *KeyManager) macStoreInKeychain(apiKey string) error {
	return fmt.Errorf("macOS keychain is not available on this platform")
}

// windowsGetFromCredentialManager is a stub for non-Windows platforms
func (k *KeyManager) windowsGetFromCredentialManager() (string, error) {
	return "", fmt.Errorf("Windows Credential Manager is not available on this platform")
}

// windowsStoreInCredentialManager is a stub for non-Windows platforms
func (k *KeyManager) windowsStoreInCredentialManager(apiKey string) error {
	return fmt.Errorf("Windows Credential Manager is not available on this platform")
}

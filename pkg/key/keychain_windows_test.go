//go:build windows
// +build windows

package key

import (
	"testing"
)

func TestWindowsCredentialManager(t *testing.T) {
	// This test will only run on Windows
	km := NewKeyManager(true)

	// Make sure we detect the platform correctly
	if km.GetPlatform() != PlatformWindows {
		t.Errorf("Expected platform to be Windows, got %v", km.GetPlatform())
	}

	// Make sure credential store is available
	if !km.CredentialStoreAvailable() {
		t.Errorf("Expected Windows Credential Manager to be available")
	}

	// Try storing a dummy API key - this is a basic test, 
	// so we won't actually check the value in the credential store
	testKey := "sk_ant_test_key_for_windows_credential_manager"
	err := km.StoreInKeychain(testKey)
	if err != nil {
		t.Errorf("Error storing key in Windows Credential Manager: %v", err)
	}

	// Try retrieving the key
	retrievedKey, err := km.GetFromKeychain()
	if err != nil {
		t.Errorf("Error retrieving key from Windows Credential Manager: %v", err)
	}

	// Verify the key matches what we stored
	if retrievedKey != testKey {
		t.Errorf("Retrieved key does not match stored key. Got %s, expected %s", retrievedKey, testKey)
	}
}

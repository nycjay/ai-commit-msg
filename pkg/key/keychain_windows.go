//go:build windows
// +build windows

package key

import (
	"fmt"

	"github.com/danieljoos/wincred"
)

// getFromWindowsCredentialManager is the platform-facing function for Windows credential access
// This is used by the platform-neutral code
func (k *KeyManager) getFromWindowsCredentialManager() (string, error) {
	return k.windowsGetFromCredentialManager()
}

// storeInWindowsCredentialManager is the platform-facing function for Windows credential storage
// This is used by the platform-neutral code
func (k *KeyManager) storeInWindowsCredentialManager(apiKey string) error {
	return k.windowsStoreInCredentialManager(apiKey)
}

// windowsGetFromCredentialManager retrieves the API key from Windows Credential Manager
// This is the internal implementation
func (k *KeyManager) windowsGetFromCredentialManager() (string, error) {
	k.log("Retrieving API key from Windows Credential Manager...")
	
	// Create a credential target name that includes the service
	targetName := fmt.Sprintf("%s:%s", k.keychainService, k.keychainAccount)
	
	// Attempt to get the credential from Windows Credential Manager
	cred, err := wincred.GetGenericCredential(targetName)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve API key from Windows Credential Manager: %v", err)
	}
	
	// Return the credential as a string
	return string(cred.CredentialBlob), nil
}

// windowsStoreInCredentialManager stores the API key in Windows Credential Manager
// This is the internal implementation
func (k *KeyManager) windowsStoreInCredentialManager(apiKey string) error {
	k.log("Storing API key in Windows Credential Manager...")
	
	// Create a credential target name that includes the service
	targetName := fmt.Sprintf("%s:%s", k.keychainService, k.keychainAccount)
	
	// First, try to delete any existing credential with the same target name
	existing, err := wincred.GetGenericCredential(targetName)
	if err == nil && existing != nil {
		k.log("Deleting existing credential...")
		err = existing.Delete()
		if err != nil {
			k.log("Warning: Could not delete existing credential: %v", err)
			// Continue anyway, we'll try to overwrite it
		}
	}
	
	// Create a new generic credential
	cred := wincred.NewGenericCredential(targetName)
	cred.UserName = k.keychainAccount
	cred.CredentialBlob = []byte(apiKey)
	cred.Persist = wincred.PersistLocalMachine
	cred.Comment = "API key for AI Commit Message Generator"
	
	// Write the credential to the Windows Credential Manager
	err = cred.Write()
	if err != nil {
		return fmt.Errorf("failed to store API key in Windows Credential Manager: %v", err)
	}
	
	return nil
}

//go:build darwin
// +build darwin

package key

import (
	"fmt"
	"os/exec"
	"strings"
)

// macGetFromKeychain retrieves the API key from Mac keychain
func (k *KeyManager) macGetFromKeychain() (string, error) {
	k.log("Executing keychain command to retrieve API key...")
	cmd := exec.Command("security", "find-generic-password", "-s", k.keychainService, "-a", k.keychainAccount, "-w")
	output, err := cmd.Output()
	if err != nil {
		// Don't return the error details as they might contain sensitive info or be verbose
		return "", fmt.Errorf("failed to retrieve API key from macOS keychain")
	}
	return strings.TrimSpace(string(output)), nil
}

// macStoreInKeychain stores the API key in Mac keychain
func (k *KeyManager) macStoreInKeychain(apiKey string) error {
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

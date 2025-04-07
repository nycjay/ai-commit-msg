package key

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

// TestKeyManager is a wrapper around KeyManager with fake implementations for testing
type TestKeyManager struct {
	*KeyManager
}

// NewTestKeyManager creates a KeyManager with fake implementations for testing
func NewTestKeyManager(verbose bool) *KeyManager {
	km := NewKeyManager(verbose)
	
	// Replace the platform functions with fakes
	km.getFromKeychainFn = func() (string, error) {
		return "", fmt.Errorf("test keychain not implemented")
	}
	
	km.storeInKeychainFn = func(apiKey string) error {
		return fmt.Errorf("test keychain not implemented")
	}
	
	return km
}

// Test retrieving API key from environment variables
func TestGetFromEnvironment(t *testing.T) {
	// Setup
	originalValue := os.Getenv(EnvVarName)
	defer os.Setenv(EnvVarName, originalValue) // Restore original value after test
	
	testCases := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "Environment variable set",
			envValue: "test-api-key-123",
			expected: "test-api-key-123",
		},
		{
			name:     "Environment variable empty",
			envValue: "",
			expected: "",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv(EnvVarName, tc.envValue)
			
			km := NewTestKeyManager(false)
			result := km.GetFromEnvironment()
			
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// Test validating API key format
func TestValidateKey(t *testing.T) {
	testCases := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{
			name:     "Valid key format",
			apiKey:   "sk_ant_1234567890abcdefghijklmn",
			expected: true,
		},
		{
			name:     "Invalid prefix",
			apiKey:   "invalid_1234567890abcdefghijklmn",
			expected: false,
		},
		{
			name:     "Too short",
			apiKey:   "sk_ant_short",
			expected: false,
		},
		{
			name:     "Empty string",
			apiKey:   "",
			expected: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			km := NewTestKeyManager(false)
			result := km.ValidateKey(tc.apiKey)
			
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for key '%s'", tc.expected, result, tc.apiKey)
			}
		})
	}
}

// Test the key precedence logic
func TestGetKey(t *testing.T) {
	// Save original environment variable
	originalValue := os.Getenv(EnvVarName)
	defer os.Setenv(EnvVarName, originalValue)
	
	// Test cases
	testCases := []struct {
		name           string
		cmdLineKey     string
		envKey         string
		keychainKey    string
		keychainError  error
		expectError    bool
		expectedResult string
	}{
		{
			name:           "Command line key takes precedence",
			cmdLineKey:     "cmd-line-key",
			envKey:         "env-key",
			keychainKey:    "keychain-key",
			expectError:    false,
			expectedResult: "cmd-line-key",
		},
		{
			name:           "Environment variable used when no command line key",
			cmdLineKey:     "",
			envKey:         "env-key",
			keychainKey:    "keychain-key",
			expectError:    false,
			expectedResult: "env-key",
		},
		{
			name:           "Keychain used when no command line or env key",
			cmdLineKey:     "",
			envKey:         "",
			keychainKey:    "keychain-key",
			expectError:    false,
			expectedResult: "keychain-key",
		},
		{
			name:           "Error when no key available",
			cmdLineKey:     "",
			envKey:         "",
			keychainKey:    "",
			expectError:    true,
			expectedResult: "",
		},
		{
			name:           "Error when keychain fails",
			cmdLineKey:     "",
			envKey:         "",
			keychainKey:    "",
			keychainError:  fmt.Errorf("keychain error"),
			expectError:    true,
			expectedResult: "",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup environment variable
			os.Setenv(EnvVarName, tc.envKey)
			
			// Create key manager with mocked keychain function
			km := NewTestKeyManager(false)
			
			// Override the keychain getter with our mock
			origFn := km.getFromKeychainFn
			defer func() { km.getFromKeychainFn = origFn }()
			
			km.getFromKeychainFn = func() (string, error) {
				if tc.keychainError != nil {
					return "", tc.keychainError
				}
				return tc.keychainKey, nil
			}
			
			// Call method under test
			result, err := km.GetKey(tc.cmdLineKey)
			
			// Check error
			if tc.expectError && err == nil {
				t.Errorf("Expected an error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Did not expect an error but got: %v", err)
			}
			
			// Check result
			if result != tc.expectedResult {
				t.Errorf("Expected '%s', got '%s'", tc.expectedResult, result)
			}
		})
	}
}

// Test storing API key in keychain
func TestStoreInKeychain(t *testing.T) {
	testCases := []struct {
		name          string
		apiKey        string
		mockError     error
		expectError   bool
		expectSuccess bool
	}{
		{
			name:          "Successful storage",
			apiKey:        "test-api-key",
			mockError:     nil,
			expectError:   false,
			expectSuccess: true,
		},
		{
			name:          "Error during storage",
			apiKey:        "test-api-key",
			mockError:     fmt.Errorf("keychain error"),
			expectError:   true,
			expectSuccess: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create key manager with mocked store function
			km := NewTestKeyManager(false)
			
			// Track if the mock was called
			var mockCalled bool
			
			// Save original function
			origFn := km.storeInKeychainFn
			defer func() { km.storeInKeychainFn = origFn }()
			
			// Override with mock
			km.storeInKeychainFn = func(key string) error {
				mockCalled = true
				
				// Verify the key passed to the function
				if key != tc.apiKey {
					t.Errorf("Expected key '%s', got '%s'", tc.apiKey, key)
				}
				
				return tc.mockError
			}
			
			// Call method under test
			err := km.StoreInKeychain(tc.apiKey)
			
			// Verify the mock was called
			if !mockCalled {
				t.Error("Mock function was not called")
			}
			
			// Check error
			if tc.expectError && err == nil {
				t.Errorf("Expected an error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Did not expect an error but got: %v", err)
			}
		})
	}
}

// Test platform detection
func TestPlatformDetection(t *testing.T) {
	km := NewTestKeyManager(false)
	platform := km.GetPlatform()
	
	// Verify that the platform matches the runtime.GOOS
	var expected Platform
	switch runtime.GOOS {
	case "darwin":
		expected = PlatformMac
	case "windows":
		expected = PlatformWindows
	case "linux":
		expected = PlatformLinux
	default:
		expected = PlatformUnknown
	}
	
	if platform != expected {
		t.Errorf("Expected platform '%s', got '%s'", expected, platform)
	}
}

// Test credential store availability
func TestCredentialStoreAvailable(t *testing.T) {
	km := NewTestKeyManager(false)
	available := km.CredentialStoreAvailable()
	
	// Verify that credential store availability matches the platform
	expected := runtime.GOOS == "darwin" || runtime.GOOS == "windows"
	
	if available != expected {
		t.Errorf("Expected credential store availability '%v', got '%v'", expected, available)
	}
}

// Test credential store name
func TestGetCredentialStoreName(t *testing.T) {
	km := NewTestKeyManager(false)
	name := km.GetCredentialStoreName()
	
	// Verify that the credential store name matches the platform
	var expected string
	switch runtime.GOOS {
	case "darwin":
		expected = "macOS Keychain"
	case "windows":
		expected = "Windows Credential Manager"
	default:
		expected = "none"
	}
	
	if name != expected {
		t.Errorf("Expected credential store name '%s', got '%s'", expected, name)
	}
}

// Test platform-specific credential store functions
func TestPlatformCredentialStore(t *testing.T) {
	// This is a more limited test that doesn't try to manipulate the methods directly
	// since that was causing issues. Instead, we just test the high-level functionality.
	
	km := NewTestKeyManager(false)
	
	// Create a mock for getFromKeychainFn
	origGetFn := km.getFromKeychainFn
	defer func() { km.getFromKeychainFn = origGetFn }()
	
	var getFromStoreCalled bool
	km.getFromKeychainFn = func() (string, error) {
		getFromStoreCalled = true
		return "test-key", nil
	}
	
	// Call GetFromKeychain which should use our mock
	result, err := km.GetFromKeychain()
	
	// Verify that the function was called and we got the expected result
	if !getFromStoreCalled {
		t.Error("getFromKeychainFn was not called")
	}
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if result != "test-key" {
		t.Errorf("Expected 'test-key', got '%s'", result)
	}
	
	// Now test storing with a mock
	origStoreFn := km.storeInKeychainFn
	defer func() { km.storeInKeychainFn = origStoreFn }()
	
	var storeInKeychainCalled bool
	km.storeInKeychainFn = func(apiKey string) error {
		storeInKeychainCalled = true
		if apiKey != "test-key-to-store" {
			t.Errorf("Expected 'test-key-to-store', got '%s'", apiKey)
		}
		return nil
	}
	
	// Call StoreInKeychain
	err = km.StoreInKeychain("test-key-to-store")
	
	// Verify that the function was called and we got the expected result
	if !storeInKeychainCalled {
		t.Error("storeInKeychainFn was not called")
	}
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

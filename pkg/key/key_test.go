package key

import (
	"fmt"
	"os"
	"testing"
)

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
			
			km := NewKeyManager(false)
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
			km := NewKeyManager(false)
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
			km := NewKeyManager(false)
			
			// Override the keychain getter with our mock
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
			km := NewKeyManager(false)
			
			// Track if the mock was called
			var mockCalled bool
			
			// Override the keychain storer with our mock
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

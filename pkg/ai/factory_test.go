package ai

import (
	"strings"
	"testing"
)

// TestNewProvider tests the creation of providers through the factory
func TestNewProvider(t *testing.T) {
	testCases := []struct {
		name          string
		providerName  string
		expectedType  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "Create Anthropic provider",
			providerName: "anthropic",
			expectedType: "*ai.AnthropicProvider",
			expectError:  false,
		},
		{
			name:         "Create OpenAI provider",
			providerName: "openai",
			expectedType: "*ai.OpenAIProvider",
			expectError:  false,
		},
		{
			name:         "Create Gemini provider",
			providerName: "gemini",
			expectedType: "*ai.GeminiProvider",
			expectError:  false,
		},
		{
			name:          "Unknown provider",
			providerName:  "unknown",
			expectedType:  "",
			expectError:   true,
			errorContains: "unknown provider",
		},
		{
			name:          "Empty provider name",
			providerName:  "",
			expectedType:  "",
			expectError:   true,
			errorContains: "provider name cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := NewProvider(tc.providerName)

			// Check error
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tc.errorContains != "" && !contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}

			// Check provider type
			if !tc.expectError {
				// Get the type of the provider
				typeName := getTypeName(provider)
				if typeName != tc.expectedType {
					t.Errorf("Expected provider type '%s', got '%s'", tc.expectedType, typeName)
				}
			}
		})
	}
}

// TestGetAllProviders tests the GetAllProviders function
func TestGetAllProviders(t *testing.T) {
	providers := GetAllProviders()

	// Check that we have all three providers
	if len(providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providers))
	}

	// Check that all provider names are present
	expectedNames := map[string]bool{
		"anthropic": false,
		"openai":    false,
		"gemini":    false,
	}

	for _, p := range providers {
		name := p.GetName()
		if _, exists := expectedNames[name]; exists {
			expectedNames[name] = true
		} else {
			t.Errorf("Unexpected provider name: %s", name)
		}
	}

	// Verify all expected names were found
	for name, found := range expectedNames {
		if !found {
			t.Errorf("Expected provider '%s' not found", name)
		}
	}
}

// TestGetProviderByName tests the GetProviderByName function
func TestGetProviderByName(t *testing.T) {
	testCases := []struct {
		name           string
		providerName   string
		expectProvider bool
	}{
		{
			name:           "Get Anthropic provider",
			providerName:   "anthropic",
			expectProvider: true,
		},
		{
			name:           "Get OpenAI provider",
			providerName:   "openai",
			expectProvider: true,
		},
		{
			name:           "Get Gemini provider",
			providerName:   "gemini",
			expectProvider: true,
		},
		{
			name:           "Unknown provider",
			providerName:   "unknown",
			expectProvider: false,
		},
		{
			name:           "Empty provider name",
			providerName:   "",
			expectProvider: false,
		},
		{
			name:           "Case insensitive - Anthropic",
			providerName:   "AnThrOpIc",
			expectProvider: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := GetProviderByName(tc.providerName)

			if tc.expectProvider {
				if provider == nil {
					t.Errorf("Expected to get a provider, got nil")
				} else {
					// Check provider name (case-insensitive)
					expectedName := tc.providerName
					if expectedName != "" {
						expectedName = expectedName[0:1] + expectedName[1:]
					}
					if !equalIgnoreCase(provider.GetName(), expectedName) {
						t.Errorf("Expected provider name '%s', got '%s'", expectedName, provider.GetName())
					}
				}
			} else {
				if provider != nil {
					t.Errorf("Expected nil, got provider: %v", provider)
				}
			}
		})
	}
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Helper function that returns the type name of an interface
func getTypeName(i interface{}) string {
	if i == nil {
		return "<nil>"
	}
	switch i.(type) {
	case *AnthropicProvider:
		return "*ai.AnthropicProvider"
	case *OpenAIProvider:
		return "*ai.OpenAIProvider"
	case *GeminiProvider:
		return "*ai.GeminiProvider"
	default:
		return "unknown type"
	}
}

// Helper function to compare strings case-insensitively
func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i]|32 != b[i]|32 { // Convert to lowercase and compare
			return false
		}
	}
	return true
}

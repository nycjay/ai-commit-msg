package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nycjay/ai-commit-msg/pkg/git"
)

// TestGeminiProvider_NewGeminiProvider tests the creation of a new Gemini provider
func TestGeminiProvider_NewGeminiProvider(t *testing.T) {
	provider := NewGeminiProvider()

	if provider == nil {
		t.Error("Expected provider to be non-nil")
	}

	// Check that the provider implements the Provider interface
	var _ Provider = provider
}

// TestGeminiProvider_GetName tests the GetName method
func TestGeminiProvider_GetName(t *testing.T) {
	provider := &GeminiProvider{}
	name := provider.GetName()

	if name != "gemini" {
		t.Errorf("Expected name 'gemini', got '%s'", name)
	}
}

// TestGeminiProvider_GetDefaultModel tests the GetDefaultModel method
func TestGeminiProvider_GetDefaultModel(t *testing.T) {
	provider := &GeminiProvider{}
	model := provider.GetDefaultModel()

	defaultModel := "gemini-1.5-pro"
	if model != defaultModel {
		t.Errorf("Expected default model '%s', got '%s'", defaultModel, model)
	}
}

// TestGeminiProvider_GetAvailableModels tests the GetAvailableModels method
func TestGeminiProvider_GetAvailableModels(t *testing.T) {
	provider := &GeminiProvider{}
	models := provider.GetAvailableModels()

	if len(models) == 0 {
		t.Error("Expected available models to be non-empty")
	}

	// Check that the default model is in the list
	defaultModel := "gemini-1.5-pro"
	found := false
	for _, model := range models {
		if model == defaultModel {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Default model '%s' not found in available models: %v", defaultModel, models)
	}
}

// TestGeminiProvider_ValidateAPIKey tests the ValidateAPIKey method
func TestGeminiProvider_ValidateAPIKey(t *testing.T) {
	provider := &GeminiProvider{}

	testCases := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "Long key",
			key:      "AIzaSyD1234567890abcdefghijklmnopqrstuvwxyz",
			expected: true,
		},
		{
			name:     "Shorter key",
			key:      "AIzaSyD123456789012345",
			expected: true,
		},
		{
			name:     "Too short",
			key:      "short",
			expected: false,
		},
		{
			name:     "Empty string",
			key:      "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := provider.ValidateAPIKey(tc.key)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for key '%s'", tc.expected, result, tc.key)
			}
		})
	}
}

// TestGeminiProvider_GenerateCommitMessage tests the GenerateCommitMessage method
func TestGeminiProvider_GenerateCommitMessage(t *testing.T) {
	// Test cases for success and error scenarios
	testCases := []struct {
		name           string
		apiKey         string
		modelName      string
		diff           git.GitDiff
		httpResponse   *http.Response
		httpError      error
		expectedMsg    string
		expectedErrMsg string
	}{
		{
			name:      "Successful commit message generation",
			apiKey:    "AIzaSyD-test-key",
			modelName: "gemini-1.5-pro",
			diff: git.GitDiff{
				StagedFiles: []string{"auth.go"},
				Diff:        "diff --git a/auth.go b/auth.go\n...",
				Branch:      "feature/auth-fix",
				SystemPrompt: "Generate a commit message based on the provided diff.",
				UserPrompt:   "Here is the diff: %s",
			},
			httpResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewBufferString(`{
					"candidates": [{"content": {"parts": [{"text": "fix(auth): correct user authentication flow"}]}}]
				}`)),
			},
			httpError:      nil,
			expectedMsg:    "fix(auth): correct user authentication flow",
			expectedErrMsg: "",
		},
		{
			name:      "HTTP error",
			apiKey:    "AIzaSyD-test-key",
			modelName: "gemini-1.5-pro",
			diff: git.GitDiff{
				StagedFiles: []string{"auth.go"},
				Diff:        "diff --git a/auth.go b/auth.go\n...",
				Branch:      "feature/auth-fix",
				SystemPrompt: "Generate a commit message based on the provided diff.",
				UserPrompt:   "Here is the diff: %s",
			},
			httpResponse:   nil,
			httpError:      io.ErrUnexpectedEOF,
			expectedMsg:    "",
			expectedErrMsg: "Post \"https://generativelanguage.googleapis.com/v1/models/gemini-1.5-pro:generateContent?key=AIzaSyD-test-key\": unexpected EOF",
		},
		{
			name:      "API error response",
			apiKey:    "AIzaSyD-test-key",
			modelName: "gemini-1.5-pro",
			diff: git.GitDiff{
				StagedFiles: []string{"auth.go"},
				Diff:        "diff --git a/auth.go b/auth.go\n...",
				Branch:      "feature/auth-fix",
				SystemPrompt: "Generate a commit message based on the provided diff.",
				UserPrompt:   "Here is the diff: %s",
			},
			httpResponse: &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": {"message": "Invalid API key"}}`)),
			},
			httpError:      nil,
			expectedMsg:    "",
			expectedErrMsg: "API error (status 400)",
		},
		{
			name:      "Empty API response",
			apiKey:    "AIzaSyD-test-key",
			modelName: "gemini-1.5-pro",
			diff: git.GitDiff{
				StagedFiles: []string{"auth.go"},
				Diff:        "diff --git a/auth.go b/auth.go\n...",
				Branch:      "feature/auth-fix",
				SystemPrompt: "Generate a commit message based on the provided diff.",
				UserPrompt:   "Here is the diff: %s",
			},
			httpResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"candidates": []}`)),
			},
			httpError:      nil,
			expectedMsg:    "",
			expectedErrMsg: "empty response from API",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a base URL for the Gemini API with the test API key
			expectedURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:generateContent?key=%s", 
				tc.modelName, tc.apiKey)
				
			// Setup mock HTTP function
			mockDoFunc = func(req *http.Request) (*http.Response, error) {
				// Validate request
				if req.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", req.Method)
				}

				if !strings.HasPrefix(req.URL.String(), expectedURL) {
					t.Errorf("Unexpected URL: got %s, expected prefix %s", req.URL.String(), expectedURL)
				}

				// Validate body
				var reqBody map[string]interface{}
				body, _ := io.ReadAll(req.Body)
				json.Unmarshal(body, &reqBody)

				if contentsParts, ok := reqBody["contents"].([]interface{}); ok && len(contentsParts) > 0 {
					// Validate the model name being sent
					parts, ok := contentsParts[0].(map[string]interface{})
					if !ok {
						t.Error("Unexpected contents structure in request")
					}
					if _, hasRole := parts["role"]; !hasRole {
						t.Error("Expected 'role' in contents")
					}
				} else {
					t.Error("Expected 'contents' in request body")
				}

				// Return the mock response
				return tc.httpResponse, tc.httpError
			}

			// Get the default provider
			provider := NewGeminiProvider()

			// Call the method being tested
			message, err := provider.GenerateCommitMessage(tc.apiKey, tc.modelName, tc.diff)

			// Check error
			if tc.expectedErrMsg != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tc.expectedErrMsg)
				} else if !strings.Contains(err.Error(), tc.expectedErrMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.expectedErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got '%s'", err.Error())
				}
			}

			// Check message
			if message != tc.expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", tc.expectedMsg, message)
			}
		})
	}
}
package ai

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nycjay/ai-commit-msg/pkg/git"
)

// TestAnthropicProvider_NewAnthropicProvider tests the creation of a new Anthropic provider
func TestAnthropicProvider_NewAnthropicProvider(t *testing.T) {
	provider := NewAnthropicProvider()

	if provider == nil {
		t.Error("Expected provider to be non-nil")
	}

	// Check that the provider implements the Provider interface
	var _ Provider = provider
}

// TestAnthropicProvider_GetName tests the GetName method
func TestAnthropicProvider_GetName(t *testing.T) {
	provider := &AnthropicProvider{}
	name := provider.GetName()

	if name != "anthropic" {
		t.Errorf("Expected name 'anthropic', got '%s'", name)
	}
}

// TestAnthropicProvider_GetDefaultModel tests the GetDefaultModel method
func TestAnthropicProvider_GetDefaultModel(t *testing.T) {
	provider := &AnthropicProvider{}
	model := provider.GetDefaultModel()

	defaultModel := "claude-3-haiku-20240307"
	if model != defaultModel {
		t.Errorf("Expected default model '%s', got '%s'", defaultModel, model)
	}
}

// TestAnthropicProvider_GetAvailableModels tests the GetAvailableModels method
func TestAnthropicProvider_GetAvailableModels(t *testing.T) {
	provider := &AnthropicProvider{}
	models := provider.GetAvailableModels()

	if len(models) == 0 {
		t.Error("Expected available models to be non-empty")
	}

	// Check that the default model is in the list
	found := false
	defaultModel := "claude-3-haiku-20240307"
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

// TestAnthropicProvider_ValidateAPIKey tests the ValidateAPIKey method
func TestAnthropicProvider_ValidateAPIKey(t *testing.T) {
	provider := &AnthropicProvider{}

	testCases := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "Valid key with sk-ant prefix",
			key:      "sk-ant-1234567890abcdefghijklmnopqrstuvwxyz",
			expected: true,
		},
		{
			name:     "Valid key with sk- prefix",
			key:      "sk-1234567890abcdefghijklmnopqrstuvwxyz",
			expected: true,
		},
		{
			name:     "Invalid prefix",
			key:      "invalid-1234567890abcdefghijklmnopqrstuvwxyz",
			expected: false,
		},
		{
			name:     "Too short",
			key:      "sk-ant-short",
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

// TestAnthropicProvider_GenerateCommitMessage tests the GenerateCommitMessage method
func TestAnthropicProvider_GenerateCommitMessage(t *testing.T) {
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
			apiKey:    "sk-ant-test",
			modelName: "claude-3-haiku-20240307",
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
					"content": [{"type": "text", "text": "fix(auth): correct user authentication flow"}]
				}`)),
			},
			httpError:      nil,
			expectedMsg:    "fix(auth): correct user authentication flow",
			expectedErrMsg: "",
		},
		{
			name:      "HTTP error",
			apiKey:    "sk-ant-test",
			modelName: "claude-3-haiku-20240307",
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
			expectedErrMsg: "Post \"https://api.anthropic.com/v1/messages\": unexpected EOF",
		},
		{
			name:      "API error response",
			apiKey:    "sk-ant-test",
			modelName: "claude-3-haiku-20240307",
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
			apiKey:    "sk-ant-test",
			modelName: "claude-3-haiku-20240307",
			diff: git.GitDiff{
				StagedFiles: []string{"auth.go"},
				Diff:        "diff --git a/auth.go b/auth.go\n...",
				Branch:      "feature/auth-fix",
				SystemPrompt: "Generate a commit message based on the provided diff.",
				UserPrompt:   "Here is the diff: %s",
			},
			httpResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"content": []}`)),
			},
			httpError:      nil,
			expectedMsg:    "",
			expectedErrMsg: "empty response from API",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock HTTP function
			mockDoFunc = func(req *http.Request) (*http.Response, error) {
				// Validate request
				if req.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", req.Method)
				}

				if req.URL.String() != anthropicAPI {
					t.Errorf("Unexpected URL: got %s, expected %s", req.URL.String(), anthropicAPI)
				}

				// Validate API key in header
				if req.Header.Get("x-api-key") != tc.apiKey {
					t.Errorf("Unexpected API key in header: got %s, expected %s", req.Header.Get("x-api-key"), tc.apiKey)
				}

				// Validate body
				var reqBody map[string]interface{}
				body, _ := io.ReadAll(req.Body)
				json.Unmarshal(body, &reqBody)

				if reqBody["model"] != tc.modelName {
					t.Errorf("Unexpected model in request body: got %s, expected %s", reqBody["model"], tc.modelName)
				}

				// Return the mock response
				return tc.httpResponse, tc.httpError
			}

			// Get the default provider
			provider := NewAnthropicProvider()

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
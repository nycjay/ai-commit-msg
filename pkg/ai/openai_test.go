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

// TestOpenAIProvider_NewOpenAIProvider tests the creation of a new OpenAI provider
func TestOpenAIProvider_NewOpenAIProvider(t *testing.T) {
	provider := NewOpenAIProvider()

	if provider == nil {
		t.Error("Expected provider to be non-nil")
	}

	// Check that the provider implements the Provider interface
	var _ Provider = provider
}

// TestOpenAIProvider_GetName tests the GetName method
func TestOpenAIProvider_GetName(t *testing.T) {
	provider := &OpenAIProvider{}
	name := provider.GetName()

	if name != "openai" {
		t.Errorf("Expected name 'openai', got '%s'", name)
	}
}

// TestOpenAIProvider_GetDefaultModel tests the GetDefaultModel method
func TestOpenAIProvider_GetDefaultModel(t *testing.T) {
	provider := &OpenAIProvider{}
	model := provider.GetDefaultModel()

	defaultModel := "gpt-4o"
	if model != defaultModel {
		t.Errorf("Expected default model '%s', got '%s'", defaultModel, model)
	}
}

// TestOpenAIProvider_GetAvailableModels tests the GetAvailableModels method
func TestOpenAIProvider_GetAvailableModels(t *testing.T) {
	provider := &OpenAIProvider{}
	models := provider.GetAvailableModels()

	if len(models) == 0 {
		t.Error("Expected available models to be non-empty")
	}

	// Check that the default model is in the list
	defaultModel := "gpt-4o"
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

// TestOpenAIProvider_ValidateAPIKey tests the ValidateAPIKey method
func TestOpenAIProvider_ValidateAPIKey(t *testing.T) {
	provider := &OpenAIProvider{}

	testCases := []struct {
		name     string
		key      string
		expected bool
	}{
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
			key:      "sk-short",
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

// TestOpenAIProvider_GenerateCommitMessage tests the GenerateCommitMessage method
func TestOpenAIProvider_GenerateCommitMessage(t *testing.T) {
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
			apiKey:    "sk-test",
			modelName: "gpt-4",
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
					"choices": [{"message": {"content": "fix(auth): correct user authentication flow"}}]
				}`)),
			},
			httpError:      nil,
			expectedMsg:    "fix(auth): correct user authentication flow",
			expectedErrMsg: "",
		},
		{
			name:      "HTTP error",
			apiKey:    "sk-test",
			modelName: "gpt-4",
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
			expectedErrMsg: "Post \"https://api.openai.com/v1/chat/completions\": unexpected EOF",
		},
		{
			name:      "API error response",
			apiKey:    "sk-test",
			modelName: "gpt-4",
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
			apiKey:    "sk-test",
			modelName: "gpt-4",
			diff: git.GitDiff{
				StagedFiles: []string{"auth.go"},
				Diff:        "diff --git a/auth.go b/auth.go\n...",
				Branch:      "feature/auth-fix",
				SystemPrompt: "Generate a commit message based on the provided diff.",
				UserPrompt:   "Here is the diff: %s",
			},
			httpResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"choices": []}`)),
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

				if req.URL.String() != openaiAPI {
					t.Errorf("Unexpected URL: got %s, expected %s", req.URL.String(), openaiAPI)
				}

				// Validate API key in header
				if req.Header.Get("Authorization") != "Bearer "+tc.apiKey {
					t.Errorf("Unexpected Authorization header: got %s, expected Bearer %s", 
						req.Header.Get("Authorization"), tc.apiKey)
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
			provider := NewOpenAIProvider()

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

				// Check message
				if message != tc.expectedMsg {
					t.Errorf("Expected message '%s', got '%s'", tc.expectedMsg, message)
				}
			}
		})
	}
}

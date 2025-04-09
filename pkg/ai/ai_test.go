package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nycjay/ai-commit-msg/pkg/git"
)

// TestGenerateCommitMessage tests the GenerateCommitMessage function
func TestGenerateCommitMessage(t *testing.T) {
	// Create a mock git diff
	diffInfo := git.GitDiff{
		StagedFiles:     []string{"test.go", "README.md"},
		Diff:            "diff --git a/test.go b/test.go\n...",
		Branch:          "feature/test",
		JiraID:          "TEST-123",
		JiraDescription: "Test feature implementation",
		SystemPrompt:    "You are a commit message generator.",
		UserPrompt:      "Generate a commit message for the following changes: {{diff}}",
	}

	// Test with a mock API key
	apiKey := "sk_test_123456789"
	
	// Call the function
	message, err := GenerateCommitMessage(apiKey, diffInfo)
	
	// We don't expect an error even though this is a stub function
	if err != nil {
		t.Errorf("GenerateCommitMessage returned error: %v", err)
	}
	
	// The stub returns a placeholder message
	expectedMessage := "feat: Sample commit message"
	if message != expectedMessage {
		t.Errorf("Expected message to be '%s', got '%s'", expectedMessage, message)
	}
}

// TestRequestStructure tests the Request and Message structs
func TestRequestStructure(t *testing.T) {
	// Create a sample request
	req := Request{
		Model:     "claude-3-haiku",
		MaxTokens: 500,
		System:    "You are a commit message generator.",
		Messages: []Message{
			{
				Role:    "user",
				Content: "Generate a commit message for these changes.",
			},
		},
	}
	
	// Marshal the request to JSON to test the structure
	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	
	// Verify JSON structure matches expected format
	jsonStr := string(jsonData)
	
	// Check for required fields
	requiredFields := []string{
		"model", "max_tokens", "system", "messages",
	}
	
	for _, field := range requiredFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing required field: %s", field)
		}
	}
	
	// Check message structure
	if !strings.Contains(jsonStr, `"role":"user"`) {
		t.Errorf("Message missing role field")
	}
	
	if !strings.Contains(jsonStr, `"content":"Generate a commit message for these changes."`) {
		t.Errorf("Message missing content field")
	}
}

// TestResponseParsing tests the Response and Content structs
func TestResponseParsing(t *testing.T) {
	// Sample JSON response from an API
	jsonResponse := `{
		"content": [
			{
				"text": "feat: Implement user authentication"
			}
		]
	}`
	
	// Parse the response
	var response Response
	err := json.Unmarshal([]byte(jsonResponse), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	// Verify parsed content
	if len(response.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(response.Content))
	}
	
	expectedText := "feat: Implement user authentication"
	if response.Content[0].Text != expectedText {
		t.Errorf("Expected text to be '%s', got '%s'", expectedText, response.Content[0].Text)
	}
}

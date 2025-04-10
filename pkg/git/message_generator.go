package git

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FormatEnhancedContext formats the enhanced git context into a string suitable for the LLM
func FormatEnhancedContext(diff EnhancedGitDiff) string {
	var contextBuilder strings.Builder

	// Project context
	contextBuilder.WriteString(diff.ProjectContext)
	contextBuilder.WriteString("\n")

	// File summaries
	contextBuilder.WriteString("File Summaries:\n")
	for file, summary := range diff.FileSummaries {
		contextBuilder.WriteString(fmt.Sprintf("- %s: %s\n", file, summary))
	}
	contextBuilder.WriteString("\n")

	// Recent commit history
	contextBuilder.WriteString("Recent Commit History:\n")
	for file, commits := range diff.CommitHistory {
		contextBuilder.WriteString(fmt.Sprintf("File: %s\n", file))
		for _, commit := range commits {
			contextBuilder.WriteString(fmt.Sprintf("  %s\n", commit))
		}
	}
	contextBuilder.WriteString("\n")

	// Related files
	contextBuilder.WriteString("Related Files:\n")
	for _, file := range diff.RelatedFiles {
		contextBuilder.WriteString(fmt.Sprintf("- %s\n", file))
	}

	return contextBuilder.String()
}

// GenerateCommitMessageWithEnhancedContext generates a commit message using enhanced git context
func GenerateCommitMessageWithEnhancedContext(apiProvider string, apiKey string, modelName string, diff EnhancedGitDiff, systemPrompt string, userPromptTemplate string) (string, error) {
	// Format the context information
	fileSummaries := ""
	for file, summary := range diff.FileSummaries {
		fileSummaries += fmt.Sprintf("- %s: %s\n", file, summary)
	}

	commitHistory := ""
	for file, commits := range diff.CommitHistory {
		commitHistory += fmt.Sprintf("File: %s\n", file)
		for _, commit := range commits {
			commitHistory += fmt.Sprintf("  %s\n", commit)
		}
	}

	relatedFiles := ""
	for _, file := range diff.RelatedFiles {
		relatedFiles += fmt.Sprintf("- %s\n", file)
	}

	// Format the user prompt with all the enhanced information
	userPrompt := fmt.Sprintf(
		userPromptTemplate,
		diff.Branch,
		strings.Join(diff.StagedFiles, "\n"),
		diff.Diff,
		diff.JiraID,
		diff.JiraDescription,
		diff.ProjectContext,
		fileSummaries,
		commitHistory,
		relatedFiles,
	)

	// Now we call the appropriate provider-specific implementation
	switch strings.ToLower(apiProvider) {
	case "anthropic", "claude":
		return generateAnthropicCommitMessage(apiKey, modelName, systemPrompt, userPrompt)
	case "openai", "gpt":
		// Placeholder for OpenAI implementation
		return "", fmt.Errorf("OpenAI implementation not yet available for enhanced context")
	case "gemini", "google":
		// Placeholder for Gemini implementation
		return "", fmt.Errorf("Gemini implementation not yet available for enhanced context")
	default:
		// Default to Anthropic
		return generateAnthropicCommitMessage(apiKey, modelName, systemPrompt, userPrompt)
	}
}

// generateAnthropicCommitMessage sends a request to the Anthropic Claude API
func generateAnthropicCommitMessage(apiKey string, modelName string, systemPrompt string, userPrompt string) (string, error) {
	const anthropicAPI = "https://api.anthropic.com/v1/messages"

	// Prepare the API request
	request := struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		System    string `json:"system"`
		Messages  []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}{
		Model:     modelName,
		MaxTokens: 1000,
		System:    systemPrompt,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "user", Content: userPrompt},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// Send the request to the Anthropic API
	req, err := http.NewRequest("POST", anthropicAPI, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorMsg := string(bodyBytes)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, errorMsg)
	}

	// Parse the API response
	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return response.Content[0].Text, nil
}

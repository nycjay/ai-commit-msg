package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nycjay/ai-commit-msg/pkg/git"
)

const anthropicAPI = "https://api.anthropic.com/v1/messages"

// AnthropicProvider implements the Provider interface for Anthropic's Claude
type AnthropicProvider struct{}

// AnthropicRequest represents a request to the Claude API
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []AnthropicMessage `json:"messages"`
}

// AnthropicMessage represents a message in the Claude API request
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents a response from the Claude API
type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider() *AnthropicProvider {
	return &AnthropicProvider{}
}

// GenerateCommitMessage generates a commit message using Claude AI
func (p *AnthropicProvider) GenerateCommitMessage(apiKey string, modelName string, diffInfo git.GitDiff) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("no API key found for Anthropic")
	}

	// Format prompts to be passed to provider
	systemPrompt := diffInfo.SystemPrompt
	userPrompt := ""
	
	// Check if we have enhanced context fields
	if diffInfo.ProjectContext != "" || len(diffInfo.FileSummaries) > 0 || len(diffInfo.CommitHistory) > 0 || len(diffInfo.RelatedFiles) > 0 {
		// Format enhanced context
		fileSummaries := ""
		for file, summary := range diffInfo.FileSummaries {
			fileSummaries += fmt.Sprintf("- %s: %s\n", file, summary)
		}
		
		commitHistory := ""
		for file, commits := range diffInfo.CommitHistory {
			commitHistory += fmt.Sprintf("File: %s\n", file)
			for _, commit := range commits {
				commitHistory += fmt.Sprintf("  %s\n", commit)
			}
		}
		
		relatedFiles := ""
		for _, file := range diffInfo.RelatedFiles {
			relatedFiles += fmt.Sprintf("- %s\n", file)
		}
		
		// Use the enhanced format with additional context
		userPrompt = fmt.Sprintf(
			diffInfo.UserPrompt,
			diffInfo.Branch,
			strings.Join(diffInfo.StagedFiles, "\n"),
			diffInfo.Diff,
			diffInfo.JiraID,
			diffInfo.JiraDescription,
			diffInfo.ProjectContext,
			fileSummaries,
			commitHistory,
			relatedFiles,
		)
	} else {
		// Use the regular format without enhanced context
		userPrompt = fmt.Sprintf(
			diffInfo.UserPrompt,
			diffInfo.Branch,
			strings.Join(diffInfo.StagedFiles, "\n"),
			diffInfo.Diff,
			diffInfo.JiraID,
			diffInfo.JiraDescription,
		)
	}

	request := AnthropicRequest{
		Model:     modelName,
		MaxTokens: 1000,
		System:    systemPrompt,
		Messages: []AnthropicMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", anthropicAPI, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Use mock transport in tests, real client in production
	client := &http.Client{Timeout: time.Second * 30}
	if mockDoFunc != nil {
		client.Transport = &MockTransport{}
	}
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

	var response AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return response.Content[0].Text, nil
}

// ValidateAPIKey validates the Anthropic API key format
func (p *AnthropicProvider) ValidateAPIKey(key string) bool {
	return len(key) >= 20 && (
		strings.HasPrefix(key, "sk_ant_") ||
		strings.HasPrefix(key, "sk-ant-") ||
		strings.HasPrefix(key, "sk-") ||
		strings.HasPrefix(key, "sk-o-") ||
		strings.HasPrefix(key, "kp-"))
}

// GetName returns the provider name
func (p *AnthropicProvider) GetName() string {
	return string(ProviderAnthropic)
}

// GetDefaultModel returns the default model name
func (p *AnthropicProvider) GetDefaultModel() string {
	return "claude-3-haiku-20240307"
}

// GetAvailableModels returns the list of available models
func (p *AnthropicProvider) GetAvailableModels() []string {
	return []string{
		"claude-3-haiku-20240307",
		"claude-3-sonnet-20240229",
		"claude-3-opus-20240229",
		"claude-3-5-sonnet-20240620",
		"claude-3-haiku-20231221",
	}
}

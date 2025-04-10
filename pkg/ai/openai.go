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

const openaiAPI = "https://api.openai.com/v1/chat/completions"

// OpenAIProvider implements the Provider interface for OpenAI models
type OpenAIProvider struct{}

// OpenAIRequest represents a request to the OpenAI API
type OpenAIRequest struct {
	Model       string         `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int            `json:"max_tokens"`
	Temperature float64        `json:"temperature"`
}

// OpenAIMessage represents a message in the OpenAI chat API
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents a response from the OpenAI API
type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{}
}

// GenerateCommitMessage generates a commit message using OpenAI
func (p *OpenAIProvider) GenerateCommitMessage(apiKey string, modelName string, diffInfo git.GitDiff) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("no API key found for OpenAI")
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

	request := OpenAIRequest{
		Model: modelName,
		Messages: []OpenAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", openaiAPI, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

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

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return response.Choices[0].Message.Content, nil
}

// ValidateAPIKey validates the OpenAI API key format
func (p *OpenAIProvider) ValidateAPIKey(key string) bool {
	// OpenAI API keys typically start with "sk-" and are 51 characters long
	return len(key) >= 20 && strings.HasPrefix(key, "sk-")
}

// GetName returns the provider name
func (p *OpenAIProvider) GetName() string {
	return string(ProviderOpenAI)
}

// GetDefaultModel returns the default model name
func (p *OpenAIProvider) GetDefaultModel() string {
	return "gpt-4o"
}

// GetAvailableModels returns the list of available models
func (p *OpenAIProvider) GetAvailableModels() []string {
	return []string{
		"gpt-4o",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
	}
}

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

// GeminiProvider implements the Provider interface for Google's Gemini models
type GeminiProvider struct{}

// GeminiContent represents a content part in the Gemini request
type GeminiContent struct {
	Role  string `json:"role"`
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
}

// GeminiRequest represents a request to the Gemini API
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
	GenerationConfig struct {
		MaxOutputTokens int     `json:"maxOutputTokens"`
		Temperature     float64 `json:"temperature"`
	} `json:"generationConfig"`
}

// GeminiResponse represents a response from the Gemini API
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{}
}

// GenerateCommitMessage generates a commit message using Gemini
func (p *GeminiProvider) GenerateCommitMessage(apiKey string, modelName string, diffInfo git.GitDiff) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("no API key found for Gemini")
	}

	// Gemini API URL with API key included
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/%s:generateContent?key=%s", 
		modelName, apiKey)

	// Format prompts to be passed to provider
	// Gemini doesn't have separate system/user roles like OpenAI/Anthropic,
	// so we combine them for Gemini
	systemPrompt := diffInfo.SystemPrompt
	userPromptTemplate := diffInfo.UserPrompt
	
	// Format the user prompt with the diff information
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
			userPromptTemplate,
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
			userPromptTemplate,
			diffInfo.Branch,
			strings.Join(diffInfo.StagedFiles, "\n"),
			diffInfo.Diff,
			diffInfo.JiraID,
			diffInfo.JiraDescription,
		)
	}
	
	// Combine system and user prompts for Gemini
	combinedPrompt := fmt.Sprintf("%s\n\n%s", systemPrompt, userPrompt)
	
	// Create Gemini request with proper structure
	request := GeminiRequest{}
	request.Contents = []GeminiContent{
		{
			Role: "user",
			Parts: []struct {
				Text string `json:"text"`
			}{
				{Text: combinedPrompt},
			},
		},
	}
	request.GenerationConfig.MaxOutputTokens = 1000
	request.GenerationConfig.Temperature = 0.7
	
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
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
	
	var response GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}
	
	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from API")
	}
	
	return response.Candidates[0].Content.Parts[0].Text, nil
}

// ValidateAPIKey validates the Gemini API key format
func (p *GeminiProvider) ValidateAPIKey(key string) bool {
	// Gemini API keys can vary in length but are typically at least 20 characters
	return len(key) >= 20
}

// GetName returns the provider name
func (p *GeminiProvider) GetName() string {
	return string(ProviderGemini)
}

// GetDefaultModel returns the default model name
func (p *GeminiProvider) GetDefaultModel() string {
	return "gemini-1.5-pro"
}

// GetAvailableModels returns the list of available models
func (p *GeminiProvider) GetAvailableModels() []string {
	return []string{
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-1.0-pro",
	}
}

package reviewer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"localhost/aireview/internal/config"
)

// ReviewRequest represents the API request structure
type ReviewRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ReviewResponse represents the API response structure
type ReviewResponse struct {
	Choices []Choice  `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a response choice
type Choice struct {
	Message Message `json:"message"`
}

// APIError represents an API error
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// Service handles code review operations
type Service struct {
	config *config.Config
	client *http.Client
}

// NewService creates a new reviewer service
func NewService(cfg *config.Config) *Service {
	return &Service{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
	}
}

// ReviewCode analyzes code using AI API
func (s *Service) ReviewCode(ctx context.Context, code string) (string, error) {
	if len(code) > int(s.config.MaxFileSize) {
		return "", fmt.Errorf("file size exceeds maximum allowed size of %d bytes", s.config.MaxFileSize)
	}

	request := ReviewRequest{
		Model: s.config.Model,
		Messages: []Message{
			{
				Role:    "system",
				Content: s.getSystemPrompt(),
			},
			{
				Role:    "user",
				Content: code,
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.APIURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "aireview/1.0")
	
	// Add API key if provided
	if s.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.APIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Provide more helpful error messages for common HTTP status codes
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return "", fmt.Errorf("authentication failed (401): check your API key")
		case http.StatusForbidden:
			return "", fmt.Errorf("access forbidden (403): insufficient permissions or invalid API key")
		case http.StatusNotFound:
			return "", fmt.Errorf("model not found (404): check if model '%s' exists and you have access to it", s.config.Model)
		case http.StatusTooManyRequests:
			return "", fmt.Errorf("rate limit exceeded (429): too many requests, please wait and try again")
		case http.StatusInternalServerError:
			return "", fmt.Errorf("server error (500): API service temporarily unavailable")
		default:
			return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, resp.Status)
		}
	}

	var reviewResponse ReviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&reviewResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if reviewResponse.Error != nil {
		return "", fmt.Errorf("API error: %s", reviewResponse.Error.Message)
	}

	if len(reviewResponse.Choices) == 0 {
		return "", fmt.Errorf("no review choices returned")
	}

	return reviewResponse.Choices[0].Message.Content, nil
}

// getSystemPrompt returns the system prompt for code review
func (s *Service) getSystemPrompt() string {
	return `You are a very experienced senior developer. Analyze the following code and provide recommendations on:
- Security vulnerabilities and best practices
- Performance optimizations and efficiency improvements
- Code correctness and potential bugs
- Code readability and maintainability
- Clean architecture principles
- Go-specific best practices

Provide only actionable, specific, and important recommendations. Be concise and focus on real issues.`
}

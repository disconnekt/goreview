package reviewer

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "sync/atomic"

    "github.com/disconnekt/goreview/internal/config"
)

type ReviewRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ReviewResponse struct {
	Choices []Choice  `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

type Service struct {
    config *config.Config
    client *http.Client
    // endpoints contains the effective list of API endpoints to use
    endpoints []string
    // rrCounter is used for round-robin selection across endpoints
    rrCounter uint64
}

func NewService(cfg *config.Config) *Service {
    return &Service{
        config: cfg,
        client: &http.Client{
            Timeout: cfg.RequestTimeout,
        },
        endpoints: cfg.EffectiveAPIURLs(),
    }
}

func (s *Service) ReviewCode(ctx context.Context, code string) (string, error) {
	if len(code) > int(s.config.MaxFileSize) {
		return "", fmt.Errorf("file size exceeds maximum allowed size of %d bytes", s.config.MaxFileSize)
	}

	// Validate content to prevent API issues
	if err := s.validateContent(code); err != nil {
		return "", fmt.Errorf("content validation failed: %w", err)
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
		MaxTokens:   4000,
		Temperature: 0.1,
		Stream:      false,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Try multiple endpoints starting from a round-robin index for failover
	eps := s.endpoints
	if len(eps) == 0 {
		eps = []string{s.config.APIURL}
	}
	start := int((atomic.AddUint64(&s.rrCounter, 1) - 1) % uint64(len(eps)))
	var lastErr error
	for i := 0; i < len(eps); i++ {
		ep := eps[(start+i)%len(eps)]
		review, err := s.attemptRequest(ctx, ep, requestBody)
		if err == nil {
			return review, nil
		}
		lastErr = fmt.Errorf("endpoint %s failed: %w", ep, err)
	}
	if lastErr != nil {
		return "", fmt.Errorf("all %d endpoints failed; last error: %w", len(eps), lastErr)
	}
	return "", fmt.Errorf("no endpoints configured")
}

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

// attemptRequest performs a single HTTP request to the given endpoint
func (s *Service) attemptRequest(ctx context.Context, endpoint string, requestBody []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "aireview/1.0")
	if s.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.APIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return "", fmt.Errorf("bad request (400): invalid request format or unsupported model '%s'", s.config.Model)
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

// validateContent validates code content to prevent API issues
func (s *Service) validateContent(code string) error {
	// Check for empty content
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("code content is empty")
	}

	// Check for extremely large content that might cause API issues
	if len(code) > 100000 { // 100KB threshold for API compatibility
		return fmt.Errorf("code content too large (%d bytes), may cause API issues", len(code))
	}

	// Check for binary content or non-text content
	if !s.isTextContent(code) {
		return fmt.Errorf("content appears to be binary or non-text")
	}

	return nil
}

// isTextContent checks if content is valid text
func (s *Service) isTextContent(content string) bool {
	// Check for null bytes which indicate binary content
	if strings.Contains(content, "\x00") {
		return false
	}

	// Check if content is mostly printable characters
	printableCount := 0
	totalCount := 0
	for _, r := range content {
		totalCount++
		if r >= 32 && r <= 126 || r == '\t' || r == '\n' || r == '\r' {
			printableCount++
		}
	}

	// Require at least 95% printable characters
	if totalCount > 0 && float64(printableCount)/float64(totalCount) < 0.95 {
		return false
	}

	return true
}

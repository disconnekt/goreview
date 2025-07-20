package config

import (
	"errors"
	"strings"
	"time"
)

// Config holds application configuration
type Config struct {
	ProjectPath    string
	APIURL         string
	APIKey         string
	Model          string
	MaxFileSize    int64
	RequestTimeout time.Duration
	MaxConcurrency int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		ProjectPath:    ".",
		APIURL:         "http://127.0.0.1:1234/v1/chat/completions",
		Model:          "devstral-small-2507-mlx",
		MaxFileSize:    1024 * 1024, // 1MB
		RequestTimeout: 30 * time.Second,
		MaxConcurrency: 3,
	}
}

// Validate validates configuration
func (c *Config) Validate() error {
	if c.APIURL == "" {
		return errors.New("API URL cannot be empty")
	}
	if c.Model == "" {
		return errors.New("model cannot be empty")
	}
	if c.MaxFileSize <= 0 {
		return errors.New("max file size must be positive")
	}
	if c.RequestTimeout <= 0 {
		return errors.New("request timeout must be positive")
	}
	// API key is optional - needed for online services but not for local models
	return nil
}

// RequiresAPIKey checks if the API endpoint likely requires authentication
func (c *Config) RequiresAPIKey() bool {
	// Check if URL contains common online AI service domains
	onlineServices := []string{
		"api.openai.com",
		"openai.azure.com",
		"api.anthropic.com",
		"generativelanguage.googleapis.com",
	}
	
	for _, service := range onlineServices {
		if strings.Contains(c.APIURL, service) {
			return true
		}
	}
	return false
}

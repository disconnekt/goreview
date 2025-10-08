package config

import (
	"errors"
	"strings"
	"time"
)

type Config struct {
	ProjectPath    string
	APIURL         string
	// APIURLs allows specifying multiple OpenAI-compatible endpoints.
	// If provided, these take precedence over APIURL and will be used in a round-robin fashion.
	APIURLs        []string
	APIKey         string
	Model          string
	MaxFileSize    int64
	RequestTimeout time.Duration
	MaxConcurrency int
	// ReportFile, if set, writes the review content (without logs) to the given file.
	// When empty, the review content is printed to stdout as before.
	ReportFile     string
}

func DefaultConfig() *Config {
	return &Config{
		ProjectPath:    ".",
		APIURL:         "http://127.0.0.1:1234/v1/chat/completions",
		Model:          "devstral-small-2507-mlx",
		MaxFileSize:    10 * 1024 * 1024, // 10MB
		RequestTimeout: 720 * time.Second,
		MaxConcurrency: 10,
	}
}

func (c *Config) Validate() error {
	urlList := c.EffectiveAPIURLs()
	if len(urlList) == 0 {
		return errors.New("at least one API URL must be provided")
	}
	for _, u := range urlList {
		if strings.TrimSpace(u) == "" {
			return errors.New("API URL cannot be empty")
		}
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

	return nil
}

// EffectiveAPIURLs returns the list of API URLs to use. If APIURLs is set,
// it takes precedence; otherwise, it falls back to the single APIURL value.
func (c *Config) EffectiveAPIURLs() []string {
	if len(c.APIURLs) > 0 {
		return c.APIURLs
	}
	if strings.TrimSpace(c.APIURL) != "" {
		return []string{c.APIURL}
	}
	return nil
}

func (c *Config) RequiresAPIKey() bool {
	onlineServices := []string{
		"api.openai.com",
		"openai.azure.com",
		"api.anthropic.com",
		"generativelanguage.googleapis.com",
	}

	urlList := c.EffectiveAPIURLs()
	for _, url := range urlList {
		for _, service := range onlineServices {
			if strings.Contains(url, service) {
				return true
			}
		}
	}
	return false
}

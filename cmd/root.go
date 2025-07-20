package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"localhost/aireview/internal/config"
	"localhost/aireview/internal/reviewer"
	"localhost/aireview/internal/scanner"
)

var cfg *config.Config

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "aireview",
	Short: "AI-powered code review tool for Go projects",
	Long: `AIReview is a command-line tool that analyzes Go code files 
and provides intelligent code review suggestions using AI.`,
	RunE: runReview,
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cfg = config.DefaultConfig()
	
	rootCmd.Flags().StringVarP(&cfg.ProjectPath, "path", "p", cfg.ProjectPath, 
		"Path to the project directory for review")
	rootCmd.Flags().StringVarP(&cfg.APIURL, "url", "u", cfg.APIURL, 
		"URL to the AI API endpoint")
	rootCmd.Flags().StringVarP(&cfg.APIKey, "api-key", "k", cfg.APIKey, 
		"API key for authentication (can also use AIREVIEW_API_KEY env var)")
	rootCmd.Flags().StringVarP(&cfg.Model, "model", "m", cfg.Model, 
		"AI model to use for code review")
	rootCmd.Flags().Int64Var(&cfg.MaxFileSize, "max-size", cfg.MaxFileSize, 
		"Maximum file size in bytes to process")
	rootCmd.Flags().IntVarP(&cfg.MaxConcurrency, "concurrency", "c", cfg.MaxConcurrency, 
		"Maximum number of concurrent reviews")
}

func runReview(cmd *cobra.Command, args []string) error {
	// Check for API key in environment variable if not provided via flag
	if cfg.APIKey == "" {
		if envKey := os.Getenv("AIREVIEW_API_KEY"); envKey != "" {
			cfg.APIKey = envKey
		}
	}
	
	// Warn if API key might be required but not provided
	if cfg.RequiresAPIKey() && cfg.APIKey == "" {
		fmt.Fprintf(os.Stderr, "Warning: This API endpoint (%s) likely requires an API key.\n", cfg.APIURL)
		fmt.Fprintf(os.Stderr, "Use --api-key flag or set AIREVIEW_API_KEY environment variable.\n\n")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Initialize services
	fileScanner := scanner.NewScanner(cfg.MaxFileSize)
	reviewService := reviewer.NewService(cfg)

	// Scan for Go files
	fmt.Printf("Scanning directory: %s\n", cfg.ProjectPath)
	files, err := fileScanner.ScanGoFiles(cfg.ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No Go files found to review")
		return nil
	}

	fmt.Printf("Found %d Go files to review\n", len(files))

	// Process files concurrently with limited concurrency
	return processFilesWithConcurrency(reviewService, files, cfg.MaxConcurrency)
}

func processFilesWithConcurrency(reviewService *reviewer.Service, files []scanner.FileInfo, maxConcurrency int) error {
	ctx := context.Background()
	
	// Create a semaphore to limit concurrency
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, file := range files {
		wg.Add(1)
		go func(f scanner.FileInfo) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("Reviewing: %s\n", f.Path)
			
			review, err := reviewService.ReviewCode(ctx, f.Content)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to review %s: %w", f.Path, err))
				mu.Unlock()
				return
			}

			if review != "" {
				mu.Lock()
				fmt.Printf("\n=== Review for %s ===\n", f.Path)
				fmt.Printf("File size: %d bytes\n", f.Size)
				fmt.Printf("Review:\n%s\n\n", review)
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()

	// Report any errors
	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nEncountered %d errors during review:\n", len(errors))
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "- %v\n", err)
		}
		return fmt.Errorf("review completed with %d errors", len(errors))
	}

	fmt.Printf("\nReview completed successfully for %d files\n", len(files))
	return nil
}

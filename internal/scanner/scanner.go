package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo represents information about a scanned file
type FileInfo struct {
	Path    string
	Size    int64
	Content string
}

// Scanner handles file system operations
type Scanner struct {
	maxFileSize int64
}

// NewScanner creates a new file scanner
func NewScanner(maxFileSize int64) *Scanner {
	return &Scanner{
		maxFileSize: maxFileSize,
	}
}

// ScanGoFiles scans directory for Go files and returns their information
func (s *Scanner) ScanGoFiles(dirPath string) ([]FileInfo, error) {
	var files []FileInfo

	// Clean and validate path
	cleanPath := filepath.Clean(dirPath)
	if !filepath.IsAbs(cleanPath) {
		abs, err := filepath.Abs(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		cleanPath = abs
	}

	err := filepath.Walk(cleanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() {
			// Skip common directories that shouldn't be analyzed
			if s.shouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// Skip test files for now (can be made configurable)
		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		// Check file size
		if info.Size() > s.maxFileSize {
			fmt.Printf("Warning: Skipping file %s (size %d exceeds limit %d)\n",
				path, info.Size(), s.maxFileSize)
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		files = append(files, FileInfo{
			Path:    path,
			Size:    info.Size(),
			Content: string(content),
		})

		return nil
	})

	return files, err
}

// shouldSkipDir determines if a directory should be skipped during scanning
func (s *Scanner) shouldSkipDir(dirName string) bool {
	skipDirs := []string{
		"vendor",
		".git",
		".vscode",
		".idea",
		"node_modules",
		"build",
		"dist",
		"bin",
		"tmp",
		".tmp",
	}

	for _, skip := range skipDirs {
		if dirName == skip {
			return true
		}
	}
	return false
}

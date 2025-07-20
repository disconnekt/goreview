package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Path    string
	Size    int64
	Content string
}

type Scanner struct {
	maxFileSize int64
}
func NewScanner(maxFileSize int64) *Scanner {
	return &Scanner{
		maxFileSize: maxFileSize,
	}
}

func (s *Scanner) ScanGoFiles(dirPath string) ([]FileInfo, error) {
	var files []FileInfo

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

		if info.IsDir() {
			if s.shouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}
		if info.Size() > s.maxFileSize {
			fmt.Printf("Warning: Skipping file %s (size %d exceeds limit %d)\n",
				path, info.Size(), s.maxFileSize)
			return nil
		}

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

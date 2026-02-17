package worktree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
)

// CopyFiles copies files from source directory to destination based on include/exclude patterns
func CopyFiles(sourceDir, destDir string, copyConfig *config.CopyConfig) error {
	if copyConfig == nil {
		return nil
	}

	// Find all files matching include patterns
	filesToCopy := make(map[string]bool)

	for _, pattern := range copyConfig.Include {
		matches, err := doublestar.Glob(os.DirFS(sourceDir), pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}

		for _, match := range matches {
			// Check if it's excluded
			if isExcluded(match, copyConfig.Exclude) {
				continue
			}
			filesToCopy[match] = true
		}
	}

	// Copy files
	for filePath := range filesToCopy {
		sourcePath := filepath.Join(sourceDir, filePath)
		destPath := filepath.Join(destDir, filePath)

		// Check if source is a directory
		info, err := os.Stat(sourcePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot stat %s: %v\n", sourcePath, err)
			continue
		}

		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(destPath, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cannot create directory %s: %v\n", destPath, err)
			}
			continue
		}

		// Copy file
		if err := copyFile(sourcePath, destPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to copy %s: %v\n", filePath, err)
			continue
		}
	}

	return nil
}

func isExcluded(path string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		// Try exact match first
		matched, err := doublestar.Match(pattern, path)
		if err != nil {
			continue
		}
		if matched {
			return true
		}

		// Check if path is within an excluded directory (pattern ending with /)
		if len(pattern) > 0 && pattern[len(pattern)-1] == '/' {
			if len(path) > len(pattern) && path[:len(pattern)] == pattern {
				return true
			}
		}
	}
	return false
}

func copyFile(sourcePath, destPath string) error {
	// Create destination directory if needed
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer destFile.Close()

	// Copy content
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy content: %w", err)
	}

	// Copy permissions
	sourceInfo, err := os.Stat(sourcePath)
	if err == nil {
		os.Chmod(destPath, sourceInfo.Mode())
	}

	return nil
}

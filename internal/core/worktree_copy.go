package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
)

// CopyWorktreeFiles copies files from project root to worktree based on copy configuration
func CopyWorktreeFiles(projectRoot, worktreePath string, copyConfig *config.WorktreeCopy) error {
	if copyConfig == nil || len(copyConfig.Include) == 0 {
		return nil
	}

	for _, pattern := range copyConfig.Include {
		if err := copyFilesMatchingPattern(projectRoot, worktreePath, pattern, copyConfig.Exclude); err != nil {
			return fmt.Errorf("failed to copy files matching pattern %q: %w", pattern, err)
		}
	}

	return nil
}

func copyFilesMatchingPattern(projectRoot, worktreePath, pattern string, excludePatterns []string) error {
	// Use doublestar for glob matching with ** support
	matches, err := doublestar.Glob(os.DirFS(projectRoot), pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}

	for _, match := range matches {
		// Skip if matches any exclude pattern
		if shouldExclude(match, excludePatterns) {
			continue
		}

		sourcePath := filepath.Join(projectRoot, match)
		targetPath := filepath.Join(worktreePath, match)

		// Get file info
		info, err := os.Stat(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to stat %q: %w", sourcePath, err)
		}

		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %q: %w", targetPath, err)
			}
			continue
		}

		// Copy file
		if err := copyFile(sourcePath, targetPath, info.Mode()); err != nil {
			return fmt.Errorf("failed to copy file %q to %q: %w", sourcePath, targetPath, err)
		}
	}

	return nil
}

func shouldExclude(path string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		// Try exact match first
		if matched, _ := doublestar.Match(pattern, path); matched {
			return true
		}

		// Try with trailing slash for directories
		if !strings.HasSuffix(pattern, "/") {
			if matched, _ := doublestar.Match(pattern+"/", path); matched {
				return true
			}
		}

		// Check if path is within an excluded directory
		if strings.HasPrefix(path, pattern+"/") || strings.HasPrefix(path, pattern+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func copyFile(sourcePath, targetPath string, mode os.FileMode) error {
	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Open source file
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	// Create target file
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer target.Close()

	// Copy content
	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

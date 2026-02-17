package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
)

func TestCopyFiles(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(sourceDir, ".env.local"), []byte("ENV=local"), 0644)
	os.MkdirAll(filepath.Join(sourceDir, "config"), 0755)
	os.WriteFile(filepath.Join(sourceDir, "config", ".env.local"), []byte("ENV=config"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "vendor", "file.php"), []byte("vendor"), 0644)

	copyConfig := &config.CopyConfig{
		Include: []string{"**/.env.local"},
		Exclude: []string{"vendor/"},
	}

	err := CopyFiles(sourceDir, destDir, copyConfig)
	if err != nil {
		t.Fatalf("CopyFiles failed: %v", err)
	}

	// Check files were copied
	if _, err := os.Stat(filepath.Join(destDir, ".env.local")); err != nil {
		t.Error("expected .env.local to be copied")
	}
	if _, err := os.Stat(filepath.Join(destDir, "config", ".env.local")); err != nil {
		t.Error("expected config/.env.local to be copied")
	}
	if _, err := os.Stat(filepath.Join(destDir, "vendor", "file.php")); err == nil {
		t.Error("expected vendor/file.php to NOT be copied (excluded)")
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		expected bool
	}{
		{"vendor/file.php", []string{"vendor/"}, true},
		{"src/file.php", []string{"vendor/"}, false},
		{"node_modules/pkg/index.js", []string{"node_modules/"}, true},
		{"vendor/autoload.php", []string{"vendor/**"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isExcluded(tt.path, tt.patterns)
			if result != tt.expected {
				t.Errorf("isExcluded(%q, %v) = %v, want %v", tt.path, tt.patterns, result, tt.expected)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	sourcePath := filepath.Join(sourceDir, "test.txt")
	destPath := filepath.Join(destDir, "subdir", "test.txt")

	content := []byte("test content")
	os.WriteFile(sourcePath, content, 0644)

	err := copyFile(sourcePath, destPath)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	copied, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}

	if string(copied) != string(content) {
		t.Errorf("copied content mismatch: got %q, want %q", string(copied), string(content))
	}
}

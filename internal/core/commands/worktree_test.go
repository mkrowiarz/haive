package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core"
)

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid normal name", "feature/test", false},
		{"empty name", "", true},
		{"invalid chars", "test;rm -rf", true},
		{"path traversal", "../etc/passwd", true},
		{"slashes", "feature/test", false},
		{"hyphens", "fix-auth-bug", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := core.ValidateBranchName(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateBranchName(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

func TestCheckPathTraversal(t *testing.T) {
	basePath := "/tmp/worktrees"

	tests := []struct {
		name      string
		resolved  string
		base      string
		wantError bool
	}{
		{"valid path", "/tmp/worktrees/feature-test", basePath, false},
		{"traversal attempt", "/tmp/worktrees/../etc/passwd", basePath, true},
		{"escape attempt", "/etc/passwd", basePath, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := core.CheckPathTraversal(tt.resolved, tt.base)
			if (err != nil) != tt.wantError {
				t.Errorf("CheckPathTraversal(%q, %q) error = %v, wantError %v", tt.resolved, tt.base, err, tt.wantError)
			}
		})
	}
}

func TestSanitizeWorktreeName(t *testing.T) {
	tests := []struct {
		input   string
		wantDir string
		wantDB  string
	}{
		{"feature/test", "feature-test", "feature_test"},
		{"fix-auth-bug", "fix-auth-bug", "fix_auth_bug"},
		{"hotfix/critical/issue", "hotfix-critical-issue", "hotfix_critical_issue"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			dirName, dbName := core.SanitizeWorktreeName(tt.input)
			if dirName != tt.wantDir {
				t.Errorf("SanitizeWorktreeName(%q) dirName = %q, want %q", tt.input, dirName, tt.wantDir)
			}
			if dbName != tt.wantDB {
				t.Errorf("SanitizeWorktreeName(%q) dbName = %q, want %q", tt.input, dbName, tt.wantDB)
			}
		})
	}
}

func TestWorktreeCreateValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "worktree-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".haive.json")

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"worktrees": {"base_path": "` + tmpDir + `/worktrees"}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("empty branch name", func(t *testing.T) {
		_, err := Create(tmpDir, "", false)
		if err == nil {
			t.Error("expected error for empty branch name")
		}
	})

	t.Run("invalid branch name", func(t *testing.T) {
		_, err := Create(tmpDir, "test;rm -rf", false)
		if err == nil {
			t.Error("expected error for invalid branch name")
		}
	})
}

func TestWorktreeRemoveValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "worktree-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".haive.json")

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"worktrees": {"base_path": "` + tmpDir + `/worktrees"}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("empty branch name", func(t *testing.T) {
		_, err := Remove(tmpDir, "")
		if err == nil {
			t.Error("expected error for empty branch name")
		}
	})

	t.Run("invalid branch name", func(t *testing.T) {
		_, err := Remove(tmpDir, "test;rm -rf")
		if err == nil {
			t.Error("expected error for invalid branch name")
		}
	})
}

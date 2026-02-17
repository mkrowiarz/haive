package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHookContext_ToEnv(t *testing.T) {
	ctx := &HookContext{
		RepoRoot:       "/project",
		ProjectName:    "myapp",
		WorktreePath:   "/project/.worktrees/feature",
		WorktreeName:   "feature",
		Branch:         "feature/test",
		DatabaseName:   "myapp_feature",
		DatabaseURL:    "mysql://user:pass@db:3306/myapp_feature",
		SourceDatabase: "myapp",
		TargetDatabase: "myapp_feature",
	}

	env := ctx.ToEnv()

	expectedVars := map[string]string{
		"REPO_ROOT=/project":       "REPO_ROOT=/project",
		"PROJECT_NAME=myapp":       "PROJECT_NAME=myapp",
		"WORKTREE_PATH=/project/.worktrees/feature": "WORKTREE_PATH=/project/.worktrees/feature",
		"WORKTREE_NAME=feature":    "WORKTREE_NAME=feature",
		"BRANCH=feature/test":      "BRANCH=feature/test",
		"DATABASE_NAME=myapp_feature": "DATABASE_NAME=myapp_feature",
		"DATABASE_URL=mysql://user:pass@db:3306/myapp_feature": "DATABASE_URL=mysql://user:pass@db:3306/myapp_feature",
		"SOURCE_DATABASE=myapp":    "SOURCE_DATABASE=myapp",
		"TARGET_DATABASE=myapp_feature": "TARGET_DATABASE=myapp_feature",
	}

	for expectedKey, expectedFull := range expectedVars {
		found := false
		for _, e := range env {
			if e == expectedFull {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected env var %s", expectedKey)
		}
	}
}

func TestExecutor_ExecuteHooks_Command(t *testing.T) {
	tmpDir := t.TempDir()

	exec := NewExecutor(tmpDir)
	ctx := &HookContext{
		RepoRoot:    tmpDir,
		ProjectName: "test",
	}

	// Test simple command that succeeds
	hooks := []string{"echo hello"}
	err := exec.ExecuteHooks(hooks, ctx, tmpDir, false)

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestExecutor_ExecuteHooks_Script(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test script
	scriptPath := filepath.Join(tmpDir, "test-hook.sh")
	scriptContent := "#!/bin/sh\necho 'hook ran'"
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)

	exec := NewExecutor(tmpDir)
	ctx := &HookContext{
		RepoRoot:    tmpDir,
		ProjectName: "test",
	}

	// Test script hook
	hooks := []string{"./test-hook.sh"}
	err := exec.ExecuteHooks(hooks, ctx, tmpDir, false)

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestExecutor_ExecuteHooks_PreHookFailure(t *testing.T) {
	tmpDir := t.TempDir()

	exec := NewExecutor(tmpDir)
	ctx := &HookContext{
		RepoRoot:    tmpDir,
		ProjectName: "test",
	}

	// Pre-hook that fails should return error
	hooks := []string{"exit 1"}
	err := exec.ExecuteHooks(hooks, ctx, tmpDir, true)

	if err == nil {
		t.Error("expected error for failed pre-hook")
	}
}

func TestIsScriptFile(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"echo hello", false},
		{"composer install", false},
		{"./hooks/setup.sh", true},
		{"../scripts/test.sh", true},
		{"/absolute/path/script.sh", true},
		{".haive/hooks/post-create", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isScriptFile(tt.input)
			if result != tt.expected {
				t.Errorf("isScriptFile(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

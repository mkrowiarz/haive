package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCreateWorktree_WithCopyAndHooks(t *testing.T) {
	// This is an integration test that requires git
	// Skip if not in CI or if git is not available
	if os.Getenv("CI") == "" {
		t.Skip("Skipping integration test - run with CI=1 to enable")
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available")
	}

	// Setup: create a temp git repo with config
	repoDir := t.TempDir()

	// Initialize git repo
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	gitCmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}

	for _, args := range gitCmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git command failed: %v", err)
		}
	}

	// Create initial commit
	os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Test"), 0644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	// Create config with worktrees and copy settings
	configContent := `{
  "project": {"name": "test-project", "type": "generic"},
  "docker": {"compose_files": ["compose.yaml"]},
  "worktrees": {
    "base_path": ".worktrees",
    "copy": {
      "include": [".env.local"]
    }
  }
}`
	os.WriteFile(filepath.Join(repoDir, ".haive.json"), []byte(configContent), 0644)
	os.WriteFile(filepath.Join(repoDir, ".env.local"), []byte("ENV=local"), 0644)

	// Stage and commit config files
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "add config")
	cmd.Dir = repoDir
	cmd.Run()

	// Create worktree using the Create command
	result, err := Create(repoDir, "feature/test", true)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Verify worktree path
	expectedPath := filepath.Join(repoDir, ".worktrees", "feature-test")
	if result.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result.Path)
	}

	// Verify .env.local was copied
	copiedEnv := filepath.Join(expectedPath, ".env.local")
	if _, err := os.Stat(copiedEnv); err != nil {
		t.Error(".env.local was not copied to worktree")
	}

	// Verify content was copied correctly
	content, err := os.ReadFile(copiedEnv)
	if err != nil {
		t.Errorf("failed to read copied .env.local: %v", err)
	}
	if string(content) != "ENV=local" {
		t.Errorf("copied content mismatch: got %q, want %q", string(content), "ENV=local")
	}
}

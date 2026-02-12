package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateIsolatedWorktreeNoDB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".haive.json")

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"worktrees": {"base_path": "` + tmpDir + `/wt"}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = CreateIsolatedWorktree(tmpDir, "feature/test", "true", "")
	if err == nil {
		t.Error("expected error (git not available)")
	}
}

func TestRemoveIsolatedWorktreeNoDB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".haive.json")

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"worktrees": {"base_path": "` + tmpDir + `/wt"}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = RemoveIsolatedWorktree(tmpDir, "feature/test", false)
	if err == nil {
		t.Error("expected error (git not available)")
	}
}

func TestRemoveIsolatedWorktreeWithDB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".haive.json")

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"database": {
			"service": "database",
			"dsn": "mysql://root:secret@database:3306/app",
			"allowed": ["app", "app_*"]
		},
		"worktrees": {
			"base_path": "` + tmpDir + `/wt",
			"db_per_worktree": true
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = RemoveIsolatedWorktree(tmpDir, "feature/test", true)
	if err == nil {
		t.Error("expected error (git not available)")
	}
}

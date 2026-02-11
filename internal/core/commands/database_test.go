package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

func TestDumpDisallowedDB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"database": {
			"service": "database",
			"dsn": "mysql://root:secret@database:3306/app",
			"allowed": ["app", "app_*"],
			"dumps_path": "var/dumps"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = Dump(tmpDir, "other_db", nil)
	if err == nil {
		t.Error("expected error for disallowed database")
	}

	cmdErr, ok := err.(*types.CommandError)
	if !ok {
		t.Errorf("expected CommandError, got %T", err)
		return
	}

	if cmdErr.Code != types.ErrDbNotAllowed {
		t.Errorf("expected ErrDbNotAllowed, got %s", cmdErr.Code)
	}
}

func TestDropDefaultDB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"database": {
			"service": "database",
			"dsn": "mysql://root:secret@database:3306/app",
			"allowed": ["app", "app_*"],
			"dumps_path": "var/dumps"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = DropDB(tmpDir, "app")
	if err == nil {
		t.Error("expected error for dropping default database")
	}

	cmdErr, ok := err.(*types.CommandError)
	if !ok {
		t.Errorf("expected CommandError, got %T", err)
		return
	}

	if cmdErr.Code != types.ErrDbIsDefault {
		t.Errorf("expected ErrDbIsDefault, got %s", cmdErr.Code)
	}
}

func TestImportFileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"database": {
			"service": "database",
			"dsn": "mysql://root:secret@database:3306/app",
			"allowed": ["app", "app_*"],
			"dumps_path": "var/dumps"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = ImportDB(tmpDir, "app_test", "/nonexistent/file.sql")
	if err == nil {
		t.Error("expected error for missing file")
	}

	cmdErr, ok := err.(*types.CommandError)
	if !ok {
		t.Errorf("expected CommandError, got %T", err)
		return
	}

	if cmdErr.Code != types.ErrFileNotFound {
		t.Errorf("expected ErrFileNotFound, got %s", cmdErr.Code)
	}
}

func TestCreateDBDisallowed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"database": {
			"service": "database",
			"dsn": "mysql://root:secret@database:3306/app",
			"allowed": ["app"],
			"dumps_path": "var/dumps"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = CreateDB(tmpDir, "other_db")
	if err == nil {
		t.Error("expected error for disallowed database")
	}

	cmdErr, ok := err.(*types.CommandError)
	if !ok {
		t.Errorf("expected CommandError, got %T", err)
		return
	}

	if cmdErr.Code != types.ErrDbNotAllowed {
		t.Errorf("expected ErrDbNotAllowed, got %s", cmdErr.Code)
	}
}

func TestWildcardAllowedPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"database": {
			"service": "database",
			"dsn": "mysql://root:secret@database:3306/app",
			"allowed": ["app_*"],
			"dumps_path": "var/dumps"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = CreateDB(tmpDir, "app_staging")
	if err == nil {
		t.Error("expected error (docker not available)")
	}

	if cmdErr, ok := err.(*types.CommandError); ok {
		if cmdErr.Code == types.ErrDbNotAllowed {
			t.Error("app_staging should match app_* pattern - guard should pass")
		}
	}
}

func TestMissingDatabaseConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = Dump(tmpDir, "app", nil)
	if err == nil {
		t.Error("expected error when database config is missing")
	}
}

func TestListDBsMissingConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = ListDBs(tmpDir)
	if err == nil {
		t.Error("expected error when database config is missing")
	}

	cmdErr, ok := err.(*types.CommandError)
	if !ok {
		t.Errorf("expected CommandError, got %T", err)
		return
	}

	if cmdErr.Code != types.ErrConfigMissing {
		t.Errorf("expected ErrConfigMissing, got %s", cmdErr.Code)
	}
}

func TestCloneDBDisallowedSource(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"database": {
			"service": "database",
			"dsn": "mysql://root:secret@database:3306/app",
			"allowed": ["app", "app_*"],
			"dumps_path": "var/dumps"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = CloneDB(tmpDir, "other_db", "app_test")
	if err == nil {
		t.Error("expected error for disallowed source database")
	}

	cmdErr, ok := err.(*types.CommandError)
	if !ok {
		t.Errorf("expected CommandError, got %T", err)
		return
	}

	if cmdErr.Code != types.ErrDbNotAllowed {
		t.Errorf("expected ErrDbNotAllowed, got %s", cmdErr.Code)
	}
}

func TestCloneDBDisallowedTarget(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"database": {
			"service": "database",
			"dsn": "mysql://root:secret@database:3306/app",
			"allowed": ["app", "app_*"],
			"dumps_path": "var/dumps"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = CloneDB(tmpDir, "app", "other_db")
	if err == nil {
		t.Error("expected error for disallowed target database")
	}

	cmdErr, ok := err.(*types.CommandError)
	if !ok {
		t.Errorf("expected CommandError, got %T", err)
		return
	}

	if cmdErr.Code != types.ErrDbNotAllowed {
		t.Errorf("expected ErrDbNotAllowed, got %s", cmdErr.Code)
	}
}

func TestListDumpsEmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	dumpsDir := filepath.Join(tmpDir, "var", "dumps")
	if err := os.MkdirAll(dumpsDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"database": {
			"service": "database",
			"dsn": "mysql://root:secret@database:3306/app",
			"allowed": ["app"],
			"dumps_path": "` + dumpsDir + `"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ListDumps(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(result.Dumps) != 0 {
		t.Errorf("expected empty list, got %d dumps", len(result.Dumps))
	}
}

func TestListDumpsValidFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, ".claude", "project.json")
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	dumpsDir := filepath.Join(tmpDir, "var", "dumps")
	if err := os.MkdirAll(dumpsDir, 0755); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(filepath.Join(dumpsDir, "app_2025-02-11T10-30.sql"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dumpsDir, "app_2025-02-10T18-00.sql"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dumpsDir, "invalid.txt"), []byte("test"), 0644)

	cfgContent := `{
		"project": {"name": "test", "type": "symfony"},
		"docker": {"compose_files": ["docker-compose.yaml"]},
		"database": {
			"service": "database",
			"dsn": "mysql://root:secret@database:3306/app",
			"allowed": ["app"],
			"dumps_path": "` + dumpsDir + `"
		}
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ListDumps(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(result.Dumps) != 2 {
		t.Errorf("expected 2 dumps, got %d", len(result.Dumps))
		return
	}

	if result.Dumps[0].Database != "app" {
		t.Errorf("expected database 'app', got %s", result.Dumps[0].Database)
	}
}

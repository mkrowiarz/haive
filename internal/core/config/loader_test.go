package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Load_TOML(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
[project]
name = "test-project"

[docker]
compose_files = ["compose.yaml"]

[worktree]
base_path = ".worktrees"

[database]
service = "database"
dsn = "mysql://user:pass@db:3306/test"
allowed = ["test", "test_*"]
`

	configPath := filepath.Join(tmpDir, "haive.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader()
	cfg, err := loader.Load(tmpDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Project.Name != "test-project" {
		t.Errorf("expected project name 'test-project', got '%s'", cfg.Project.Name)
	}

	if cfg.Worktree == nil || cfg.Worktree.BasePath != ".worktrees" {
		t.Error("expected worktree config with base_path '.worktrees'")
	}
}

func TestLoader_Load_Priority(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config in .haive/config.toml
	os.MkdirAll(filepath.Join(tmpDir, ".haive"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".haive", "config.toml"), []byte(`
[project]
name = "nested-config"
`), 0644)

	loader := NewLoader()
	cfg, err := loader.Load(tmpDir)

	if err != nil {
		t.Fatal(err)
	}

	if cfg.Project.Name != "nested-config" {
		t.Errorf("expected 'nested-config', got: %s", cfg.Project.Name)
	}
}

func TestLoader_Load_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	loader := NewLoader()
	_, err := loader.Load(tmpDir)

	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestHaiveConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     HaiveConfig
		wantErr bool
	}{
		{
			name: "valid minimal",
			cfg: HaiveConfig{
				Project: ProjectConfig{Name: "test"},
				Docker:  DockerConfig{ComposeFiles: []string{"compose.yaml"}},
			},
			wantErr: false,
		},
		{
			name:    "missing project name",
			cfg:     HaiveConfig{},
			wantErr: true,
		},
		{
			name: "worktree without base_path",
			cfg: HaiveConfig{
				Project:  ProjectConfig{Name: "test"},
				Worktree: &WorktreeConfig{},
			},
			wantErr: true,
		},
		{
			name: "env without database",
			cfg: HaiveConfig{
				Project: ProjectConfig{Name: "test"},
				Worktree: &WorktreeConfig{
					BasePath: ".worktrees",
					Env:      &EnvConfig{File: ".env.local", VarName: "DATABASE_URL"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

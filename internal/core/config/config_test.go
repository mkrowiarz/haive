package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name          string
		projectRoot   string
		setupFunc     func() (string, error)
		cleanupFunc   func()
		expectedError types.ErrCode
	}{
		{
			name:        "valid config loads successfully",
			projectRoot: "./testdata",
			setupFunc: func() (string, error) {
				configPath := filepath.Join("./testdata", ".haive.json")
				sampleConfig, err := os.ReadFile(filepath.Join("./testdata", "sample-config.json"))
				if err != nil {
					return "", err
				}
				return "", os.WriteFile(configPath, sampleConfig, 0644)
			},
			cleanupFunc: func() {
				os.Remove(filepath.Join("./testdata", ".haive.json"))
			},
			expectedError: "",
		},
		{
			name:        "missing config returns ErrConfigMissing",
			projectRoot: "/nonexistent/path",
			setupFunc: func() (string, error) {
				return "", nil
			},
			cleanupFunc:   func() {},
			expectedError: types.ErrConfigMissing,
		},
		{
			name: "malformed JSON returns ErrConfigInvalid",
			setupFunc: func() (string, error) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, ".haive.json")
				return tmpDir, os.WriteFile(configPath, []byte("{invalid json"), 0644)
			},
			cleanupFunc:   func() {},
			expectedError: types.ErrConfigInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				root, err := tt.setupFunc()
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				if root != "" {
					tt.projectRoot = root
				}
			}
			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			cfg, err := Load(tt.projectRoot)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error %s, got nil", tt.expectedError)
					return
				}
				cmdErr, ok := err.(*types.CommandError)
				if !ok {
					t.Errorf("expected CommandError, got %T", err)
					return
				}
				if cmdErr.Code != tt.expectedError {
					t.Errorf("expected error code %s, got %s", tt.expectedError, cmdErr.Code)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Error("expected config, got nil")
				return
			}

			if cfg.Project == nil {
				t.Error("expected Project section, got nil")
			} else {
				if cfg.Project.Name != "facility-saas" {
					t.Errorf("expected project name 'facility-saas', got '%s'", cfg.Project.Name)
				}
				if cfg.Project.Type != "symfony" {
					t.Errorf("expected project type 'symfony', got '%s'", cfg.Project.Type)
				}
			}

			if cfg.Docker == nil {
				t.Error("expected Docker section, got nil")
			} else {
				if len(cfg.Docker.ComposeFiles) != 1 || cfg.Docker.ComposeFiles[0] != "docker-compose.yaml" {
					t.Errorf("expected compose_files ['docker-compose.yaml'], got '%v'", cfg.Docker.ComposeFiles)
				}
			}
		})
	}
}

package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInfo(t *testing.T) {
	t.Run("missing config returns info with nil summary", func(t *testing.T) {
		tmpDir := t.TempDir()
		info, err := Info(tmpDir)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if info.ConfigSummary != nil {
			t.Error("expected nil ConfigSummary for missing config")
		}

		if info.DockerComposeExists {
			t.Error("expected DockerComposeExists to be false in empty dir")
		}
	})

	t.Run("existing config returns populated info", func(t *testing.T) {
		tmpDir := t.TempDir()

		configPath := filepath.Join(tmpDir, ".claude", "project.json")
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}

		sampleConfig, err := os.ReadFile(filepath.Join("../config/testdata", "sample-config.json"))
		if err != nil {
			t.Fatalf("failed to read sample config: %v", err)
		}

		if err := os.WriteFile(configPath, sampleConfig, 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		composePath := filepath.Join(tmpDir, "docker-compose.yaml")
		composeContent := `services:
  php:
    image: php:8.3-fpm
`
		if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
			t.Fatalf("failed to write compose file: %v", err)
		}

		info, err := Info(tmpDir)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if info.ConfigSummary == nil {
			t.Error("expected ConfigSummary to be populated")
			return
		}

		if info.ConfigSummary.Name != "facility-saas" {
			t.Errorf("expected name 'facility-saas', got '%s'", info.ConfigSummary.Name)
		}

		if info.ConfigSummary.Type != "symfony" {
			t.Errorf("expected type 'symfony', got '%s'", info.ConfigSummary.Type)
		}

		if !info.DockerComposeExists {
			t.Error("expected DockerComposeExists to be true")
		}
	})
}

func TestDetectEnvFiles(t *testing.T) {
	t.Run("no env files returns empty slice", func(t *testing.T) {
		tmpDir := t.TempDir()
		files := detectEnvFiles(tmpDir)

		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})

	t.Run("detects .env file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, ".env")
		os.WriteFile(envPath, []byte("VAR=value\n"), 0644)

		files := detectEnvFiles(tmpDir)

		if len(files) != 1 {
			t.Errorf("expected 1 file, got %d", len(files))
		}
	})

	t.Run("detects both .env and .env.local", func(t *testing.T) {
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, ".env")
		envLocalPath := filepath.Join(tmpDir, ".env.local")
		os.WriteFile(envPath, []byte("VAR=value\n"), 0644)
		os.WriteFile(envLocalPath, []byte("VAR2=value\n"), 0644)

		files := detectEnvFiles(tmpDir)

		if len(files) != 2 {
			t.Errorf("expected 2 files, got %d", len(files))
		}
	})
}

func TestDockerComposeExists(t *testing.T) {
	t.Run("no compose file returns false", func(t *testing.T) {
		tmpDir := t.TempDir()
		exists := dockerComposeExists(tmpDir)

		if exists {
			t.Error("expected DockerComposeExists to be false")
		}
	})

	t.Run("docker-compose.yaml returns true", func(t *testing.T) {
		tmpDir := t.TempDir()
		composePath := filepath.Join(tmpDir, "docker-compose.yaml")
		os.WriteFile(composePath, []byte("services:\n"), 0644)

		exists := dockerComposeExists(tmpDir)

		if !exists {
			t.Error("expected DockerComposeExists to be true")
		}
	})
}

func TestDetectDockerServices(t *testing.T) {
	t.Run("detects services from docker-compose.yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		composePath := filepath.Join(tmpDir, "docker-compose.yaml")
		composeContent := `services:
  php:
    image: php:8.3-fpm
  database:
    image: mariadb:11.4
  redis:
    image: redis:7-alpine
`
		os.WriteFile(composePath, []byte(composeContent), 0644)

		services, err := detectDockerServices(tmpDir)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if services == nil {
			t.Error("expected services map, got nil")
			return
		}

		if len(services) != 3 {
			t.Errorf("expected 3 services, got %d", len(services))
		}

		if services["database"] != "mariadb:11.4" {
			t.Errorf("expected database service image 'mariadb:11.4', got '%s'", services["database"])
		}
	})

	t.Run("no compose file returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		services, err := detectDockerServices(tmpDir)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if services != nil {
			t.Error("expected nil services, got map")
		}
	})
}

func TestDetectProjectType(t *testing.T) {
	t.Run("symfony project returns symfony", func(t *testing.T) {
		tmpDir := t.TempDir()
		composerPath := filepath.Join(tmpDir, "composer.json")
		composerContent := `{
  "type": "project",
  "require": {
    "symfony/console": "^7.0"
  }
}`
		os.WriteFile(composerPath, []byte(composerContent), 0644)

		projectType := detectProjectType(tmpDir)

		if projectType != "symfony" {
			t.Errorf("expected 'symfony', got '%s'", projectType)
		}
	})

	t.Run("laravel project returns laravel", func(t *testing.T) {
		tmpDir := t.TempDir()
		composerPath := filepath.Join(tmpDir, "composer.json")
		composerContent := `{
  "type": "laravel",
  "require": {
    "laravel/framework": "^11.0"
  }
}`
		os.WriteFile(composerPath, []byte(composerContent), 0644)

		projectType := detectProjectType(tmpDir)

		if projectType != "laravel" {
			t.Errorf("expected 'laravel', got '%s'", projectType)
		}
	})

	t.Run("no composer.json returns generic", func(t *testing.T) {
		tmpDir := t.TempDir()
		projectType := detectProjectType(tmpDir)

		if projectType != "generic" {
			t.Errorf("expected 'generic', got '%s'", projectType)
		}
	})
}

func TestDetectEnvVars(t *testing.T) {
	t.Run("parses env variables from .env", func(t *testing.T) {
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, ".env")
		envContent := `# Comment line
VAR1=value1
VAR2=value2

# Another comment
VAR3=value with spaces
`
		os.WriteFile(envPath, []byte(envContent), 0644)

		vars, err := detectEnvVars(tmpDir)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if len(vars) != 3 {
			t.Errorf("expected 3 variables, got %d", len(vars))
		}
	})

	t.Run("no .env file returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		vars, err := detectEnvVars(tmpDir)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if vars != nil {
			t.Error("expected nil, got slice")
		}
	})
}

func TestInit(t *testing.T) {
	t.Run("returns init suggestion", func(t *testing.T) {
		tmpDir := t.TempDir()

		composePath := filepath.Join(tmpDir, "docker-compose.yaml")
		composeContent := `services:
  php:
    image: php:8.3-fpm
  database:
    image: mariadb:11.4
`
		os.WriteFile(composePath, []byte(composeContent), 0644)

		composerPath := filepath.Join(tmpDir, "composer.json")
		composerContent := `{
  "type": "project",
  "require": {
    "symfony/console": "^7.0"
  }
}`
		os.WriteFile(composerPath, []byte(composerContent), 0644)

		envPath := filepath.Join(tmpDir, ".env")
		os.WriteFile(envPath, []byte("DATABASE_URL=mysql://root:pw@db/app\nAPP_ENV=dev\n"), 0644)

		suggestion, err := Init(tmpDir)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if suggestion == nil {
			t.Error("expected InitSuggestion, got nil")
			return
		}

		if len(suggestion.DetectedEnvVars) != 2 {
			t.Errorf("expected 2 env vars, got %d", len(suggestion.DetectedEnvVars))
		}

		if suggestion.DetectedServices == nil {
			t.Error("expected DetectedServices map, got nil")
		}

		if suggestion.SuggestedConfig == "" {
			t.Error("expected non-empty SuggestedConfig")
		}
	})
}

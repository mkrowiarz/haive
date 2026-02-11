package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEnvVars(t *testing.T) {
	t.Run("resolves from environment", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := ResolveEnvVars("prefix_${TEST_VAR}_suffix", "/tmp")
		if result != "prefix_test_value_suffix" {
			t.Errorf("expected 'prefix_test_value_suffix', got '%s'", result)
		}
	})

	t.Run("resolves from .env file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, ".env")
		os.WriteFile(envPath, []byte("MY_VAR=my_value\n"), 0644)

		result := ResolveEnvVars("${MY_VAR}", tmpDir)
		if result != "my_value" {
			t.Errorf("expected 'my_value', got '%s'", result)
		}
	})

	t.Run("resolves from .env.local over .env", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("VAR=env_value\n"), 0644)
		os.WriteFile(filepath.Join(tmpDir, ".env.local"), []byte("VAR=local_value\n"), 0644)

		result := ResolveEnvVars("${VAR}", tmpDir)
		if result != "local_value" {
			t.Errorf("expected 'local_value' from .env.local, got '%s'", result)
		}
	})

	t.Run("returns original if not found", func(t *testing.T) {
		result := ResolveEnvVars("${NONEXISTENT}", "/tmp")
		if result != "${NONEXISTENT}" {
			t.Errorf("expected original string, got '%s'", result)
		}
	})

	t.Run("handles quoted values", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(`QUOTED="quoted value"`+"\n"), 0644)

		result := ResolveEnvVars("${QUOTED}", tmpDir)
		if result != "quoted value" {
			t.Errorf("expected 'quoted value', got '%s'", result)
		}
	})
}

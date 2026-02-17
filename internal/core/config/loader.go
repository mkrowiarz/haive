package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Loader handles config file discovery and parsing
type Loader struct {
	searchPaths []string
}

// NewLoader creates a new config loader with default search paths
func NewLoader() *Loader {
	return &Loader{
		searchPaths: []string{
			"haive.toml",
			".haive/config.toml",
		},
	}
}

// Load discovers and loads config from project root or parent directories
func (l *Loader) Load(startDir string) (*HaiveConfig, error) {
	searchDir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	for {
		for _, path := range l.searchPaths {
			configPath := filepath.Join(searchDir, path)

			if _, err := os.Stat(configPath); err != nil {
				continue // File doesn't exist
			}

			cfg, err := l.parseFile(configPath)
			if err != nil {
				// File exists but is invalid - continue searching
				continue
			}

			if cfg == nil || !l.hasContent(cfg) {
				continue // File exists but has no haive content
			}

			cfg.ProjectRoot = searchDir
			return cfg, nil
		}

		// Move to parent directory
		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			break // Reached root
		}
		searchDir = parent
	}

	return nil, fmt.Errorf("config file not found in %s or parent directories", startDir)
}

func (l *Loader) parseFile(path string) (*HaiveConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg HaiveConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return &cfg, nil
}

func (l *Loader) hasContent(cfg *HaiveConfig) bool {
	return cfg.Project.Name != "" ||
		cfg.Docker.ComposeFiles != nil ||
		cfg.Worktree != nil ||
		cfg.Database != nil
}

// LoadHaive is the convenience function for loading HaiveConfig
func LoadHaive(projectRoot string) (*HaiveConfig, error) {
	loader := NewLoader()
	cfg, err := loader.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	// Resolve environment variables in DSN
	if cfg.Database != nil {
		cfg.Database.DSN = ResolveEnvVars(cfg.Database.DSN, cfg.ProjectRoot)
	}

	// Set default dumps path
	if cfg.Database != nil && cfg.Database.DumpsPath == "" {
		cfg.Database.DumpsPath = "var/dumps"
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/dsn"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

type Config struct {
	Project   *Project   `json:"project"`
	Docker    *Docker    `json:"docker"`
	Database  *Database  `json:"database,omitempty"`
	Worktrees *Worktrees `json:"worktrees,omitempty"`
	// ProjectRoot is the directory where the config file was found
	// Used to resolve relative paths (e.g., docker-compose files)
	ProjectRoot string `json:"-"`
}

type Project struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Docker struct {
	ComposeFiles []string `json:"compose_files,omitempty"`
}

type Database struct {
	Service   string   `json:"service"`
	DSN       string   `json:"dsn"`
	Allowed   []string `json:"allowed"`
	DumpsPath string   `json:"dumps_path,omitempty"`
}

type Worktrees struct {
	BasePath      string `json:"base_path"`
	DBPerWorktree bool   `json:"db_per_worktree,omitempty"`
	DBPrefix      string `json:"db_prefix,omitempty"`
}

// Phase 2: Database and Worktrees sections are not validated/used in phase 1
// Database operations and worktree commands will be implemented in phase 2

func Load(projectRoot string) (*Config, error) {
	// Search current directory and parent directories (for worktrees)
	// Get absolute path to handle relative paths like "."
	searchDir, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigInvalid,
			Message: fmt.Sprintf("failed to resolve path: %v", err),
		}
	}
	
	for {
		configPaths := []string{
			filepath.Join(searchDir, ".claude", "project.json"),
			filepath.Join(searchDir, ".haive", "config.json"),
			filepath.Join(searchDir, ".haive.json"),
		}

		var lastErr error
		var foundFiles []string

		for _, configPath := range configPaths {
			data, err := os.ReadFile(configPath)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, &types.CommandError{
					Code:    types.ErrConfigInvalid,
					Message: fmt.Sprintf("failed to read config file %s: %v", configPath, err),
				}
			}

			foundFiles = append(foundFiles, configPath)

			// First, try to parse with namespace (allows sharing .haive.json with other tools)
			var wrapper struct {
				PM *Config `json:"pm"`
			}
			var cfg Config

			if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.PM != nil && hasPMContent(wrapper.PM) {
				// Set the project root where config was found
				wrapper.PM.ProjectRoot = searchDir
				return validateConfig(wrapper.PM, searchDir)
			}

			// Fall back to direct config format (for .claude/project.json and legacy configs)
			if err := json.Unmarshal(data, &cfg); err != nil {
				// Config file exists but has wrong format - could be for a different tool
				// Continue to try other config files instead of failing immediately
				lastErr = &types.CommandError{
					Code:    types.ErrConfigInvalid,
					Message: fmt.Sprintf("invalid JSON in config file %s: %v", configPath, err),
				}
				continue
			}

			if !hasPMContent(&cfg) {
				continue
			}

			// Set the project root where config was found
			cfg.ProjectRoot = searchDir
			return validateConfig(&cfg, searchDir)
		}

		if len(foundFiles) > 0 && lastErr != nil {
			// Found config file(s) but none were valid
			return nil, lastErr
		}

		// Move to parent directory
		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			// Reached root
			break
		}
		searchDir = parent
	}

	return nil, &types.CommandError{
		Code:    types.ErrConfigMissing,
		Message: fmt.Sprintf("config file not found in %s or parent directories", projectRoot),
	}
}

func hasPMContent(cfg *Config) bool {
	return cfg.Project != nil || cfg.Database != nil || cfg.Docker != nil || cfg.Worktrees != nil
}

func validateConfig(cfg *Config, projectRoot string) (*Config, error) {

	if cfg.Worktrees != nil {
		if cfg.Worktrees.BasePath == "" {
			return nil, &types.CommandError{
				Code:    types.ErrConfigInvalid,
				Message: "worktrees.base_path is required when worktrees section is present",
			}
		}
	}

	if cfg.Database != nil {
		if cfg.Database.Service == "" {
			return nil, &types.CommandError{
				Code:    types.ErrConfigInvalid,
				Message: "database.service is required when database section is present",
			}
		}
		if cfg.Database.DSN == "" {
			return nil, &types.CommandError{
				Code:    types.ErrConfigInvalid,
				Message: "database.dsn is required when database section is present",
			}
		}
		if len(cfg.Database.Allowed) == 0 {
			return nil, &types.CommandError{
				Code:    types.ErrConfigInvalid,
				Message: "database.allowed is required when database section is present (e.g., [\"myapp\", \"myapp_*\"])",
			}
		}
		cfg.Database.DSN = ResolveEnvVars(cfg.Database.DSN, projectRoot)
		if cfg.Database.DumpsPath == "" {
			cfg.Database.DumpsPath = "var/dumps"
		}
	}

	if cfg.Worktrees != nil && cfg.Worktrees.DBPrefix == "" && cfg.Database != nil {
		parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
		if err == nil && parsedDSN.Database != "" {
			cfg.Worktrees.DBPrefix = parsedDSN.Database + "_wt_"
		}
	}

	return cfg, nil
}

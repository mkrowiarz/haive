package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/dsn"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

type Config struct {
	Docker    *Docker    `json:"docker"`
	Database  *Database  `json:"database,omitempty"`
	Worktrees *Worktrees `json:"worktrees,omitempty"`
	Serve     *Serve     `json:"serve,omitempty"`
	// ProjectRoot is the directory where the config file was found
	// Used to resolve relative paths (e.g., docker-compose files)
	ProjectRoot string `json:"-"`
}

type Docker struct {
	ComposeFiles []string `json:"compose_files,omitempty" toml:"compose_files,omitempty"`
	ProjectName  string   `json:"project_name,omitempty" toml:"project_name,omitempty"`
}

type Database struct {
	Service   string         `json:"service" toml:"service"`
	DSN       string         `json:"dsn" toml:"dsn"`
	Allowed   []string       `json:"allowed" toml:"allowed"`
	DumpsPath string         `json:"dumps_path,omitempty" toml:"dumps_path,omitempty"`
	Prefix    string         `json:"prefix,omitempty" toml:"prefix,omitempty"`
	Hooks     *DatabaseHooks `json:"hooks,omitempty" toml:"hooks,omitempty"`
}

type Worktrees struct {
	BasePath      string            `json:"base_path" toml:"base_path"`
	DBPerWorktree bool              `json:"db_per_worktree,omitempty" toml:"db_per_worktree,omitempty"`
	DBPrefix      string            `json:"db_prefix,omitempty" toml:"db_prefix,omitempty"`
	Copy          *WorktreeCopy     `json:"copy,omitempty" toml:"copy,omitempty"`
	Hooks         *WorktreeHooks    `json:"hooks,omitempty" toml:"hooks,omitempty"`
	Env           *EnvConfig        `json:"env,omitempty" toml:"env,omitempty"`
}

type WorktreeCopy struct {
	Include []string `json:"include,omitempty" toml:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty" toml:"exclude,omitempty"`
}

// Serve holds serve command configuration
type Serve struct {
	ComposeFiles []string       `json:"compose_files" toml:"compose_files"`
	Worktree     *ServeWorktree `json:"worktree,omitempty" toml:"worktree,omitempty"`
}

// ServeWorktree holds worktree-specific serve configuration
type ServeWorktree struct {
	ComposeFiles []string `json:"compose_files" toml:"compose_files"`
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
		// First, try TOML config (new standard)
		tomlPath := filepath.Join(searchDir, ".haive.toml")
		if _, err := os.Stat(tomlPath); err == nil {
			cfg, err := loadTOML(tomlPath)
			if err == nil && cfg != nil {
				cfg.ProjectRoot = searchDir
				return validateConfig(cfg, searchDir)
			}
		}

		// Fall back to JSON configs (legacy support)
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
	return cfg.Database != nil || cfg.Docker != nil || cfg.Worktrees != nil || cfg.Serve != nil
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

// loadTOML loads configuration from a TOML file
func loadTOML(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	return &cfg, nil
}

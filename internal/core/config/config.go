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
	configPath := filepath.Join(projectRoot, ".claude", "project.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &types.CommandError{
				Code:    types.ErrConfigMissing,
				Message: fmt.Sprintf("config file not found at %s", configPath),
			}
		}
		return nil, &types.CommandError{
			Code:    types.ErrConfigInvalid,
			Message: fmt.Sprintf("failed to read config file: %v", err),
		}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigInvalid,
			Message: fmt.Sprintf("invalid JSON in config file: %v", err),
		}
	}

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
				Message: "database.allowed must have at least one pattern",
			}
		}
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

	return &cfg, nil
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	ComposeFile string `json:"compose_file,omitempty"`
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

	return &cfg, nil
}

package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pmcore "github.com/mkrowiarz/mcp-symfony-stack/internal/core"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/executor"
)

func List(projectRoot string) ([]types.WorktreeInfo, error) {
	externalWorktrees, err := executor.GitWorktreeList()
	if err != nil {
		return nil, err
	}

	result := make([]types.WorktreeInfo, len(externalWorktrees))
	for i, wt := range externalWorktrees {
		result[i] = types.WorktreeInfo{
			Path:   wt.Path,
			Branch: wt.Branch,
			IsMain: wt.IsMain,
		}
	}

	return result, nil
}

func Create(projectRoot string, branch string, newBranch bool) (*types.WorktreeCreateResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Worktrees == nil {
		// Prompt user for worktrees base path
		basePath, err := promptForWorktreesPath(projectRoot)
		if err != nil {
			return nil, err
		}
		
		// Update config with worktrees section
		cfg.Worktrees = &config.Worktrees{
			BasePath: basePath,
		}
		if err := updateConfigWorktrees(projectRoot, basePath); err != nil {
			return nil, fmt.Errorf("failed to update config: %w", err)
		}
	}

	if err := pmcore.ValidateBranchName(branch); err != nil {
		return nil, err
	}

	dirName, _ := pmcore.SanitizeWorktreeName(branch)
	worktreePath := filepath.Join(cfg.Worktrees.BasePath, dirName)

	if err := pmcore.CheckPathTraversal(worktreePath, cfg.Worktrees.BasePath); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create worktree directory: %w", err)
	}

	if err := executor.GitWorktreeAdd(worktreePath, branch, newBranch); err != nil {
		return nil, err
	}

	return &types.WorktreeCreateResult{
		Path:   worktreePath,
		Branch: branch,
	}, nil
}

func Remove(projectRoot string, branch string) (*types.WorktreeRemoveResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Worktrees == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "worktrees configuration is required for worktree operations",
		}
	}

	if err := pmcore.ValidateBranchName(branch); err != nil {
		return nil, err
	}

	dirName, _ := pmcore.SanitizeWorktreeName(branch)
	worktreePath := filepath.Join(cfg.Worktrees.BasePath, dirName)

	if err := pmcore.CheckPathTraversal(worktreePath, cfg.Worktrees.BasePath); err != nil {
		return nil, err
	}

	if err := executor.GitWorktreeRemove(worktreePath); err != nil {
		return nil, err
	}

	return &types.WorktreeRemoveResult{
		Path: worktreePath,
	}, nil
}

func promptForWorktreesPath(projectRoot string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Println()
	fmt.Println("Worktrees configuration not found.")
	fmt.Println()
	fmt.Println("Worktrees allow you to work on multiple branches simultaneously")
	fmt.Println("by checking them out into separate directories.")
	fmt.Println()
	fmt.Printf("Where would you like to store worktrees? (default: ../worktrees): ")
	
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	
	input = strings.TrimSpace(input)
	if input == "" {
		input = "../worktrees"
	}
	
	// Make it relative to project root if not absolute
	if !filepath.IsAbs(input) {
		input = filepath.Join(projectRoot, input)
	}
	
	// Clean up the path
	input = filepath.Clean(input)
	
	return input, nil
}

func updateConfigWorktrees(projectRoot, basePath string) error {
	// Find the config file
	configPaths := []string{
		filepath.Join(projectRoot, ".claude", "project.json"),
		filepath.Join(projectRoot, ".haive", "config.json"),
		filepath.Join(projectRoot, ".haive.json"),
	}
	
	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}
	
	if configPath == "" {
		return fmt.Errorf("no config file found")
	}
	
	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	
	// Parse config
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	
	// Check if it's namespaced
	if pmConfig, ok := cfg["pm"].(map[string]interface{}); ok {
		// Update namespaced config
		pmConfig["worktrees"] = map[string]string{
			"base_path": basePath,
		}
		cfg["pm"] = pmConfig
	} else {
		// Update direct config
		cfg["worktrees"] = map[string]string{
			"base_path": basePath,
		}
	}
	
	// Write back
	data, err = json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	
	fmt.Printf("✓ Updated %s with worktrees configuration\n", configPath)
	return nil
}

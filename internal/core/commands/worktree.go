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
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/hooks"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
	worktreepkg "github.com/mkrowiarz/mcp-symfony-stack/internal/core/worktree"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/executor"
)

// isTerminal checks if stdin is a terminal (not piped or redirected)
func isTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fileInfo.Mode()&os.ModeCharDevice != 0
}

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

// PreValidateWorktree checks if worktree creation prerequisites are met
func PreValidateWorktree(cfg *config.HaiveConfig, branch string) error {
	// Check branch name is valid
	if branch == "" {
		return &types.CommandError{
			Code:    types.ErrInvalidWorktree,
			Message: "branch name cannot be empty",
		}
	}

	if err := pmcore.ValidateBranchName(branch); err != nil {
		return err
	}

	// Check for path traversal
	dirName, _ := pmcore.SanitizeWorktreeName(branch)
	worktreePath := filepath.Join(cfg.Worktree.BasePath, dirName)
	if err := pmcore.CheckPathTraversal(worktreePath, cfg.Worktree.BasePath); err != nil {
		return err
	}

	// Check if worktree already exists (git or directory)
	gitWorktrees, err := executor.GitWorktreeList()
	if err != nil {
		return err
	}

	for _, wt := range gitWorktrees {
		if wt.Branch == branch || strings.Contains(wt.Path, dirName) {
			return &types.CommandError{
				Code:    types.ErrInvalidWorktree,
				Message: fmt.Sprintf("worktree for branch %s already exists", branch),
			}
		}
	}

	if _, err := os.Stat(worktreePath); err == nil {
		return &types.CommandError{
			Code:    types.ErrInvalidWorktree,
			Message: fmt.Sprintf("directory %s already exists", worktreePath),
		}
	}

	// If worktree.env is configured, validate database section exists
	if cfg.Worktree.Env != nil && cfg.Database == nil {
		return &types.CommandError{
			Code:    types.ErrConfigInvalid,
			Message: "worktree.env requires database configuration",
		}
	}

	return nil
}

func Create(projectRoot string, branch string, newBranch bool) (*types.WorktreeCreateResult, error) {
	cfg, err := config.LoadHaive(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Worktree == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "worktree not configured. Add worktree.base_path to your config first",
		}
	}

	// Pre-validation
	if err := PreValidateWorktree(cfg, branch); err != nil {
		return nil, err
	}

	dirName, _ := pmcore.SanitizeWorktreeName(branch)
	worktreePath := filepath.Join(cfg.Worktree.BasePath, dirName)

	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create worktree directory: %w", err)
	}

	if err := executor.GitWorktreeAdd(worktreePath, branch, newBranch); err != nil {
		return nil, err
	}

	// 1. Copy files
	if cfg.Worktree.Copy != nil {
		if err := worktreepkg.CopyFiles(cfg.ProjectRoot, worktreePath, cfg.Worktree.Copy); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to copy worktree files: %v\n", err)
		}
	}

	// 2. Run postCreate hooks
	if cfg.Worktree.Hooks != nil && len(cfg.Worktree.Hooks.PostCreate) > 0 {
		hookExec := hooks.NewExecutor(cfg.ProjectRoot)
		hookCtx := &hooks.HookContext{
			RepoRoot:     cfg.ProjectRoot,
			ProjectName:  cfg.Project.Name,
			WorktreePath: worktreePath,
			WorktreeName: dirName,
			Branch:       branch,
		}

		if err := hookExec.ExecuteHooks(cfg.Worktree.Hooks.PostCreate, hookCtx, worktreePath, false); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: postCreate hook failed: %v\n", err)
		}
	}

	return &types.WorktreeCreateResult{
		Path:   worktreePath,
		Branch: branch,
	}, nil
}

func Remove(projectRoot string, branch string) (*types.WorktreeRemoveResult, error) {
	cfg, err := config.LoadHaive(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Worktree == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "worktree configuration is required for worktree operations",
		}
	}

	if err := pmcore.ValidateBranchName(branch); err != nil {
		return nil, err
	}

	dirName, _ := pmcore.SanitizeWorktreeName(branch)
	worktreePath := filepath.Join(cfg.Worktree.BasePath, dirName)

	if err := pmcore.CheckPathTraversal(worktreePath, cfg.Worktree.BasePath); err != nil {
		return nil, err
	}

	// Run preRemove hooks (can abort removal)
	if cfg.Worktree.Hooks != nil && len(cfg.Worktree.Hooks.PreRemove) > 0 {
		hookExec := hooks.NewExecutor(cfg.ProjectRoot)
		hookCtx := &hooks.HookContext{
			RepoRoot:     cfg.ProjectRoot,
			ProjectName:  cfg.Project.Name,
			WorktreePath: worktreePath,
			WorktreeName: dirName,
			Branch:       branch,
		}

		if err := hookExec.ExecuteHooks(cfg.Worktree.Hooks.PreRemove, hookCtx, cfg.ProjectRoot, true); err != nil {
			return nil, fmt.Errorf("preRemove hook prevented removal: %w", err)
		}
	}

	if err := executor.GitWorktreeRemove(worktreePath); err != nil {
		return nil, err
	}

	// Run postRemove hooks (fire and forget)
	if cfg.Worktree.Hooks != nil && len(cfg.Worktree.Hooks.PostRemove) > 0 {
		hookExec := hooks.NewExecutor(cfg.ProjectRoot)
		hookCtx := &hooks.HookContext{
			RepoRoot:     cfg.ProjectRoot,
			ProjectName:  cfg.Project.Name,
			WorktreeName: dirName,
			Branch:       branch,
		}

		if err := hookExec.ExecuteHooks(cfg.Worktree.Hooks.PostRemove, hookCtx, cfg.ProjectRoot, false); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: postRemove hook failed: %v\n", err)
		}
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
	fmt.Printf("Where would you like to store worktrees? (default: .worktrees): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		input = ".worktrees"
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

	fmt.Printf("Updated %s with worktrees configuration\n", configPath)

	// Add to .gitignore if path is inside project root
	if strings.HasPrefix(basePath, projectRoot) || !filepath.IsAbs(basePath) {
		relPath, _ := filepath.Rel(projectRoot, basePath)
		if relPath == "" {
			relPath = basePath
		}
		if err := addToGitignore(projectRoot, relPath); err == nil {
			fmt.Printf("Added %s to .gitignore\n", relPath)
		}
	}

	return nil
}

func addToGitignore(projectRoot, path string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	// Normalize path for .gitignore (use forward slashes)
	path = strings.ReplaceAll(path, "\\", "/")

	// Read existing .gitignore if it exists
	content := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		content = string(data)
		// Check if already present
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == path || line == "/"+path || line == path+"/" {
				return nil // Already present
			}
		}
	}

	// Add new line
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "# Worktrees directory\n" + path + "/\n"

	return os.WriteFile(gitignorePath, []byte(content), 0644)
}

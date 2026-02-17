package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

type ServeResult struct {
	Branch       string `json:"branch"`
	WorktreePath string `json:"worktree_path"`
	Hostname     string `json:"hostname"`
	URL          string `json:"url"`
}

// Serve starts containers using the configured compose files
func Serve(projectRoot string) (*ServeResult, error) {
	// 1. Detect if we're in a worktree
	branch, isWorktree, err := detectWorktree(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to detect worktree: %w", err)
	}

	if !isWorktree {
		return nil, &types.CommandError{
			Code:    types.ErrInvalidWorktree,
			Message: "not in a worktree directory",
		}
	}

	// 2. Load config and check for [serve] section
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: fmt.Sprintf("failed to load config: %v", err),
		}
	}

	// 3. Determine which compose files to use
	composeFiles := getComposeFiles(cfg, true)
	if len(composeFiles) == 0 {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "[serve] section not configured or compose_files is empty. Add [serve] with compose_files to your config.",
		}
	}

	// 4. Verify all compose files exist
	for _, f := range composeFiles {
		composePath := filepath.Join(projectRoot, f)
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			return nil, &types.CommandError{
				Code:    types.ErrConfigInvalid,
				Message: fmt.Sprintf("compose file not found: %s", f),
			}
		}
	}

	// 5. Generate unique project name
	projectName := generateProjectName(cfg, branch)

	// 6. Start containers
	if err := startContainers(projectRoot, projectName, composeFiles); err != nil {
		return nil, fmt.Errorf("failed to start containers: %w", err)
	}

	// 6. Build result with hostname
	hostname := fmt.Sprintf("%s-app.orb.local", projectName)

	return &ServeResult{
		Branch:       branch,
		WorktreePath: projectRoot,
		Hostname:     hostname,
		URL:          fmt.Sprintf("http://%s", hostname),
	}, nil
}

// Stop stops containers using the configured compose files
func Stop(projectRoot string) error {
	// Detect worktree
	branch, isWorktree, err := detectWorktree(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to detect worktree: %w", err)
	}

	if !isWorktree {
		return &types.CommandError{
			Code:    types.ErrInvalidWorktree,
			Message: "not in a worktree directory",
		}
	}

	// Load config and check for [serve] section
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: fmt.Sprintf("failed to load config: %v", err),
		}
	}

	// Determine which compose files to use
	composeFiles := getComposeFiles(cfg, true)
	if len(composeFiles) == 0 {
		return &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "[serve] section not configured or compose_files is empty. Add [serve] with compose_files to your config.",
		}
	}

	// Generate project name
	projectName := generateProjectName(cfg, branch)

	// Build docker compose command with all compose files
	args := []string{"compose"}
	for _, f := range composeFiles {
		args = append(args, "-f", f)
	}
	args = append(args, "-p", projectName, "down")

	cmd := exec.Command("docker", args...)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	return nil
}

// detectWorktree checks if the current directory is a worktree and returns the branch name
func detectWorktree(projectRoot string) (string, bool, error) {
	// Check if .git is a file (worktrees have .git file pointing to main repo)
	gitPath := filepath.Join(projectRoot, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", false, err
	}

	// If .git is a directory, we're in the main repo
	if info.IsDir() {
		return "", false, nil
	}

	// Get branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	if err != nil {
		return "", false, fmt.Errorf("failed to get branch name: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	return branch, true, nil
}

// generateProjectName creates a unique docker compose project name for the worktree
func generateProjectName(cfg *config.Config, branch string) string {
	// Sanitize branch name for docker project name
	sanitized := strings.ReplaceAll(branch, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, "_", "-")
	sanitized = strings.ToLower(sanitized)

	// Use project name from config or default
	baseProject := "app"
	if cfg != nil && cfg.Docker != nil && cfg.Docker.ProjectName != "" {
		baseProject = cfg.Docker.ProjectName
	}

	return fmt.Sprintf("%s-wt-%s", baseProject, sanitized)
}

// startContainers starts containers with docker compose using configured files
func startContainers(projectRoot, projectName string, composeFiles []string) error {
	args := []string{"compose"}
	for _, f := range composeFiles {
		args = append(args, "-f", f)
	}
	args = append(args, "-p", projectName, "up", "-d")

	cmd := exec.Command("docker", args...)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// getComposeFiles returns the compose files to use based on context
// If inWorktree is true and [serve.worktree] is configured, use worktree-specific files
func getComposeFiles(cfg *config.Config, inWorktree bool) []string {
	if cfg.Serve == nil {
		return nil
	}

	// If in worktree and worktree-specific config exists, use it
	if inWorktree && cfg.Serve.Worktree != nil && len(cfg.Serve.Worktree.ComposeFiles) > 0 {
		return cfg.Serve.Worktree.ComposeFiles
	}

	// Otherwise use default serve config
	return cfg.Serve.ComposeFiles
}

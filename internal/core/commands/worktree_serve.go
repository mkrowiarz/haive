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
	ProjectName  string `json:"project_name"`
	Branch       string `json:"branch"`
	WorktreePath string `json:"worktree_path"`
	Hostname     string `json:"hostname"`
	URL          string `json:"url"`
}

// Serve starts the app container for a worktree using compose.worktree.yaml
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

	// 2. Check for compose.worktree.yaml
	worktreeComposeFile := filepath.Join(projectRoot, "compose.worktree.yaml")
	if _, err := os.Stat(worktreeComposeFile); os.IsNotExist(err) {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "compose.worktree.yaml not found. See README.md for setup instructions.",
		}
	}

	// 3. Find main project's docker-compose.yml
	mainComposeFile, err := findMainComposeFile(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find main docker-compose.yml: %w", err)
	}

	// 4. Generate unique project name
	cfg, _ := config.Load(projectRoot) // Ignore error, use defaults
	projectName := generateProjectName(cfg, branch)

	// 5. Start the app container
	if err := startAppContainer(projectRoot, projectName, mainComposeFile, worktreeComposeFile); err != nil {
		return nil, fmt.Errorf("failed to start app container: %w", err)
	}

	// 6. Build result with OrbStack hostname
	hostname := fmt.Sprintf("%s-app.orb.local", projectName)

	return &ServeResult{
		ProjectName:  projectName,
		Branch:       branch,
		WorktreePath: projectRoot,
		Hostname:     hostname,
		URL:          fmt.Sprintf("http://%s", hostname),
	}, nil
}

// Stop stops the app container for a worktree
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

	// Load config for project name generation
	cfg, _ := config.Load(projectRoot) // Ignore error, use defaults
	projectName := generateProjectName(cfg, branch)

	// Stop and remove containers
	cmd := exec.Command("docker", "compose", "-p", projectName, "down")
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

// findMainComposeFile finds the docker-compose.yml in the main project
func findMainComposeFile(projectRoot string) (string, error) {
	mainRoot, err := findProjectRoot(projectRoot)
	if err != nil {
		return "", err
	}

	composeFile := filepath.Join(mainRoot, "docker-compose.yml")
	if _, err := os.Stat(composeFile); err != nil {
		return "", fmt.Errorf("docker-compose.yml not found in main project: %w", err)
	}

	return composeFile, nil
}

// findProjectRoot finds the main project root from a worktree
func findProjectRoot(worktreeRoot string) (string, error) {
	// Read .git file to find main repo location
	gitFile := filepath.Join(worktreeRoot, ".git")
	content, err := os.ReadFile(gitFile)
	if err != nil {
		return "", err
	}

	// .git file contains: gitdir: /path/to/main/.git/worktrees/branch-name
	line := strings.TrimSpace(string(content))
	if !strings.HasPrefix(line, "gitdir: ") {
		return "", fmt.Errorf("invalid .git file format")
	}

	gitDir := strings.TrimPrefix(line, "gitdir: ")
	// gitDir is like: /path/to/main/.git/worktrees/branch-name
	// We need: /path/to/main
	mainGitDir := filepath.Dir(filepath.Dir(gitDir)) // Remove /worktrees/branch-name
	mainRoot := filepath.Dir(mainGitDir)              // Remove /.git

	return mainRoot, nil
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

// startAppContainer starts the app service with docker compose
func startAppContainer(projectRoot, projectName, mainComposeFile, worktreeComposeFile string) error {
	cmd := exec.Command("docker", "compose",
		"-p", projectName,
		"-f", mainComposeFile,
		"-f", worktreeComposeFile,
		"up", "-d", "app")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

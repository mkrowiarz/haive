package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HookContext provides environment variables and working directory for hooks
type HookContext struct {
	// Common
	RepoRoot    string
	ProjectName string

	// Worktree-specific
	WorktreePath string
	WorktreeName string
	Branch       string

	// Database-specific
	DatabaseName   string
	DatabaseURL    string
	SourceDatabase string
	TargetDatabase string
}

// ToEnv returns the context as environment variable slice
func (c *HookContext) ToEnv() []string {
	env := os.Environ()

	// Common
	env = append(env, fmt.Sprintf("REPO_ROOT=%s", c.RepoRoot))
	env = append(env, fmt.Sprintf("PROJECT_NAME=%s", c.ProjectName))

	// Worktree
	if c.WorktreePath != "" {
		env = append(env, fmt.Sprintf("WORKTREE_PATH=%s", c.WorktreePath))
	}
	if c.WorktreeName != "" {
		env = append(env, fmt.Sprintf("WORKTREE_NAME=%s", c.WorktreeName))
	}
	if c.Branch != "" {
		env = append(env, fmt.Sprintf("BRANCH=%s", c.Branch))
	}

	// Database
	if c.DatabaseName != "" {
		env = append(env, fmt.Sprintf("DATABASE_NAME=%s", c.DatabaseName))
	}
	if c.DatabaseURL != "" {
		env = append(env, fmt.Sprintf("DATABASE_URL=%s", c.DatabaseURL))
	}
	if c.SourceDatabase != "" {
		env = append(env, fmt.Sprintf("SOURCE_DATABASE=%s", c.SourceDatabase))
	}
	if c.TargetDatabase != "" {
		env = append(env, fmt.Sprintf("TARGET_DATABASE=%s", c.TargetDatabase))
	}

	return env
}

// Executor runs hook commands
type Executor struct {
	ProjectRoot string
}

// NewExecutor creates a new hook executor
func NewExecutor(projectRoot string) *Executor {
	return &Executor{ProjectRoot: projectRoot}
}

// ExecuteHooks runs a list of hooks and returns error if any fail
// For pre-hooks, non-zero exit stops execution and returns error
// For post-hooks, non-zero exit is logged but doesn't stop
func (e *Executor) ExecuteHooks(hooks []string, ctx *HookContext, workingDir string, isPre bool) error {
	for _, hook := range hooks {
		if err := e.executeHook(hook, ctx, workingDir, isPre); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) executeHook(hook string, ctx *HookContext, workingDir string, isPre bool) error {
	// Check if hook is a script file or command
	var cmd *exec.Cmd

	if isScriptFile(hook) {
		scriptPath := hook
		if !filepath.IsAbs(scriptPath) {
			scriptPath = filepath.Join(e.ProjectRoot, scriptPath)
		}

		// Check if script exists
		if _, err := os.Stat(scriptPath); err != nil {
			if isPre {
				return fmt.Errorf("pre-hook script not found: %s", scriptPath)
			}
			fmt.Fprintf(os.Stderr, "Warning: post-hook script not found: %s\n", scriptPath)
			return nil
		}

		cmd = exec.Command(scriptPath)
	} else {
		// Run as shell command
		cmd = exec.Command("sh", "-c", hook)
	}

	cmd.Dir = workingDir
	cmd.Env = ctx.ToEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if isPre {
			return fmt.Errorf("pre-hook failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Warning: post-hook failed: %v\n", err)
	}

	return nil
}

func isScriptFile(hook string) bool {
	// If it contains a path separator and the file exists, treat as script
	if strings.Contains(hook, "/") || strings.Contains(hook, "\\") {
		return true
	}
	return false
}

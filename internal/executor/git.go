package executor

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

type GitExecutor struct{}

func NewGitExecutor() Executor {
	return &GitExecutor{}
}

type worktreeOutput struct {
	Path   string
	Head   string
	Branch string
}

func (g *GitExecutor) GitWorktreeList() ([]types.WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git worktree list failed: %w", err)
	}

	cmdPath := exec.Command("git", "rev-parse", "--show-toplevel")
	toplevelOutput, err := cmdPath.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git rev-parse failed: %w", err)
	}
	toplevelPath := strings.TrimSpace(string(toplevelOutput))

	return parseWorktreeListOutput(string(output), toplevelPath)
}

func (g *GitExecutor) GitWorktreeAdd(path, branch string, newBranch bool) error {
	args := []string{"worktree", "add"}
	if newBranch {
		args = append(args, "-b", branch)
	} else {
		args = append(args, branch)
	}
	args = append(args, path)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (g *GitExecutor) GitWorktreeRemove(path string) error {
	cmd := exec.Command("git", "worktree", "remove", path, "--force")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git worktree remove failed: %w", err)
	}

	return nil
}

func (g *GitExecutor) ReadFile(path string) ([]byte, error) {
	panic("GitExecutor.ReadFile not implemented - use FileExecutor for file operations")
}

func (g *GitExecutor) WriteFile(path string, data []byte) error {
	panic("GitExecutor.WriteFile not implemented - use FileExecutor for file operations")
}

func (g *GitExecutor) FileExists(path string) bool {
	panic("GitExecutor.FileExists not implemented - use FileExecutor for file operations")
}

func parseWorktreeListOutput(output, toplevelPath string) ([]types.WorktreeInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var worktrees []types.WorktreeInfo

	var current *worktreeOutput

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			if current != nil {
				worktrees = append(worktrees, types.WorktreeInfo{
					Path:   current.Path,
					Branch: strings.TrimPrefix(current.Branch, "refs/heads/"),
					IsMain: current.Path == toplevelPath,
				})
			}
			current = nil
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, types.WorktreeInfo{
					Path:   current.Path,
					Branch: strings.TrimPrefix(current.Branch, "refs/heads/"),
					IsMain: current.Path == toplevelPath,
				})
			}
			current = &worktreeOutput{
				Path: strings.TrimSpace(strings.TrimPrefix(line, "worktree ")),
			}
		} else if strings.HasPrefix(line, "HEAD ") {
			if current != nil {
				current.Head = strings.TrimSpace(strings.TrimPrefix(line, "HEAD "))
			}
		} else if strings.HasPrefix(line, "branch ") {
			if current != nil {
				current.Branch = strings.TrimSpace(strings.TrimPrefix(line, "branch "))
			}
		} else if line == "detached" {
			if current != nil {
				current.Branch = "detached"
			}
		}
	}

	if current != nil {
		worktrees = append(worktrees, types.WorktreeInfo{
			Path:   current.Path,
			Branch: strings.TrimPrefix(current.Branch, "refs/heads/"),
			IsMain: current.Path == toplevelPath,
		})
	}

	return worktrees, nil
}

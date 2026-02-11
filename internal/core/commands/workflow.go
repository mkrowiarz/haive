package commands

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

func CreateIsolatedWorktree(projectRoot, branch, newBranch, newDB string) (*types.WorkflowCreateResult, error) {
	_ = newDB // unused - controlled by config.Worktrees.DBPerWorktree

	newBranchBool, _ := strconv.ParseBool(newBranch)
	result, err := Create(projectRoot, branch, newBranchBool)
	if err != nil {
		return nil, err
	}

	workflowResult := &types.WorkflowCreateResult{
		WorktreePath:   result.Path,
		WorktreeBranch: result.Branch,
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return workflowResult, nil
	}

	if cfg.Worktrees == nil || !cfg.Worktrees.DBPerWorktree || cfg.Database == nil {
		return workflowResult, nil
	}

	_, dbName := core.SanitizeWorktreeName(branch)
	targetDB := cfg.Worktrees.DBPrefix + dbName

	cloneResult, err := CloneDB(projectRoot, "", targetDB)
	if err != nil {
		return workflowResult, fmt.Errorf("worktree created but database clone failed: %w", err)
	}

	envPath := filepath.Join(result.Path, ".env.local")
	newDSN, err := replaceDSNDatabase(cfg.Database.DSN, cloneResult.Source, cloneResult.Target)
	if err != nil {
		return workflowResult, fmt.Errorf("worktree and DB created but DSN replacement failed: %w", err)
	}

	if err := os.WriteFile(envPath, []byte("DATABASE_URL="+newDSN+"\n"), 0644); err != nil {
		return workflowResult, fmt.Errorf("worktree and DB created but .env.local patch failed: %w", err)
	}

	workflowResult.DatabaseName = cloneResult.Target
	workflowResult.ClonedFrom = cloneResult.Source

	return workflowResult, nil
}

func RemoveIsolatedWorktree(projectRoot, branch string, dropDB bool) (*types.WorkflowRemoveResult, error) {
	result, err := Remove(projectRoot, branch)
	if err != nil {
		return nil, err
	}

	workflowResult := &types.WorkflowRemoveResult{
		WorktreePath: result.Path,
	}

	if !dropDB {
		return workflowResult, nil
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return workflowResult, nil
	}

	if cfg.Worktrees == nil || cfg.Database == nil {
		return workflowResult, nil
	}

	_, dbName := core.SanitizeWorktreeName(branch)
	targetDB := cfg.Worktrees.DBPrefix + dbName

	_, err = DropDB(projectRoot, targetDB)
	if err != nil {
		if cmdErr, ok := err.(*types.CommandError); ok && cmdErr.Code == types.ErrDbNotAllowed {
			return workflowResult, nil
		}
		return workflowResult, fmt.Errorf("worktree removed but database drop failed: %w", err)
	}

	workflowResult.DatabaseName = targetDB

	return workflowResult, nil
}

func replaceDSNDatabase(dsnString, oldDB, newDB string) (string, error) {
	u, err := url.Parse(dsnString)
	if err != nil {
		return "", fmt.Errorf("failed to parse DSN: %w", err)
	}

	if u.Path == "" || u.Path == "/" {
		return "", fmt.Errorf("DSN has no database path")
	}

	u.Path = "/" + newDB
	return u.String(), nil
}

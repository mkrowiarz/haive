package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/commands"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

func registerWorktreeTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("worktree.list",
		mcp.WithDescription("List all git worktrees"),
		mcp.WithString("project_root", mcp.Description("Project root directory (optional, defaults to cwd)")),
	), handleWorktreeList)

	s.AddTool(mcp.NewTool("worktree.create",
		mcp.WithDescription("Create a new git worktree"),
		mcp.WithString("project_root", mcp.Description("Project root directory (optional, defaults to cwd)")),
		mcp.WithString("branch", mcp.Required(), mcp.Description("Branch name")),
		mcp.WithBoolean("new_branch", mcp.Description("Create new branch (default false)")),
	), handleWorktreeCreate)

	s.AddTool(mcp.NewTool("worktree.remove",
		mcp.WithDescription("Remove a git worktree (destructive)"),
		mcp.WithString("project_root", mcp.Description("Project root directory (optional, defaults to cwd)")),
		mcp.WithString("branch", mcp.Required(), mcp.Description("Branch name")),
		mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("Must be true to confirm destructive operation")),
	), handleWorktreeRemove)
}

func handleWorktreeList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot := getProjectRoot(request)
	result, err := commands.List(projectRoot)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleWorktreeCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot := getProjectRoot(request)
	args := request.GetArguments()
	branch := args["branch"].(string)

	newBranch := false
	if v, ok := args["new_branch"].(bool); ok {
		newBranch = v
	}

	result, err := commands.Create(projectRoot, branch, newBranch)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleWorktreeRemove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	confirm, _ := args["confirm"].(bool)
	if !confirm {
		return nil, toMCPError(&types.CommandError{
			Code:    types.ErrConfigInvalid,
			Message: "confirm must be true to remove worktree",
		})
	}

	projectRoot := getProjectRoot(request)
	branch := args["branch"].(string)

	result, err := commands.Remove(projectRoot, branch)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

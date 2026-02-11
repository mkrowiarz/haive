package mcp

import (
	"context"
	"encoding/json"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/commands"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

func registerWorkflowTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("workflow.create",
		mcp.WithDescription("Create isolated worktree with database (if db_per_worktree enabled)"),
		mcp.WithString("branch", mcp.Required(), mcp.Description("Branch name")),
		mcp.WithBoolean("new_branch", mcp.Description("Create new branch (default false)")),
	), handleWorkflowCreate)

	s.AddTool(mcp.NewTool("workflow.remove",
		mcp.WithDescription("Remove worktree and optionally drop database (destructive)"),
		mcp.WithString("branch", mcp.Required(), mcp.Description("Branch name")),
		mcp.WithBoolean("drop_db", mcp.Description("Drop associated database (default true)")),
		mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("Must be true to confirm destructive operation")),
	), handleWorkflowRemove)
}

func handleWorkflowCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, _ := os.Getwd()
	args := request.GetArguments()
	branch := args["branch"].(string)

	newBranch := "false"
	if v, ok := args["new_branch"].(bool); ok && v {
		newBranch = "true"
	}

	result, err := commands.CreateIsolatedWorktree(projectRoot, branch, newBranch, "")
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleWorkflowRemove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	confirm, _ := args["confirm"].(bool)
	if !confirm {
		return nil, toMCPError(&types.CommandError{
			Code:    types.ErrConfigInvalid,
			Message: "confirm must be true to remove worktree",
		})
	}

	projectRoot, _ := os.Getwd()
	branch := args["branch"].(string)

	dropDB := true
	if v, ok := args["drop_db"].(bool); ok {
		dropDB = v
	}

	result, err := commands.RemoveIsolatedWorktree(projectRoot, branch, dropDB)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

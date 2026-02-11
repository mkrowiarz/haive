package mcp

import (
	"context"
	"encoding/json"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/commands"
)

func registerProjectTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("project.info",
		mcp.WithDescription("Get project configuration and status"),
	), handleProjectInfo)

	s.AddTool(mcp.NewTool("project.init",
		mcp.WithDescription("Generate suggested project configuration"),
	), handleProjectInit)
}

func handleProjectInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, _ := os.Getwd()
	result, err := commands.Info(projectRoot)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleProjectInit(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, _ := os.Getwd()
	result, err := commands.Init(projectRoot)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

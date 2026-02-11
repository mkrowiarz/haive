package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

func Run() error {
	s := server.NewMCPServer(
		"mcp-project-manager",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	registerProjectTools(s)
	registerDatabaseTools(s)
	registerWorktreeTools(s)
	registerWorkflowTools(s)

	return server.ServeStdio(s)
}

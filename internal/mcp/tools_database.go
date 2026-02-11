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

func registerDatabaseTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("db.list",
		mcp.WithDescription("List all databases in the container"),
	), handleDbList)

	s.AddTool(mcp.NewTool("db.dump",
		mcp.WithDescription("Dump a database to a SQL file"),
		mcp.WithString("database", mcp.Description("Database name (optional, defaults to DSN database)")),
		mcp.WithArray("tables", mcp.Description("Specific tables to dump (optional)")),
	), handleDbDump)

	s.AddTool(mcp.NewTool("db.import",
		mcp.WithDescription("Import a SQL file into a database"),
		mcp.WithString("database", mcp.Required(), mcp.Description("Target database name")),
		mcp.WithString("sql_path", mcp.Required(), mcp.Description("Path to SQL file")),
	), handleDbImport)

	s.AddTool(mcp.NewTool("db.create",
		mcp.WithDescription("Create a new empty database"),
		mcp.WithString("database", mcp.Required(), mcp.Description("Database name")),
	), handleDbCreate)

	s.AddTool(mcp.NewTool("db.drop",
		mcp.WithDescription("Drop a database (destructive)"),
		mcp.WithString("database", mcp.Required(), mcp.Description("Database name")),
		mcp.WithBoolean("confirm", mcp.Required(), mcp.Description("Must be true to confirm destructive operation")),
	), handleDbDrop)

	s.AddTool(mcp.NewTool("db.clone",
		mcp.WithDescription("Clone a database (dump + create + import)"),
		mcp.WithString("source", mcp.Description("Source database (optional, defaults to DSN database)")),
		mcp.WithString("target", mcp.Required(), mcp.Description("Target database name")),
	), handleDbClone)

	s.AddTool(mcp.NewTool("db.dumps",
		mcp.WithDescription("List available SQL dump files"),
	), handleDbDumps)
}

func handleDbList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, _ := os.Getwd()
	result, err := commands.ListDBs(projectRoot)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDbDump(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, _ := os.Getwd()
	args := request.GetArguments()

	database := ""
	if v, ok := args["database"].(string); ok {
		database = v
	}

	var tables []string
	if v, ok := args["tables"].([]interface{}); ok {
		for _, t := range v {
			if s, ok := t.(string); ok {
				tables = append(tables, s)
			}
		}
	}

	result, err := commands.Dump(projectRoot, database, tables)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDbImport(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, _ := os.Getwd()
	args := request.GetArguments()

	database := args["database"].(string)
	sqlPath := args["sql_path"].(string)

	result, err := commands.ImportDB(projectRoot, database, sqlPath)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDbCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, _ := os.Getwd()
	args := request.GetArguments()
	database := args["database"].(string)

	result, err := commands.CreateDB(projectRoot, database)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDbDrop(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	confirm, _ := args["confirm"].(bool)
	if !confirm {
		return nil, toMCPError(&types.CommandError{
			Code:    types.ErrConfigInvalid,
			Message: "confirm must be true to drop database",
		})
	}

	projectRoot, _ := os.Getwd()
	database := args["database"].(string)

	result, err := commands.DropDB(projectRoot, database)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDbClone(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, _ := os.Getwd()
	args := request.GetArguments()

	source := ""
	if v, ok := args["source"].(string); ok {
		source = v
	}
	target := args["target"].(string)

	result, err := commands.CloneDB(projectRoot, source, target)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func handleDbDumps(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectRoot, _ := os.Getwd()
	result, err := commands.ListDumps(projectRoot)
	if err != nil {
		return nil, toMCPError(err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

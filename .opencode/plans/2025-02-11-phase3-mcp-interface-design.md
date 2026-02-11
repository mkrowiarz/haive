# Phase 3: MCP Interface - Design

> **Status:** Approved
> **Date:** 2025-02-11
> **Scope:** MCP server with all Phase 1-2C commands exposed as tools

---

## Overview

Phase 3 exposes all implemented commands as MCP tools via stdio transport. The MCP server runs when `pm --mcp` is invoked, using the mcp-go SDK for protocol handling.

**Technology:** `github.com/mark3labs/mcp-go` SDK with stdio transport

---

## Architecture

### Entry Point

```go
// cmd/pm/main.go
if *mcpFlag {
    mcpServer.Run()
    return
}
```

### MCP Server Package

```go
// internal/mcp/server.go
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
```

### Project Root Resolution

Tools use `os.Getwd()` as projectRoot. MCP clients set cwd when registering the server.

---

## Error Handling

### Error Code Mapping

Our `ErrCode` strings map to JSON-RPC -32000 range (reserved for implementation-defined server errors):

| ErrCode | JSON-RPC Code | Description |
|---------|---------------|-------------|
| `ErrConfigMissing` | -32001 | Config file not found |
| `ErrConfigInvalid` | -32002 | Invalid config structure |
| `ErrInvalidName` | -32003 | Invalid branch/database name |
| `ErrPathTraversal` | -32004 | Path traversal attempt detected |
| `ErrDbNotAllowed` | -32005 | Database not in allowed list |
| `ErrDbIsDefault` | -32006 | Cannot drop default database |
| `ErrFileNotFound` | -32007 | SQL file not found |

### Error Response Structure

```json
{
  "code": -32005,
  "message": "database 'other_db' is not in allowed list",
  "data": {
    "code": "DB_NOT_ALLOWED",
    "database": "other_db"
  }
}
```

---

## Tool Definitions

### Project Tools (2)

| Tool | Description | Parameters |
|------|-------------|------------|
| `project.info` | Get project config and status | none |
| `project.init` | Generate suggested config | none |

### Database Tools (7)

| Tool | Description | Parameters |
|------|-------------|------------|
| `db.list` | List databases | none |
| `db.dump` | Dump database to file | `database: string`, `tables?: string[]` |
| `db.import` | Import SQL file | `database: string`, `sql_path: string` |
| `db.create` | Create empty database | `database: string` |
| `db.drop` | Drop database | `database: string`, `confirm: boolean` |
| `db.clone` | Clone database | `source?: string`, `target: string` |
| `db.dumps` | List dump files | none |

### Worktree Tools (3)

| Tool | Description | Parameters |
|------|-------------|------------|
| `worktree.list` | List worktrees | none |
| `worktree.create` | Create worktree | `branch: string`, `new_branch?: boolean` |
| `worktree.remove` | Remove worktree | `branch: string`, `confirm: boolean` |

### Workflow Tools (2)

| Tool | Description | Parameters |
|------|-------------|------------|
| `workflow.create` | Create worktree with DB | `branch: string`, `new_branch?: boolean` |
| `workflow.remove` | Remove worktree with DB | `branch: string`, `drop_db?: boolean`, `confirm: boolean` |

**Total: 14 tools**

---

## Safety: Confirmation Parameters

Destructive operations require explicit confirmation:

| Tool | Confirmation Param |
|------|-------------------|
| `db.drop` | `confirm: boolean` (required, must be `true`) |
| `worktree.remove` | `confirm: boolean` (required, must be `true`) |
| `workflow.remove` | `confirm: boolean` (required, must be `true`) |

Handler validates `confirm === true` before executing. Returns error if missing or false.

---

## Implementation Approach

### Files to Create

| File | Purpose |
|------|---------|
| `internal/mcp/server.go` | MCP server setup, tool registration, main Run() |
| `internal/mcp/tools_project.go` | project.info, project.init handlers |
| `internal/mcp/tools_database.go` | db.* handlers |
| `internal/mcp/tools_worktree.go` | worktree.* handlers |
| `internal/mcp/tools_workflow.go` | workflow.* handlers |
| `internal/mcp/errors.go` | Error code mapping, toMCPError() function |
| `internal/mcp/server_test.go` | Unit tests |

### Files to Modify

| File | Changes |
|------|---------|
| `cmd/pm/main.go` | Add `--mcp` flag, call `mcp.Run()` |
| `go.mod` | Add `github.com/mark3labs/mcp-go` |

### Tool Handler Pattern

```go
func handleDbList(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
    projectRoot, _ := os.Getwd()
    result, err := commands.ListDBs(projectRoot)
    if err != nil {
        return nil, toMCPError(err)
    }
    return mcp.NewToolResultText(toJSON(result)), nil
}
```

---

## Testing Strategy

### Unit Tests

1. **Error mapping** - Verify each `ErrCode` maps to correct JSON-RPC code
2. **Argument parsing** - Test extraction of params from arguments map
3. **Result formatting** - Verify JSON serialization of result types

### Manual Testing

```bash
# List tools
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./pm --mcp

# Call a tool
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"db.list","arguments":{}}}' | ./pm --mcp
```

---

## Success Criteria

Phase 3 is complete when:
- ✅ All 14 tools registered and callable
- ✅ Error responses use -32000 range codes with structured data
- ✅ Destructive tools require `confirm: true`
- ✅ `pm --mcp` starts stdio server successfully
- ✅ All tests passing
- ✅ Works with Claude Code MCP registration

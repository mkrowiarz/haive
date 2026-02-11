# mcp-symfony-stack

A standalone, reusable tool for managing Docker Compose-based development projects. Provides both an interactive TUI (Terminal UI) and an MCP server for Claude Code, enabling database operations and git worktree management.

## Purpose

- **TUI Mode**: Interactive terminal interface for database dumps/imports, worktree creation/removal, and project status
- **MCP Mode**: Stdio-based MCP server for Claude Code to manage infrastructure through `.claude/project.json`
- **CLI Mode**: One-shot commands for scripting and automation

Initially targeting Symfony (7/8, PHP 8.3+) but designed to be framework-agnostic where possible.

## Main Assumptions

- **Docker Compose is the runtime** — all database interactions happen via `docker compose exec`
- **Config-driven** — project-specific knowledge lives in `.claude/project.json`; the tool is stateless and generic
- **Env var resolution** — config values can reference `.env`/`.env.local` variables using `${VAR_NAME}` syntax
- **Safety by default** — database operations restricted to an explicit allowlist; default database cannot be dropped
- **Refuse without config** — any infrastructure operation requires valid project configuration

## Technology Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Language | **Go** | Single binary, fast startup, excellent exec/process handling |
| TUI | **Bubble Tea** + **Lip Gloss** | Industry standard for Go TUIs (lazygit, lazydocker) |
| MCP | **mcp-go** (`mark3labs/mcp-go`) | Go MCP SDK with stdio transport support |
| Config | **JSON + JSON Schema** | Native parsing, editor autocompletion via `$schema` |
| CLI | **cobra** or bare `os.Args` | One-shot commands for scripting |

## Phase 1: Core Library (Project Commands)

Current implementation supports project information and initialization commands.

### Project Commands

**`project.info`**: Get project configuration and status

```go
package main

import (
    "fmt"
    "github.com/mkrowiarz/mcp-symfony-stack/internal/core/commands"
)

func main() {
    info, err := commands.Info(".")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Project: %s (%s)\n", info.ConfigSummary.Name, info.ConfigSummary.Type)
    fmt.Printf("Docker Compose: %v\n", info.DockerComposeExists)
    fmt.Printf("Env files: %v\n", info.EnvFiles)
}
```

**`project.init`**: Generate suggested project configuration

```go
package main

import (
    "fmt"
    "github.com/mkrowiarz/mcp-symfony-stack/internal/core/commands"
)

func main() {
    suggestion, err := commands.Init(".")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Println("Suggested configuration:")
    fmt.Println(suggestion.SuggestedConfig)
    fmt.Printf("\nDetected services: %v\n", suggestion.DetectedServices)
    fmt.Printf("Detected env vars: %v\n", suggestion.DetectedEnvVars)
}
```

### Configuration File

Project configuration lives in `.claude/project.json`:

```json
{
  "project": {
    "name": "your-project",
    "type": "symfony"
  },
  "docker": {
    "compose_file": "docker-compose.yaml"
  }
}
```

*Note: Phase 1 supports minimal configuration only. Database and worktrees sections will be added in phase 2.*

# pm - Project Manager

A standalone tool for managing Docker Compose-based development projects. Provides TUI, MCP server, and CLI interfaces for database operations and git worktree management.

## Installation

### Build

```bash
git clone https://github.com/mkrowiarz/mcp-symfony-stack.git
cd mcp-symfony-stack
go build -o pm ./cmd/pm
```

### Install to PATH

**Option 1: User-local (recommended)**

```bash
mkdir -p ~/.local/bin
cp pm ~/.local/bin/
```

**Option 2: System-wide**

```bash
sudo cp pm /usr/local/bin/
```

### Add to PATH (if needed)

**Bash** (~/.bashrc or ~/.bash_profile):

```bash
export PATH="$HOME/.local/bin:$PATH"
```

**Fish** (~/.config/fish/config.fish):

```fish
set -gx PATH $HOME/.local/bin $PATH
```

Then reload your shell or run:

```bash
# Bash
source ~/.bashrc

# Fish
source ~/.config/fish/config.fish
```

## Quick Start

```bash
# Initialize config for your project
cd /path/to/your/project
pm init > .haive/config.json

# Run interactive TUI
pm

# Or use as MCP server for Claude Code
pm --mcp
```

## Configuration

Config file locations (checked in order):
1. `.haive/config.json`
2. `.haive.json`

### Minimal Config

```json
{
  "$schema": "https://raw.githubusercontent.com/mkrowiarz/mcp-symfony-stack/main/schema.json",
  "project": {
    "name": "my-project",
    "type": "symfony"
  },
  "docker": {
    "compose_files": ["docker-compose.yaml"]
  }
}
```

### Full Config with Database

```json
{
  "$schema": "https://raw.githubusercontent.com/mkrowiarz/mcp-symfony-stack/main/schema.json",
  "project": {
    "name": "my-project",
    "type": "symfony"
  },
  "docker": {
    "compose_files": [
      "compose.yaml",
      "docker/dev/compose/compose.app.yaml",
      "docker/dev/compose/compose.database.yaml"
    ]
  },
  "database": {
    "service": "database",
    "dsn": "${DATABASE_URL}",
    "dumps_path": "var/dumps"
  },
  "worktrees": {
    "base_path": "/path/to/worktrees",
    "db_per_worktree": true
  }
}
```

### Configuration Fields

| Field | Required | Description |
|-------|----------|-------------|
| `project.name` | Yes | Project name for display |
| `project.type` | Yes | Project type: `symfony`, `laravel`, `generic` |
| `docker.compose_files` | Yes | Array of compose file paths (relative to project root) |
| `database.service` | If database section exists | Docker Compose service name |
| `database.dsn` | If database section exists | Database URL (supports `${VAR}` interpolation) |
| `database.allowed` | No | Glob patterns for allowed databases (omit = allow all) |
| `database.dumps_path` | No | Directory for SQL dumps (default: `var/dumps`) |
| `worktrees.base_path` | If worktrees section exists | Directory for worktrees |
| `worktrees.db_per_worktree` | No | Auto-create database per worktree |

### Environment Variables

Config values support `${VAR_NAME}` syntax, resolved from:
1. OS environment
2. `.env.local`
3. `.env`

## MCP Server Setup

MCP servers are configured in `.claude/mcp.json` files.

**Option 1: Global** (~/.claude/mcp.json) - applies to all projects:

```json
{
  "mcpServers": {
    "pm": {
      "command": "/path/to/pm",
      "args": ["--mcp"]
    }
  }
}
```

**Option 2: Project-specific** (/path/to/project/.claude/mcp.json) - only for this project:

```json
{
  "mcpServers": {
    "pm": {
      "command": "/path/to/pm",
      "args": ["--mcp"]
    }
  }
}
```

Project-specific config is loaded in addition to (not instead of) global config.

**Note:** MCP servers are configured in `mcp.json`, not `settings.local.json`. The `settings.local.json` file is for other settings like `approvedCommandPatterns` and permission modes.

### Available MCP Tools

- `project_info` - Get project configuration and status
- `project_init` - Generate suggested configuration
- `worktree_list` - List git worktrees
- `worktree_create` - Create a worktree
- `worktree_remove` - Remove a worktree
- `db_list` - List databases
- `db_dump` - Dump database to SQL file
- `db_import` - Import SQL file into database
- `db_create` - Create empty database
- `db_drop` - Drop database
- `db_clone` - Clone database
- `db_dumps_list` - List available dump files
- `workflow_create_isolated_worktree` - Create worktree with optional database
- `workflow_remove_isolated_worktree` - Remove worktree with optional database cleanup

## TUI Keyboard Shortcuts

Press `?` in TUI to see all shortcuts.

| Key | Action |
|-----|--------|
| `Tab`/`Shift+Tab` | Cycle panes |
| `1-4` | Jump to pane |
| `j`/`k` or arrows | Navigate items |
| `n` | New worktree (pane 2) |
| `o` | Open worktree in terminal (pane 2) |
| `x` | Remove worktree / Drop database / Delete dump |
| `d` | Dump database (pane 3) |
| `c` | Clone database (pane 3) |
| `i` | Import dump (pane 4) |
| `r` | Refresh current pane |
| `R` | Refresh all panes |
| `q` | Quit |

## Safety Guards

- Default database (from DSN) cannot be dropped
- If `allowed` is set, only matching databases can be operated on
- Path traversal attempts are blocked for worktrees

## Supported Databases

- MySQL (port 3306)
- MariaDB (detected via `serverVersion` query param)
- PostgreSQL (port 5432)

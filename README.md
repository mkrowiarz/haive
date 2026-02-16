# pm - Project Manager

A standalone tool for managing Docker Compose-based development projects. Provides TUI, MCP server, and CLI interfaces for database operations and git worktree management.

## Installation

### Option 1: Install from Source (Recommended for Development)

```bash
git clone https://github.com/mkrowiarz/mcp-symfony-stack.git
cd mcp-symfony-stack
make install
```

Binary is installed to `$HOME/go/bin/pm`. Add to PATH if needed:

**Bash**: `export PATH="$HOME/go/bin:$PATH"`

**Fish**: `set -gx PATH $HOME/go/bin $PATH`

### Option 2: Install to ~/.local/bin

```bash
git clone https://github.com/mkrowiarz/mcp-symfony-stack.git
cd mcp-symfony-stack
make install-local
```

Binary is installed to `~/.local/bin/pm`. Make sure `~/.local/bin` is in your PATH.

### Option 3: Install Latest Release

```bash
go install github.com/mkrowiarz/mcp-symfony-stack/cmd/pm@latest
```

Binary is installed to `$HOME/go/bin/pm`.

### Option 4: Manual Build

```bash
git clone https://github.com/mkrowiarz/mcp-symfony-stack.git
cd mcp-symfony-stack
go build -o pm ./cmd/pm

# Move to system PATH
sudo mv pm /usr/local/bin/
# Or to user-local bin
mkdir -p ~/.local/bin && mv pm ~/.local/bin/
```

## Quick Start

```bash
# Initialize config for your project (preview)
cd /path/to/your/project
pm init

# Write config directly to .haive/config.json
pm init --write

# Output with "pm" namespace (for adding to existing .haive.json)
pm init --namespace

# Write namespaced config directly to .haive/config.json
pm init --namespace --write

# Switch to a branch with automatic database switching
pm checkout feature/pf-1234-demo --create

# Just switch database for current branch
pm switch

# Run interactive TUI
pm

# Or use as MCP server for Claude Code
pm --mcp
```

## Configuration

Config file locations (checked in order):
1. `.claude/project.json` (recommended)
2. `.haive/config.json`
3. `.haive.json`

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
    "allowed": ["myapp", "myapp_*"],
    "dumps_path": "var/dumps"
  },
  "worktrees": {
    "base_path": "/path/to/worktrees",
    "db_per_worktree": true
  }
}
```

**Note:** `database.allowed` is required when the database section is present. Use glob patterns like `["app", "app_*"]` to specify which databases can be operated on.

### Shared Config with Other Tools

If you use `.haive.json` for multiple tools, you can namespace the `pm` config:

```json
{
  "project": "other-tool-config",
  "agents": ["claude"],
  "pm": {
    "project": {
      "name": "my-project",
      "type": "symfony"
    },
    "docker": {
      "compose_files": ["docker-compose.yaml"]
    },
    "database": {
      "service": "db",
      "dsn": "${DATABASE_URL}",
      "allowed": ["myapp", "myapp_*"]
    }
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
| `database.allowed` | If database section exists | Glob patterns for allowed databases (e.g., `["app", "app_*"]`) |
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
      "command": "pm",
      "args": ["--mcp"]
    }
  }
}
```

If `pm` is not in PATH, use the full path: `"command": "/home/user/go/bin/pm"`.

**Option 2: Project-specific** (/path/to/project/.claude/mcp.json) - only for this project:

```json
{
  "mcpServers": {
    "pm": {
      "command": "pm",
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
- `database.allowed` restricts which databases can be operated on (required when database section is present)
- Path traversal attempts are blocked for worktrees

## CLI Commands

### `pm checkout <branch>` - Switch branch with database

Switches to a git branch and automatically switches to the corresponding database.

```bash
# Checkout existing branch and its database
pm checkout feature/my-feature

# Create new branch with new database
pm checkout feature/new-feature --create

# Create branch and clone data from specific database
pm checkout feature/demo --create --clone-from=symfony
```

**Database naming:** Branch `feature/pf-1234-demo` gets database `symfony_feature_pf_1234_demo` (based on your default DB name).

### `pm switch` - Switch database for current branch

Switches the database for your current branch without changing git branches.

```bash
# Switch to database for current branch
pm switch

# Switch and clone from specific database
pm switch --clone-from=symfony
```

**Automatic behavior:**
- On `main` or `master`: uses the default database
- On feature branches: creates/uses `<default_db>_<branch_name>`
- If database doesn't exist: automatically creates it
- If feature branch: automatically clones from default database

### `pm worktree` - Manage git worktrees

List, create, and remove git worktrees from the command line.

```bash
# List all worktrees
pm worktree list
pm wt ls

# Create worktree for existing branch
pm worktree create feature/my-feature
pm wt create feature/my-feature

# Create worktree with new branch
pm worktree create feature/new-feature --new-branch
pm wt add feature/new-feature -n

# Remove worktree
pm worktree remove feature/my-feature
pm wt rm feature/my-feature
```

**Note:** Worktree commands require the `worktrees` section in your config.

### `pm serve` - Run app container for worktrees

Start and stop the app container for a worktree with isolated dependencies. Designed for OrbStack environments where each container gets automatic DNS (`.orb.local`).

```bash
# From within a worktree directory
cd .worktrees/feature-my-feature

# Start the app container
pm serve

# Stop the app container
pm serve stop
```

**How it works:**
1. Detects you're in a worktree (checks `.git` file format)
2. Looks for `compose.worktree.yaml` in the worktree root
3. Starts container with unique project name: `<project>-wt-<branch>`
4. Returns OrbStack hostname: `<project>-wt-<branch>-app.orb.local`

**Setup: Create `compose.worktree.yaml` in each worktree**

This file customizes how the app runs for this specific worktree:

```yaml
services:
  app:
    # No port mapping - OrbStack provides automatic hostname
    ports: []

    # Isolate dependencies from other worktrees/main
    volumes:
      - .:/app:delegated
      - /app/var/cache       # Isolated cache
      - /app/vendor          # Isolated vendor/
      - /app/node_modules    # Isolated node_modules (optional)
      - ~/.composer/auth.json:/root/.config/composer/auth.json

    # Override environment for this worktree
    environment:
      # Example: Use worktree-specific database
      DATABASE_URL: "mysql://user:pass@db:3306/myapp_wt_feature_x"
      TZ: 'Europe/Berlin'

# Connect to shared services (db, redis, etc.) from main project
networks:
  pf-network:
    external: true
    name: professionfit-symfony_pf-network
  local:
    external: true
    name: professionfit-symfony_local
```

**Network names:** Update `professionfit-symfony` to match your main project's Docker Compose project name. Check with: `docker network ls`

**Dependencies:** Run `composer install` and `npm install` in the worktree before starting the container. The isolated volumes ensure each worktree has its own dependencies.

**Why separate dependencies?**
- Branches may require different package versions
- Prevents conflicts between worktrees
- Avoids breaking main project when testing experimental packages

## Troubleshooting

### "Dump failed" or "Import failed" errors

If you see errors mentioning `mysqldump: [Warning] Using a password...`, this is a MySQL warning that was being captured into the SQL output. This has been fixed - update to the latest version.

### Config not found

The tool searches for config in this order:
1. `.claude/project.json` (recommended)
2. `.haive/config.json`
3. `.haive.json`

If you have an existing `.haive.json` with other tool configs, add the `pm` namespace (see "Shared Config with Other Tools" above).

### Database operations fail with "not in allowed list"

The `database.allowed` field is required when the database section is present. It specifies which databases can be operated on for safety:

```json
"database": {
  "allowed": ["myapp", "myapp_*"]
}
```

## Supported Databases

- MySQL (port 3306)
- MariaDB (detected via `serverVersion` query param)
- PostgreSQL (port 5432)

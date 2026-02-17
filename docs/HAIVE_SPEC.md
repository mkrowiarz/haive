# Haive Specification

> This document serves as the authoritative specification for the Haive project. Future agents should read this before making changes.

## Overview

**Haive** is a modular development environment manager focused on:
- Git worktree management
- Database lifecycle management (dump, import, clone, drop)
- Project-specific workflow automation
- Multi-branch development workflows

**Key Principles:**
1. Modular design - each feature is independent
2. Configuration over code - behavior driven by config files
3. AI-friendly - works well with AI assistants via MCP
4. Project-type agnostic with presets for common frameworks

## Project Naming

- **Binary name**: `haive`
- **Go module**: `github.com/mkrowiarz/mcp-symfony-stack` (keep repo name)
- **Config file names**: `haive.toml`, `haive.yaml`, or legacy `haive.json`

## Configuration System

### Config File Discovery (Priority Order)

Haive searches for config files in this order:
1. `haive.toml` (project root)
2. `.haive/config.toml`
3. `haive.yaml` / `.haive/config.yaml`
4. `haive.json` / `.haive/config.json`
5. Legacy: `.claude/project.json` (for backward compatibility)

### Master Config Structure

```toml
# haive.toml
[project]
name = "my-project"
preset = "symfony"  # optional: loads preset defaults

[docker]
compose_files = ["compose.yaml"]

# Module configuration - either inline or external file
[worktree]
config = ".haive/worktree.toml"  # external file reference
# OR inline:
# base_path = ".worktrees"
# db_per_worktree = true

[database]
config = ".haive/database.toml"  # external file reference
# OR inline configuration
```

### Module Config Structure

When using external files (e.g., `.haive/worktree.toml`), the content is **NOT** wrapped in a `[worktree]` section:

**Correct** (`.haive/worktree.toml`):
```toml
base_path = ".worktrees"
db_per_worktree = true

[copy]
include = ["**/.env.local"]
exclude = ["vendor/", "node_modules/"]

[hooks]
postCreate = ["composer install"]
```

**Incorrect**:
```toml
[worktree]  # DON'T DO THIS in module files
base_path = ".worktrees"
```

### Supported Formats

1. **TOML** (primary, recommended)
   - Best for complex nested structures
   - Supports comments
   - Good array formatting

2. **YAML** (secondary)
   - Familiar to many developers
   - Good for simple configs

3. **JSON** (legacy, backward compatibility only)
   - No comments
   - Verbose for complex structures
   - Legacy namespace `pm` still supported: `{ "pm": { ... } }`

## Modules

### Module Interface

Each module implements:
- `Name() string` - module identifier
- `ValidateConfig() error` - validate module-specific config
- `RegisterHooks(registry HookRegistry)` - register default hooks

### Core Modules

#### 1. Worktree Module

Purpose: Manage Git worktrees for multi-branch development.

**Config Fields:**
```toml
base_path = ".worktrees"        # required: where to create worktrees
db_per_worktree = true          # optional: auto-create DB per worktree
db_prefix = "app_wt_"           # optional: DB name prefix

[copy]                          # optional: file copying
include = ["**/.env.local"]     # glob patterns to copy from main repo
exclude = ["vendor/"]           # glob patterns to exclude

[hooks]                         # optional: lifecycle hooks
postCreate = ["composer install"]
preRemove = ["make cleanup"]
postRemove = ["echo 'Cleaned up'"]
```

**Supported Hooks:**
- `postCreate`: Runs after worktree is created (and files are copied)
- `preRemove`: Runs before worktree is removed. Can abort with non-zero exit.
- `postRemove`: Runs after worktree is removed

**Environment Variables Available to Hooks:**
- `REPO_ROOT`: Absolute path to project root
- `PROJECT_NAME`: Project name from config
- `WORKTREE_PATH`: Absolute path to worktree directory
- `WORKTREE_NAME`: Directory name (sanitized branch name)
- `BRANCH`: Git branch name

#### 2. Database Module

Purpose: Manage database lifecycle (dump, import, clone, drop).

**Config Fields:**
```toml
service = "database"                # required: Docker service name
dsn = "${DATABASE_URL}"             # required: connection string
allowed = ["myapp", "myapp_*"]      # required: allowed DB patterns
dumps_path = "var/dumps"            # optional: dump storage location

[hooks]
postClone = ["bin/console doctrine:migrations:migrate"]
preDrop = ["echo 'About to drop $DATABASE_NAME'"]
```

**Supported Hooks:**
- `postClone`: Runs after database is cloned
- `preDrop`: Runs before database is dropped. Can abort with non-zero exit.

**Environment Variables Available to Hooks:**
- `DATABASE_NAME`: Name of the database
- `DATABASE_URL`: Full DSN connection string
- `DATABASE_HOST`: Database host
- `DATABASE_PORT`: Database port
- `SOURCE_DATABASE`: Source DB name (for postClone)
- `TARGET_DATABASE`: Target DB name (for postClone)
- `REPO_ROOT`: Absolute path to project root
- `PROJECT_NAME`: Project name from config

#### 3. Docker Module

Purpose: Docker Compose integration.

**Config Fields:**
```toml
compose_files = ["compose.yaml", "compose.override.yaml"]
project_name = "myapp"  # optional
```

### Future Modules

- `task`: Task runner integration (Make, Taskfile, npm scripts)
- `server`: Development server management
- `test`: Test runner integration

## Hooks System

### Hook Execution

Hooks are shell commands executed with the following behavior:

1. **Execution Context**: 
   - Worktree hooks run in the worktree directory
   - Database hooks run in project root

2. **Environment Variables**: Injected based on hook type (see module docs)

3. **Error Handling**:
   - Non-zero exit code = error
   - `preRemove` and `preDrop` hooks can abort the operation
   - Use `--force` flag to skip `pre*` hooks on error

4. **Output**: Streamed to user (TUI shows in logs, CLI shows directly)

### Hook Definition

In TOML:
```toml
[worktree.hooks]
postCreate = [
    "composer install --no-interaction",
    "npm ci",
    "bin/console cache:clear"
]
```

In YAML:
```yaml
worktree:
  hooks:
    postCreate:
      - "composer install --no-interaction"
      - "npm ci"
```

### Predefined Hooks (Built-in)

Some behaviors that could be hooks are built-in for convenience:

1. **File Copy** (`worktree.copy`): Built-in, runs before `postCreate`
   - Uses `doublestar` library for glob matching
   - Respects `include` and `exclude` patterns
   - Maintains directory structure when copying

2. **Database Auto-Creation**: When `worktree.db_per_worktree = true`, automatically creates database for worktree

## Presets

### What Are Presets?

Presets are default configurations for specific project types. They:
- Provide sensible defaults
- Include common hooks
- Set up copy patterns

### Preset Resolution

```toml
[project]
preset = "symfony"  # Can be:
                    # - Built-in name: "generic", "symfony", "laravel"
                    # - Relative path: "./custom-preset.toml"
                    # - Absolute path: "/path/to/preset.toml"
                    # - Omit for no preset
```

### Preset Merging

User config **merges with** preset (preset provides defaults):

1. Load preset config
2. Load user config
3. Deep merge: user values override preset values
4. Validate final config

### Built-in Presets

#### `generic` (Default)
Minimal configuration, no assumptions.

#### `symfony`
```toml
# Default copy patterns for Symfony
[worktree.copy]
include = ["**/.env.local", "**/.env.*.local"]
exclude = ["vendor/", "node_modules/", ".git/", "var/cache/", "var/log/"]

# Default hooks
[worktree.hooks]
postCreate = ["composer install --no-interaction"]

[database.hooks]
postClone = ["bin/console doctrine:migrations:migrate --no-interaction"]

# Default database settings
[database]
dumps_path = "var/dumps"
```

#### `laravel`
Similar to Symfony but with Laravel-specific defaults:
```toml
[worktree.hooks]
postCreate = ["composer install", "npm install"]

[database.hooks]
postClone = ["php artisan migrate"]
```

### Custom Presets

Users can create custom presets:

**`.haive/presets/team-default.toml`**:
```toml
[worktree]
db_per_worktree = true

[worktree.hooks]
postCreate = [
    "task install",      # Taskfile
    "mise trust && mise install"
]
```

**Usage**:
```toml
[project]
preset = ".haive/presets/team-default.toml"
```

## File Copy System

### Purpose
Copy untracked/local files from main repository to new worktrees.

### How It Works

1. After `git worktree add` creates the worktree
2. Before `postCreate` hooks run
3. Copy files matching `include` patterns, excluding `exclude` patterns
4. Maintain directory structure

### Configuration

```toml
[worktree.copy]
include = [
    "**/.env.local",        # Any .env.local at any depth
    "**/.env.*.local",      # Environment-specific .env files
    "config/secrets.yaml"    # Specific config file
]

exclude = [
    "vendor/",               # Never copy vendor directories
    "node_modules/",         # Never copy node_modules
    ".git/",                 # Never copy git directory
    "**/cache/**"            # Never copy cache directories
]
```

### Implementation Notes

- Uses `github.com/bmatcuk/doublestar/v4` for glob matching
- Supports `**` for recursive matching
- Directories are created as needed
- Files are copied (not symlinked) for isolation
- If copy fails, warning is logged but worktree creation succeeds

## Architecture

### Directory Structure

```
internal/
├── core/
│   ├── config/              # Config loading, merging, validation
│   │   ├── loader.go        # File discovery and parsing
│   │   ├── merge.go         # Config merging logic
│   │   ├── preset.go        # Preset resolution
│   │   └── validation.go    # Config validation
│   ├── hooks/
│   │   ├── executor.go      # Hook execution engine
│   │   └── registry.go      # Hook registration
│   ├── modules/
│   │   ├── module.go        # Module interface
│   │   ├── worktree/        # Worktree module
│   │   │   ├── config.go    # Worktree config types
│   │   │   ├── commands.go  # Worktree operations
│   │   │   └── hooks.go     # Default hooks
│   │   └── database/        # Database module
│   │       ├── config.go
│   │       ├── commands.go
│   │       └── hooks.go
│   └── types/
│       └── types.go         # Shared types
├── executor/                # Shell command execution
├── tui/                     # Bubble Tea TUI
├── cli/                     # Cobra CLI
├── mcp/                     # MCP server
└── presets/                 # Built-in presets (embedded)
    ├── generic.toml
    ├── symfony.toml
    └── laravel.toml
```

### Module Independence

Modules are designed to be independent:
- `haive worktree create` works without database module
- `haive database dump` works without worktree module
- Modules communicate through hooks, not direct calls

### Orchestration (Future)

For multi-module workflows:

```toml
[workflow.create]
steps = [
    { module = "worktree", action = "create" },
    { module = "database", action = "clone", from = "main" },
]
```

Command: `haive workflow create feature/xyz`

## Error Handling

### Typed Errors

All errors use `types.CommandError`:

```go
type CommandError struct {
    Code    ErrCode `json:"code"`
    Message string  `json:"message"`
}
```

Error codes:
- `CONFIG_MISSING`: Config file not found
- `CONFIG_INVALID`: Config has invalid values
- `ERR_DB_NOT_ALLOWED`: Database name not in allowed list
- `ERR_DB_IS_DEFAULT`: Attempted to drop default database
- `ERR_INVALID_WORKTREE`: Invalid worktree name/path
- `ERR_PATH_TRAVERSAL`: Path traversal attempt detected

### Hook Error Behavior

- `preCreate`: Non-zero exit stops worktree creation
- `postCreate`: Non-zero exit logs warning, continues
- `preRemove`: Non-zero exit stops removal (use `--force` to override)
- `postRemove`: Non-zero exit logs warning, continues

## Security Considerations

1. **Path Traversal**: All paths checked against traversal attacks
2. **Database Guards**: `allowed` patterns prevent accidental DB operations
3. **Default DB Protection**: Cannot drop default database
4. **Hook Safety**: Hooks run with user permissions (no sandboxing)

## Dependencies

### Required
- `github.com/BurntSushi/toml` (TOML parsing)
- `github.com/bmatcuk/doublestar/v4` (glob matching)
- Standard library for JSON, YAML already present

### Optional (Future)
- `github.com/mitchellh/mapstructure` (generic config mapping)

## Migration from "pm"

### Config Migration

Old `.claude/project.json`:
```json
{
  "pm": {
    "project": { "name": "myapp", "type": "symfony" },
    "docker": { "compose_files": ["compose.yaml"] },
    "worktrees": { "base_path": ".worktrees" }
  }
}
```

New `haive.toml`:
```toml
[project]
name = "myapp"
preset = "symfony"

[docker]
compose_files = ["compose.yaml"]

[worktree]
base_path = ".worktrees"
```

### Breaking Changes

1. `worktrees` → `worktree` (singular)
2. `project.type` → `project.preset`
3. Remove `"pm"` namespace wrapper
4. Config location: `.claude/` → root or `.haive/`

## Future Considerations

### Plugin System (Not Now)
- WASM modules for custom presets
- Go plugins for built-in modules

### Remote Presets
```toml
preset = "https://company.com/haive-presets/symfony.toml"
```

### Multi-Project Workspaces
```toml
[workspace]
projects = ["api/", "frontend/", "worker/"]
```

---

## Quick Reference

### Commands
```bash
# Worktree management
haive worktree create feature/xyz
haive worktree remove feature/xyz
haive worktree list

# Database management  
haive database dump mydb
haive database import mydb dump.sql
haive database clone source target
haive database drop mydb
haive database list

# Workflow (future)
haive workflow create feature/xyz
haive workflow remove feature/xyz
```

### Config Locations Checked
1. `haive.toml`
2. `.haive/config.toml`
3. `.haive/config.yaml`
4. `haive.yaml`
5. Legacy: `.claude/project.json`

### Hook Environment Variables
All hooks: `REPO_ROOT`, `PROJECT_NAME`
Worktree: `WORKTREE_PATH`, `WORKTREE_NAME`, `BRANCH`
Database: `DATABASE_NAME`, `DATABASE_URL`, `SOURCE_DATABASE`, `TARGET_DATABASE`

# MCP Project Manager — Specification

## Overview

A standalone, reusable tool that gives both developers (via TUI) and Claude (via MCP) the ability to manage Docker Compose-based development projects. Initially targeting Symfony (7/8, PHP 8.3+) but designed to be framework-agnostic where possible.

The tool ships as a **single binary** with two modes:

- **TUI mode** (default) — an interactive terminal interface in the style of lazygit/lazydocker
- **MCP mode** (`--mcp`) — a stdio-based MCP server for Claude Code and other MCP clients
- **CLI mode** (`pm <command>`) — one-shot commands for scripting and automation

All three interfaces share the same core library. Project-specific configuration lives in `.claude/project.json`.

**Repository:** `github.com/<user>/mcp-project-manager`

---

## Design Principles

1. **Project config is the contract.** The tool is stateless and generic. All project-specific knowledge lives in `.claude/project.json`. No hardcoded paths, credentials, or assumptions.

2. **Refuse without config.** Any operation that touches infrastructure (database, worktrees) requires a valid config file. If missing or incomplete, the tool returns a clear error with setup instructions.

3. **Docker Compose is the runtime.** The tool does not require host-installed database clients. All database interactions happen via `docker compose exec -T` into the appropriate service container.

4. **Env var resolution.** Config values can reference environment variables from the project's `.env` / `.env.local` files using the `${VAR_NAME}` syntax. Plain strings are used as-is.

5. **Safety by default.** Database operations are restricted to an explicit allowlist. The default/primary database cannot be dropped. Worktree paths are validated against traversal.

6. **Interface-agnostic core.** The core library knows nothing about MCP, TUI, or CLI. It accepts typed inputs, returns typed outputs, and emits progress events. Each interface is a thin adapter.

---

## Technology

| Layer     | Technology                          | Rationale                                                                    |
|-----------|-------------------------------------|------------------------------------------------------------------------------|
| Language  | **Go**                              | Single binary, fast startup, excellent exec/process handling                 |
| TUI       | **Bubble Tea** + **Lip Gloss**      | Industry standard for Go TUIs (lazygit, lazydocker). Elm architecture.       |
| MCP       | **mcp-go** (`mark3labs/mcp-go`)     | Go MCP SDK, stdio transport support                                          |
| Config    | JSON + JSON Schema                  | Native parsing, editor autocompletion via `$schema`                          |
| CLI       | **cobra** or bare `os.Args`         | One-shot commands for scripting (`pm dump`, `pm worktree create`)            |

### Why Go over TypeScript

- **Single binary** — no Node.js runtime dependency, trivial installation (`go install` / download binary)
- **Bubble Tea** — the TUI framework that lazygit and lazydocker use; nothing in the Node/TypeScript ecosystem matches its quality
- **Startup time** — instant; Node.js TUI apps have a noticeable cold start
- **Cross-platform** — compiles to Linux, macOS, Windows with no runtime requirements
- **`os/exec`** — first-class support for shelling out to `git`, `docker`, etc.

---

## Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                        Interfaces                              │
│                                                                │
│  ┌────────────┐    ┌────────────────┐    ┌──────────────────┐  │
│  │ MCP Server │    │   TUI (Bubble  │    │   CLI (cobra /   │  │
│  │  (stdio)   │    │     Tea)       │    │    one-shot)     │  │
│  └─────┬──────┘    └───────┬────────┘    └────────┬─────────┘  │
│        │                   │                      │            │
│        └───────────┬───────┴──────────────────────┘            │
│                    ▼                                           │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Core Library                          │   │
│  │                                                         │   │
│  │  config    guard    dsn    env    commands/*             │   │
│  │                                                         │   │
│  │  • Typed inputs and outputs (structs)                   │   │
│  │  • Progress callbacks for long operations               │   │
│  │  • Result types with structured errors                  │   │
│  │  • No I/O formatting opinions                           │   │
│  └────────────────────────┬────────────────────────────────┘   │
│                           │                                    │
│                           ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                     Executors                           │   │
│  │                                                         │   │
│  │  docker (compose exec)    git (worktree)    filesystem  │   │
│  └─────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────┘
```

### Core Design Contracts

**Commands return typed results, never formatted strings:**

```go
type DumpResult struct {
    Path     string        `json:"path"`
    Size     int64         `json:"size"`
    Database string        `json:"database"`
    Tables   []string      `json:"tables"`
    Duration time.Duration `json:"duration"`
}
```

**Errors are typed with codes:**

```go
type ErrCode string

const (
    ErrConfigMissing    ErrCode = "CONFIG_MISSING"
    ErrConfigInvalid    ErrCode = "CONFIG_INVALID"
    ErrDbNotAllowed     ErrCode = "DB_NOT_ALLOWED"
    ErrDbIsDefault      ErrCode = "DB_IS_DEFAULT"
    ErrDockerNotRunning ErrCode = "DOCKER_NOT_RUNNING"
    ErrServiceNotFound  ErrCode = "SERVICE_NOT_FOUND"
    ErrFileNotFound     ErrCode = "FILE_NOT_FOUND"
    ErrPathTraversal    ErrCode = "PATH_TRAVERSAL"
    ErrInvalidName      ErrCode = "INVALID_NAME"
    ErrEnvVarNotFound   ErrCode = "ENV_VAR_NOT_FOUND"
)

type CommandError struct {
    Code    ErrCode `json:"code"`
    Message string  `json:"message"`
}
```

**Long operations emit progress:**

```go
type ProgressStage string

const (
    StageDumping   ProgressStage = "dumping"
    StageCreating  ProgressStage = "creating"
    StageImporting ProgressStage = "importing"
    StageCloning   ProgressStage = "cloning"
    StagePatching  ProgressStage = "patching"
)

type ProgressFunc func(stage ProgressStage, detail string)
```

Each interface adapter handles progress differently:
- **MCP:** ignores progress (stdio protocol has no progress mechanism)
- **TUI:** renders a spinner or progress bar via Bubble Tea messages
- **CLI:** prints progress lines to stderr

---

## Configuration File

**Location:** `.claude/project.json` (relative to project root)

A JSON Schema is published at the repository root (`schema.json`) for editor autocompletion and validation.

### Env Var Resolution

Any string value in the config may use `${VAR_NAME}` syntax. The tool resolves these by reading the project's `.env` and `.env.local` files (`.env.local` takes precedence). If a referenced variable is not found, the tool fails with a descriptive error.

Plain strings without `${}` are used literally.

```json
"dsn": "${DATABASE_URL}"
"dsn": "mysql://root:pw@db/app"
```

### Full Example

```json
{
  "$schema": "https://raw.githubusercontent.com/<user>/mcp-project-manager/main/schema.json",
  "project": {
    "name": "facility-saas",
    "type": "symfony"
  },
  "docker": {
    "compose_file": "docker-compose.yaml"
  },
  "database": {
    "service": "database",
    "dsn": "${DATABASE_URL}",
    "allowed": [
      "facility_app",
      "facility_app_test",
      "facility_app_wt_*"
    ],
    "dumps_path": "var/dumps"
  },
  "worktrees": {
    "base_path": "../worktrees",
    "db_per_worktree": true,
    "db_prefix": "facility_app_wt_"
  }
}
```

### Schema Reference

#### `project` (required)

| Field  | Type   | Required | Description                                           |
|--------|--------|----------|-------------------------------------------------------|
| `name` | string | yes      | Human-readable project name                           |
| `type` | string | yes      | Project type: `symfony`, `laravel`, `generic`         |

#### `docker` (required)

| Field          | Type   | Required | Default                | Description                                        |
|----------------|--------|----------|------------------------|----------------------------------------------------|
| `compose_file` | string | no       | `docker-compose.yaml`  | Path to compose file, relative to project root     |

#### `database` (optional)

If omitted, all database commands are disabled.

| Field        | Type     | Required | Default     | Description                                                        |
|--------------|----------|----------|-------------|--------------------------------------------------------------------|
| `service`    | string   | yes      | —           | Docker Compose service name for the database container             |
| `dsn`        | string   | yes      | —           | Database connection string (standard URI format). Supports `${VAR}`. |
| `allowed`    | string[] | yes      | —           | Database names or glob patterns permitted for operations           |
| `dumps_path` | string   | no       | `var/dumps` | Directory for SQL dumps, relative to project root                  |

**DSN format:** `<engine>://<user>:<password>@<host>:<port>/<database>?serverVersion=<version>`

Parsed using Go's `net/url` package. The `host` field is the Docker Compose service name (only relevant inside the container network). The `serverVersion` parameter distinguishes MariaDB from MySQL when the scheme is `mysql://`.

**Allowed patterns:** Exact names (`facility_app`) or glob wildcards (`facility_app_wt_*`). A database name must match at least one pattern to be operated on.

#### `worktrees` (optional)

If omitted, worktree commands are disabled.

| Field             | Type    | Required | Default            | Description                                                      |
|-------------------|---------|----------|--------------------|------------------------------------------------------------------|
| `base_path`       | string  | yes      | —                  | Directory where worktrees are created, relative to project root  |
| `db_per_worktree` | boolean | no       | `false`            | Automatically create/drop an isolated DB per worktree            |
| `db_prefix`       | string  | no       | `<default_db>_wt_` | Prefix for auto-created worktree databases                       |

When `db_per_worktree` is `true`, creating a worktree for branch `feature/foo` will:
1. Create the worktree directory
2. Create database `<db_prefix>feature_foo` (branch name sanitized: `/` and `-` become `_`)
3. Clone the default database (from DSN) into the new database
4. Patch `.env.local` in the worktree to point `DATABASE_URL` to the new database

Removing a worktree reverses this process.

---

## Commands

All commands are defined in the core library and exposed through all three interfaces.

### Project Commands

#### `project.info`

Returns the current project configuration and status. Always available, even without config (reports that no config was found).

**Parameters:** none

**Returns:** `ProjectInfo` — config summary, Docker Compose service statuses, detected `.env` files.

#### `project.init`

Inspects the project structure and generates a suggested `.claude/project.json`. Reads `docker-compose.yaml` to detect database services, reads `composer.json` to detect framework type, reads `.env` for credential variable names. Does not write any files — returns the suggestion.

**Parameters:** none

**Returns:** `InitSuggestion` — suggested JSON config, detected services, detected env vars.

### Database Commands

All database commands require a valid `database` section in the config.

#### `db.list`

List all databases in the container (excluding system databases).

**Parameters:** none

**Returns:** `[]DatabaseInfo` — name, size (if available), whether it's the default.

#### `db.dump`

Dump a database to a SQL file.

| Parameter  | Type     | Required | Default             | Description                              |
|------------|----------|----------|---------------------|------------------------------------------|
| `database` | string   | no       | Default DB from DSN | Database to dump (must be in `allowed`)  |
| `tables`   | string[] | no       | All tables          | Specific tables to dump                  |

**Returns:** `DumpResult` — file path, size, database, tables, duration.

**Implementation:** `docker compose exec -T <service> <dump_command>` piped to host filesystem at `<dumps_path>/<db>_<timestamp>.sql`.

Engine-specific dump commands:
- MariaDB: `mariadb-dump`
- MySQL: `mysqldump`
- PostgreSQL: `pg_dump`

#### `db.import`

Import a SQL file into a database. **Destructive.**

| Parameter  | Type   | Required | Default             | Description                             |
|------------|--------|----------|---------------------|-----------------------------------------|
| `sql_path` | string | yes      | —                   | Path to the `.sql` file                 |
| `database` | string | no       | Default DB from DSN | Target database (must be in `allowed`)  |

**Returns:** `ImportResult` — imported file, database, duration.

#### `db.create`

Create a new empty database.

| Parameter  | Type   | Required | Description                                    |
|------------|--------|----------|------------------------------------------------|
| `database` | string | yes      | Database name (must match `allowed` patterns)  |

**Returns:** `CreateResult` — database name, confirmation.

#### `db.drop`

Drop a database. **Destructive.**

| Parameter  | Type   | Required | Description                                    |
|------------|--------|----------|------------------------------------------------|
| `database` | string | yes      | Database name (must match `allowed` patterns)  |

**Safety:** Refuses to drop the default database.

**Returns:** `DropResult` — database name, confirmation.

#### `db.clone`

Clone one database into another (dump + create + import in one step).

| Parameter | Type   | Required | Default             | Description          |
|-----------|--------|----------|---------------------|----------------------|
| `source`  | string | no       | Default DB from DSN | Source database       |
| `target`  | string | yes      | —                   | Target database name  |

Creates the target if it doesn't exist. Both must match `allowed` patterns. Emits progress events.

**Returns:** `CloneResult` — source, target, size, duration.

#### `db.dumps`

List available SQL dump files.

**Parameters:** none

**Returns:** `[]DumpFileInfo` — name, size, modification time, sorted by most recent.

### Worktree Commands

All worktree commands require a valid `worktrees` section in the config.

#### `worktree.list`

List all git worktrees for the project.

**Parameters:** none

**Returns:** `[]WorktreeInfo` — path, branch, associated database (if any), whether it's the main worktree.

#### `worktree.create`

Create a new git worktree, optionally with an isolated database.

| Parameter    | Type    | Required | Default                        | Description                                   |
|--------------|---------|----------|--------------------------------|-----------------------------------------------|
| `branch`     | string  | yes      | —                              | Branch name to checkout or create              |
| `new_branch` | boolean | no       | `false`                        | Create a new branch (vs checkout existing)     |
| `clone_db`   | boolean | no       | Value of `db_per_worktree`     | Clone the default DB for this worktree         |

**Worktree directory name:** Branch name with `/` replaced by `-` (e.g., `feature/booking-calendar` → `feature-booking-calendar`).

**Database name (when cloning):** `<db_prefix>` + branch name with `/` and `-` replaced by `_` (e.g., `facility_app_wt_feature_booking_calendar`).

Emits progress events across stages: creating worktree → creating database → cloning data → patching env.

**Returns:** `WorktreeCreateResult` — worktree path, branch, database info (if cloned).

#### `worktree.remove`

Remove a git worktree and optionally drop its associated database. **Destructive.**

| Parameter | Type    | Required | Default | Description                            |
|-----------|---------|----------|---------|----------------------------------------|
| `branch`  | string  | yes      | —       | Branch name of the worktree            |
| `drop_db` | boolean | no       | `true`  | Also drop the worktree's database      |

**Returns:** `WorktreeRemoveResult` — removed path, dropped database (if applicable).

---

## TUI Design

The TUI uses **Bubble Tea** (Elm architecture: Model → Update → View) with **Lip Gloss** for styling and **Bubbles** for standard components (lists, spinners, text inputs, viewports).

### Navigation

The interface is structured as tabbed panels, navigable with `Tab` / `Shift+Tab` or number keys.

```
 [1] Dashboard    [2] Worktrees    [3] Databases    [4] Dumps
```

### Dashboard Screen

Overview of the project status at a glance.

```
┌─ pm ── facility-saas ── symfony ───────────────────────────────────┐
│                                                                     │
│  Docker   ● database (mariadb:11)  ● php (running)  ○ redis (down) │
│                                                                     │
│  Worktrees   3 active     Databases   4 found     Dumps   2 files  │
│                                                                     │
│  Config   .claude/project.json ✔                                    │
│  DSN      mysql://root:***@database:3306/facility_app               │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  [1-4] navigate  [q] quit  [?] help                                │
└─────────────────────────────────────────────────────────────────────┘
```

### Worktrees Screen

```
┌─ Worktrees ─────────────────────────────────────────────────────────┐
│                                                                     │
│  Branch                      Path                         Database  │
│  ─────────────────────────────────────────────────────────────────── │
│  ▸ main                      ./                           (default) │
│    feature/booking-calendar  ../worktrees/feature-booking  wt_…cal  │
│    fix/auth-redirect         ../worktrees/fix-auth-redir   wt_…red  │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  [c] create  [d] delete  [enter] details  [q] back                 │
└─────────────────────────────────────────────────────────────────────┘
```

Creating a worktree opens an inline form:

```
┌─ Create Worktree ───────────────────────────────────────────────────┐
│                                                                     │
│  Branch name:  feature/█                                            │
│  New branch:   [x] yes  [ ] no                                     │
│  Clone DB:     [x] yes  [ ] no                                     │
│                                                                     │
│  Database will be: facility_app_wt_feature_…                        │
│                                                                     │
│  [enter] confirm  [esc] cancel                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Databases Screen

```
┌─ Databases ─────────────────────────────────────────────────────────┐
│                                                                     │
│  Name                                    Size      Status           │
│  ─────────────────────────────────────────────────────────────────── │
│  ▸ facility_app                          142 MB    default          │
│    facility_app_test                      12 MB                     │
│    facility_app_wt_feature_booking       142 MB    worktree         │
│    facility_app_wt_fix_auth_redirect     142 MB    worktree         │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  [d] dump  [i] import  [c] clone  [n] new  [D] drop  [q] back     │
└─────────────────────────────────────────────────────────────────────┘
```

Destructive actions show a confirmation dialog:

```
┌─ Confirm ────────────────────────────────────┐
│                                               │
│  Drop database "facility_app_wt_feature…"?   │
│                                               │
│  This action is irreversible.                 │
│  Type the database name to confirm:           │
│                                               │
│  > █                                          │
│                                               │
│  [enter] confirm  [esc] cancel                │
└───────────────────────────────────────────────┘
```

### Dumps Screen

```
┌─ Dumps ─────────────────────────────────────────────────────────────┐
│                                                                     │
│  File                                      Size     Created         │
│  ─────────────────────────────────────────────────────────────────── │
│  ▸ facility_app_2026-02-11T10-30.sql       48 MB    2 hours ago    │
│    facility_app_2026-02-10T18-00.sql       47 MB    yesterday      │
│    facility_app_test_2026-02-09.sql         5 MB    2 days ago     │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│  [i] import to…  [x] delete  [q] back                              │
└─────────────────────────────────────────────────────────────────────┘
```

### Progress Rendering

Long operations render as an inline spinner with stage information:

```
  ⠋ Cloning facility_app → facility_app_wt_feature_booking
    Dumping source database…                             [3s]
```

Completed:

```
  ✔ Cloned facility_app → facility_app_wt_feature_booking
    142 MB in 8.2s
```

---

## CLI Mode

One-shot commands for scripting and CI.

```bash
# Project
pm info
pm init

# Database
pm db list
pm db dump [--database=<n>] [--tables=<t1,t2>]
pm db import <file> [--database=<n>]
pm db create <n>
pm db drop <n> [--confirm]
pm db clone [--source=<n>] --target=<n>
pm db dumps

# Worktrees
pm wt list
pm wt create <branch> [--new-branch] [--no-db]
pm wt remove <branch> [--keep-db]
```

Destructive commands require `--confirm` flag or interactive confirmation (if TTY is attached).

Exit codes: `0` success, `1` command error, `2` config error.

Output: human-readable by default, `--json` flag for machine-readable output.

---

## Safety & Validation

### Config Validation

On startup (or first command), the tool:
1. Locates `.claude/project.json` relative to `PROJECT_ROOT` (or current working directory)
2. Validates against the JSON Schema
3. Resolves all `${VAR}` references from `.env` / `.env.local`
4. Parses the DSN
5. Verifies Docker Compose file exists

If any step fails, commands that depend on the failed section return a typed `CommandError`.

### Database Guards

- Every database operation checks the name against `allowed` patterns
- `db.drop` has an additional hard guard: refuses to drop the default database
- `db.import` verifies the SQL file exists before executing
- DSN credentials are never included in logs, TUI display (masked), or MCP tool responses

### Worktree Guards

- Branch/directory names are validated against `^[a-zA-Z0-9_\-\/]+$`
- Resolved worktree paths are checked to fall under `base_path` (prevents path traversal)
- `worktree.remove` uses `git worktree remove --force`

### Docker Guards

- All database commands use `docker compose exec -T` (no TTY allocation)
- The compose file path is validated on startup
- If the target service is not running, the command fails with `ErrDockerNotRunning`

---

## File Structure

```
mcp-project-manager/
├── cmd/
│   └── pm/
│       └── main.go               # Entry point: routes to TUI, MCP, or CLI
│
├── internal/
│   ├── core/                     # Pure logic, no I/O opinions
│   │   ├── config.go             # Load, validate .claude/project.json
│   │   ├── config_test.go
│   │   ├── env.go                # .env / .env.local parsing, ${VAR} resolution
│   │   ├── env_test.go
│   │   ├── dsn.go                # DSN parsing via net/url
│   │   ├── dsn_test.go
│   │   ├── guard.go              # Safety: allowed check, path traversal, default DB
│   │   ├── guard_test.go
│   │   ├── types.go              # Shared types: results, errors, progress
│   │   └── commands/
│   │       ├── database.go       # db.list, db.dump, db.import, db.create, db.drop, db.clone
│   │       ├── database_test.go
│   │       ├── worktree.go       # worktree.list, worktree.create, worktree.remove
│   │       ├── worktree_test.go
│   │       ├── project.go        # project.info, project.init
│   │       └── project_test.go
│   │
│   ├── executor/                 # Shell command wrappers
│   │   ├── docker.go             # docker compose exec, docker compose config
│   │   ├── git.go                # git worktree add/remove/list
│   │   └── executor.go           # Interface for testability (mock exec in tests)
│   │
│   ├── mcp/                      # MCP interface adapter
│   │   └── server.go             # Tool registration, core → MCP response mapping
│   │
│   ├── tui/                      # TUI interface adapter (Bubble Tea)
│   │   ├── app.go                # Root model, tab navigation
│   │   ├── screens/
│   │   │   ├── dashboard.go
│   │   │   ├── databases.go
│   │   │   ├── worktrees.go
│   │   │   └── dumps.go
│   │   ├── components/
│   │   │   ├── confirm.go        # Destructive action confirmation dialog
│   │   │   ├── form.go           # Inline form (create worktree, etc.)
│   │   │   ├── statusbar.go
│   │   │   ├── table.go
│   │   │   └── progress.go       # Spinner + stage info for long ops
│   │   └── styles/
│   │       └── theme.go          # Lip Gloss styles
│   │
│   └── cli/                      # CLI interface adapter
│       └── commands.go           # Cobra command tree → core calls
│
├── schema.json                   # JSON Schema for .claude/project.json
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Distribution

### Installation

```bash
# Go install
go install github.com/<user>/mcp-project-manager/cmd/pm@latest

# Or download binary from releases
curl -sL https://github.com/<user>/mcp-project-manager/releases/latest/download/pm-$(uname -s)-$(uname -m) -o pm
chmod +x pm
sudo mv pm /usr/local/bin/
```

### Usage

```bash
# Interactive TUI (default)
pm

# MCP server mode (for Claude Code)
pm --mcp

# One-shot CLI
pm db dump
pm wt create feature/foo --new-branch
```

### Claude Code Registration

```bash
claude mcp add project-manager \
  -e PROJECT_ROOT=/path/to/project \
  -- pm --mcp
```

For multiple projects, register with different `PROJECT_ROOT` values:

```bash
claude mcp add pm-facility  -e PROJECT_ROOT=/home/user/facility-saas  -- pm --mcp
claude mcp add pm-client-x  -e PROJECT_ROOT=/home/user/client-x      -- pm --mcp
```

---

## Testing Strategy

All core logic must be unit tested. The `Executor` interface enables testing without Docker, Git, or a filesystem.

### Executor Interface (for testability)

```go
type Executor interface {
    DockerExec(service string, command []string) (string, error)
    DockerComposeConfig() (*ComposeConfig, error)
    GitWorktreeList() ([]WorktreeInfo, error)
    GitWorktreeAdd(path, branch string, newBranch bool) error
    GitWorktreeRemove(path string) error
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte) error
    FileExists(path string) bool
}
```

The core library accepts an `Executor` — real implementation calls `os/exec`, test implementation returns canned responses.

### Test Layers

**Pure unit tests (no mocks needed):** Config parsing, env resolution, DSN parsing, guard checks, name sanitization. These are pure functions: input → output.

**Unit tests with mock executor:** All commands. Verify correct shell commands are built, correct sequence of executor calls, guards are checked before any executor call.

**Integration tests (optional, CI only):** Spin up real containers, run actual operations. Confirm mock assumptions match reality.

### Unit Test Cases

#### Config Parsing (`config_test.go`)

| # | Case | Input | Expected |
|---|------|-------|----------|
| 1 | Valid minimal config | `project` + `docker` only | Parses successfully |
| 2 | Valid full config | All sections and fields | Parses successfully |
| 3 | Missing `project` section | No `project` key | `ErrConfigInvalid` |
| 4 | Missing `project.name` | `project` without `name` | `ErrConfigInvalid` |
| 5 | Missing `project.type` | `project` without `type` | `ErrConfigInvalid` |
| 6 | Unknown `project.type` | `"type": "django"` | `ErrConfigInvalid` |
| 7 | Missing `docker` section | No `docker` key | `ErrConfigInvalid` |
| 8 | Missing `database.service` | `database` section without `service` | `ErrConfigInvalid` |
| 9 | Missing `database.dsn` | `database` section without `dsn` | `ErrConfigInvalid` |
| 10 | Missing `database.allowed` | `database` section without `allowed` | `ErrConfigInvalid` |
| 11 | Empty `database.allowed` | `"allowed": []` | `ErrConfigInvalid` |
| 12 | Missing `worktrees.base_path` | `worktrees` section without `base_path` | `ErrConfigInvalid` |
| 13 | Config file not found | Non-existent path | `ErrConfigMissing` |
| 14 | Invalid JSON | Malformed JSON | `ErrConfigInvalid` |
| 15 | No `database` section | Omitted entirely | Valid, `config.Database` is nil |
| 16 | No `worktrees` section | Omitted entirely | Valid, `config.Worktrees` is nil |
| 17 | Default `compose_file` | `docker` section without `compose_file` | Defaults to `docker-compose.yaml` |
| 18 | Default `dumps_path` | `database` section without `dumps_path` | Defaults to `var/dumps` |
| 19 | Default `db_prefix` | `worktrees` section without `db_prefix` | Defaults to `<default_db>_wt_` |

#### Env Resolution (`env_test.go`)

| # | Case | Input | Expected |
|---|------|-------|----------|
| 1 | Resolve from `.env` | `${DB_NAME}`, `.env` has `DB_NAME=app` | `"app"` |
| 2 | Resolve from `.env.local` | `${DB_NAME}`, `.env.local` has `DB_NAME=local_app` | `"local_app"` |
| 3 | `.env.local` takes precedence | Both files define `DB_NAME` | Value from `.env.local` |
| 4 | Plain string passthrough | `"mysql://root:pw@db/app"` | `"mysql://root:pw@db/app"` |
| 5 | Missing variable | `${NONEXISTENT}` | `ErrEnvVarNotFound` |
| 6 | Double-quoted value | `VAR="value"` | `"value"` |
| 7 | Single-quoted value | `VAR='value'` | `"value"` |
| 8 | Value containing `=` | `VAR=foo=bar=baz` | `"foo=bar=baz"` |
| 9 | Empty value | `VAR=` | `""` |
| 10 | Comment lines skipped | `# this is a comment` | Skipped |
| 11 | Empty lines skipped | Blank lines between vars | Skipped |
| 12 | Multiple `${}` in one string | `${USER}:${PASS}@${HOST}` | All three resolved |
| 13 | `.env.local` missing | Only `.env` exists | Resolves from `.env` |
| 14 | Both files missing | Neither exists | `ErrEnvVarNotFound` for any `${}` |
| 15 | Var with leading/trailing whitespace in value | `VAR= value ` | Trimmed or preserved (decide convention) |
| 16 | Line with only whitespace | `   ` | Skipped |
| 17 | Var name with no `${}` wrapping | `DATABASE_URL` (literal) | Returned as-is (plain string) |
| 18 | Partial `${` without closing `}` | `${BROKEN` | Returned as-is (not a valid reference) |

#### DSN Parsing (`dsn_test.go`)

| # | Case | Input | Expected |
|---|------|-------|----------|
| 1 | Standard MySQL DSN | `mysql://root:secret@database:3306/app` | engine=`mysql`, user=`root`, pass=`secret`, host=`database`, port=`3306`, db=`app` |
| 2 | MariaDB via serverVersion | `mysql://root:secret@db:3306/app?serverVersion=mariadb-11.4` | engine=`mariadb` |
| 3 | MySQL via serverVersion | `mysql://root:secret@db:3306/app?serverVersion=8.0` | engine=`mysql` |
| 4 | PostgreSQL DSN | `postgresql://user:pass@db:5432/app` | engine=`postgres`, port=`5432` |
| 5 | Missing port (MySQL) | `mysql://root:pass@db/app` | port defaults to `3306` |
| 6 | Missing port (Postgres) | `postgresql://user:pass@db/app` | port defaults to `5432` |
| 7 | URL-encoded password | `mysql://root:p%40ss%3Aw%2Frd@db:3306/app` | password=`p@ss:w/rd` |
| 8 | No serverVersion param | `mysql://root:pass@db:3306/app` | engine=`mysql` (default for scheme) |
| 9 | Extra query params | `mysql://root:pass@db:3306/app?serverVersion=mariadb-11.4&charset=utf8mb4` | Parsed, extra params ignored |
| 10 | Empty string | `""` | Error |
| 11 | Not a URL | `not-a-dsn` | Error |
| 12 | Missing database name | `mysql://root:pass@db:3306/` | Error |
| 13 | Missing user | `mysql://:pass@db:3306/app` | user=`""` (or error — decide convention) |
| 14 | `postgres://` alias | `postgres://user:pass@db/app` | engine=`postgres` (alias for `postgresql`) |

#### Guard: Allowed Database Check (`guard_test.go`)

| # | Case | DB Name | Allowed Patterns | Expected |
|---|------|---------|-----------------|----------|
| 1 | Exact match | `facility_app` | `["facility_app"]` | Allowed |
| 2 | No match | `other_db` | `["facility_app"]` | `ErrDbNotAllowed` |
| 3 | Wildcard match | `facility_app_wt_foo` | `["facility_app_wt_*"]` | Allowed |
| 4 | Wildcard no match | `other_wt_foo` | `["facility_app_wt_*"]` | `ErrDbNotAllowed` |
| 5 | Multiple patterns, one matches | `facility_app_test` | `["facility_app", "facility_app_test"]` | Allowed |
| 6 | Multiple patterns, none match | `random_db` | `["facility_app", "facility_app_test"]` | `ErrDbNotAllowed` |
| 7 | Wildcard `*` alone | `anything` | `["*"]` | Allowed |
| 8 | Multiple wildcards | `foo_test_bar` | `["*_test_*"]` | Allowed |
| 9 | Mixed exact and wildcard | `facility_app_wt_xyz` | `["facility_app", "facility_app_wt_*"]` | Allowed |
| 10 | Empty allowed list | `facility_app` | `[]` | `ErrDbNotAllowed` (defensive, normally caught by config validation) |

#### Guard: Default DB Protection (`guard_test.go`)

| # | Case | Operation | DB Name | Default DB | Expected |
|---|------|-----------|---------|------------|----------|
| 1 | Drop default DB | `drop` | `facility_app` | `facility_app` | `ErrDbIsDefault` |
| 2 | Drop non-default DB | `drop` | `facility_app_wt_foo` | `facility_app` | Allowed |
| 3 | Dump default DB | `dump` | `facility_app` | `facility_app` | Allowed |
| 4 | Import into default DB | `import` | `facility_app` | `facility_app` | Allowed |
| 5 | Clone from default DB | `clone` (source) | `facility_app` | `facility_app` | Allowed |

#### Guard: Path Traversal (`guard_test.go`)

| # | Case | Branch Name | Base Path | Expected |
|---|------|-------------|-----------|----------|
| 1 | Normal branch | `feature/foo` | `../worktrees` | Valid |
| 2 | Path traversal with `..` | `feature/../../../etc/passwd` | `../worktrees` | `ErrPathTraversal` |
| 3 | Resolved path outside base | (constructed to escape) | `../worktrees` | `ErrPathTraversal` |
| 4 | Resolved path inside base | `feature/foo` | `../worktrees` | Valid |
| 5 | Invalid chars (semicolon) | `feature/foo;rm -rf` | `../worktrees` | `ErrInvalidName` |
| 6 | Invalid chars (space) | `feature foo` | `../worktrees` | `ErrInvalidName` |
| 7 | Empty branch name | `""` | `../worktrees` | `ErrInvalidName` |
| 8 | Branch with dots (valid) | `fix/1.2.3` | `../worktrees` | Decide: valid or invalid |

#### Name Sanitization (`sanitize_test.go`)

| # | Case | Input | Dir Output | DB Output |
|---|------|-------|------------|-----------|
| 1 | Slashes to hyphens/underscores | `feature/booking-calendar` | `feature-booking-calendar` | `feature_booking_calendar` |
| 2 | Simple branch | `main` | `main` | `main` |
| 3 | Nested slashes | `feature/api/v2` | `feature-api-v2` | `feature_api_v2` |
| 4 | Already clean | `hotfix` | `hotfix` | `hotfix` |
| 5 | Hyphens in branch | `fix-auth-bug` | `fix-auth-bug` | `fix_auth_bug` |
| 6 | Mixed slashes and hyphens | `feature/my-cool-feature` | `feature-my-cool-feature` | `feature_my_cool_feature` |

#### Commands with Mock Executor

These tests verify that commands build the correct executor calls and enforce guards.

##### `db.dump` (`database_test.go`)

| # | Case | Expected |
|---|------|----------|
| 1 | Dump default DB (MariaDB) | Calls `DockerExec("database", ["mariadb-dump", ...])` |
| 2 | Dump default DB (MySQL) | Calls `DockerExec("database", ["mysqldump", ...])` |
| 3 | Dump default DB (Postgres) | Calls `DockerExec("database", ["pg_dump", ...])` |
| 4 | Dump specific tables | Table names included in command args |
| 5 | Dump disallowed DB | `ErrDbNotAllowed` before any executor call |
| 6 | No database config | `ErrConfigMissing` (or appropriate section-missing error) |
| 7 | Result contains path, size, duration | Verify `DumpResult` struct fields populated |

##### `db.import` (`database_test.go`)

| # | Case | Expected |
|---|------|----------|
| 1 | Import into default DB | Calls `DockerExec` with correct import command |
| 2 | Import into specific DB | Correct DB name in command |
| 3 | SQL file not found | `ErrFileNotFound` before any executor call |
| 4 | Target DB not allowed | `ErrDbNotAllowed` before any executor call |

##### `db.create` (`database_test.go`)

| # | Case | Expected |
|---|------|----------|
| 1 | Create allowed DB | Calls `DockerExec` with `CREATE DATABASE` |
| 2 | Create disallowed DB | `ErrDbNotAllowed` before any executor call |
| 3 | DB name validated | Invalid characters → `ErrInvalidName` |

##### `db.drop` (`database_test.go`)

| # | Case | Expected |
|---|------|----------|
| 1 | Drop allowed non-default DB | Calls `DockerExec` with `DROP DATABASE` |
| 2 | Drop default DB | `ErrDbIsDefault` before any executor call |
| 3 | Drop disallowed DB | `ErrDbNotAllowed` before any executor call |

##### `db.clone` (`database_test.go`)

| # | Case | Expected |
|---|------|----------|
| 1 | Clone default → new DB | Calls dump, create, import in sequence |
| 2 | Target not allowed | `ErrDbNotAllowed` before any executor call |
| 3 | Source not allowed | `ErrDbNotAllowed` before any executor call |
| 4 | Progress callback fired | `StageDumping`, `StageCreating`, `StageImporting` in order |

##### `db.list` (`database_test.go`)

| # | Case | Expected |
|---|------|----------|
| 1 | Lists databases | Calls `DockerExec` with `SHOW DATABASES`, parses output |
| 2 | System databases filtered | `information_schema`, `mysql`, `performance_schema`, `sys` excluded |
| 3 | Default DB marked | Result marks which DB is the default |

##### `worktree.create` (`worktree_test.go`)

| # | Case | Expected |
|---|------|----------|
| 1 | Create with existing branch | Calls `GitWorktreeAdd(path, branch, false)` |
| 2 | Create with new branch | Calls `GitWorktreeAdd(path, branch, true)` |
| 3 | Create with `clone_db=true` | Calls worktree add, then db.create, db.clone, patches `.env.local` |
| 4 | Create with `clone_db=false` | Only calls worktree add, no DB operations |
| 5 | Invalid branch name | `ErrInvalidName` before any executor call |
| 6 | Path traversal in branch | `ErrPathTraversal` before any executor call |
| 7 | Worktree dir already exists | Error from executor, propagated |
| 8 | Progress callback fired | Correct stages in order |
| 9 | `.env.local` patching | `WriteFile` called with updated `DATABASE_URL` |

##### `worktree.remove` (`worktree_test.go`)

| # | Case | Expected |
|---|------|----------|
| 1 | Remove with `drop_db=true` | Calls `GitWorktreeRemove`, then `db.drop` |
| 2 | Remove with `drop_db=false` | Only calls `GitWorktreeRemove` |
| 3 | Invalid branch name | `ErrInvalidName` before any executor call |
| 4 | Path traversal in branch | `ErrPathTraversal` before any executor call |

##### `worktree.list` (`worktree_test.go`)

| # | Case | Expected |
|---|------|----------|
| 1 | Lists worktrees | Calls `GitWorktreeList`, returns parsed results |
| 2 | Main worktree identified | Result marks which is the main worktree |
| 3 | Associated databases detected | Worktrees with matching `db_prefix` databases linked |

### Integration Tests (CI only)

These are optional and run against real services. They validate that the mock executor's assumptions match reality.

- **Docker + MariaDB:** Spin up a container, run dump/import/create/drop, verify SQL files and databases
- **Docker + PostgreSQL:** Same as above for Postgres-specific commands
- **Git:** Create a temp repo with commits, test worktree add/remove/list
- **Config discovery:** Create temp directories with various `.env` / `.env.local` combinations, verify resolution

---

## Future Considerations

These are not part of the initial implementation but are logical extensions:

- **`console` command** — run Symfony `bin/console` or Laravel `artisan` commands inside the app container
- **`docker` commands** — start/stop/restart Docker Compose services from the TUI
- **`db.snapshot` / `db.restore`** — named snapshots for quick save/restore during development
- **PostgreSQL support** — DSN parsing and guard logic are engine-agnostic; only dump/import commands need engine-specific variants. TODO: Implement `PostgresEngine` with `pg_dump`, `psql`, `createdb`, `dropdb` commands. (Out of scope for current phases)
- **Multi-compose support** — projects with multiple compose files (`docker-compose.yaml` + `docker-compose.override.yaml`)
- **Hooks** — pre/post hooks for operations (e.g., `composer install` after creating a worktree, `bin/console doctrine:migrations:migrate` after cloning a DB)
- **Config profiles** — multiple database sections for projects with multiple database services
- **Remote project support** — SSH-based executor for managing projects on remote servers

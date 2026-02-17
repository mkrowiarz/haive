# AGENTS.md

This file contains guidelines for agents working on this repository.

## Build / Lint / Test Commands

```bash
# Build the main binary (local development)
go build -o haive ./cmd/haive

# Install to ~/go/bin using 'go install' (recommended for use)
make install

# Install to ~/.local/bin
make install-local

# Build for multiple platforms
go build -o haive-linux ./cmd/haive
GOOS=darwin GOARCH=arm64 go build -o haive-mac-arm64 ./cmd/haive

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests in verbose mode
go test -v ./...

# Run a specific test
go test -v ./internal/core -run TestDumpDefaultDB

# Run tests in a specific package
go test ./internal/core/commands

# Benchmark tests
go test -bench=. ./...

# Lint with golangci-lint
golangci-lint run

# Format code
go fmt ./...

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify
```

## Code Style Guidelines

### Project Structure

- `cmd/haive/` - Application entry point (routes to TUI, MCP, or CLI)
- `internal/core/` - Pure logic, no I/O opinions (config, commands, types)
- `internal/executor/` - Shell command wrappers (docker, git, filesystem)
- `internal/mcp/` - MCP interface adapter
- `internal/tui/` - TUI interface adapter (Bubble Tea)
- `internal/cli/` - CLI interface adapter

### Formatting

- Use `gofmt` or `go fmt ./...` for standard formatting
- Maximum line length: 100 characters (not 80)
- Use `gofumpt` for stricter formatting if available

### Imports

- Standard library → third-party → internal packages
- Use blank lines between import groups
- Prefer named imports for packages with generic names:
  ```go
  import (
      "fmt"
      teamodels "github.com/charmbracelet/bubbletea"
      "github.com/charmbracelet/lipgloss"
      pmcore "mcp-project-manager/internal/core"
  )
  ```

### Naming Conventions

- **Interfaces**: Single verb or noun ending in -er: `Executor`, `Parser`, `Validator`
- **Package names**: Lowercase, single word, descriptive: `core`, `executor`, `tui`
- **Constants**: PascalCase: `ErrConfigMissing`, `StageDumping`
- **Variables**: camelCase
- **Private fields**: camelCase with lowercase first letter
- **Public fields**: PascalCase
- **Files**: Lowercase, match package name: `config.go`, `executor.go`

### Types

- Use structs for data containers
- Use typed errors with codes: `type ErrCode string` + `CommandError` struct
- Return typed results, not formatted strings (core library)
- Define progress callbacks as types: `type ProgressFunc func(stage ProgressStage, detail string)`

### Error Handling

- Never ignore errors (use `_` only if explicitly justified)
- Wrap errors with context: `fmt.Errorf("failed to load config: %w", err)`
- Return typed `CommandError` from core commands with error codes
- Define error codes as constants: `ErrConfigMissing`, `ErrDbNotAllowed`, etc.
- Use `errors.Is()` and `errors.As()` for error checking

### Testing

- Write table-driven tests for functions with multiple cases
- Test pure functions without mocks (config parsing, env resolution, DSN parsing)
- Mock `Executor` interface for command tests (no real Docker/Git calls)
- Test file naming: `<source>_test.go` (e.g., `config_test.go`)
- Run single test: `go test -v ./internal/core -run TestSpecificCase`
- Use `t.Parallel()` for independent tests

### Bubble Tea TUI

- Use Elm architecture: Model → Update → View
- Define message types as structs: `type TickMsg time.Time`, `type ErrMsg error`
- Keep models pure; use `tea.Cmd` for side effects
- Use Lip Gloss for consistent styling
- Define styles in `tui/styles/theme.go`

### CLI (Cobra)

- Use snake_case for command names: `db dump`, `worktree create`
- Provide `--help` for all commands
- Use flags for options: `--database`, `--tables`
- Return exit codes: 0 (success), 1 (command error), 2 (config error)

### Config & DSN

- Config file: `.claude/project.json` (relative to project root)
- Resolve `${VAR_NAME}` from `.env` and `.env.local` files
- Parse DSN using `net/url` package
- Mask credentials in logs and TUI display

### Safety & Guards

- Check `allowed` patterns before any database operation
- Refuse to drop the default database
- Validate worktree paths against traversal attacks
- Return typed errors before calling executor (fail fast)

### Git & Jujutsu Conventions

This repository supports both Git and Jujutsu workflows. Both can push to GitHub.

**Conventional commits**: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`
- Keep commits/changesets atomic (one logical change per commit)
- Write descriptive commit messages explaining "why", not just "what"

**Git workflow**:
```bash
git status          # Check working tree status
git diff            # View staged and unstaged changes
git add .           # Stage changes
git commit -m "..." # Commit with message
git push            # Push to GitHub
```

**Jujutsu workflow**:
```bash
jj status           # Check working copy status
jj diff             # View changes
jj new <description> # Create new change (no editor needed)
jj describe <change-id> # Update change description
jj git push         # Push to GitHub (syncs Git HEAD)
```

**Jujutsu with GitHub**:
- Use `jj git clone <url>` to clone from GitHub
- Use `jj git push` to push changes back to GitHub
- `jj` automatically creates Git commits when pushing
- Use `jj git fetch` to pull changes from GitHub
- Use `gh` CLI for GitHub operations (PRs, issues) alongside `jj`

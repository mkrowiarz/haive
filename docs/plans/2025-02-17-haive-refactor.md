# Haive Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform `pm` into `haive` - a modular development environment manager with TOML-based configuration, worktree management with hooks/copy patterns, and database lifecycle operations.

**Architecture:** Modular design with Worktree and Database modules. Config loader supports TOML (primary), YAML, and JSON (legacy). Hooks system executes shell commands at lifecycle points. Docker remains an execution mechanism, not a module.

**Tech Stack:** Go 1.25, TOML parsing (BurntSushi/toml), YAML (gopkg.in/yaml.v3), glob matching (doublestar/v4), Bubble Tea TUI, Cobra CLI, MCP server.

---

## Pre-Requisites

- Review current codebase structure: `internal/core/`, `internal/executor/`, `internal/tui/`, `internal/mcp/`
- Understand existing config loading in `internal/core/config/config.go`
- Understand existing types in `internal/core/types/types.go`

---

## Task 1: Add TOML Dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add TOML parser dependency**

```bash
go get github.com/BurntSushi/toml
```

**Step 2: Verify go.mod updated**

Run: `cat go.mod | grep toml`
Expected: Line showing `github.com/BurntSushi/toml`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add BurntSushi/toml for TOML config support"
```

---

## Task 2: Create Modular Config Types

**Files:**
- Create: `internal/core/config/types.go`

**Step 1: Define module interfaces**

```go
package config

// Module is the interface all modules implement
type Module interface {
	Name() string
	Validate() error
}

// WorktreeConfig holds worktree module configuration
type WorktreeConfig struct {
	BasePath      string           `toml:"base_path" yaml:"base_path" json:"base_path"`
	DBPerWorktree bool             `toml:"db_per_worktree,omitempty" yaml:"db_per_worktree,omitempty" json:"db_per_worktree,omitempty"`
	DBPrefix      string           `toml:"db_prefix,omitempty" yaml:"db_prefix,omitempty" json:"db_prefix,omitempty"`
	Copy          *CopyConfig      `toml:"copy,omitempty" yaml:"copy,omitempty" json:"copy,omitempty"`
	Hooks         *WorktreeHooks   `toml:"hooks,omitempty" yaml:"hooks,omitempty" json:"hooks,omitempty"`
	Env           *EnvConfig       `toml:"env,omitempty" yaml:"env,omitempty" json:"env,omitempty"`
}

func (w *WorktreeConfig) Name() string { return "worktree" }

func (w *WorktreeConfig) Validate() error {
	if w.BasePath == "" {
		return fmt.Errorf("worktree.base_path is required")
	}
	return nil
}

// CopyConfig holds file copy patterns
type CopyConfig struct {
	Include []string `toml:"include,omitempty" yaml:"include,omitempty" json:"include,omitempty"`
	Exclude []string `toml:"exclude,omitempty" yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// WorktreeHooks holds worktree lifecycle hooks
type WorktreeHooks struct {
	PostCreate []string `toml:"postCreate,omitempty" yaml:"postCreate,omitempty" json:"postCreate,omitempty"`
	PreRemove  []string `toml:"preRemove,omitempty" yaml:"preRemove,omitempty" json:"preRemove,omitempty"`
	PostRemove []string `toml:"postRemove,omitempty" yaml:"postRemove,omitempty" json:"postRemove,omitempty"`
}

// EnvConfig holds per-worktree environment configuration
type EnvConfig struct {
	File    string `toml:"file" yaml:"file" json:"file"`
	VarName string `toml:"var_name" yaml:"var_name" json:"var_name"`
}

// DatabaseConfig holds database module configuration
type DatabaseConfig struct {
	Service   string          `toml:"service" yaml:"service" json:"service"`
	DSN       string          `toml:"dsn" yaml:"dsn" json:"dsn"`
	Allowed   []string        `toml:"allowed" yaml:"allowed" json:"allowed"`
	DumpsPath string          `toml:"dumps_path,omitempty" yaml:"dumps_path,omitempty" json:"dumps_path,omitempty"`
	Hooks     *DatabaseHooks  `toml:"hooks,omitempty" yaml:"hooks,omitempty" json:"hooks,omitempty"`
}

func (d *DatabaseConfig) Name() string { return "database" }

func (d *DatabaseConfig) Validate() error {
	if d.Service == "" {
		return fmt.Errorf("database.service is required")
	}
	if d.DSN == "" {
		return fmt.Errorf("database.dsn is required")
	}
	if len(d.Allowed) == 0 {
		return fmt.Errorf("database.allowed is required")
	}
	return nil
}

// DatabaseHooks holds database lifecycle hooks
type DatabaseHooks struct {
	PostClone []string `toml:"postClone,omitempty" yaml:"postClone,omitempty" json:"postClone,omitempty"`
	PreDrop   []string `toml:"preDrop,omitempty" yaml:"preDrop,omitempty" json:"preDrop,omitempty"`
}

// ProjectConfig holds project metadata
type ProjectConfig struct {
	Name   string `toml:"name" yaml:"name" json:"name"`
	Preset string `toml:"preset,omitempty" yaml:"preset,omitempty" json:"preset,omitempty"`
}

// DockerConfig holds Docker settings
type DockerConfig struct {
	ComposeFiles []string `toml:"compose_files,omitempty" yaml:"compose_files,omitempty" json:"compose_files,omitempty"`
	ProjectName  string   `toml:"project_name,omitempty" yaml:"project_name,omitempty" json:"project_name,omitempty"`
}

// HaiveConfig is the top-level configuration structure
type HaiveConfig struct {
	Project    ProjectConfig     `toml:"project" yaml:"project" json:"project"`
	Docker     DockerConfig      `toml:"docker" yaml:"docker" json:"docker"`
	Worktree   *WorktreeConfig   `toml:"worktree,omitempty" yaml:"worktree,omitempty" json:"worktree,omitempty"`
	Database   *DatabaseConfig   `toml:"database,omitempty" yaml:"database,omitempty" json:"database,omitempty"`
	ProjectRoot string           `toml:"-" yaml:"-" json:"-"` // Set at runtime
}

// Validate validates the entire configuration
func (c *HaiveConfig) Validate() error {
	if c.Project.Name == "" {
		return fmt.Errorf("project.name is required")
	}
	
	if c.Worktree != nil {
		if err := c.Worktree.Validate(); err != nil {
			return err
		}
	}
	
	if c.Database != nil {
		if err := c.Database.Validate(); err != nil {
			return err
		}
	}
	
	// Validate worktree.env requires database
	if c.Worktree != nil && c.Worktree.Env != nil && c.Database == nil {
		return fmt.Errorf("worktree.env requires database configuration")
	}
	
	return nil
}
```

**Step 2: Run go build to check for errors**

Run: `go build ./internal/core/config/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/core/config/types.go
git commit -m "feat: add modular config types for Haive"
```

---

## Task 3: Create TOML/YAML/JSON Config Loader

**Files:**
- Create: `internal/core/config/loader.go`
- Create: `internal/core/config/loader_test.go`

**Step 1: Write the config loader**

```go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// ConfigFile represents a discovered config file
type ConfigFile struct {
	Path     string
	Format   string // "toml", "yaml", "json"
	Priority int
}

// Loader handles config file discovery and parsing
type Loader struct {
	searchPaths []ConfigFile
}

// NewLoader creates a new config loader with default search paths
func NewLoader() *Loader {
	return &Loader{
		searchPaths: []ConfigFile{
			{Path: "haive.toml", Format: "toml", Priority: 1},
			{Path: ".haive/config.toml", Format: "toml", Priority: 2},
			{Path: "haive.yaml", Format: "yaml", Priority: 3},
			{Path: ".haive/config.yaml", Format: "yaml", Priority: 4},
			{Path: "haive.json", Format: "json", Priority: 5},
			{Path: ".haive/config.json", Format: "json", Priority: 6},
			{Path: ".claude/project.json", Format: "json", Priority: 7},
		},
	}
}

// Load discovers and loads config from project root or parent directories
func (l *Loader) Load(startDir string) (*HaiveConfig, error) {
	searchDir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	for {
		for _, cf := range l.searchPaths {
			configPath := filepath.Join(searchDir, cf.Path)
			
			if _, err := os.Stat(configPath); err != nil {
				continue // File doesn't exist
			}
			
			cfg, err := l.parseFile(configPath, cf.Format)
			if err != nil {
				// File exists but is invalid - continue searching
				continue
			}
			
			if cfg == nil || !l.hasContent(cfg) {
				continue // File exists but has no haive content
			}
			
			cfg.ProjectRoot = searchDir
			return cfg, nil
		}
		
		// Move to parent directory
		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			break // Reached root
		}
		searchDir = parent
	}
	
	return nil, fmt.Errorf("config file not found in %s or parent directories", startDir)
}

func (l *Loader) parseFile(path string, format string) (*HaiveConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg HaiveConfig
	
	switch format {
	case "toml":
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse TOML: %w", err)
		}
	case "yaml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case "json":
		// Try namespaced format first (legacy .haive.json)
		var wrapper struct {
			PM *HaiveConfig `json:"pm"`
		}
		if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.PM != nil && l.hasContent(wrapper.PM) {
			return wrapper.PM, nil
		}
		
		// Try direct format
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
	
	return &cfg, nil
}

func (l *Loader) hasContent(cfg *HaiveConfig) bool {
	return cfg.Project.Name != "" ||
		cfg.Docker.ComposeFiles != nil ||
		cfg.Worktree != nil ||
		cfg.Database != nil
}

// Load is the convenience function (replaces old Load)
func Load(projectRoot string) (*HaiveConfig, error) {
	loader := NewLoader()
	cfg, err := loader.Load(projectRoot)
	if err != nil {
		return nil, err
	}
	
	// Resolve environment variables in DSN
	if cfg.Database != nil {
		cfg.Database.DSN = ResolveEnvVars(cfg.Database.DSN, cfg.ProjectRoot)
	}
	
	// Set default dumps path
	if cfg.Database != nil && cfg.Database.DumpsPath == "" {
		cfg.Database.DumpsPath = "var/dumps"
	}
	
	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	
	return cfg, nil
}
```

**Step 2: Write tests**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Load_TOML(t *testing.T) {
	tmpDir := t.TempDir()
	
	configContent := `
[project]
name = "test-project"

[docker]
compose_files = ["compose.yaml"]

[worktree]
base_path = ".worktrees"

[database]
service = "database"
dsn = "mysql://user:pass@db:3306/test"
allowed = ["test", "test_*"]
`
	
	configPath := filepath.Join(tmpDir, "haive.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	loader := NewLoader()
	cfg, err := loader.Load(tmpDir)
	
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	
	if cfg.Project.Name != "test-project" {
		t.Errorf("expected project name 'test-project', got '%s'", cfg.Project.Name)
	}
	
	if cfg.Worktree == nil || cfg.Worktree.BasePath != ".worktrees" {
		t.Error("expected worktree config with base_path '.worktrees'")
	}
}

func TestLoader_Load_Priority(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create both TOML and JSON configs
	tomlContent := `[project]
name = "toml-project"`
	jsonContent := `{"project": {"name": "json-project"}}`
	
	os.WriteFile(filepath.Join(tmpDir, "haive.toml"), []byte(tomlContent), 0644)
	os.WriteFile(filepath.Join(tmpDir, "haive.json"), []byte(jsonContent), 0644)
	
	loader := NewLoader()
	cfg, err := loader.Load(tmpDir)
	
	if err != nil {
		t.Fatal(err)
	}
	
	// TOML should take priority
	if cfg.Project.Name != "toml-project" {
		t.Errorf("expected TOML config to win, got: %s", cfg.Project.Name)
	}
}

func TestLoader_Load_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	
	loader := NewLoader()
	_, err := loader.Load(tmpDir)
	
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestLoader_Load_NamespacedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	
	configContent := `{
		"pm": {
			"project": {"name": "namespaced"},
			"docker": {"compose_files": ["docker-compose.yaml"]}
		}
	}`
	
	os.WriteFile(filepath.Join(tmpDir, ".claude", "project.json"), []byte(configContent), 0644)
	os.MkdirAll(filepath.Join(tmpDir, ".claude"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".claude", "project.json"), []byte(configContent), 0644)
	
	loader := NewLoader()
	cfg, err := loader.Load(tmpDir)
	
	if err != nil {
		t.Fatal(err)
	}
	
	if cfg.Project.Name != "namespaced" {
		t.Errorf("expected 'namespaced', got '%s'", cfg.Project.Name)
	}
}

func TestHaiveConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     HaiveConfig
		wantErr bool
	}{
		{
			name: "valid minimal",
			cfg: HaiveConfig{
				Project: ProjectConfig{Name: "test"},
				Docker:  DockerConfig{ComposeFiles: []string{"compose.yaml"}},
			},
			wantErr: false,
		},
		{
			name:    "missing project name",
			cfg:     HaiveConfig{},
			wantErr: true,
		},
		{
			name: "worktree without base_path",
			cfg: HaiveConfig{
				Project:  ProjectConfig{Name: "test"},
				Worktree: &WorktreeConfig{},
			},
			wantErr: true,
		},
		{
			name: "env without database",
			cfg: HaiveConfig{
				Project: ProjectConfig{Name: "test"},
				Worktree: &WorktreeConfig{
					BasePath: ".worktrees",
					Env:      &EnvConfig{File: ".env.local", VarName: "DATABASE_URL"},
				},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/core/config -v -run TestLoader`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/core/config/loader.go internal/core/config/loader_test.go
git commit -m "feat: add modular config loader with TOML/YAML/JSON support"
```

---

## Task 4: Create Hook Executor

**Files:**
- Create: `internal/core/hooks/executor.go`
- Create: `internal/core/hooks/executor_test.go`

**Step 1: Write hook executor**

```go
package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HookContext provides environment variables and working directory for hooks
type HookContext struct {
	// Common
	RepoRoot    string
	ProjectName string
	
	// Worktree-specific
	WorktreePath string
	WorktreeName string
	Branch       string
	
	// Database-specific
	DatabaseName   string
	DatabaseURL    string
	SourceDatabase string
	TargetDatabase string
}

// ToEnv returns the context as environment variable slice
func (c *HookContext) ToEnv() []string {
	env := os.Environ()
	
	// Common
	env = append(env, fmt.Sprintf("REPO_ROOT=%s", c.RepoRoot))
	env = append(env, fmt.Sprintf("PROJECT_NAME=%s", c.ProjectName))
	
	// Worktree
	if c.WorktreePath != "" {
		env = append(env, fmt.Sprintf("WORKTREE_PATH=%s", c.WorktreePath))
	}
	if c.WorktreeName != "" {
		env = append(env, fmt.Sprintf("WORKTREE_NAME=%s", c.WorktreeName))
	}
	if c.Branch != "" {
		env = append(env, fmt.Sprintf("BRANCH=%s", c.Branch))
	}
	
	// Database
	if c.DatabaseName != "" {
		env = append(env, fmt.Sprintf("DATABASE_NAME=%s", c.DatabaseName))
	}
	if c.DatabaseURL != "" {
		env = append(env, fmt.Sprintf("DATABASE_URL=%s", c.DatabaseURL))
	}
	if c.SourceDatabase != "" {
		env = append(env, fmt.Sprintf("SOURCE_DATABASE=%s", c.SourceDatabase))
	}
	if c.TargetDatabase != "" {
		env = append(env, fmt.Sprintf("TARGET_DATABASE=%s", c.TargetDatabase))
	}
	
	return env
}

// Executor runs hook commands
type Executor struct {
	ProjectRoot string
}

// NewExecutor creates a new hook executor
func NewExecutor(projectRoot string) *Executor {
	return &Executor{ProjectRoot: projectRoot}
}

// ExecuteHooks runs a list of hooks and returns error if any fail
// For pre-hooks, non-zero exit stops execution and returns error
// For post-hooks, non-zero exit is logged but doesn't stop
func (e *Executor) ExecuteHooks(hooks []string, ctx *HookContext, workingDir string, isPre bool) error {
	for _, hook := range hooks {
		if err := e.executeHook(hook, ctx, workingDir, isPre); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) executeHook(hook string, ctx *HookContext, workingDir string, isPre bool) error {
	// Check if hook is a script file or command
	var cmd *exec.Cmd
	
	if isScriptFile(hook) {
		scriptPath := hook
		if !filepath.IsAbs(scriptPath) {
			scriptPath = filepath.Join(e.ProjectRoot, scriptPath)
		}
		
		// Check if script exists
		if _, err := os.Stat(scriptPath); err != nil {
			if isPre {
				return fmt.Errorf("pre-hook script not found: %s", scriptPath)
			}
			fmt.Fprintf(os.Stderr, "Warning: post-hook script not found: %s\n", scriptPath)
			return nil
		}
		
		cmd = exec.Command(scriptPath)
	} else {
		// Run as shell command
		cmd = exec.Command("sh", "-c", hook)
	}
	
	cmd.Dir = workingDir
	cmd.Env = ctx.ToEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		if isPre {
			return fmt.Errorf("pre-hook failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Warning: post-hook failed: %v\n", err)
	}
	
	return nil
}

func isScriptFile(hook string) bool {
	// If it contains a path separator and the file exists, treat as script
	if strings.Contains(hook, "/") || strings.Contains(hook, "\\") {
		return true
	}
	return false
}
```

**Step 2: Write tests**

```go
package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHookContext_ToEnv(t *testing.T) {
	ctx := &HookContext{
		RepoRoot:       "/project",
		ProjectName:    "myapp",
		WorktreePath:   "/project/.worktrees/feature",
		WorktreeName:   "feature",
		Branch:         "feature/test",
		DatabaseName:   "myapp_feature",
		DatabaseURL:    "mysql://user:pass@db:3306/myapp_feature",
		SourceDatabase: "myapp",
		TargetDatabase: "myapp_feature",
	}
	
	env := ctx.ToEnv()
	
	expectedVars := map[string]string{
		"REPO_ROOT=/project",
		"PROJECT_NAME=myapp",
		"WORKTREE_PATH=/project/.worktrees/feature",
		"WORKTREE_NAME=feature",
		"BRANCH=feature/test",
		"DATABASE_NAME=myapp_feature",
		"DATABASE_URL=mysql://user:pass@db:3306/myapp_feature",
		"SOURCE_DATABASE=myapp",
		"TARGET_DATABASE=myapp_feature",
	}
	
	for expected := range expectedVars {
		found := false
		for _, e := range env {
			if strings.HasPrefix(e, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected env var with prefix %s", expected)
		}
	}
}

func TestExecutor_ExecuteHooks_Command(t *testing.T) {
	tmpDir := t.TempDir()
	
	exec := NewExecutor(tmpDir)
	ctx := &HookContext{
		RepoRoot:    tmpDir,
		ProjectName: "test",
	}
	
	// Test simple command that succeeds
	hooks := []string{"echo hello"}
	err := exec.ExecuteHooks(hooks, ctx, tmpDir, false)
	
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestExecutor_ExecuteHooks_Script(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a test script
	scriptPath := filepath.Join(tmpDir, "test-hook.sh")
	scriptContent := "#!/bin/sh\necho 'hook ran'"
	os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	
	exec := NewExecutor(tmpDir)
	ctx := &HookContext{
		RepoRoot:    tmpDir,
		ProjectName: "test",
	}
	
	// Test script hook
	hooks := []string{"./test-hook.sh"}
	err := exec.ExecuteHooks(hooks, ctx, tmpDir, false)
	
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestExecutor_ExecuteHooks_PreHookFailure(t *testing.T) {
	tmpDir := t.TempDir()
	
	exec := NewExecutor(tmpDir)
	ctx := &HookContext{
		RepoRoot:    tmpDir,
		ProjectName: "test",
	}
	
	// Pre-hook that fails should return error
	hooks := []string{"exit 1"}
	err := exec.ExecuteHooks(hooks, ctx, tmpDir, true)
	
	if err == nil {
		t.Error("expected error for failed pre-hook")
	}
}

func TestIsScriptFile(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"echo hello", false},
		{"composer install", false},
		{"./hooks/setup.sh", true},
		{"../scripts/test.sh", true},
		{"/absolute/path/script.sh", true},
		{".haive/hooks/post-create", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isScriptFile(tt.input)
			if result != tt.expected {
				t.Errorf("isScriptFile(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/core/hooks -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/core/hooks/
git commit -m "feat: add hook executor with environment variables"
```

---

## Task 5: Create File Copy System

**Files:**
- Create: `internal/core/worktree/copy.go`
- Create: `internal/core/worktree/copy_test.go`

**Step 1: Write copy system**

```go
package worktree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
)

// CopyFiles copies files from source directory to destination based on include/exclude patterns
func CopyFiles(sourceDir, destDir string, copyConfig *config.CopyConfig) error {
	if copyConfig == nil {
		return nil
	}
	
	// Find all files matching include patterns
	filesToCopy := make(map[string]bool)
	
	for _, pattern := range copyConfig.Include {
		matches, err := doublestar.Glob(os.DirFS(sourceDir), pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		
		for _, match := range matches {
			// Check if it's excluded
			if isExcluded(match, copyConfig.Exclude) {
				continue
			}
			filesToCopy[match] = true
		}
	}
	
	// Copy files
	for filePath := range filesToCopy {
		sourcePath := filepath.Join(sourceDir, filePath)
		destPath := filepath.Join(destDir, filePath)
		
		// Check if source is a directory
		info, err := os.Stat(sourcePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot stat %s: %v\n", sourcePath, err)
			continue
		}
		
		if info.IsDir() {
			// Create directory
			if err := os.MkdirAll(destPath, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cannot create directory %s: %v\n", destPath, err)
			}
			continue
		}
		
		// Copy file
		if err := copyFile(sourcePath, destPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to copy %s: %v\n", filePath, err)
			continue
		}
	}
	
	return nil
}

func isExcluded(path string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		matched, err := doublestar.Match(pattern, path)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func copyFile(sourcePath, destPath string) error {
	// Create destination directory if needed
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer sourceFile.Close()
	
	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer destFile.Close()
	
	// Copy content
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy content: %w", err)
	}
	
	// Copy permissions
	sourceInfo, err := os.Stat(sourcePath)
	if err == nil {
		os.Chmod(destPath, sourceInfo.Mode())
	}
	
	return nil
}
```

**Step 2: Write tests**

```go
package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
)

func TestCopyFiles(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	
	// Create test files
	os.WriteFile(filepath.Join(sourceDir, ".env.local"), []byte("ENV=local"), 0644)
	os.MkdirAll(filepath.Join(sourceDir, "config"), 0755)
	os.WriteFile(filepath.Join(sourceDir, "config", ".env.local"), []byte("ENV=config"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "vendor", "file.php"), []byte("vendor"), 0644)
	
	copyConfig := &config.CopyConfig{
		Include: []string{"**/.env.local"},
		Exclude: []string{"vendor/"},
	}
	
	err := CopyFiles(sourceDir, destDir, copyConfig)
	if err != nil {
		t.Fatalf("CopyFiles failed: %v", err)
	}
	
	// Check files were copied
	if _, err := os.Stat(filepath.Join(destDir, ".env.local")); err != nil {
		t.Error("expected .env.local to be copied")
	}
	if _, err := os.Stat(filepath.Join(destDir, "config", ".env.local")); err != nil {
		t.Error("expected config/.env.local to be copied")
	}
	if _, err := os.Stat(filepath.Join(destDir, "vendor", "file.php")); err == nil {
		t.Error("expected vendor/file.php to NOT be copied (excluded)")
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		expected bool
	}{
		{"vendor/file.php", []string{"vendor/"}, true},
		{"src/file.php", []string{"vendor/"}, false},
		{"node_modules/pkg/index.js", []string{"node_modules/"}, true},
		{"vendor/autoload.php", []string{"vendor/**"}, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isExcluded(tt.path, tt.patterns)
			if result != tt.expected {
				t.Errorf("isExcluded(%q, %v) = %v, want %v", tt.path, tt.patterns, result, tt.expected)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()
	
	sourcePath := filepath.Join(sourceDir, "test.txt")
	destPath := filepath.Join(destDir, "subdir", "test.txt")
	
	content := []byte("test content")
	os.WriteFile(sourcePath, content, 0644)
	
	err := copyFile(sourcePath, destPath)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	
	copied, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}
	
	if string(copied) != string(content) {
		t.Errorf("copied content mismatch: got %q, want %q", string(copied), string(content))
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/core/worktree -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/core/worktree/
git commit -m "feat: add file copy system for worktrees"
```

---

## Task 6: Update Worktree Commands for New Config

**Files:**
- Modify: `internal/core/commands/worktree.go`

**Step 1: Review current worktree.go structure**

Read the file to understand current implementation before modifying.

**Step 2: Update to use new config types and add copy/hooks**

Key changes needed:
- Import `internal/core/config`, `internal/core/hooks`, `internal/core/worktree`
- Update `CreateWorktree` to accept `*config.HaiveConfig`
- Add file copy after `git worktree add`
- Add hook execution after file copy
- Add pre-validation before any operations

**Step 3: Implement pre-validation function**

```go
// PreValidateWorktree checks if worktree creation prerequisites are met
func PreValidateWorktree(cfg *config.HaiveConfig, worktreeName string) error {
	// Check worktree name is valid
	if worktreeName == "" {
		return &types.CommandError{
			Code:    types.ErrInvalidWorktree,
			Message: "worktree name cannot be empty",
		}
	}
	
	// Check for path traversal
	if strings.Contains(worktreeName, "..") || strings.Contains(worktreeName, "/") {
		return &types.CommandError{
			Code:    types.ErrPathTraversal,
			Message: fmt.Sprintf("invalid worktree name: %s", worktreeName),
		}
	}
	
	// Check if worktree already exists (git or directory)
	gitWorktrees, err := executor.ListWorktrees(cfg.ProjectRoot)
	if err != nil {
		return err
	}
	
	for _, wt := range gitWorktrees {
		if wt.Branch == worktreeName || strings.Contains(wt.Path, worktreeName) {
			return &types.CommandError{
				Code:    types.ErrInvalidWorktree,
				Message: fmt.Sprintf("worktree %s already exists", worktreeName),
			}
		}
	}
	
	worktreePath := filepath.Join(cfg.ProjectRoot, cfg.Worktree.BasePath, worktreeName)
	if _, err := os.Stat(worktreePath); err == nil {
		return &types.CommandError{
			Code:    types.ErrInvalidWorktree,
			Message: fmt.Sprintf("directory %s already exists", worktreePath),
		}
	}
	
	// If worktree.env is configured, validate database section exists
	if cfg.Worktree.Env != nil && cfg.Database == nil {
		return &types.CommandError{
			Code:    types.ErrConfigInvalid,
			Message: "worktree.env requires database configuration",
		}
	}
	
	return nil
}
```

**Step 4: Update CreateWorktree function**

Add file copy and hook execution after git worktree add:

```go
// After git worktree add succeeds:

// 1. Copy files
if cfg.Worktree.Copy != nil {
	if err := worktree.CopyFiles(cfg.ProjectRoot, worktreePath, cfg.Worktree.Copy); err != nil {
		// Log warning but continue
		fmt.Fprintf(os.Stderr, "Warning: failed to copy files: %v\n", err)
	}
}

// 2. Run postCreate hooks
if cfg.Worktree.Hooks != nil && len(cfg.Worktree.Hooks.PostCreate) > 0 {
	hookExec := hooks.NewExecutor(cfg.ProjectRoot)
	hookCtx := &hooks.HookContext{
		RepoRoot:     cfg.ProjectRoot,
		ProjectName:  cfg.Project.Name,
		WorktreePath: worktreePath,
		WorktreeName: worktreeName,
		Branch:       branch,
	}
	
	if err := hookExec.ExecuteHooks(cfg.Worktree.Hooks.PostCreate, hookCtx, worktreePath, false); err != nil {
		// Log warning but worktree exists
		fmt.Fprintf(os.Stderr, "Warning: postCreate hook failed: %v\n", err)
	}
}
```

**Step 5: Run tests**

Run: `go test ./internal/core/commands -v -run TestWorktree`
Expected: Tests pass (may need to update existing tests)

**Step 6: Commit**

```bash
git add internal/core/commands/worktree.go
git commit -m "feat: update worktree commands with copy patterns and hooks"
```

---

## Task 7: Rename Binary (pm → haive)

**Files:**
- Rename: `cmd/pm/` to `cmd/haive/`
- Modify: `Makefile`
- Modify: `go.mod` module path (optional - keep same)
- Modify: All imports that reference binary name (if any)

**Step 1: Rename directory**

```bash
git mv cmd/pm cmd/haive
```

**Step 2: Update Makefile**

```makefile
# Before
BINARY_NAME=pm

# After
BINARY_NAME=haive
```

Update all targets that reference `pm` to use `$(BINARY_NAME)` or `haive`.

**Step 3: Update README references**

Search for `pm ` (with space) or `pm/` to find binary references:
- Installation instructions
- Command examples
- MCP configuration

**Step 4: Test build**

```bash
go build -o haive ./cmd/haive
./haive --help
```

**Step 5: Commit**

```bash
git add cmd/haive/ Makefile README.md
git commit -m "feat: rename binary from pm to haive"
```

---

## Task 8: Update MCP Server and CLI Entry Points

**Files:**
- Modify: `internal/mcp/server.go` (tool names/descriptions)
- Modify: `cmd/haive/main.go` (CLI setup)

**Step 1: Update MCP tool names and descriptions**

Change references from "pm" to "haive" in tool descriptions.

**Step 2: Update main.go if needed**

Check if there are hardcoded references to "pm" in the CLI setup.

**Step 3: Test**

```bash
go build -o haive ./cmd/haive
./haive --help
./haive --mcp  # test MCP mode
```

**Step 4: Commit**

```bash
git add internal/mcp/ cmd/haive/
git commit -m "feat: update MCP and CLI for haive rename"
```

---

## Task 9: Add Database Hook Support

**Files:**
- Modify: `internal/core/commands/database.go`

**Step 1: Add postClone hook after database clone**

After successful database clone operation:

```go
// Run postClone hooks
if cfg.Database.Hooks != nil && len(cfg.Database.Hooks.PostClone) > 0 {
	hookExec := hooks.NewExecutor(cfg.ProjectRoot)
	
	// Parse DSN to build DATABASE_URL for target
	targetDSN := // build from source DSN with new database name
	
	hookCtx := &hooks.HookContext{
		RepoRoot:       cfg.ProjectRoot,
		ProjectName:    cfg.Project.Name,
		DatabaseName:   targetDB,
		DatabaseURL:    targetDSN,
		SourceDatabase: sourceDB,
		TargetDatabase: targetDB,
	}
	
	if err := hookExec.ExecuteHooks(cfg.Database.Hooks.PostClone, hookCtx, cfg.ProjectRoot, false); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: postClone hook failed: %v\n", err)
	}
}
```

**Step 2: Add preDrop hook before database drop**

Before executing drop:

```go
// Run preDrop hooks
if cfg.Database.Hooks != nil && len(cfg.Database.Hooks.PreDrop) > 0 {
	hookExec := hooks.NewExecutor(cfg.ProjectRoot)
	hookCtx := &hooks.HookContext{
		RepoRoot:     cfg.ProjectRoot,
		ProjectName:  cfg.Project.Name,
		DatabaseName: dbName,
		DatabaseURL:  cfg.Database.DSN, // original DSN
	}
	
	if err := hookExec.ExecuteHooks(cfg.Database.Hooks.PreDrop, hookCtx, cfg.ProjectRoot, true); err != nil {
		return fmt.Errorf("preDrop hook prevented drop: %w", err)
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/core/commands -v -run TestDatabase`
Expected: Tests pass

**Step 4: Commit**

```bash
git add internal/core/commands/database.go
git commit -m "feat: add database hooks (postClone, preDrop)"
```

---

## Task 10: Update Tests for New Config Types

**Files:**
- Modify: `internal/core/config/config_test.go`
- Modify: All test files that use old config types

**Step 1: Review existing tests**

Find all references to old `Config` type from `internal/core/config` and update to use `HaiveConfig`.

**Step 2: Update test fixtures**

If tests use JSON config fixtures, add TOML equivalents.

**Step 3: Run all tests**

```bash
go test ./... -v
```

Fix any failing tests.

**Step 4: Commit**

```bash
git add internal/core/config/config_test.go
git commit -m "test: update tests for new config types"
```

---

## Task 11: Add Worktree Env (Database URL) Management

**Files:**
- Create: `internal/core/worktree/env.go`
- Create: `internal/core/worktree/env_test.go`

**Step 1: Write env management functions**

```go
package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/dsn"
)

// SetWorktreeDatabase configures the database for a worktree
// - Sets git config haive.database
// - Updates .env.local with new DATABASE_URL
func SetWorktreeDatabase(worktreePath string, cfg *config.HaiveConfig, dbName string) error {
	// Set git config
	cmd := exec.Command("git", "config", "--local", "haive.database", dbName)
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git config: %w", err)
	}
	
	// Update .env.local if configured
	if cfg.Worktree.Env != nil {
		if err := updateEnvFile(worktreePath, cfg, dbName); err != nil {
			return fmt.Errorf("failed to update env file: %w", err)
		}
	}
	
	return nil
}

// updateEnvFile updates the DATABASE_URL in the env file
func updateEnvFile(worktreePath string, cfg *config.HaiveConfig, dbName string) error {
	if cfg.Worktree.Env == nil || cfg.Database == nil {
		return nil
	}
	
	envFile := cfg.Worktree.Env.File
	varName := cfg.Worktree.Env.VarName
	
	envPath := filepath.Join(worktreePath, envFile)
	
	// Check if file exists
	_, err := os.Stat(envPath)
	if err != nil {
		return fmt.Errorf("env file not found: %s", envPath)
	}
	
	// Parse original DSN to get connection details
	parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("failed to parse DSN: %w", err)
	}
	
	// Build new DSN with worktree database name
	parsedDSN.Database = dbName
	newDSN := parsedDSN.String()
	
	// Read existing file
	content, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}
	
	// Replace the variable
	lines := strings.Split(string(content), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, varName+"=") {
			lines[i] = fmt.Sprintf("%s=%s", varName, newDSN)
			found = true
			break
		}
	}
	
	if !found {
		// Variable not found, append it
		lines = append(lines, fmt.Sprintf("%s=%s", varName, newDSN))
	}
	
	// Write back
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(envPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}
	
	return nil
}

// GetWorktreeDatabase reads the configured database for a worktree
func GetWorktreeDatabase(worktreePath string) (string, error) {
	cmd := exec.Command("git", "config", "--local", "haive.database")
	cmd.Dir = worktreePath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no database configured for this worktree")
	}
	return strings.TrimSpace(string(out)), nil
}
```

**Step 2: Write tests**

```go
package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
)

func TestUpdateEnvFile(t *testing.T) {
	worktreePath := t.TempDir()
	
	// Create .env.local
	envContent := `APP_ENV=dev
DATABASE_URL=mysql://user:pass@db:3306/main_db
OTHER_VAR=value
`
	envPath := filepath.Join(worktreePath, ".env.local")
	os.WriteFile(envPath, []byte(envContent), 0644)
	
	cfg := &config.HaiveConfig{
		Project: config.ProjectConfig{Name: "test"},
		Database: &config.DatabaseConfig{
			DSN: "mysql://user:pass@db:3306/main_db",
		},
		Worktree: &config.WorktreeConfig{
			Env: &config.EnvConfig{
				File:    ".env.local",
				VarName: "DATABASE_URL",
			},
		},
	}
	
	err := updateEnvFile(worktreePath, cfg, "main_db_feature_x")
	if err != nil {
		t.Fatalf("updateEnvFile failed: %v", err)
	}
	
	// Read updated file
	updated, _ := os.ReadFile(envPath)
	updatedStr := string(updated)
	
	if !strings.Contains(updatedStr, "DATABASE_URL=mysql://user:pass@db:3306/main_db_feature_x") {
		t.Errorf("DATABASE_URL not updated correctly:\n%s", updatedStr)
	}
	
	// Other vars should be preserved
	if !strings.Contains(updatedStr, "APP_ENV=dev") {
		t.Error("APP_ENV was not preserved")
	}
}

func TestUpdateEnvFile_AppendNew(t *testing.T) {
	worktreePath := t.TempDir()
	
	// Create .env.local without DATABASE_URL
	envContent := `APP_ENV=dev
OTHER_VAR=value
`
	envPath := filepath.Join(worktreePath, ".env.local")
	os.WriteFile(envPath, []byte(envContent), 0644)
	
	cfg := &config.HaiveConfig{
		Project: config.ProjectConfig{Name: "test"},
		Database: &config.DatabaseConfig{
			DSN: "mysql://user:pass@db:3306/main_db",
		},
		Worktree: &config.WorktreeConfig{
			Env: &config.EnvConfig{
				File:    ".env.local",
				VarName: "DATABASE_URL",
			},
		},
	}
	
	err := updateEnvFile(worktreePath, cfg, "main_db_feature_x")
	if err != nil {
		t.Fatalf("updateEnvFile failed: %v", err)
	}
	
	updated, _ := os.ReadFile(envPath)
	if !strings.Contains(string(updated), "DATABASE_URL=mysql://user:pass@db:3306/main_db_feature_x") {
		t.Error("DATABASE_URL not appended")
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/core/worktree -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/core/worktree/env.go internal/core/worktree/env_test.go
git commit -m "feat: add worktree database environment management"
```

---

## Task 12: Integration Test - Full Worktree Flow

**Files:**
- Create: `internal/core/commands/worktree_integration_test.go`

**Step 1: Write integration test**

This test exercises the full worktree creation flow with copy and hooks.

```go
package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
)

func TestCreateWorktree_WithCopyAndHooks(t *testing.T) {
	// This is an integration test that requires git
	// Skip if not in CI or if git is not available
	if os.Getenv("CI") == "" {
		t.Skip("Skipping integration test")
	}
	
	// Setup: create a temp git repo with haive.toml
	repoDir := t.TempDir()
	
	// Initialize git repo
	os.Chdir(repoDir)
	os.Exec("git", "init")
	os.Exec("git", "config", "user.email", "test@test.com")
	os.Exec("git", "config", "user.name", "Test")
	
	// Create initial commit
	os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Test"), 0644)
	os.Exec("git", "add", ".")
	os.Exec("git", "commit", "-m", "initial")
	
	// Create haive.toml
	configContent := `
[project]
name = "test-project"

[docker]
compose_files = ["compose.yaml"]

[worktree]
base_path = ".worktrees"

[worktree.copy]
include = [".env.local"]

[worktree.hooks]
postCreate = ["echo 'Worktree created: ${WORKTREE_NAME}'"]
`
	os.WriteFile(filepath.Join(repoDir, "haive.toml"), []byte(configContent), 0644)
	os.WriteFile(filepath.Join(repoDir, ".env.local"), []byte("ENV=local"), 0644)
	
	// Load config
	cfg, err := config.Load(repoDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	
	// Create worktree
	result, err := CreateWorktree(cfg, "feature/test", true)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	
	// Verify worktree path
	expectedPath := filepath.Join(repoDir, ".worktrees", "feature-test")
	if result.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result.Path)
	}
	
	// Verify .env.local was copied
	copiedEnv := filepath.Join(expectedPath, ".env.local")
	if _, err := os.Stat(copiedEnv); err != nil {
		t.Error(".env.local was not copied to worktree")
	}
}
```

**Step 2: Commit**

```bash
git add internal/core/commands/worktree_integration_test.go
git commit -m "test: add integration test for worktree creation flow"
```

---

## Task 13: Final Verification

**Step 1: Run all tests**

```bash
go test ./... -v
```

**Step 2: Build and test binary**

```bash
go build -o haive ./cmd/haive
./haive --help
./haive worktree --help
./haive database --help
```

**Step 3: Test TUI mode**

```bash
./haive
```

**Step 4: Test MCP mode**

```bash
./haive --mcp
# In another terminal, send MCP requests
```

**Step 5: Update AGENTS.md with new info**

Add documentation about:
- New config format (TOML)
- New binary name (haive)
- Hook system
- Copy patterns

**Step 6: Commit**

```bash
git add AGENTS.md
git commit -m "docs: update AGENTS.md for haive refactor"
```

---

## Summary

This plan transforms `pm` into `haive` with:

1. **Modular config system** - TOML primary, supports YAML/JSON legacy
2. **Worktree module** - Copy patterns, hooks (postCreate, preRemove, postRemove), env management
3. **Database module** - Hooks (postClone, preDrop)
4. **Hook executor** - Environment variables, script/command support
5. **File copy system** - Glob patterns using doublestar
6. **Binary rename** - pm → haive

**Phase 2 (Future):**
- Presets system
- Config-defined workflows
- More hooks (preCreate, postDump, etc.)
- Remote presets

---

**Execution choice:**

1. **Subagent-Driven (this session)** - Dispatch fresh subagent per task
2. **Parallel Session (separate)** - Open new session with executing-plans

Which approach would you prefer?

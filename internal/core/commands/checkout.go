package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/dsn"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/executor"
)

type CheckoutResult struct {
	Branch   string `json:"branch"`
	Database string `json:"database"`
	Created  bool   `json:"created"`
	Cloned   bool   `json:"cloned"`
}

type SwitchResult struct {
	Branch   string `json:"branch"`
	Database string `json:"database"`
	Created  bool   `json:"created"`
	Cloned   bool   `json:"cloned"`
}

// Checkout switches to a git branch and sets up the corresponding database
func Checkout(projectRoot, branch string, create bool, cloneFrom string) (*CheckoutResult, error) {
	// First, switch git branch
	if create {
		// Create new branch
		cmd := exec.Command("git", "checkout", "-b", branch)
		cmd.Dir = projectRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to create branch: %w\nOutput: %s", err, string(output))
		}
	} else {
		// Checkout existing branch
		cmd := exec.Command("git", "checkout", branch)
		cmd.Dir = projectRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to checkout branch: %w\nOutput: %s", err, string(output))
		}
	}

	// Then switch database for this branch
	switchResult, err := Switch(projectRoot, cloneFrom)
	if err != nil {
		return nil, err
	}

	return &CheckoutResult{
		Branch:   branch,
		Database: switchResult.Database,
		Created:  switchResult.Created,
		Cloned:   switchResult.Cloned,
	}, nil
}

// Switch switches the database for the current branch (without changing git branch)
func Switch(projectRoot string, cloneFrom string) (*SwitchResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}
	
	// For docker operations, use the config's project root (handles worktrees)
	dockerRoot := projectRoot
	if cfg.ProjectRoot != "" {
		dockerRoot = cfg.ProjectRoot
	}

	if cfg.Database == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "database configuration is required for switch operations",
		}
	}

	// Get current branch name
	branch, err := getCurrentBranch(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Parse DSN to get default database
	parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	defaultDB := parsedDSN.Database

	// Generate branch-specific database name
	branchDB := generateBranchDBName(defaultDB, branch)

	// Check if database exists (use dockerRoot for docker operations)
	dbExists, err := databaseExists(cfg, dockerRoot, branchDB)
	if err != nil {
		return nil, err
	}

	result := &SwitchResult{
		Branch:   branch,
		Database: branchDB,
	}

	// Create database if it doesn't exist
	if !dbExists {
		if err := createBranchDB(cfg, dockerRoot, branchDB); err != nil {
			return nil, err
		}
		result.Created = true

		// Clone data if requested
		if cloneFrom != "" {
			if err := cloneDatabaseData(cfg, dockerRoot, cloneFrom, branchDB); err != nil {
				return nil, err
			}
			result.Cloned = true
		} else if branch != "main" && branch != "master" {
			// Auto-clone from default db for feature branches
			if err := cloneDatabaseData(cfg, dockerRoot, defaultDB, branchDB); err != nil {
				return nil, err
			}
			result.Cloned = true
		}
	}

	// Update .env.local with new database
	if err := updateEnvLocalDatabase(projectRoot, cfg.Database.DSN, branchDB); err != nil {
		return nil, fmt.Errorf("failed to update .env.local: %w", err)
	}

	return result, nil
}

func getCurrentBranch(projectRoot string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func generateBranchDBName(defaultDB, branch string) string {
	// Sanitize branch name for database name
	sanitized := strings.ReplaceAll(branch, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")

	// If on main/master, use default db
	if branch == "main" || branch == "master" {
		return defaultDB
	}

	return defaultDB + "_" + sanitized
}

func databaseExists(cfg *config.Config, projectRoot, dbName string) (bool, error) {
	parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return false, err
	}

	engine := getEngine(parsedDSN.Engine)
	dbExecutor := executor.NewDockerDatabaseExecutor(engine, cfg.Docker.ComposeFiles, projectRoot)

	result, err := dbExecutor.List(cfg.Database.Service, parsedDSN, parsedDSN.Database)
	if err != nil {
		return false, err
	}

	for _, db := range result.Databases {
		if db.Name == dbName {
			return true, nil
		}
	}
	return false, nil
}

func createBranchDB(cfg *config.Config, projectRoot, dbName string) error {
	// Check if database is allowed
	if err := core.IsDatabaseAllowed(dbName, cfg.Database.Allowed); err != nil {
		// Auto-add branch pattern to allowed if wildcard exists
		allowed := false
		for _, pattern := range cfg.Database.Allowed {
			if pattern == cfg.Database.Allowed[0]+"_*" || pattern == "*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return err
		}
	}

	parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return err
	}

	engine := getEngine(parsedDSN.Engine)
	dbExecutor := executor.NewDockerDatabaseExecutor(engine, cfg.Docker.ComposeFiles, projectRoot)

	_, err = dbExecutor.Create(cfg.Database.Service, parsedDSN, dbName)
	return err
}

func cloneDatabaseData(cfg *config.Config, projectRoot, sourceDB, targetDB string) error {
	// Use the existing CloneDB command
	_, err := CloneDB(projectRoot, sourceDB, targetDB)
	return err
}

func updateEnvLocalDatabase(projectRoot, originalDSN, newDBName string) error {
	envLocalPath := filepath.Join(projectRoot, ".env.local")

	// Parse original DSN and create new one with different database
	parsedDSN, err := dsn.ParseDSN(originalDSN)
	if err != nil {
		return err
	}

	// Build new DSN with new database name
	newDSN := originalDSN
	if parsedDSN.Database != "" {
		newDSN = strings.Replace(originalDSN, "/"+parsedDSN.Database, "/"+newDBName, 1)
		newDSN = strings.Replace(newDSN, "/"+parsedDSN.Database+"?", "/"+newDBName+"?", 1)
	}

	// Read existing .env.local
	content := ""
	if data, err := os.ReadFile(envLocalPath); err == nil {
		content = string(data)
	}

	// Replace or add DATABASE_URL
	lines := strings.Split(content, "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "DATABASE_URL=") {
			lines[i] = "DATABASE_URL=" + newDSN
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, "DATABASE_URL="+newDSN)
	}

	newContent := strings.Join(lines, "\n")
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	return os.WriteFile(envLocalPath, []byte(newContent), 0644)
}



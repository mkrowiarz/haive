package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/dsn"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/hooks"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/executor"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/executor/engines"
)

func Dump(projectRoot, dbName string, tables []string) (*types.DumpResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Database == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "database configuration is required for dump operations",
		}
	}

	if err := core.IsDatabaseAllowed(dbName, cfg.Database.Allowed); err != nil {
		return nil, err
	}

	parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	engine := getEngine(parsedDSN.Engine)

	dbExecutor := executor.NewDockerDatabaseExecutor(engine, cfg.Docker.ComposeFiles, projectRoot)

	destPath := filepath.Join(cfg.Database.DumpsPath,
		fmt.Sprintf("%s_%s.sql", dbName, time.Now().Format("2006-01-02T15-04")))

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create dumps directory: %w", err)
	}

	return dbExecutor.Dump(cfg.Database.Service, parsedDSN, destPath, tables)
}

func CreateDB(projectRoot, dbName string) (*types.CreateResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Database == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "database configuration is required for create operations",
		}
	}

	if err := core.IsDatabaseAllowed(dbName, cfg.Database.Allowed); err != nil {
		return nil, err
	}

	parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	engine := getEngine(parsedDSN.Engine)

	dbExecutor := executor.NewDockerDatabaseExecutor(engine, cfg.Docker.ComposeFiles, projectRoot)

	return dbExecutor.Create(cfg.Database.Service, parsedDSN, dbName)
}

func ImportDB(projectRoot, dbName, sourcePath string) (*types.ImportResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Database == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "database configuration is required for import operations",
		}
	}

	if err := core.IsDatabaseAllowed(dbName, cfg.Database.Allowed); err != nil {
		return nil, err
	}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, &types.CommandError{
			Code:    types.ErrFileNotFound,
			Message: fmt.Sprintf("SQL file not found: %s", sourcePath),
		}
	}

	parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	engine := getEngine(parsedDSN.Engine)

	dbExecutor := executor.NewDockerDatabaseExecutor(engine, cfg.Docker.ComposeFiles, projectRoot)

	return dbExecutor.Import(cfg.Database.Service, parsedDSN, sourcePath, dbName)
}

func DropDB(projectRoot, dbName string) (*types.DropResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Database == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "database configuration is required for drop operations",
		}
	}

	if err := core.IsDatabaseAllowed(dbName, cfg.Database.Allowed); err != nil {
		return nil, err
	}

	parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	if err := core.IsNotDefaultDB(dbName, parsedDSN.Database); err != nil {
		return nil, err
	}

	// Run preDrop hooks
	if cfg.Database.Hooks != nil && len(cfg.Database.Hooks.PreDrop) > 0 {
		hookExec := hooks.NewExecutor(cfg.ProjectRoot)
		hookCtx := &hooks.HookContext{
			RepoRoot:     cfg.ProjectRoot,
			DatabaseName: dbName,
			DatabaseURL:  cfg.Database.DSN, // original DSN
		}

		if err := hookExec.ExecuteHooks(cfg.Database.Hooks.PreDrop, hookCtx, cfg.ProjectRoot, true); err != nil {
			return nil, fmt.Errorf("preDrop hook prevented drop: %w", err)
		}
	}

	engine := getEngine(parsedDSN.Engine)

	dbExecutor := executor.NewDockerDatabaseExecutor(engine, cfg.Docker.ComposeFiles, projectRoot)

	return dbExecutor.Drop(cfg.Database.Service, parsedDSN, dbName)
}

func ListDBs(projectRoot string) (*types.DatabaseListResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Database == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "database configuration is required for list operations",
		}
	}

	parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	engine := getEngine(parsedDSN.Engine)

	dbExecutor := executor.NewDockerDatabaseExecutor(engine, cfg.Docker.ComposeFiles, projectRoot)

	return dbExecutor.List(cfg.Database.Service, parsedDSN, parsedDSN.Database)
}

func CloneDB(projectRoot, sourceDB, targetDB string) (*types.CloneResult, error) {
	start := time.Now()

	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Database == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "database configuration is required for clone operations",
		}
	}

	parsedDSN, err := dsn.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	if sourceDB == "" {
		sourceDB = parsedDSN.Database
	}

	if err := core.IsDatabaseAllowed(sourceDB, cfg.Database.Allowed); err != nil {
		return nil, err
	}

	if err := core.IsDatabaseAllowed(targetDB, cfg.Database.Allowed); err != nil {
		return nil, err
	}

	engine := getEngine(parsedDSN.Engine)

	dbExecutor := executor.NewDockerDatabaseExecutor(engine, cfg.Docker.ComposeFiles, projectRoot)

	_, err = dbExecutor.Create(cfg.Database.Service, parsedDSN, targetDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create target database: %w", err)
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("clone_%s_%d.sql", targetDB, time.Now().UnixNano()))

	sourceDSN := *parsedDSN
	sourceDSN.Database = sourceDB

	dumpResult, err := dbExecutor.Dump(cfg.Database.Service, &sourceDSN, tmpFile, nil)
	if err != nil {
		dbExecutor.Drop(cfg.Database.Service, parsedDSN, targetDB)
		os.Remove(tmpFile)
		return nil, fmt.Errorf("failed to dump source database: %w", err)
	}

	_, err = dbExecutor.Import(cfg.Database.Service, parsedDSN, tmpFile, targetDB)
	if err != nil {
		dbExecutor.Drop(cfg.Database.Service, parsedDSN, targetDB)
		os.Remove(tmpFile)
		return nil, fmt.Errorf("failed to import into target database: %w", err)
	}

	os.Remove(tmpFile)

	// Run postClone hooks
	if cfg.Database.Hooks != nil && len(cfg.Database.Hooks.PostClone) > 0 {
		hookExec := hooks.NewExecutor(cfg.ProjectRoot)

		// Build target DSN by modifying the existing parsed DSN
		targetDSN := *parsedDSN
		targetDSN.Database = targetDB

		hookCtx := &hooks.HookContext{
			RepoRoot:       cfg.ProjectRoot,
			DatabaseName:   targetDB,
			DatabaseURL:    targetDSN.String(),
			SourceDatabase: sourceDB,
			TargetDatabase: targetDB,
		}

		if err := hookExec.ExecuteHooks(cfg.Database.Hooks.PostClone, hookCtx, cfg.ProjectRoot, false); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: postClone hook failed: %v\n", err)
		}
	}

	return &types.CloneResult{
		Source:   sourceDB,
		Target:   targetDB,
		Size:     dumpResult.Size,
		Duration: time.Since(start),
	}, nil
}

func ListDumps(projectRoot string) (*types.DumpsListResult, error) {
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, err
	}

	if cfg.Database == nil {
		return nil, &types.CommandError{
			Code:    types.ErrConfigMissing,
			Message: "database configuration is required for dumps list operations",
		}
	}

	dumpsPath := cfg.Database.DumpsPath
	if dumpsPath == "" {
		dumpsPath = "var/dumps"
	}

	if !filepath.IsAbs(dumpsPath) {
		dumpsPath = filepath.Join(projectRoot, dumpsPath)
	}

	if _, err := os.Stat(dumpsPath); os.IsNotExist(err) {
		return &types.DumpsListResult{Dumps: []types.DumpFileInfo{}}, nil
	}

	files, err := filepath.Glob(filepath.Join(dumpsPath, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to read dumps directory: %w", err)
	}

	var dumps []types.DumpFileInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		filename := filepath.Base(file)
		dbName, _ := parseDumpFilename(filename)
		if dbName == "" {
			continue
		}

		dumps = append(dumps, types.DumpFileInfo{
			Name:     filename,
			Database: dbName,
			Size:     info.Size(),
			Modified: info.ModTime().Format(time.RFC3339),
		})
	}

	sort.Slice(dumps, func(i, j int) bool {
		return dumps[i].Modified > dumps[j].Modified
	})

	return &types.DumpsListResult{Dumps: dumps}, nil
}

func parseDumpFilename(filename string) (dbName, timestamp string) {
	ext := filepath.Ext(filename)
	if ext != ".sql" {
		return "", ""
	}

	name := strings.TrimSuffix(filename, ext)

	idx := strings.LastIndex(name, "_")
	if idx == -1 {
		return "", ""
	}

	dbName = name[:idx]
	timestamp = name[idx+1:]

	return dbName, timestamp
}

func getEngine(engineType string) engines.DatabaseEngine {
	if engineType == "mariadb" {
		return engines.NewMySQLEngine(true)
	}
	return engines.NewMySQLEngine(false)
}

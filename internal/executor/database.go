package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/executor/engines"
)

type DatabaseExecutor interface {
	Dump(service string, dsn *types.DSN, destPath string, tables []string) (*types.DumpResult, error)
	Create(service string, dsn *types.DSN, dbName string) (*types.CreateResult, error)
	Import(service string, dsn *types.DSN, sourcePath string, dbName string) (*types.ImportResult, error)
	Drop(service string, dsn *types.DSN, dbName string) (*types.DropResult, error)
	List(service string, dsn *types.DSN, defaultDB string) (*types.DatabaseListResult, error)
}

type DockerDatabaseExecutor struct {
	engine       engines.DatabaseEngine
	composeFiles []string
}

func NewDockerDatabaseExecutor(engine engines.DatabaseEngine, composeFiles []string) *DockerDatabaseExecutor {
	return &DockerDatabaseExecutor{
		engine:       engine,
		composeFiles: composeFiles,
	}
}

func (d *DockerDatabaseExecutor) buildComposeArgs(subcmd ...string) []string {
	var args []string
	args = append(args, "compose")
	for _, f := range d.composeFiles {
		args = append(args, "-f", f)
	}
	args = append(args, subcmd...)
	return args
}

func (d *DockerDatabaseExecutor) Dump(service string, dsn *types.DSN, destPath string, tables []string) (*types.DumpResult, error) {
	start := time.Now()

	cmd := d.engine.BuildDumpCommand(dsn, tables)
	args := append(d.buildComposeArgs("exec", "-T", service), cmd...)

	execCmd := exec.Command("docker", args...)
	output, err := execCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("dump failed: %w", err)
	}

	if err := os.WriteFile(destPath, output, 0644); err != nil {
		return nil, fmt.Errorf("failed to write dump file: %w", err)
	}

	stat, _ := os.Stat(destPath)

	return &types.DumpResult{
		Path:     destPath,
		Size:     stat.Size(),
		Database: dsn.Database,
		Duration: time.Since(start),
	}, nil
}

func (d *DockerDatabaseExecutor) Create(service string, dsn *types.DSN, dbName string) (*types.CreateResult, error) {
	cmd := d.engine.BuildCreateCommand(dsn, dbName)
	args := append(d.buildComposeArgs("exec", "-T", service), cmd...)

	execCmd := exec.Command("docker", args...)
	if err := execCmd.Run(); err != nil {
		return nil, fmt.Errorf("create database failed: %w", err)
	}

	return &types.CreateResult{Database: dbName}, nil
}

func (d *DockerDatabaseExecutor) Import(service string, dsn *types.DSN, sourcePath string, dbName string) (*types.ImportResult, error) {
	start := time.Now()

	sqlData, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SQL file: %w", err)
	}

	cmd := d.engine.BuildImportCommand(dsn, dbName)
	args := append(d.buildComposeArgs("exec", "-T", service), cmd...)

	execCmd := exec.Command("docker", args...)
	execCmd.Stdin = bytes.NewReader(sqlData)

	if err := execCmd.Run(); err != nil {
		return nil, fmt.Errorf("import failed: %w", err)
	}

	return &types.ImportResult{
		Path:     sourcePath,
		Database: dbName,
		Duration: time.Since(start),
	}, nil
}

func (d *DockerDatabaseExecutor) Drop(service string, dsn *types.DSN, dbName string) (*types.DropResult, error) {
	cmd := d.engine.BuildDropCommand(dsn, dbName)
	args := append(d.buildComposeArgs("exec", "-T", service), cmd...)

	execCmd := exec.Command("docker", args...)
	if err := execCmd.Run(); err != nil {
		return nil, fmt.Errorf("drop database failed: %w", err)
	}

	return &types.DropResult{Database: dbName}, nil
}

func (d *DockerDatabaseExecutor) List(service string, dsn *types.DSN, defaultDB string) (*types.DatabaseListResult, error) {
	cmd := d.engine.BuildListCommand(dsn)
	args := append(d.buildComposeArgs("exec", "-T", service), cmd...)

	execCmd := exec.Command("docker", args...)
	output, err := execCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list databases failed: %w", err)
	}

	return parseDatabaseList(string(output), defaultDB)
}

func parseDatabaseList(output, defaultDB string) (*types.DatabaseListResult, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var databases []types.DatabaseInfo

	systemDBs := map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" || name == "Database" {
			continue
		}
		if systemDBs[name] {
			continue
		}

		databases = append(databases, types.DatabaseInfo{
			Name:      name,
			IsDefault: name == defaultDB,
		})
	}

	return &types.DatabaseListResult{Databases: databases}, nil
}

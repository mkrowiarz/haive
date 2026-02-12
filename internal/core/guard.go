package core

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

func ValidateBranchName(name string) error {
	if name == "" {
		return &types.CommandError{Code: types.ErrInvalidName, Message: "branch name cannot be empty"}
	}

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\/]+$`, name)
	if !matched {
		return &types.CommandError{Code: types.ErrInvalidName, Message: "branch name contains invalid characters"}
	}

	return nil
}

func CheckPathTraversal(resolvedPath, basePath string) error {
	absResolved, err := filepath.Abs(resolvedPath)
	if err != nil {
		return &types.CommandError{Code: types.ErrPathTraversal, Message: "failed to resolve path: " + err.Error()}
	}

	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return &types.CommandError{Code: types.ErrPathTraversal, Message: "failed to resolve base path: " + err.Error()}
	}

	rel, err := filepath.Rel(absBase, absResolved)
	if err != nil || strings.Contains(rel, "..") {
		return &types.CommandError{Code: types.ErrPathTraversal, Message: "path traversal attempt detected"}
	}

	return nil
}

func SanitizeWorktreeName(branchName string) (dirName, dbName string) {
	dirName = strings.ReplaceAll(branchName, "/", "-")

	dbName = strings.ReplaceAll(branchName, "/", "_")
	dbName = strings.ReplaceAll(dbName, "-", "_")

	return
}

func IsDatabaseAllowed(dbName string, allowed []string) error {
	if len(allowed) == 0 {
		return nil
	}
	for _, pattern := range allowed {
		matched, err := filepath.Match(pattern, dbName)
		if err == nil && matched {
			return nil
		}
	}
	return &types.CommandError{
		Code:    types.ErrDbNotAllowed,
		Message: fmt.Sprintf("database '%s' is not in allowed list", dbName),
	}
}

func IsNotDefaultDB(dbName string, defaultDB string) error {
	if dbName == defaultDB {
		return &types.CommandError{
			Code:    types.ErrDbIsDefault,
			Message: "cannot drop the default database",
		}
	}
	return nil
}

package mcp

import (
	"testing"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

func TestToMCPCode(t *testing.T) {
	tests := []struct {
		input    types.ErrCode
		expected int
	}{
		{types.ErrConfigMissing, ErrCodeConfigMissing},
		{types.ErrConfigInvalid, ErrCodeConfigInvalid},
		{types.ErrInvalidName, ErrCodeInvalidName},
		{types.ErrPathTraversal, ErrCodePathTraversal},
		{types.ErrDbNotAllowed, ErrCodeDbNotAllowed},
		{types.ErrDbIsDefault, ErrCodeDbIsDefault},
		{types.ErrFileNotFound, ErrCodeFileNotFound},
		{types.ErrCode("UNKNOWN"), -32000},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := toMCPCode(tt.input)
			if result != tt.expected {
				t.Errorf("toMCPCode(%s) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToMCPError(t *testing.T) {
	cmdErr := &types.CommandError{
		Code:    types.ErrDbNotAllowed,
		Message: "database 'other' not allowed",
	}

	err := toMCPError(cmdErr)
	if err == nil {
		t.Error("expected error, got nil")
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("error message is empty")
	}
}

package mcp

import (
	"fmt"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
)

const (
	ErrCodeConfigMissing = -32001
	ErrCodeConfigInvalid = -32002
	ErrCodeInvalidName   = -32003
	ErrCodePathTraversal = -32004
	ErrCodeDbNotAllowed  = -32005
	ErrCodeDbIsDefault   = -32006
	ErrCodeFileNotFound  = -32007
)

func toMCPCode(code types.ErrCode) int {
	switch code {
	case types.ErrConfigMissing:
		return ErrCodeConfigMissing
	case types.ErrConfigInvalid:
		return ErrCodeConfigInvalid
	case types.ErrInvalidName:
		return ErrCodeInvalidName
	case types.ErrPathTraversal:
		return ErrCodePathTraversal
	case types.ErrDbNotAllowed:
		return ErrCodeDbNotAllowed
	case types.ErrDbIsDefault:
		return ErrCodeDbIsDefault
	case types.ErrFileNotFound:
		return ErrCodeFileNotFound
	default:
		return -32000
	}
}

func toMCPError(err error) error {
	if cmdErr, ok := err.(*types.CommandError); ok {
		return fmt.Errorf("mcp error %d: %s (data: {\"code\":\"%s\"})",
			toMCPCode(cmdErr.Code), cmdErr.Message, cmdErr.Code)
	}
	return fmt.Errorf("mcp error -32000: %s", err.Error())
}

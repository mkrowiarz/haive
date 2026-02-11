package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

func ResolveEnvVars(value, projectRoot string) string {
	return envVarRegex.ReplaceAllStringFunc(value, func(match string) string {
		varName := strings.Trim(match, "${}")

		// First check environment
		if val := os.Getenv(varName); val != "" {
			return val
		}

		// Then check .env.local
		if val := getEnvFromFile(filepath.Join(projectRoot, ".env.local"), varName); val != "" {
			return val
		}

		// Then check .env
		if val := getEnvFromFile(filepath.Join(projectRoot, ".env"), varName); val != "" {
			return val
		}

		return match
	})
}

func getEnvFromFile(filePath, varName string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == varName {
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"'")
			return val
		}
	}
	return ""
}

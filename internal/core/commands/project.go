package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/config"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
	"gopkg.in/yaml.v3"
)

func Info(projectRoot string) (*types.ProjectInfo, error) {
	if projectRoot == "" {
		projectRoot = "."
	}

	cfg, err := config.Load(projectRoot)
	if err != nil {
		if cmdErr, ok := err.(*types.CommandError); ok {
			if cmdErr.Code == types.ErrConfigMissing {
				return &types.ProjectInfo{
					ConfigSummary:       nil,
					EnvFiles:            detectEnvFiles(projectRoot),
					DockerComposeExists: dockerComposeExists(projectRoot),
				}, nil
			}
		}
		return nil, err
	}

	summary := &types.ConfigSummary{
		Name: cfg.Project.Name,
		Type: cfg.Project.Type,
	}

	return &types.ProjectInfo{
		ConfigSummary:       summary,
		EnvFiles:            detectEnvFiles(projectRoot),
		DockerComposeExists: dockerComposeExists(projectRoot),
	}, nil
}

func Init(projectRoot string) (*types.InitSuggestion, error) {
	if projectRoot == "" {
		projectRoot = "."
	}

	detectedServices, err := detectDockerServices(projectRoot)
	if err != nil {
		return nil, err
	}

	projectType := detectProjectType(projectRoot)

	detectedEnvVars, err := detectEnvVars(projectRoot)
	if err != nil {
		return nil, err
	}

	suggestedConfig := generateSuggestedConfig(projectType, detectedServices)

	return &types.InitSuggestion{
		SuggestedConfig:  suggestedConfig,
		DetectedServices: detectedServices,
		DetectedEnvVars:  detectedEnvVars,
	}, nil
}

func detectEnvFiles(projectRoot string) []string {
	var envFiles []string

	envPaths := []string{".env", ".env.local"}
	for _, envPath := range envPaths {
		fullPath := filepath.Join(projectRoot, envPath)
		if _, err := os.Stat(fullPath); err == nil {
			envFiles = append(envFiles, fullPath)
		}
	}

	return envFiles
}

func dockerComposeExists(projectRoot string) bool {
	paths := []string{"docker-compose.yaml", "docker-compose.yml", "docker-compose.yml.yaml"}
	for _, path := range paths {
		fullPath := filepath.Join(projectRoot, path)
		if _, err := os.Stat(fullPath); err == nil {
			return true
		}
	}
	return false
}

type DockerCompose struct {
	Services map[string]struct {
		Image string `yaml:"image"`
	} `yaml:"services"`
}

func detectDockerServices(projectRoot string) (map[string]string, error) {
	paths := []string{"docker-compose.yaml", "docker-compose.yml"}
	for _, path := range paths {
		fullPath := filepath.Join(projectRoot, path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		var dc DockerCompose
		if err := yaml.Unmarshal(data, &dc); err != nil {
			continue
		}

		services := make(map[string]string)
		for name, svc := range dc.Services {
			services[name] = svc.Image
		}

		return services, nil
	}

	return nil, nil
}

type ComposerJSON struct {
	Type string `json:"type"`
}

func detectProjectType(projectRoot string) string {
	composerPath := filepath.Join(projectRoot, "composer.json")
	data, err := os.ReadFile(composerPath)
	if err != nil {
		return "generic"
	}

	var composer ComposerJSON
	if err := json.Unmarshal(data, &composer); err != nil {
		return "generic"
	}

	projectType := strings.ToLower(composer.Type)
	switch projectType {
	case "project", "symfony":
		return "symfony"
	case "laravel":
		return "laravel"
	default:
		return "generic"
	}
}

func detectEnvVars(projectRoot string) ([]string, error) {
	envPath := filepath.Join(projectRoot, ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return nil, nil
	}

	lines := strings.Split(string(data), "\n")
	var vars []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimSpace(parts[0])
				if varName != "" {
					vars = append(vars, varName)
				}
			}
		}
	}

	return vars, nil
}

func generateSuggestedConfig(projectType string, services map[string]string) string {
	var dbService string
	for name, img := range services {
		if strings.Contains(img, "mysql") || strings.Contains(img, "mariadb") || strings.Contains(img, "postgres") {
			dbService = name
			break
		}
	}

	suggested := fmt.Sprintf(`{
  "$schema": "https://raw.githubusercontent.com/mkrowiarz/mcp-symfony-stack/main/schema.json",
  "project": {
    "name": "<your-project-name>",
    "type": "%s"
  },
  "docker": {
    "compose_files": ["docker-compose.yaml"]
  }
}
// Note: schema.json will be generated in phase 2 from config structs
`, projectType)

	if dbService != "" {
		suggested += `,
  "database": {
    "service": "%s",
    "dsn": "${DATABASE_URL}",
    "allowed": ["<your-database-name>", "<your-database-name>_test", "<your-database-name>_wt_*"],
    "dumps_path": "var/dumps"
  }
`
	}

	suggested += `
}`

	if dbService != "" {
		return fmt.Sprintf(suggested, dbService)
	}
	return suggested
}

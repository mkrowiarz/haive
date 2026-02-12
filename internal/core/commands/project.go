package commands

import (
	"encoding/json"
	"io/fs"
	"net/url"
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

	if cfg.Project == nil {
		return &types.ProjectInfo{
			ConfigSummary:       nil,
			EnvFiles:            detectEnvFiles(projectRoot),
			DockerComposeExists: dockerComposeExists(projectRoot),
		}, nil
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

	composeFiles := findComposeFiles(projectRoot)
	services, err := detectDockerServices(projectRoot, composeFiles)
	if err != nil {
		return nil, err
	}

	projectType := detectProjectType(projectRoot)
	projectName := detectProjectName(projectRoot)
	dbService, dbName := detectDatabase(projectRoot, services)

	detectedEnvVars, err := detectEnvVars(projectRoot)
	if err != nil {
		return nil, err
	}

	suggestedConfig := generateSuggestedConfig(projectName, projectType, composeFiles, dbService, dbName)

	return &types.InitSuggestion{
		SuggestedConfig:  suggestedConfig,
		DetectedServices: services,
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
	paths := []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}
	for _, path := range paths {
		fullPath := filepath.Join(projectRoot, path)
		if _, err := os.Stat(fullPath); err == nil {
			return true
		}
	}
	return false
}

func findComposeFiles(projectRoot string) []string {
	var files []string

	rootPatterns := []string{
		"compose.yaml", "compose.yml",
		"docker-compose.yaml", "docker-compose.yml",
	}
	for _, p := range rootPatterns {
		fullPath := filepath.Join(projectRoot, p)
		if _, err := os.Stat(fullPath); err == nil {
			if !strings.Contains(strings.ToLower(p), "prod") {
				files = append(files, p)
			}
		}
	}

	dockerDir := filepath.Join(projectRoot, "docker")
	filepath.WalkDir(dockerDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if (strings.HasPrefix(name, "compose") || strings.HasPrefix(name, "docker-compose")) &&
			(strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")) &&
			!strings.Contains(name, "prod") {
			relPath, _ := filepath.Rel(projectRoot, path)
			files = append(files, relPath)
		}
		return nil
	})

	return files
}

type DockerCompose struct {
	Services map[string]struct {
		Image string `yaml:"image"`
	} `yaml:"services"`
}

func detectDockerServices(projectRoot string, composeFiles []string) (map[string]string, error) {
	services := make(map[string]string)

	for _, composeFile := range composeFiles {
		fullPath := filepath.Join(projectRoot, composeFile)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		var dc DockerCompose
		if err := yaml.Unmarshal(data, &dc); err != nil {
			continue
		}

		for name, svc := range dc.Services {
			services[name] = svc.Image
		}
	}

	return services, nil
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

func detectProjectName(projectRoot string) string {
	composerPath := filepath.Join(projectRoot, "composer.json")
	if data, err := os.ReadFile(composerPath); err == nil {
		var composer struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(data, &composer) == nil && composer.Name != "" {
			parts := strings.Split(composer.Name, "/")
			if len(parts) == 2 {
				return parts[1]
			}
			return composer.Name
		}
	}

	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return "project"
	}
	return filepath.Base(absPath)
}

func detectDatabase(projectRoot string, services map[string]string) (service, dbName string) {
	for name, img := range services {
		imgLower := strings.ToLower(img)
		if strings.Contains(imgLower, "mysql") ||
			strings.Contains(imgLower, "mariadb") ||
			strings.Contains(imgLower, "postgres") {
			service = name
			break
		}
	}

	if service == "" {
		return "", ""
	}

	envFiles := []string{".env", ".env.local"}
	for _, envFile := range envFiles {
		envPath := filepath.Join(projectRoot, envFile)
		if data, err := os.ReadFile(envPath); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATABASE_URL=") {
					dsn := strings.TrimPrefix(line, "DATABASE_URL=")
					dsn = strings.Trim(dsn, "\"'")
					if parsed, err := url.Parse(dsn); err == nil {
						dbName = strings.TrimPrefix(parsed.Path, "/")
					}
					return service, dbName
				}
			}
		}
	}

	return service, dbName
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

func generateSuggestedConfig(projectName, projectType string, composeFiles []string, dbService, dbName string) string {
	cfg := map[string]interface{}{
		"$schema": "https://raw.githubusercontent.com/mkrowiarz/mcp-symfony-stack/main/schema.json",
		"project": map[string]string{
			"name": projectName,
			"type": projectType,
		},
		"docker": map[string]interface{}{
			"compose_files": composeFiles,
		},
	}

	if dbService != "" && dbName != "" {
		cfg["database"] = map[string]interface{}{
			"service":    dbService,
			"dsn":        "${DATABASE_URL}",
			"dumps_path": "var/dumps",
		}
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	return string(data)
}

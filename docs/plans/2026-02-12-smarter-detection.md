# Smarter Project Detection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Generate complete, ready-to-use config from `project.init` by detecting compose files, project name, and database info automatically.

**Architecture:** Extend `internal/core/commands/project.go` with new detection functions. Find compose files recursively in `docker/`, parse all found files for services, detect project name from composer.json, extract database name from DATABASE_URL.

**Tech Stack:** Go, standard library (os, filepath, path/filepath, net/url), gopkg.in/yaml.v3

---

## Task 1: Add findComposeFiles Function

**Files:**
- Modify: `internal/core/commands/project.go`

**Step 1: Add findComposeFiles function after dockerComposeExists**

```go
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
```

**Step 2: Add missing import**

Add `"io/fs"` to imports.

**Step 3: Build and verify**

```bash
go build ./internal/core/commands
```

Expected: No errors

**Step 4: Commit**

```bash
git add internal/core/commands/project.go
git commit -m "feat(detection): add findComposeFiles function"
```

---

## Task 2: Add detectProjectName Function

**Files:**
- Modify: `internal/core/commands/project.go`

**Step 1: Add detectProjectName function after detectProjectType**

```go
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

	return filepath.Base(projectRoot)
}
```

**Step 2: Build and verify**

```bash
go build ./internal/core/commands
```

Expected: No errors

**Step 3: Commit**

```bash
git add internal/core/commands/project.go
git commit -m "feat(detection): add detectProjectName function"
```

---

## Task 3: Add detectDatabase Function

**Files:**
- Modify: `internal/core/commands/project.go`

**Step 1: Add detectDatabase function after detectProjectName**

```go
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

	envPath := filepath.Join(projectRoot, ".env")
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
				break
			}
		}
	}

	return service, dbName
}
```

**Step 2: Add missing import**

Add `"net/url"` to imports.

**Step 3: Build and verify**

```bash
go build ./internal/core/commands
```

Expected: No errors

**Step 4: Commit**

```bash
git add internal/core/commands/project.go
git commit -m "feat(detection): add detectDatabase function"
```

---

## Task 4: Update detectDockerServices to Accept Compose Files

**Files:**
- Modify: `internal/core/commands/project.go`

**Step 1: Replace detectDockerServices function**

```go
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
```

**Step 2: Build and verify**

```bash
go build ./internal/core/commands
```

Expected: No errors

**Step 3: Commit**

```bash
git add internal/core/commands/project.go
git commit -m "refactor(detection): detectDockerServices accepts compose files list"
```

---

## Task 5: Add generateSuggestedConfig Function

**Files:**
- Modify: `internal/core/commands/project.go`

**Step 1: Replace generateSuggestedConfig function**

```go
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
			"allowed":    []string{},
			"dumps_path": "var/dumps",
		}
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	return string(data)
}
```

**Step 2: Build and verify**

```bash
go build ./internal/core/commands
```

Expected: No errors

**Step 3: Commit**

```bash
git add internal/core/commands/project.go
git commit -m "feat(detection): generateSuggestedConfig creates complete config"
```

---

## Task 6: Update Init Function

**Files:**
- Modify: `internal/core/commands/project.go`

**Step 1: Replace Init function**

```go
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
```

**Step 2: Build and verify**

```bash
go build ./internal/core/commands
```

Expected: No errors

**Step 3: Commit**

```bash
git add internal/core/commands/project.go
git commit -m "feat(detection): Init uses new detection functions"
```

---

## Task 7: Add Tests for Detection Functions

**Files:**
- Modify: `internal/core/commands/project_test.go`

**Step 1: Add tests for findComposeFiles**

```go
func TestFindComposeFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test structure
	os.WriteFile(filepath.Join(tmpDir, "compose.yaml"), []byte{}, 0644)
	os.MkdirAll(filepath.Join(tmpDir, "docker", "dev"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "docker", "dev", "compose.app.yaml"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tmpDir, "docker", "dev", "compose.prod.yaml"), []byte{}, 0644)

	files := findComposeFiles(tmpDir)

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(files), files)
	}

	for _, f := range files {
		if strings.Contains(f, "prod") {
			t.Errorf("should not include prod files: %s", f)
		}
	}
}
```

**Step 2: Add tests for detectProjectName**

```go
func TestDetectProjectName(t *testing.T) {
	t.Run("from composer.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte(`{"name": "acme/phoenix"}`), 0644)

		name := detectProjectName(tmpDir)
		if name != "phoenix" {
			t.Errorf("expected 'phoenix', got '%s'", name)
		}
	})

	t.Run("from directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		name := detectProjectName(tmpDir)
		expected := filepath.Base(tmpDir)
		if name != expected {
			t.Errorf("expected '%s', got '%s'", expected, name)
		}
	})
}
```

**Step 3: Add tests for detectDatabase**

```go
func TestDetectDatabase(t *testing.T) {
	t.Run("detects mysql service", func(t *testing.T) {
		tmpDir := t.TempDir()
		services := map[string]string{"database": "mariadb:10.5", "php": "app-php"}
		os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(`DATABASE_URL=mysql://root:pass@localhost:3306/mytower_eu`), 0644)

		service, dbName := detectDatabase(tmpDir, services)
		if service != "database" {
			t.Errorf("expected 'database', got '%s'", service)
		}
		if dbName != "mytower_eu" {
			t.Errorf("expected 'mytower_eu', got '%s'", dbName)
		}
	})

	t.Run("no database service", func(t *testing.T) {
		services := map[string]string{"php": "app-php"}
		service, dbName := detectDatabase(".", services)
		if service != "" || dbName != "" {
			t.Errorf("expected empty, got '%s' '%s'", service, dbName)
		}
	})
}
```

**Step 4: Run tests**

```bash
go test ./internal/core/commands -v -run "TestFind|TestDetect"
```

Expected: All tests pass

**Step 5: Commit**

```bash
git add internal/core/commands/project_test.go
git commit -m "test(detection): add tests for detection functions"
```

---

## Task 8: Final Verification

**Step 1: Run all tests**

```bash
go test ./...
```

Expected: All tests pass

**Step 2: Build binary**

```bash
go build -o pm ./cmd/pm
```

Expected: No errors

**Step 3: Manual test**

```bash
# Test in a real project directory
cd /path/to/your/project
/path/to/pm --mcp
# Call project.init and verify output
```

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat(detection): complete smarter project detection"
```

---

## Success Criteria

- ✅ `findComposeFiles` finds compose files in root and `docker/**/`
- ✅ `findComposeFiles` skips files with "prod" in name
- ✅ `detectProjectName` extracts name from composer.json
- ✅ `detectProjectName` falls back to directory name
- ✅ `detectDatabase` finds database service by image
- ✅ `detectDatabase` extracts DB name from DATABASE_URL
- ✅ `Init` generates complete config with all detected values
- ✅ All tests pass

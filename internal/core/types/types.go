package types

import (
	"fmt"
	"net/url"
	"time"
)

type ErrCode string

const (
	ErrConfigMissing       ErrCode = "CONFIG_MISSING"
	ErrConfigInvalid       ErrCode = "CONFIG_INVALID"
	ErrInvalidName         ErrCode = "INVALID_NAME"
	ErrPathTraversal       ErrCode = "PATH_TRAVERSAL"
	ErrDbNotAllowed        ErrCode = "DB_NOT_ALLOWED"
	ErrDbIsDefault         ErrCode = "DB_IS_DEFAULT"
	ErrFileNotFound        ErrCode = "FILE_NOT_FOUND"
	ErrInvalidWorktree     ErrCode = "INVALID_WORKTREE"
	ErrDependenciesMissing ErrCode = "DEPENDENCIES_MISSING"
)

type CommandError struct {
	Code    ErrCode `json:"code"`
	Message string  `json:"message"`
}

func (e *CommandError) Error() string {
	return e.Message
}

type ProgressStage string

const (
	StageDumping   ProgressStage = "dumping"
	StageCreating  ProgressStage = "creating"
	StageImporting ProgressStage = "importing"
	StageCloning   ProgressStage = "cloning"
	StagePatching  ProgressStage = "patching"
)

type ProgressFunc func(stage ProgressStage, detail string)

type ProjectInfo struct {
	ConfigSummary       *ConfigSummary `json:"config_summary"`
	EnvFiles            []string       `json:"env_files"`
	DockerComposeExists bool           `json:"docker_compose_exists"`
}

type ConfigSummary struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type InitSuggestion struct {
	SuggestedConfig  string            `json:"suggested_config"`
	DetectedServices map[string]string `json:"detected_services"`
	DetectedEnvVars  []string          `json:"detected_env_vars"`
}

type Config struct {
	Project   *Project   `json:"project"`
	Docker    *Docker    `json:"docker"`
	Database  *Database  `json:"database,omitempty"`
	Worktrees *Worktrees `json:"worktrees,omitempty"`
}

type Project struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Docker struct {
	ComposeFiles []string `json:"compose_files,omitempty"`
	ProjectName  string   `json:"project_name,omitempty"`
}

type Database struct {
	Service   string   `json:"service"`
	DSN       string   `json:"dsn"`
	Allowed   []string `json:"allowed"`
	DumpsPath string   `json:"dumps_path,omitempty"`
	Prefix    string   `json:"prefix,omitempty"`
}

type Worktrees struct {
	BasePath      string        `json:"base_path"`
	DBPerWorktree bool          `json:"db_per_worktree,omitempty"`
	DBPrefix      string        `json:"db_prefix,omitempty"`
	Copy          *WorktreeCopy `json:"copy,omitempty"`
}

type WorktreeCopy struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

type WorktreeInfo struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
	IsMain bool   `json:"is_main"`
}

type WorktreeCreateResult struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
}

type WorktreeRemoveResult struct {
	Path string `json:"path"`
}

type WorkflowCreateResult struct {
	WorktreePath   string `json:"worktree_path"`
	WorktreeBranch string `json:"worktree_branch"`
	DatabaseName   string `json:"database_name,omitempty"`
	ClonedFrom     string `json:"cloned_from,omitempty"`
}

type WorkflowRemoveResult struct {
	WorktreePath string `json:"worktree_path"`
	DatabaseName string `json:"database_name,omitempty"`
}

type DumpResult struct {
	Path     string        `json:"path"`
	Size     int64         `json:"size"`
	Database string        `json:"database"`
	Duration time.Duration `json:"duration"`
}

type CreateResult struct {
	Database string `json:"database"`
}

type ImportResult struct {
	Path     string        `json:"path"`
	Database string        `json:"database"`
	Duration time.Duration `json:"duration"`
}

type DropResult struct {
	Database string `json:"database"`
}

type DSN struct {
	Engine        string
	User          string
	Password      string
	Host          string
	Port          string
	Database      string
	ServerVersion string
}

// String returns the DSN as a URL string
func (d *DSN) String() string {
	scheme := d.Engine
	if scheme == "postgresql" {
		scheme = "postgres"
	}

	hostPort := d.Host
	if d.Port != "" {
		hostPort = fmt.Sprintf("%s:%s", d.Host, d.Port)
	}

	path := "/" + d.Database

	u := &url.URL{
		Scheme: scheme,
		Host:   hostPort,
		Path:   path,
	}

	if d.User != "" || d.Password != "" {
		u.User = url.UserPassword(d.User, d.Password)
	}

	if d.ServerVersion != "" {
		q := u.Query()
		q.Set("serverVersion", d.ServerVersion)
		u.RawQuery = q.Encode()
	}

	return u.String()
}

type DatabaseInfo struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

type DatabaseListResult struct {
	Databases []DatabaseInfo `json:"databases"`
}

type CloneResult struct {
	Source   string        `json:"source"`
	Target   string        `json:"target"`
	Size     int64         `json:"size"`
	Duration time.Duration `json:"duration"`
}

type DumpFileInfo struct {
	Name     string `json:"name"`
	Database string `json:"database"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

type DumpsListResult struct {
	Dumps []DumpFileInfo `json:"dumps"`
}

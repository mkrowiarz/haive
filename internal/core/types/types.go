package types

type ErrCode string

const (
	ErrConfigMissing ErrCode = "CONFIG_MISSING"
	ErrConfigInvalid ErrCode = "CONFIG_INVALID"
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

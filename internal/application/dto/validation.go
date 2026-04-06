// Package dto provides Data Transfer Objects for API request/response structures.
package dto

import (
	"time"

	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// ============================================
// Validation Configuration DTOs
// ============================================

// ValidationConfigDTO represents execution validation configuration.
type ValidationConfigDTO struct {
	Enabled       bool                   `json:"enabled" example:"true"`
	Formatting    FormattingConfigDTO    `json:"formatting"`
	Linting       LintingConfigDTO       `json:"linting"`
	Testing       TestingConfigDTO       `json:"testing"`
	AutoDetection AutoDetectionConfigDTO `json:"autoDetection"`
} // @name ValidationConfig

// FormattingConfigDTO represents formatting validation configuration.
type FormattingConfigDTO struct {
	Enabled         bool            `json:"enabled" example:"true"`
	Tools           []ToolCommandDTO `json:"tools"`
	FailureStrategy string          `json:"failureStrategy" example:"fail_fast"`
} // @name FormattingConfig

// LintingConfigDTO represents lint validation configuration.
type LintingConfigDTO struct {
	Enabled         bool            `json:"enabled" example:"true"`
	Tools           []ToolCommandDTO `json:"tools"`
	FailureStrategy string          `json:"failureStrategy" example:"warn_continue"`
	AutoFix         bool            `json:"autoFix" example:"false"`
} // @name LintingConfig

// TestingConfigDTO represents test validation configuration.
type TestingConfigDTO struct {
	Enabled         bool           `json:"enabled" example:"true"`
	Command         ToolCommandDTO `json:"command"`
	FailureStrategy string         `json:"failureStrategy" example:"fail_fast"`
	NoTestsStrategy string         `json:"noTestsStrategy" example:"warn"`
} // @name TestingConfig

// AutoDetectionConfigDTO represents auto detection configuration.
type AutoDetectionConfigDTO struct {
	Enabled         bool               `json:"enabled" example:"true"`
	Mode            string             `json:"mode" example:"auto_full"`
	DetectedProject *DetectedProjectDTO `json:"detectedProject,omitempty"`
} // @name AutoDetectionConfig

// ToolCommandDTO represents a tool command configuration.
type ToolCommandDTO struct {
	Name            string            `json:"name" example:"gofmt"`
	Command         string            `json:"command" example:"gofmt"`
	Args            []string          `json:"args" example:"-s,-w"`
	Env             map[string]string `json:"env,omitempty"`
	WorkingDir      string            `json:"workingDir,omitempty" example:"./"`
	Timeout         int               `json:"timeout" example:"30"`       // Timeout in seconds (1-3600)
	CheckConfigFile string            `json:"checkConfigFile,omitempty" example:".golangci.yml"`
} // @name ToolCommand

// ============================================
// Detection Result DTOs
// ============================================

// DetectedProjectDTO represents detected project information.
type DetectedProjectDTO struct {
	Type            string           `json:"type" example:"go"`
	PrimaryLanguage string           `json:"primaryLanguage" example:"Go"`
	Languages       []string         `json:"languages" example:"Go,TypeScript"`
	DetectedTools   []DetectedToolDTO `json:"detectedTools"`
	DetectedAt      string           `json:"detectedAt" example:"2024-01-15T10:30:00Z"`
	Confidence      int              `json:"confidence" example:"85"`
} // @name DetectedProject

// DetectedToolDTO represents a detected tool.
type DetectedToolDTO struct {
	Type               string          `json:"type" example:"formatter"`
	Name               string          `json:"name" example:"gofmt"`
	ConfigFile         string          `json:"configFile,omitempty" example:".golangci.yml"`
	RecommendedCommand ToolCommandDTO  `json:"recommendedCommand"`
	DetectionBasis     string          `json:"detectionBasis" example:".golangci.yml exists"`
} // @name DetectedTool

// ============================================
// Conversion Functions (DTO -> Domain)
// ============================================

// ToValidationConfig converts DTO to domain value object.
func ToValidationConfig(dto ValidationConfigDTO) *valueobject.ExecutionValidationConfig {
	config := &valueobject.ExecutionValidationConfig{
		Enabled:       dto.Enabled,
		Formatting:    ToFormattingConfig(dto.Formatting),
		Linting:       ToLintingConfig(dto.Linting),
		Testing:       ToTestingConfig(dto.Testing),
		AutoDetection: ToAutoDetectionConfig(dto.AutoDetection),
	}
	return config
}

// ToFormattingConfig converts DTO to domain value object.
func ToFormattingConfig(dto FormattingConfigDTO) valueobject.FormattingConfig {
	strategy := valueobject.FailureStrategy(dto.FailureStrategy)
	if !strategy.IsValid() {
		strategy = valueobject.FailFast // default to fail_fast for invalid values
	}
	return valueobject.FormattingConfig{
		Enabled:         dto.Enabled,
		Tools:           ToToolCommands(dto.Tools),
		FailureStrategy: strategy,
	}
}

// ToLintingConfig converts DTO to domain value object.
func ToLintingConfig(dto LintingConfigDTO) valueobject.LintingConfig {
	strategy := valueobject.FailureStrategy(dto.FailureStrategy)
	if !strategy.IsValid() {
		strategy = valueobject.WarnContinue // default to warn_continue for linting
	}
	return valueobject.LintingConfig{
		Enabled:         dto.Enabled,
		Tools:           ToToolCommands(dto.Tools),
		FailureStrategy: strategy,
		AutoFix:         dto.AutoFix,
	}
}

// ToTestingConfig converts DTO to domain value object.
func ToTestingConfig(dto TestingConfigDTO) valueobject.TestingConfig {
	strategy := valueobject.FailureStrategy(dto.FailureStrategy)
	if !strategy.IsValid() {
		strategy = valueobject.FailFast // default to fail_fast for testing
	}
	noTestsStrategy := valueobject.NoTestsStrategy(dto.NoTestsStrategy)
	if !noTestsStrategy.IsValid() {
		noTestsStrategy = valueobject.WarnNoTests // default to warn for no tests
	}
	return valueobject.TestingConfig{
		Enabled:         dto.Enabled,
		Command:         ToToolCommand(dto.Command),
		FailureStrategy: strategy,
		NoTestsStrategy: noTestsStrategy,
	}
}

// ToAutoDetectionConfig converts DTO to domain value object.
func ToAutoDetectionConfig(dto AutoDetectionConfigDTO) valueobject.AutoDetectionConfig {
	mode := valueobject.DetectionMode(dto.Mode)
	if !mode.IsValid() {
		mode = valueobject.AutoDetectFull // default to full auto detection
	}
	config := valueobject.AutoDetectionConfig{
		Enabled: dto.Enabled,
		Mode:    mode,
	}
	if dto.DetectedProject != nil {
		config.DetectedProject = ToDetectedProject(*dto.DetectedProject)
	}
	return config
}

// ToToolCommand converts DTO to domain value object.
// Note: This does not validate the command. Use valueobject.ToolCommand.Validate() for validation.
func ToToolCommand(dto ToolCommandDTO) valueobject.ToolCommand {
	return valueobject.ToolCommand{
		Name:            dto.Name,
		Command:         dto.Command,
		Args:            dto.Args,
		Env:             dto.Env,
		WorkingDir:      dto.WorkingDir,
		Timeout:         dto.Timeout,
		CheckConfigFile: dto.CheckConfigFile,
	}
}

// ToToolCommands converts a slice of DTOs to domain value objects.
func ToToolCommands(dtos []ToolCommandDTO) []valueobject.ToolCommand {
	if dtos == nil {
		return nil
	}
	result := make([]valueobject.ToolCommand, len(dtos))
	for i, dto := range dtos {
		result[i] = ToToolCommand(dto)
	}
	return result
}

// ToDetectedProject converts DTO to domain value object.
func ToDetectedProject(dto DetectedProjectDTO) *valueobject.DetectedProject {
	// Parse detectedAt, use current time as fallback if parsing fails
	var detectedAt time.Time
	if dto.DetectedAt != "" {
		var err error
		detectedAt, err = time.Parse(time.RFC3339, dto.DetectedAt)
		if err != nil {
			detectedAt = time.Now()
		}
	} else {
		detectedAt = time.Now()
	}

	project := valueobject.NewDetectedProject(
		valueobject.ProjectType(dto.Type),
		dto.PrimaryLanguage,
		dto.Confidence,
	)

	project.Languages = dto.Languages
	project.DetectedAt = detectedAt

	for _, tool := range dto.DetectedTools {
		project.AddTool(ToDetectedTool(tool))
	}

	return project
}

// ToDetectedTool converts DTO to domain value object.
func ToDetectedTool(dto DetectedToolDTO) valueobject.DetectedTool {
	tool := valueobject.NewDetectedTool(
		valueobject.ToolType(dto.Type),
		dto.Name,
		dto.DetectionBasis,
		ToToolCommand(dto.RecommendedCommand),
	)
	if dto.ConfigFile != "" {
		tool = tool.WithConfigFile(dto.ConfigFile)
	}
	return tool
}

// ============================================
// Conversion Functions (Domain -> DTO)
// ============================================

// FromValidationConfig converts domain value object to DTO.
func FromValidationConfig(config *valueobject.ExecutionValidationConfig) ValidationConfigDTO {
	if config == nil {
		return ValidationConfigDTO{}
	}
	return ValidationConfigDTO{
		Enabled:       config.Enabled,
		Formatting:    FromFormattingConfig(config.Formatting),
		Linting:       FromLintingConfig(config.Linting),
		Testing:       FromTestingConfig(config.Testing),
		AutoDetection: FromAutoDetectionConfig(config.AutoDetection),
	}
}

// FromFormattingConfig converts domain value object to DTO.
func FromFormattingConfig(config valueobject.FormattingConfig) FormattingConfigDTO {
	return FormattingConfigDTO{
		Enabled:         config.Enabled,
		Tools:           FromToolCommands(config.Tools),
		FailureStrategy: string(config.FailureStrategy),
	}
}

// FromLintingConfig converts domain value object to DTO.
func FromLintingConfig(config valueobject.LintingConfig) LintingConfigDTO {
	return LintingConfigDTO{
		Enabled:         config.Enabled,
		Tools:           FromToolCommands(config.Tools),
		FailureStrategy: string(config.FailureStrategy),
		AutoFix:         config.AutoFix,
	}
}

// FromTestingConfig converts domain value object to DTO.
func FromTestingConfig(config valueobject.TestingConfig) TestingConfigDTO {
	return TestingConfigDTO{
		Enabled:         config.Enabled,
		Command:         FromToolCommand(config.Command),
		FailureStrategy: string(config.FailureStrategy),
		NoTestsStrategy: string(config.NoTestsStrategy),
	}
}

// FromAutoDetectionConfig converts domain value object to DTO.
func FromAutoDetectionConfig(config valueobject.AutoDetectionConfig) AutoDetectionConfigDTO {
	dto := AutoDetectionConfigDTO{
		Enabled: config.Enabled,
		Mode:    string(config.Mode),
	}
	if config.DetectedProject != nil {
		dto.DetectedProject = FromDetectedProject(config.DetectedProject)
	}
	return dto
}

// FromToolCommand converts domain value object to DTO.
func FromToolCommand(cmd valueobject.ToolCommand) ToolCommandDTO {
	return ToolCommandDTO{
		Name:            cmd.Name,
		Command:         cmd.Command,
		Args:            cmd.Args,
		Env:             cmd.Env,
		WorkingDir:      cmd.WorkingDir,
		Timeout:         cmd.Timeout,
		CheckConfigFile: cmd.CheckConfigFile,
	}
}

// FromToolCommands converts a slice of domain value objects to DTOs.
func FromToolCommands(cmds []valueobject.ToolCommand) []ToolCommandDTO {
	if cmds == nil {
		return nil
	}
	result := make([]ToolCommandDTO, len(cmds))
	for i, cmd := range cmds {
		result[i] = FromToolCommand(cmd)
	}
	return result
}

// FromDetectedProject converts domain value object to DTO.
func FromDetectedProject(project *valueobject.DetectedProject) *DetectedProjectDTO {
	if project == nil {
		return nil
	}

	tools := make([]DetectedToolDTO, len(project.DetectedTools))
	for i, tool := range project.DetectedTools {
		tools[i] = FromDetectedTool(tool)
	}

	return &DetectedProjectDTO{
		Type:            string(project.Type),
		PrimaryLanguage: project.PrimaryLanguage,
		Languages:       project.Languages,
		DetectedTools:   tools,
		DetectedAt:      project.DetectedAt.Format(time.RFC3339),
		Confidence:      project.Confidence,
	}
}

// FromDetectedTool converts domain value object to DTO.
func FromDetectedTool(tool valueobject.DetectedTool) DetectedToolDTO {
	return DetectedToolDTO{
		Type:               string(tool.Type),
		Name:               tool.Name,
		ConfigFile:         tool.ConfigFile,
		RecommendedCommand: FromToolCommand(tool.RecommendedCommand),
		DetectionBasis:     tool.DetectionBasis,
	}
}
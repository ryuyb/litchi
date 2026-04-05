package valueobject

import (
	"errors"
	"strings"
	"time"
)

// ============================================
// Validation Status Enums
// ============================================

// ValidationStatus represents the overall validation status.
type ValidationStatus string

const (
	ValidationPassed    ValidationStatus = "passed"
	ValidationFailed    ValidationStatus = "failed"
	ValidationWarned    ValidationStatus = "warned"
	ValidationSkipped   ValidationStatus = "skipped"
)

// FailureStrategy defines how to handle validation failures.
type FailureStrategy string

const (
	// FailFast - Stop immediately on failure
	FailFast FailureStrategy = "fail_fast"
	// AutoFix - Attempt automatic fix and retry
	AutoFix FailureStrategy = "auto_fix"
	// WarnContinue - Log warning and continue
	WarnContinue FailureStrategy = "warn_continue"
	// Skip - Skip this validation step
	Skip FailureStrategy = "skip"
)

// NoTestsStrategy defines how to handle when no tests are found.
type NoTestsStrategy string

const (
	// SkipNoTests - Skip testing
	SkipNoTests NoTestsStrategy = "skip"
	// WarnNoTests - Log warning and continue
	WarnNoTests NoTestsStrategy = "warn"
	// FailNoTests - Fail and require tests to be added
	FailNoTests NoTestsStrategy = "fail"
)

// DetectionMode defines the project detection mode.
type DetectionMode string

const (
	// AutoDetectFull - Full automatic detection
	AutoDetectFull DetectionMode = "auto_full"
	// AutoDetectBasic - Basic detection (language and framework only)
	AutoDetectBasic DetectionMode = "auto_basic"
	// ManualOnly - Disable auto detection, use manual config only
	ManualOnly DetectionMode = "manual_only"
)

// ============================================
// Project and Tool Enums
// ============================================

// ProjectType represents the type of project.
type ProjectType string

const (
	ProjectTypeGo      ProjectType = "go"
	ProjectTypeNodeJS  ProjectType = "nodejs"
	ProjectTypePython  ProjectType = "python"
	ProjectTypeRust    ProjectType = "rust"
	ProjectTypeJava    ProjectType = "java"
	ProjectTypeMixed   ProjectType = "mixed"
	ProjectTypeUnknown ProjectType = "unknown"
)

// ToolType represents the type of tool.
type ToolType string

const (
	ToolTypeFormatter ToolType = "formatter"
	ToolTypeLinter    ToolType = "linter"
	ToolTypeTester    ToolType = "tester"
)

// ============================================
// Configuration Structures
// ============================================

// ToolCommand defines a tool execution command configuration.
type ToolCommand struct {
	// Name is the display name of the tool
	Name string `json:"name"`
	// Command is the executable command
	Command string `json:"command"`
	// Args are the command arguments
	Args []string `json:"args"`
	// Env are environment variables to set
	Env map[string]string `json:"env"`
	// WorkingDir is the working directory relative to worktree root
	WorkingDir string `json:"workingDir"`
	// Timeout is the execution timeout in seconds
	Timeout int `json:"timeout"`
	// CheckConfigFile is the config file to check before execution (optional)
	CheckConfigFile string `json:"checkConfigFile"`
}

// NewToolCommand creates a new ToolCommand.
func NewToolCommand(name, command string, args []string, timeout int) ToolCommand {
	return ToolCommand{
		Name:    name,
		Command: command,
		Args:    args,
		Timeout: timeout,
		Env:     map[string]string{},
	}
}

// WithEnv returns a new ToolCommand with the given environment variables.
// Note: This returns a new instance; the original ToolCommand is not modified.
func (tc ToolCommand) WithEnv(env map[string]string) ToolCommand {
	newTc := tc
	newTc.Env = env
	return newTc
}

// WithWorkingDir returns a new ToolCommand with the given working directory.
// Note: This returns a new instance; the original ToolCommand is not modified.
func (tc ToolCommand) WithWorkingDir(dir string) ToolCommand {
	newTc := tc
	newTc.WorkingDir = dir
	return newTc
}

// WithConfigCheck returns a new ToolCommand with the given config file check.
// Note: This returns a new instance; the original ToolCommand is not modified.
func (tc ToolCommand) WithConfigCheck(configFile string) ToolCommand {
	newTc := tc
	newTc.CheckConfigFile = configFile
	return newTc
}

// Validate validates the tool command configuration.
// Returns error if the command or args contain dangerous characters or is empty.
func (tc ToolCommand) Validate() error {
	if tc.Command == "" {
		return errors.New("command cannot be empty")
	}
	// Prevent command injection - disallow shell metacharacters in command
	dangerousChars := ";&|`$()<>\\"
	if strings.ContainsAny(tc.Command, dangerousChars) {
		return errors.New("command contains dangerous characters: " + dangerousChars)
	}
	// Also check args for dangerous characters
	for _, arg := range tc.Args {
		if strings.ContainsAny(arg, dangerousChars) {
			return errors.New("arg contains dangerous characters: " + dangerousChars)
		}
	}
	// Validate WorkingDir to prevent path traversal
	if tc.WorkingDir != "" {
		if strings.Contains(tc.WorkingDir, "..") {
			return errors.New("workingDir cannot contain path traversal")
		}
		// Check for absolute path (starts with / on Unix or drive letter on Windows)
		if strings.HasPrefix(tc.WorkingDir, "/") || (len(tc.WorkingDir) > 1 && tc.WorkingDir[1] == ':') {
			return errors.New("workingDir must be relative")
		}
	}
	return nil
}

// ExecutionValidationConfig is the complete execution validation configuration.
type ExecutionValidationConfig struct {
	// Enabled indicates whether execution validation is enabled
	Enabled bool `json:"enabled"`
	// Formatting is the formatting configuration
	Formatting FormattingConfig `json:"formatting"`
	// Linting is the linting configuration
	Linting LintingConfig `json:"linting"`
	// Testing is the testing configuration
	Testing TestingConfig `json:"testing"`
	// AutoDetection is the auto detection configuration
	AutoDetection AutoDetectionConfig `json:"autoDetection"`
}

// FormattingConfig defines formatting validation configuration.
type FormattingConfig struct {
	// Enabled indicates whether formatting validation is enabled
	Enabled bool `json:"enabled"`
	// Tools is the list of formatting tools to execute (in order)
	Tools []ToolCommand `json:"tools"`
	// FailureStrategy defines how to handle formatting failures
	FailureStrategy FailureStrategy `json:"failureStrategy"`
}

// LintingConfig defines lint validation configuration.
type LintingConfig struct {
	// Enabled indicates whether lint validation is enabled
	Enabled bool `json:"enabled"`
	// Tools is the list of lint tools to execute (in order)
	Tools []ToolCommand `json:"tools"`
	// FailureStrategy defines how to handle lint failures
	FailureStrategy FailureStrategy `json:"failureStrategy"`
	// AutoFix indicates whether to attempt auto-fix on lint issues
	AutoFix bool `json:"autoFix"`
}

// TestingConfig defines test validation configuration.
type TestingConfig struct {
	// Enabled indicates whether test validation is enabled
	Enabled bool `json:"enabled"`
	// Command is the test command to execute
	Command ToolCommand `json:"command"`
	// FailureStrategy defines how to handle test failures
	FailureStrategy FailureStrategy `json:"failureStrategy"`
	// NoTestsStrategy defines how to handle when no tests are found
	NoTestsStrategy NoTestsStrategy `json:"noTestsStrategy"`
}

// AutoDetectionConfig defines auto detection configuration.
type AutoDetectionConfig struct {
	// Enabled indicates whether auto detection is enabled
	Enabled bool `json:"enabled"`
	// Mode defines the detection mode
	Mode DetectionMode `json:"mode"`
	// DetectedProject contains the detected project info (read-only, filled by system)
	DetectedProject *DetectedProject `json:"detectedProject,omitempty"`
}

// ============================================
// Detection Result Structures
// ============================================

// DetectedTool represents a detected tool.
type DetectedTool struct {
	// Type is the tool type
	Type ToolType `json:"type"`
	// Name is the tool name
	Name string `json:"name"`
	// ConfigFile is the config file path (if detected)
	ConfigFile string `json:"configFile"`
	// RecommendedCommand is the recommended command for this tool
	RecommendedCommand ToolCommand `json:"recommendedCommand"`
	// DetectionBasis is the reason for detection (e.g., ".golangci.yml exists")
	DetectionBasis string `json:"detectionBasis"`
}

// NewDetectedTool creates a new DetectedTool.
func NewDetectedTool(toolType ToolType, name, basis string, cmd ToolCommand) DetectedTool {
	return DetectedTool{
		Type:             toolType,
		Name:             name,
		DetectionBasis:   basis,
		RecommendedCommand: cmd,
	}
}

// WithConfigFile sets the config file for the detected tool.
func (dt DetectedTool) WithConfigFile(configFile string) DetectedTool {
	dt.ConfigFile = configFile
	return dt
}

// DetectedProject represents detected project information.
type DetectedProject struct {
	// Type is the project type
	Type ProjectType `json:"type"`
	// PrimaryLanguage is the primary programming language
	PrimaryLanguage string `json:"primaryLanguage"`
	// Languages is the list of all languages (for multi-language projects)
	Languages []string `json:"languages"`
	// DetectedTools is the list of detected tools
	DetectedTools []DetectedTool `json:"detectedTools"`
	// DetectedAt is the detection timestamp
	DetectedAt time.Time `json:"detectedAt"`
	// Confidence is the detection confidence (0-100)
	Confidence int `json:"confidence"`
}

// NewDetectedProject creates a new DetectedProject.
func NewDetectedProject(projectType ProjectType, primaryLang string, confidence int) *DetectedProject {
	return &DetectedProject{
		Type:            projectType,
		PrimaryLanguage: primaryLang,
		Languages:       []string{primaryLang},
		DetectedTools:   []DetectedTool{},
		DetectedAt:      time.Now(),
		Confidence:      confidence,
	}
}

// AddLanguage adds a language to the project.
func (dp *DetectedProject) AddLanguage(lang string) {
	for _, l := range dp.Languages {
		if l == lang {
			return
		}
	}
	dp.Languages = append(dp.Languages, lang)
}

// AddTool adds a detected tool to the project.
func (dp *DetectedProject) AddTool(tool DetectedTool) {
	dp.DetectedTools = append(dp.DetectedTools, tool)
}

// GetToolsByType returns tools of a specific type.
func (dp *DetectedProject) GetToolsByType(toolType ToolType) []DetectedTool {
	tools := []DetectedTool{}
	for _, t := range dp.DetectedTools {
		if t.Type == toolType {
			tools = append(tools, t)
		}
	}
	return tools
}

// HasToolType checks if any tool of the given type exists.
func (dp *DetectedProject) HasToolType(toolType ToolType) bool {
	for _, t := range dp.DetectedTools {
		if t.Type == toolType {
			return true
		}
	}
	return false
}

// ============================================
// Validation Result Structures
// ============================================

// FormatResult represents formatting validation result.
type FormatResult struct {
	// Success indicates whether formatting passed
	Success bool `json:"success"`
	// Output is the command output
	Output string `json:"output"`
	// Duration is the execution duration in milliseconds
	Duration int64 `json:"duration"`
	// ToolName is the tool that was executed
	ToolName string `json:"toolName"`
}

// NewFormatResult creates a new FormatResult.
func NewFormatResult(success bool, output, toolName string, duration int64) *FormatResult {
	return &FormatResult{
		Success:  success,
		Output:   output,
		Duration: duration,
		ToolName: toolName,
	}
}

// LintResult represents lint validation result.
type LintResult struct {
	// Success indicates whether lint passed
	Success bool `json:"success"`
	// Output is the command output
	Output string `json:"output"`
	// IssuesFound is the number of issues found
	IssuesFound int `json:"issuesFound"`
	// IssuesFixed is the number of issues fixed (if autoFix was enabled)
	IssuesFixed int `json:"issuesFixed"`
	// Duration is the execution duration in milliseconds
	Duration int64 `json:"duration"`
	// ToolName is the tool that was executed
	ToolName string `json:"toolName"`
}

// NewLintResult creates a new LintResult.
func NewLintResult(success bool, output, toolName string, issuesFound, issuesFixed int, duration int64) *LintResult {
	return &LintResult{
		Success:     success,
		Output:      output,
		IssuesFound: issuesFound,
		IssuesFixed: issuesFixed,
		Duration:    duration,
		ToolName:    toolName,
	}
}

// TestFailure represents a single test failure.
type TestFailure struct {
	// Name is the test name
	Name string `json:"name"`
	// Message is the failure message
	Message string `json:"message"`
}

// ValidationTestResult represents test validation result.
// Note: This extends the existing TestResult value object.
type ValidationTestResult struct {
	// Success indicates whether tests passed
	Success bool `json:"success"`
	// Output is the command output
	Output string `json:"output"`
	// Passed is the number of passed tests
	Passed int `json:"passed"`
	// Failed is the number of failed tests
	Failed int `json:"failed"`
	// Duration is the execution duration in milliseconds
	Duration int64 `json:"duration"`
	// ToolName is the tool that was executed
	ToolName string `json:"toolName"`
	// Failures is the list of test failures
	Failures []TestFailure `json:"failures"`
}

// NewValidationTestResult creates a new ValidationTestResult.
func NewValidationTestResult(success bool, output, toolName string, passed, failed int, duration int64) *ValidationTestResult {
	return &ValidationTestResult{
		Success:  success,
		Output:   output,
		Passed:   passed,
		Failed:   failed,
		Duration: duration,
		ToolName: toolName,
		Failures: []TestFailure{},
	}
}

// AddFailure adds a test failure.
func (r *ValidationTestResult) AddFailure(name, message string) {
	r.Failures = append(r.Failures, TestFailure{
		Name:    name,
		Message: message,
	})
}

// ValidationResult represents the complete validation result.
type ValidationResult struct {
	// FormatResult is the formatting result
	FormatResult *FormatResult `json:"formatResult"`
	// LintResult is the lint result
	LintResult *LintResult `json:"lintResult"`
	// TestResult is the test result
	TestResult *ValidationTestResult `json:"testResult"`
	// OverallStatus is the overall validation status
	OverallStatus ValidationStatus `json:"overallStatus"`
	// Warnings is the list of warning messages
	Warnings []string `json:"warnings"`
	// Duration is the total execution duration in milliseconds
	Duration int64 `json:"duration"`
}

// NewValidationResult creates a new ValidationResult.
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		OverallStatus: ValidationPassed,
		Warnings:      []string{},
	}
}

// AddWarning adds a warning message.
func (r *ValidationResult) AddWarning(warning string) {
	r.Warnings = append(r.Warnings, warning)
}

// IsPassed returns true if validation passed.
func (r *ValidationResult) IsPassed() bool {
	return r.OverallStatus == ValidationPassed
}

// IsFailed returns true if validation failed.
func (r *ValidationResult) IsFailed() bool {
	return r.OverallStatus == ValidationFailed
}

// HasWarnings returns true if there are warnings.
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}
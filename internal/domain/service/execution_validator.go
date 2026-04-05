package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// ValidationRequest represents a validation execution request.
type ValidationRequest struct {
	// SessionID is the work session ID
	SessionID uuid.UUID
	// TaskID is the task ID being validated
	TaskID uuid.UUID
	// WorktreePath is the path to the git worktree
	WorktreePath string
	// Config is the validation configuration
	Config *valueobject.ExecutionValidationConfig
}

// ExecutionValidator executes the complete validation workflow.
// It orchestrates formatting, linting, and testing validators.
type ExecutionValidator interface {
	// Validate executes the complete validation workflow.
	// Executes format, lint, and test in sequence based on configuration.
	// Handles failure strategies according to configuration.
	Validate(ctx context.Context, req *ValidationRequest) (*valueobject.ValidationResult, error)

	// FormatOnly executes only formatting validation.
	// Useful for testing or partial validation.
	FormatOnly(ctx context.Context, req *ValidationRequest) (*valueobject.FormatResult, error)

	// LintOnly executes only lint validation.
	// Useful for testing or partial validation.
	LintOnly(ctx context.Context, req *ValidationRequest) (*valueobject.LintResult, error)

	// TestOnly executes only test validation.
	// Useful for testing or partial validation.
	TestOnly(ctx context.Context, req *ValidationRequest) (*valueobject.ValidationTestResult, error)
}

// FormatValidator executes formatting commands.
type FormatValidator interface {
	// Execute executes formatting tools in sequence.
	// Each tool is executed with the configured timeout.
	// Returns the first failure result or combined success.
	Execute(ctx context.Context, worktreePath string, tools []valueobject.ToolCommand) (*valueobject.FormatResult, error)
}

// LintValidator executes lint commands.
type LintValidator interface {
	// Execute executes lint tools in sequence.
	// If autoFix is true, tools are executed with fix flags.
	// Returns aggregated lint results.
	Execute(ctx context.Context, worktreePath string, tools []valueobject.ToolCommand, autoFix bool) (*valueobject.LintResult, error)
}

// TestValidator executes test commands.
type TestValidator interface {
	// Execute executes the test command.
	// Returns test pass/fail status and detailed results.
	Execute(ctx context.Context, worktreePath string, command valueobject.ToolCommand) (*valueobject.ValidationTestResult, error)
}

// OutputParser parses tool output to extract structured results.
type OutputParser interface {
	// ParseFormatOutput parses formatting tool output.
	// Returns success status and summary message.
	ParseFormatOutput(output string, toolName string) (success bool, message string)

	// ParseLintOutput parses lint tool output.
	// Returns issues found, issues fixed, and summary message.
	ParseLintOutput(output string, toolName string) (issuesFound int, issuesFixed int, message string)

	// ParseTestOutput parses test tool output.
	// Returns passed count, failed count, and failure details.
	ParseTestOutput(output string, toolName string) (passed int, failed int, failures []valueobject.TestFailure)

	// SupportsTool checks if this parser supports the given tool.
	SupportsTool(toolName string) bool
}

// DefaultConfigGenerator generates default validation config from detected project.
type DefaultConfigGenerator interface {
	// Generate generates validation configuration from detected project.
	// Returns a complete ExecutionValidationConfig with recommended tools.
	Generate(detected *valueobject.DetectedProject) *valueobject.ExecutionValidationConfig

	// GenerateForLanguage generates default config for a specific language.
	// Used as fallback when detection fails but language is known.
	GenerateForLanguage(language string) *valueobject.ExecutionValidationConfig
}
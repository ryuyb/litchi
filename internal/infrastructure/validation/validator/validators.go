package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/validation/command"
	"go.uber.org/zap"
)

// FormatValidatorImpl executes formatting commands.
type FormatValidatorImpl struct {
	executor *command.Executor
	parser   service.OutputParser
	logger   *zap.Logger
}

// FormatValidatorParams contains dependencies for FormatValidator.
type FormatValidatorParams struct {
	Executor *command.Executor
	Parser   service.OutputParser
	Logger   *zap.Logger
}

// NewFormatValidator creates a new format validator.
func NewFormatValidator(p FormatValidatorParams) service.FormatValidator {
	return &FormatValidatorImpl{
		executor: p.Executor,
		parser:   p.Parser,
		logger:   p.Logger.Named("format-validator"),
	}
}

// Execute executes formatting tools in sequence.
func (v *FormatValidatorImpl) Execute(ctx context.Context, worktreePath string, tools []valueobject.ToolCommand) (*valueobject.FormatResult, error) {
	if len(tools) == 0 {
		return &valueobject.FormatResult{
			Success:  true,
			Output:   "no formatting tools configured",
			ToolName: "none",
		}, nil
	}

	// Execute each tool, return first failure
	for _, tool := range tools {
		result, err := v.executeTool(ctx, worktreePath, tool)
		if err != nil {
			return nil, err
		}
		if !result.Success {
			return result, nil
		}
	}

	// All tools passed
	lastTool := tools[len(tools)-1]
	return &valueobject.FormatResult{
		Success:  true,
		Output:   "all formatting tools passed",
		Duration: 0,
		ToolName: lastTool.Name,
	}, nil
}

// executeTool executes a single formatting tool.
func (v *FormatValidatorImpl) executeTool(ctx context.Context, worktreePath string, tool valueobject.ToolCommand) (*valueobject.FormatResult, error) {
	// Validate tool command for security
	if err := tool.Validate(); err != nil {
		return nil, fmt.Errorf("invalid tool configuration: %w", err)
	}

	// Check config file if specified
	if tool.CheckConfigFile != "" && !v.executor.CheckFileExists(worktreePath, tool.CheckConfigFile) {
		v.logger.Debug("skipping tool, config file not found",
			zap.String("tool", tool.Name),
			zap.String("config", tool.CheckConfigFile),
		)
		return &valueobject.FormatResult{
			Success:  true,
			Output:   fmt.Sprintf("skipped (config file %s not found)", tool.CheckConfigFile),
			ToolName: tool.Name,
		}, nil
	}

	v.logger.Debug("executing format tool",
		zap.String("tool", tool.Name),
		zap.String("command", tool.Command),
		zap.Strings("args", tool.Args),
	)

	output, duration, err := v.executor.ExecWithOutput(ctx, worktreePath, tool.Command, tool.Args, tool.Env, tool.Timeout)
	if err != nil {
		return nil, err
	}

	success, message := v.parser.ParseFormatOutput(output, tool.Name)

	return &valueobject.FormatResult{
		Success:  success,
		Output:   message,
		Duration: duration,
		ToolName: tool.Name,
	}, nil
}

// LintValidatorImpl executes lint commands.
type LintValidatorImpl struct {
	executor *command.Executor
	parser   service.OutputParser
	logger   *zap.Logger
}

// LintValidatorParams contains dependencies for LintValidator.
type LintValidatorParams struct {
	Executor *command.Executor
	Parser   service.OutputParser
	Logger   *zap.Logger
}

// NewLintValidator creates a new lint validator.
func NewLintValidator(p LintValidatorParams) service.LintValidator {
	return &LintValidatorImpl{
		executor: p.Executor,
		parser:   p.Parser,
		logger:   p.Logger.Named("lint-validator"),
	}
}

// Execute executes lint tools in sequence.
func (v *LintValidatorImpl) Execute(ctx context.Context, worktreePath string, tools []valueobject.ToolCommand, autoFix bool) (*valueobject.LintResult, error) {
	if len(tools) == 0 {
		return &valueobject.LintResult{
			Success:  true,
			Output:   "no lint tools configured",
			ToolName: "none",
		}, nil
	}

	totalIssues := 0
	totalFixed := 0
	totalDuration := int64(0)
	outputs := []string{}
	allSuccess := true

	for _, tool := range tools {
		result, err := v.executeTool(ctx, worktreePath, tool, autoFix)
		if err != nil {
			return nil, err
		}

		totalIssues += result.IssuesFound
		totalFixed += result.IssuesFixed
		totalDuration += result.Duration
		if result.Output != "" {
			outputs = append(outputs, fmt.Sprintf("[%s] %s", tool.Name, result.Output))
		}

		if !result.Success {
			allSuccess = false
			// Return first failure
			return result, nil
		}
	}

	return &valueobject.LintResult{
		Success:     allSuccess && totalIssues == 0,
		Output:      strings.Join(outputs, "\n"),
		IssuesFound: totalIssues,
		IssuesFixed: totalFixed,
		Duration:    totalDuration,
		ToolName:    "combined",
	}, nil
}

// executeTool executes a single lint tool.
func (v *LintValidatorImpl) executeTool(ctx context.Context, worktreePath string, tool valueobject.ToolCommand, autoFix bool) (*valueobject.LintResult, error) {
	// Validate tool command for security
	if err := tool.Validate(); err != nil {
		return nil, fmt.Errorf("invalid tool configuration: %w", err)
	}

	// Check config file if specified
	if tool.CheckConfigFile != "" && !v.executor.CheckFileExists(worktreePath, tool.CheckConfigFile) {
		v.logger.Debug("skipping lint tool, config file not found",
			zap.String("tool", tool.Name),
			zap.String("config", tool.CheckConfigFile),
		)
		return &valueobject.LintResult{
			Success:  true,
			Output:   fmt.Sprintf("skipped (config file %s not found)", tool.CheckConfigFile),
			ToolName: tool.Name,
		}, nil
	}

	// Add --fix flag if autoFix is enabled and not already present
	args := tool.Args
	if autoFix && !containsFlag(args, "--fix") {
		args = append(args, "--fix")
	}

	v.logger.Debug("executing lint tool",
		zap.String("tool", tool.Name),
		zap.String("command", tool.Command),
		zap.Strings("args", args),
	)

	output, duration, err := v.executor.ExecWithOutput(ctx, worktreePath, tool.Command, args, tool.Env, tool.Timeout)
	if err != nil {
		return nil, err
	}

	issuesFound, issuesFixed, message := v.parser.ParseLintOutput(output, tool.Name)

	return &valueobject.LintResult{
		Success:     issuesFound == 0,
		Output:      message,
		IssuesFound: issuesFound,
		IssuesFixed: issuesFixed,
		Duration:    duration,
		ToolName:    tool.Name,
	}, nil
}

// containsFlag checks if args contains a flag.
func containsFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}

// TestValidatorImpl executes test commands.
type TestValidatorImpl struct {
	executor *command.Executor
	parser   service.OutputParser
	logger   *zap.Logger
}

// TestValidatorParams contains dependencies for TestValidator.
type TestValidatorParams struct {
	Executor *command.Executor
	Parser   service.OutputParser
	Logger   *zap.Logger
}

// NewTestValidator creates a new test validator.
func NewTestValidator(p TestValidatorParams) service.TestValidator {
	return &TestValidatorImpl{
		executor: p.Executor,
		parser:   p.Parser,
		logger:   p.Logger.Named("test-validator"),
	}
}

// Execute executes test command.
func (v *TestValidatorImpl) Execute(ctx context.Context, worktreePath string, command valueobject.ToolCommand) (*valueobject.ValidationTestResult, error) {
	v.logger.Debug("executing test command",
		zap.String("name", command.Name),
		zap.String("command", command.Command),
		zap.Strings("args", command.Args),
	)

	output, duration, err := v.executor.ExecWithOutput(ctx, worktreePath, command.Command, command.Args, command.Env, command.Timeout)
	if err != nil {
		return nil, err
	}

	passed, failed, failures := v.parser.ParseTestOutput(output, command.Name)

	return &valueobject.ValidationTestResult{
		Success:  failed == 0,
		Output:   output,
		Passed:   passed,
		Failed:   failed,
		Duration: duration,
		ToolName: command.Name,
		Failures: failures,
	}, nil
}
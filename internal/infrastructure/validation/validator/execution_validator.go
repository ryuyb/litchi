package validator

import (
	"context"
	"time"

	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"go.uber.org/zap"
)

// ExecutionValidatorImpl executes the complete validation workflow.
type ExecutionValidatorImpl struct {
	formatValidator service.FormatValidator
	lintValidator   service.LintValidator
	testValidator   service.TestValidator
	logger          *zap.Logger
}

// ExecutionValidatorParams contains dependencies for ExecutionValidator.
type ExecutionValidatorParams struct {
	FormatValidator service.FormatValidator
	LintValidator   service.LintValidator
	TestValidator   service.TestValidator
	Logger          *zap.Logger
}

// NewExecutionValidator creates a new execution validator.
func NewExecutionValidator(p ExecutionValidatorParams) service.ExecutionValidator {
	return &ExecutionValidatorImpl{
		formatValidator: p.FormatValidator,
		lintValidator:   p.LintValidator,
		testValidator:   p.TestValidator,
		logger:          p.Logger.Named("execution-validator"),
	}
}

// Validate executes the complete validation workflow.
func (v *ExecutionValidatorImpl) Validate(ctx context.Context, req *service.ValidationRequest) (*valueobject.ValidationResult, error) {
	result := valueobject.NewValidationResult()
	startTime := time.Now()

	v.logger.Debug("starting validation",
		zap.String("sessionID", req.SessionID.String()),
		zap.String("taskID", req.TaskID.String()),
	)

	// 1. Formatting validation
	if req.Config != nil && req.Config.Formatting.Enabled {
		formatResult, err := v.FormatOnly(ctx, req)
		if err != nil {
			return nil, err
		}
		result.FormatResult = formatResult

		if !formatResult.Success {
			switch req.Config.Formatting.FailureStrategy {
			case valueobject.FailFast:
				result.OverallStatus = valueobject.ValidationFailed
				result.Duration = time.Since(startTime).Milliseconds()
				v.logger.Warn("format validation failed, stopping",
					zap.String("output", formatResult.Output),
				)
				return result, nil

			case valueobject.WarnContinue:
				result.AddWarning("formatting: " + formatResult.Output)
				v.logger.Warn("format validation failed, continuing",
					zap.String("output", formatResult.Output),
				)

			case valueobject.AutoFix:
				// Formatter should have auto-fixed, continue
				v.logger.Info("format validation completed with auto-fix")

			case valueobject.Skip:
				v.logger.Debug("format validation skipped")
			}
		}
	}

	// 2. Lint validation
	if req.Config != nil && req.Config.Linting.Enabled {
		lintResult, err := v.LintOnly(ctx, req)
		if err != nil {
			return nil, err
		}
		result.LintResult = lintResult

		if !lintResult.Success {
			switch req.Config.Linting.FailureStrategy {
			case valueobject.FailFast:
				result.OverallStatus = valueobject.ValidationFailed
				result.Duration = time.Since(startTime).Milliseconds()
				v.logger.Warn("lint validation failed, stopping",
					zap.Int("issues", lintResult.IssuesFound),
				)
				return result, nil

			case valueobject.WarnContinue:
				result.AddWarning("linting: " + lintResult.Output)
				v.logger.Warn("lint validation failed, continuing",
					zap.Int("issues", lintResult.IssuesFound),
				)

			case valueobject.AutoFix:
				// Linter with auto-fix should have fixed, continue
				v.logger.Info("lint validation completed with auto-fix",
					zap.Int("fixed", lintResult.IssuesFixed),
				)

			case valueobject.Skip:
				v.logger.Debug("lint validation skipped")
			}
		}
	}

	// 3. Test validation
	if req.Config != nil && req.Config.Testing.Enabled {
		testResult, err := v.TestOnly(ctx, req)
		if err != nil {
			return nil, err
		}
		result.TestResult = testResult

		if !testResult.Success {
			switch req.Config.Testing.FailureStrategy {
			case valueobject.FailFast:
				result.OverallStatus = valueobject.ValidationFailed
				result.Duration = time.Since(startTime).Milliseconds()
				v.logger.Warn("test validation failed, stopping",
					zap.Int("passed", testResult.Passed),
					zap.Int("failed", testResult.Failed),
				)
				return result, nil

			case valueobject.WarnContinue:
				result.AddWarning("testing: " + testResult.Output)
				v.logger.Warn("test validation failed, continuing",
					zap.Int("passed", testResult.Passed),
					zap.Int("failed", testResult.Failed),
				)

			case valueobject.AutoFix:
				// Tests need manual fix, mark as failed
				result.OverallStatus = valueobject.ValidationFailed
				result.Duration = time.Since(startTime).Milliseconds()
				v.logger.Warn("test validation failed, auto-fix not supported for tests")
				return result, nil

			case valueobject.Skip:
				v.logger.Debug("test validation skipped")
			}
		}
	}

	// 4. Determine final status
	if len(result.Warnings) > 0 && result.OverallStatus != valueobject.ValidationFailed {
		result.OverallStatus = valueobject.ValidationWarned
	} else if result.OverallStatus == valueobject.ValidationPassed {
		v.logger.Info("validation passed",
			zap.Int64("duration_ms", time.Since(startTime).Milliseconds()),
		)
	}

	result.Duration = time.Since(startTime).Milliseconds()
	return result, nil
}

// FormatOnly executes only formatting validation.
func (v *ExecutionValidatorImpl) FormatOnly(ctx context.Context, req *service.ValidationRequest) (*valueobject.FormatResult, error) {
	if req.Config == nil || !req.Config.Formatting.Enabled {
		return &valueobject.FormatResult{
			Success:  true,
			Output:   "formatting disabled",
			ToolName: "none",
		}, nil
	}

	return v.formatValidator.Execute(ctx, req.WorktreePath, req.Config.Formatting.Tools)
}

// LintOnly executes only lint validation.
func (v *ExecutionValidatorImpl) LintOnly(ctx context.Context, req *service.ValidationRequest) (*valueobject.LintResult, error) {
	if req.Config == nil || !req.Config.Linting.Enabled {
		return &valueobject.LintResult{
			Success:  true,
			Output:   "linting disabled",
			ToolName: "none",
		}, nil
	}

	// AutoFix config controls whether to attempt automatic fix
	autoFix := req.Config.Linting.AutoFix
	return v.lintValidator.Execute(ctx, req.WorktreePath, req.Config.Linting.Tools, autoFix)
}

// TestOnly executes only test validation.
func (v *ExecutionValidatorImpl) TestOnly(ctx context.Context, req *service.ValidationRequest) (*valueobject.ValidationTestResult, error) {
	if req.Config == nil || !req.Config.Testing.Enabled {
		return &valueobject.ValidationTestResult{
			Success:  true,
			Output:   "testing disabled",
			ToolName: "none",
		}, nil
	}

	return v.testValidator.Execute(ctx, req.WorktreePath, req.Config.Testing.Command)
}
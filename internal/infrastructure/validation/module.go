package validation

import (
	"time"

	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/validation/command"
	"github.com/ryuyb/litchi/internal/infrastructure/validation/detector"
	"github.com/ryuyb/litchi/internal/infrastructure/validation/validator"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Module provides all validation-related dependencies.
var Module = fx.Module("validation",
	// Command executor
	fx.Provide(NewCommandExecutorFromConfig),

	// Output parser
	fx.Provide(validator.NewDefaultOutputParser),

	// Language detectors (grouped for injection into CompositeDetector)
	fx.Provide(
		fx.Annotate(detector.NewGoProjectDetector, fx.ResultTags(`group:"detectors"`)),
		fx.Annotate(detector.NewNodeJSProjectDetector, fx.ResultTags(`group:"detectors"`)),
		fx.Annotate(detector.NewPythonProjectDetector, fx.ResultTags(`group:"detectors"`)),
		fx.Annotate(detector.NewRustProjectDetector, fx.ResultTags(`group:"detectors"`)),
	),

	// Composite detector
	fx.Provide(
		fx.Annotate(
			detector.NewCompositeDetector,
			fx.As(new(service.CompositeProjectDetector)),
		),
	),

	// Individual validators
	fx.Provide(
		validator.NewFormatValidator,
		validator.NewLintValidator,
		validator.NewTestValidator,
	),

	// Execution validator
	fx.Provide(
		fx.Annotate(
			validator.NewExecutionValidator,
			fx.As(new(service.ExecutionValidator)),
		),
	),

	// Default config generator
	fx.Provide(NewDefaultConfigGenerator),

	// Validation result repository
	fx.Provide(
		fx.Annotate(
			NewValidationResultRepository,
			fx.As(new(repository.ValidationResultRepository)),
		),
	),
)

// CommandExecutorParams contains config for CommandExecutor.
type CommandExecutorParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

// NewCommandExecutorFromConfig creates a CommandExecutor from config.
func NewCommandExecutorFromConfig(p CommandExecutorParams) *command.Executor {
	timeout := 5 * time.Minute
	if p.Config != nil && p.Config.Failure.Timeout.TestRun != "" {
		if parsed, err := time.ParseDuration(p.Config.Failure.Timeout.TestRun); err == nil {
			timeout = parsed
		} else {
			p.Logger.Warn("failed to parse timeout config, using default",
				zap.String("timeout", p.Config.Failure.Timeout.TestRun),
				zap.Duration("default", timeout),
				zap.Error(err),
			)
		}
	}
	return command.NewExecutor(command.ExecutorParams{
		Logger:  p.Logger,
		Timeout: timeout,
	})
}

// ValidationResultRepoParams contains dependencies for ValidationResultRepository.
type ValidationResultRepoParams struct {
	fx.In

	DB     *gorm.DB
	Logger *zap.Logger
}

// NewValidationResultRepository creates a new validation result repository.
func NewValidationResultRepository(p ValidationResultRepoParams) repository.ValidationResultRepository {
	return NewGormValidationResultRepository(p.DB, p.Logger)
}

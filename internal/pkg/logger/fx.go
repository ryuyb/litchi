package logger

import (
	"context"
	"io"

	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "logger",
		Provides: []string{"*zap.Logger", "*zap.SugaredLogger", "[]io.Writer"},
		Invokes:  []string{"RegisterLifecycle"},
		Depends:  []string{"*config.Config"},
	})
}

// Module provides the logger module for Fx.
var Module = fx.Module("logger",
	fx.Provide(NewLoggerFromConfig, NewSugaredLogger),
	fx.Invoke(RegisterLifecycle),
)

// Params for NewLoggerFromConfig.
type Params struct {
	fx.In

	Cfg *config.Config
}

// LoggerResult holds the logger and writers created.
type LoggerResult struct {
	fx.Out

	Logger  *zap.Logger
	Writers []io.Writer
}

// NewLoggerFromConfig creates a Zap logger based on configuration.
func NewLoggerFromConfig(p Params) (LoggerResult, error) {
	builder := NewLoggerBuilder(&p.Cfg.Logging)
	logger, err := builder.Build()
	if err != nil {
		return LoggerResult{}, err
	}
	return LoggerResult{
		Logger:  logger,
		Writers: builder.Writers(),
	}, nil
}

// NewSugaredLogger creates a sugared logger.
func NewSugaredLogger(logger *zap.Logger) *zap.SugaredLogger {
	return logger.Sugar()
}

// RegisterLifecycle registers logger lifecycle hooks.
func RegisterLifecycle(lifecycle fx.Lifecycle, logger *zap.Logger, writers []io.Writer) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("logger initialized")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Sync logger
			_ = logger.Sync()

			// Close writers that need cleanup (lumberjack, file handles)
			for _, w := range writers {
				if closer, ok := w.(interface{ Close() error }); ok {
					if err := closer.Close(); err != nil {
						logger.Error("failed to close writer on shutdown", zap.Error(err))
					}
				}
			}
			return nil
		},
	})
}

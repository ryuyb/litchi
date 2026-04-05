package logger

import (
	"context"

	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "logger",
		Provides: []string{"*zap.Logger", "*zap.SugaredLogger"},
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

// NewLoggerFromConfig creates a Zap logger based on configuration.
func NewLoggerFromConfig(p Params) (*zap.Logger, error) {
	logCfg := p.Cfg.Logging

	var zapConfig zap.Config
	if logCfg.Format == "json" {
		zapConfig = zap.NewProductionConfig()
		zapConfig.EncoderConfig.TimeKey = "timestamp"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(logCfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Set output paths
	switch logCfg.Output {
	case "stdout":
		zapConfig.OutputPaths = []string{"stdout"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	case "stderr":
		zapConfig.OutputPaths = []string{"stderr"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	default:
		zapConfig.OutputPaths = []string{logCfg.Output}
		zapConfig.ErrorOutputPaths = []string{logCfg.Output}
	}

	return zapConfig.Build()
}

// NewSugaredLogger creates a sugared logger.
func NewSugaredLogger(logger *zap.Logger) *zap.SugaredLogger {
	return logger.Sugar()
}

// RegisterLifecycle registers logger lifecycle hooks.
func RegisterLifecycle(lifecycle fx.Lifecycle, logger *zap.Logger, cfg *config.Config) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("logger initialized")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Sync is only meaningful for file outputs.
			// For stdout/stderr, Sync typically returns EINVAL which is expected.
			if cfg.Logging.Output != "stdout" && cfg.Logging.Output != "stderr" {
				if err := logger.Sync(); err != nil {
					// Log the error but don't fail shutdown
					logger.Error("failed to sync logger on shutdown", zap.Error(err))
				}
			}
			return nil
		},
	})
}

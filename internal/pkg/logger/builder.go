package logger

import (
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ryuyb/litchi/internal/infrastructure/config"
)

// LoggerBuilder builds zap.Logger from LoggingConfig.
type LoggerBuilder struct {
	cfg     *config.LoggingConfig
	level   zapcore.Level
	writers []io.Writer // track writers for cleanup
}

// NewLoggerBuilder creates a new logger builder.
func NewLoggerBuilder(cfg *config.LoggingConfig) *LoggerBuilder {
	return &LoggerBuilder{cfg: cfg}
}

// Build constructs the zap.Logger.
func (b *LoggerBuilder) Build() (*zap.Logger, error) {
	// Parse level
	if b.cfg.Level != "" {
		if err := b.level.UnmarshalText([]byte(b.cfg.Level)); err != nil {
			return nil, fmt.Errorf("invalid log level %q: %w", b.cfg.Level, err)
		}
	} else {
		b.level = zapcore.InfoLevel
	}

	// Validate outputs
	if len(b.cfg.Outputs) == 0 {
		return nil, errors.New("logging.outputs is required")
	}

	// Build cores for each output
	cores := []zapcore.Core{}
	for i, output := range b.cfg.Outputs {
		core, writer, err := b.buildCoreForOutput(output)
		if err != nil {
			return nil, fmt.Errorf("outputs[%d]: %w", i, err)
		}
		cores = append(cores, core)
		if writer != nil {
			b.writers = append(b.writers, writer)
		}
	}

	// Combine cores using Tee (allows multiple outputs)
	combinedCore := zapcore.NewTee(cores...)

	// Build options
	opts := b.buildOptions()

	return zap.New(combinedCore, opts...), nil
}

// Writers returns all writers created during build for lifecycle management.
func (b *LoggerBuilder) Writers() []io.Writer {
	return b.writers
}

// buildCoreForOutput creates a zapcore.Core for a specific output.
func (b *LoggerBuilder) buildCoreForOutput(output config.OutputConfig) (zapcore.Core, io.Writer, error) {
	// Get encoder
	encoder := b.buildEncoder(output)

	// Get writer
	writer, err := b.buildWriter(output)
	if err != nil {
		return nil, nil, err
	}

	// Create core with level enabler
	core := zapcore.NewCore(encoder, zapcore.AddSync(writer), b.level)

	return core, writer, nil
}

// buildEncoder creates the appropriate encoder based on format and output type.
func (b *LoggerBuilder) buildEncoder(output config.OutputConfig) zapcore.Encoder {
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     b.getTimeEncoder(),
		EncodeDuration: b.getDurationEncoder(),
		EncodeLevel:    b.getLevelEncoder(output),
	}

	// Configure caller encoder
	if b.cfg.Caller.Enabled {
		encCfg.EncodeCaller = zapcore.ShortCallerEncoder
	} else {
		encCfg.CallerKey = zapcore.OmitKey
	}

	// Determine format
	format := output.Format
	if format == "" {
		format = "json" // default to json
	}

	if format == "json" {
		return zapcore.NewJSONEncoder(encCfg)
	}
	return zapcore.NewConsoleEncoder(encCfg)
}

// buildWriter creates the io.Writer for an output.
func (b *LoggerBuilder) buildWriter(output config.OutputConfig) (io.Writer, error) {
	switch output.Type {
	case "console":
		return b.getConsoleWriter(output.Console), nil
	case "file":
		return b.getFileWriter(output.Path, output.Rotation)
	default:
		return os.Stdout, nil
	}
}

// getConsoleWriter returns console output writer.
func (b *LoggerBuilder) getConsoleWriter(console config.ConsoleOutputConfig) io.Writer {
	if console.Stream == "stderr" {
		return os.Stderr
	}
	return os.Stdout
}

// getFileWriter returns file output writer with optional rotation.
func (b *LoggerBuilder) getFileWriter(path string, rotation config.RotationConfig) (io.Writer, error) {
	if path == "" {
		return nil, errors.New("file path is required")
	}

	if !rotation.Enabled {
		// Direct file output without rotation
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		return file, nil
	}

	// Use lumberjack for rotation, apply default max_size if not set
	maxSize := rotation.MaxSize
	if maxSize <= 0 {
		maxSize = 100 // default 100MB
	}

	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    maxSize,    // MB
		MaxBackups: rotation.MaxBackups,
		MaxAge:     rotation.MaxAge,     // Days
		Compress:   rotation.Compress,
		LocalTime:  rotation.LocalTime,
	}, nil
}

// buildOptions creates zap.Option slice.
func (b *LoggerBuilder) buildOptions() []zap.Option {
	opts := []zap.Option{}

	// Add caller if enabled
	if b.cfg.Caller.Enabled {
		opts = append(opts, zap.AddCaller())
		if b.cfg.Caller.Skip > 0 {
			opts = append(opts, zap.AddCallerSkip(b.cfg.Caller.Skip))
		}
	}

	// Add stacktrace if enabled
	if b.cfg.Stacktrace.Enabled {
		stackLevel := zapcore.ErrorLevel
		if b.cfg.Stacktrace.Level != "" {
			if err := stackLevel.UnmarshalText([]byte(b.cfg.Stacktrace.Level)); err != nil {
				stackLevel = zapcore.ErrorLevel
			}
		}
		opts = append(opts, zap.AddStacktrace(stackLevel))
	}

	return opts
}

// getTimeEncoder returns the time encoder based on config.
func (b *LoggerBuilder) getTimeEncoder() zapcore.TimeEncoder {
	switch b.cfg.Encoder.TimeFormat {
	case "epoch":
		return zapcore.EpochTimeEncoder
	case "epochMillis":
		return zapcore.EpochMillisTimeEncoder
	case "epochNanos":
		return zapcore.EpochNanosTimeEncoder
	case "iso8601", "":
		return zapcore.ISO8601TimeEncoder
	default:
		// Custom format
		return zapcore.TimeEncoderOfLayout(b.cfg.Encoder.TimeFormat)
	}
}

// getDurationEncoder returns the duration encoder based on config.
func (b *LoggerBuilder) getDurationEncoder() zapcore.DurationEncoder {
	switch b.cfg.Encoder.DurationFormat {
	case "seconds":
		return zapcore.SecondsDurationEncoder
	case "nanos":
		return zapcore.NanosDurationEncoder
	case "string", "":
		return zapcore.StringDurationEncoder
	default:
		return zapcore.StringDurationEncoder
	}
}

// getLevelEncoder returns the level encoder based on format and color setting.
func (b *LoggerBuilder) getLevelEncoder(output config.OutputConfig) zapcore.LevelEncoder {
	format := output.Format
	if format == "" {
		format = "json"
	}
	color := output.Type == "console" && output.Console.Color

	// console format with color enabled
	if format == "console" && color {
		return zapcore.CapitalColorLevelEncoder
	}

	// Use encoder config level format if specified
	switch b.cfg.Encoder.LevelFormat {
	case "uppercase", "capital":
		return zapcore.CapitalLevelEncoder
	case "capitalColor":
		return zapcore.CapitalColorLevelEncoder
	case "lowercase", "":
		return zapcore.LowercaseLevelEncoder
	default:
		return zapcore.LowercaseLevelEncoder
	}
}
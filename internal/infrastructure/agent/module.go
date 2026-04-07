// Package agent provides Agent execution infrastructure.
package agent

import (
	"context"

	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/infrastructure/agent/claude"
	"github.com/ryuyb/litchi/internal/infrastructure/agent/parser"
	"github.com/ryuyb/litchi/internal/infrastructure/agent/permission"
	"github.com/ryuyb/litchi/internal/infrastructure/agent/retry"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides Agent execution via Fx.
var Module = fx.Module("agent",
	fx.Provide(
		NewClaudeCodeAgentFromConfig,
		// Provide concrete types as interfaces
		fx.Annotate(
			parser.NewDefaultOutputParser,
			fx.As(new(parser.OutputParser)),
		),
		fx.Annotate(
			permission.NewDefaultPermissionController,
			fx.As(new(permission.PermissionController)),
		),
		fx.Annotate(
			retry.NewDefaultRetryHandler,
			fx.As(new(retry.RetryHandler)),
		),
	),
	fx.Invoke(RegisterAgentLifecycle),
)

// ClaudeCodeAgentParams contains dependencies for creating a ClaudeCodeAgent.
type ClaudeCodeAgentParams struct {
	fx.In

	Config         *config.Config
	Logger         *zap.Logger
	CacheRepo      repository.CacheRepository
	OutputParser   parser.OutputParser
	PermissionCtrl permission.PermissionController
	RetryHandler   retry.RetryHandler
}

// NewClaudeCodeAgentFromConfig creates a new ClaudeCodeAgent with config.
func NewClaudeCodeAgentFromConfig(p ClaudeCodeAgentParams) service.AgentRunner {
	return claude.NewClaudeCodeAgent(claude.ClaudeCodeAgentParams{
		ClaudeBinary:   p.Config.Agent.Type, // "claude" by default
		OutputParser:   p.OutputParser,
		PermissionCtrl: p.PermissionCtrl,
		RetryHandler:   p.RetryHandler,
		CacheRepo:      p.CacheRepo,
		Logger:         p.Logger,
	})
}

// RegisterAgentLifecycle registers the lifecycle hooks for the Agent.
func RegisterAgentLifecycle(lc fx.Lifecycle, runner service.AgentRunner, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("shutting down agent runner")
			return runner.Shutdown(ctx)
		},
	})
}
package service

import (
	"context"

	"github.com/ryuyb/litchi/internal/domain/event"
	domainservice "github.com/ryuyb/litchi/internal/domain/service"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides application services for Fx.
var Module = fx.Module("application-service",
	// Application services
	fx.Provide(NewIssueService),
	fx.Provide(NewConsistencyService),
	fx.Provide(NewClarificationService),
	fx.Provide(NewDesignService),
	fx.Provide(NewTaskService),
	fx.Provide(NewPRService),
	fx.Provide(NewRepositoryService),
	fx.Provide(NewAuditService),
	fx.Provide(NewRecoveryService),
	// Domain services
	fx.Provide(
		fx.Annotate(
			domainservice.NewDefaultSessionControlService,
			fx.As(new(domainservice.SessionControlService)),
		),
	),
	// Provide *event.Dispatcher as service.EventDispatcher for RecoveryService
	fx.Provide(
		fx.Annotate(
			func(d *event.Dispatcher) EventDispatcher { return d },
			fx.ResultTags(`name:"event_dispatcher"`),
		),
	),

	// Lifecycle hooks for session recovery on startup
	fx.Invoke(RegisterRecoveryLifecycle),
)

// RegisterRecoveryLifecycle registers the startup recovery lifecycle hook.
func RegisterRecoveryLifecycle(lc fx.Lifecycle, recovery *RecoveryService, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Run recovery in background to not block startup
			go func() {
				recoveryLogger := logger.Named("recovery_startup")
				recoveryLogger.Info("starting session recovery on service startup")
				if err := recovery.RecoverOnStartup(ctx); err != nil {
					recoveryLogger.Error("session recovery on startup failed", zap.Error(err))
				}
			}()
			return nil
		},
	})
}
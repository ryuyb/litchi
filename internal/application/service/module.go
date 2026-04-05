package service

import (
	"context"

	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "application-service",
		Provides: []string{"*IssueService", "*ConsistencyService", "*ClarificationService", "*DesignService", "*TaskService", "*PRService", "*RepositoryService", "*AuditService", "*RecoveryService"},
		Depends:  []string{"*zap.Logger", "*config.Config", "*event.Dispatcher", "*repository.WorkSessionRepository", "*repository.RepositoryRepository", "*repository.AuditLogRepository", "*github.IssueService", "*github.PullRequestService", "AgentRunner", "ConflictDetector", "BranchService", "*service.DefaultComplexityEvaluator", "*service.DefaultSessionControlService"},
	})
}

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
package service

import (
	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "application-service",
		Provides: []string{"*IssueService", "*ConsistencyService", "*ClarificationService", "*DesignService", "*TaskService", "*PRService", "*RepositoryService", "*AuditService"},
		Depends:  []string{"*zap.Logger", "*config.Config", "*event.Dispatcher", "*repository.WorkSessionRepository", "*repository.RepositoryRepository", "*repository.AuditLogRepository", "*github.IssueService", "*github.PullRequestService", "AgentRunner", "ConflictDetector", "BranchService", "*service.DefaultComplexityEvaluator"},
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
)
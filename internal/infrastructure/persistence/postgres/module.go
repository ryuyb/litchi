// Package postgres provides database connection and repository implementations.
package postgres

import (
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/postgres/repositories"
	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "repositories",
		Provides: []string{"repository.AuditLogRepository", "repository.WebhookDeliveryRepository", "repository.RepositoryRepository"},
		Invokes:  []string{},
		Depends:  []string{"*gorm.DB", "*zap.Logger", "*config.Config"},
	})
}

// RepositoriesModule provides repository implementations via Fx.
var RepositoriesModule = fx.Module("repositories",
	// Provide AuditLogRepository with proper interface binding
	fx.Provide(
		fx.Annotate(
			repositories.NewAuditLogRepository,
			fx.As(new(repository.AuditLogRepository)),
		),
		// Provide WebhookDeliveryRepository with proper interface binding
		fx.Annotate(
			repositories.NewWebhookDeliveryRepository,
			fx.As(new(repository.WebhookDeliveryRepository)),
		),
		// Provide RepositoryRepository with proper interface binding
		fx.Annotate(
			repositories.NewRepositoryRepo,
			fx.As(new(repository.RepositoryRepository)),
		),
	),
)

// Package postgres provides database connection and repository implementations.
package postgres

import (
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/postgres/repositories"
	"go.uber.org/fx"
)

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
		// Provide UserRepository with proper interface binding
		fx.Annotate(
			repositories.NewUserRepo,
			fx.As(new(repository.UserRepository)),
		),
	),
)

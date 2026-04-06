// Package infrastructure provides aggregated infrastructure modules for Fx.
package infrastructure

import (
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/infrastructure/agent"
	"github.com/ryuyb/litchi/internal/infrastructure/cache"
	"github.com/ryuyb/litchi/internal/infrastructure/git"
	"github.com/ryuyb/litchi/internal/infrastructure/github"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/postgres"
	"go.uber.org/fx"
)

// Module aggregates all infrastructure modules for dependency injection.
var Module = fx.Module("infrastructure",
	// Database connection
	postgres.DatabaseModule,
	// Database migrations (runs if auto_migrate is enabled)
	postgres.MigrateModule,
	// WorkSession repository
	persistence.WorkSessionRepositoryModule,
	// Other repositories (AuditLog, WebhookDelivery, Repository)
	postgres.RepositoriesModule,
	// Event dispatcher
	event.Module,
	// GitHub integration
	github.Module,
	// Git operations
	git.Module,
	// Agent execution
	agent.Module,
	// Cache
	cache.CacheModule,
)
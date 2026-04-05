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
		Provides: []string{"repository.AuditLogRepository"},
		Invokes:  []string{},
		Depends:  []string{"*gorm.DB", "*zap.Logger"},
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
	),
)

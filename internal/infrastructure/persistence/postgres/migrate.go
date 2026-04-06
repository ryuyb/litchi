// Package postgres provides database connection and migration functionality.
// It is integrated as an Fx module for dependency injection.
package postgres

import (
	"context"
	"embed"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/ryuyb/litchi/internal/infrastructure/config"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrateModule provides database migration via Fx.
// It only runs migrations when auto_migrate is enabled in config.
var MigrateModule = fx.Module("migrate",
	fx.Provide(NewMigrator),
	fx.Invoke(registerMigrateLifecycle),
)

// Migrator wraps the golang-migrate instance with additional functionality.
type Migrator struct {
	migrate *migrate.Migrate
	logger  *zap.Logger
}

// MigratorParams holds the dependencies for creating a Migrator instance.
type MigratorParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

// NewMigrator creates a new Migrator instance with embedded migrations.
func NewMigrator(p MigratorParams) (*Migrator, error) {
	dbConfig := &p.Config.Database

	// Skip if auto-migrate is disabled
	if !dbConfig.AutoMigrate {
		p.Logger.Info("auto-migrate is disabled, skipping migrator initialization")
		return nil, nil
	}

	// Build connection string for migrate with proper URL encoding
	// x-migrations-table specifies the migration history table name
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&x-migrations-table=litchi_migrations",
		url.PathEscape(dbConfig.User),
		url.PathEscape(dbConfig.Password),
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Name,
		dbConfig.SSLMode,
	)

	// Use embedded migrations from filesystem
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	p.Logger.Info("migrator initialized with embedded migrations")

	return &Migrator{
		migrate: m,
		logger:  p.Logger,
	}, nil
}

// registerMigrateLifecycle registers migration lifecycle hooks with Fx.
func registerMigrateLifecycle(lc fx.Lifecycle, m *Migrator) {
	// Skip if migrator is nil (auto_migrate disabled)
	if m == nil {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			m.logger.Info("running database migrations on startup")
			if err := m.Up(); err != nil {
				m.logger.Error("failed to run migrations", zap.Error(err))
				return err
			}
			m.logger.Info("database migrations completed successfully")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			m.logger.Info("closing migrate connection")
			// Use channel to handle context cancellation during close
			done := make(chan error, 1)
			go func() { done <- m.Close() }()
			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				m.logger.Warn("migrate close timeout, context cancelled")
				return ctx.Err()
			}
		},
	})
}

// Up runs all available migrations.
func (m *Migrator) Up() error {
	err := m.migrate.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			m.logger.Info("no new migrations to apply")
			return nil
		}
		return fmt.Errorf("migration up failed: %w", err)
	}
	return nil
}

// Down rolls back the last migration.
func (m *Migrator) Down() error {
	err := m.migrate.Down()
	if err != nil {
		if err == migrate.ErrNoChange {
			m.logger.Info("no migrations to rollback")
			return nil
		}
		return fmt.Errorf("migration down failed: %w", err)
	}
	return nil
}

// Drop drops all tables and migrations history.
func (m *Migrator) Drop() error {
	err := m.migrate.Drop()
	if err != nil {
		return fmt.Errorf("migration drop failed: %w", err)
	}
	return nil
}

// Version returns the current migration version.
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

// Steps applies n migrations (positive for up, negative for down).
func (m *Migrator) Steps(n int) error {
	err := m.migrate.Steps(n)
	if err != nil {
		return fmt.Errorf("migration steps failed: %w", err)
	}
	return nil
}

// Force sets the migration version but does not run migrations.
// Useful for fixing dirty database state.
func (m *Migrator) Force(version int) error {
	err := m.migrate.Force(version)
	if err != nil {
		return fmt.Errorf("migration force failed: %w", err)
	}
	return nil
}

// Close closes the migrate connection.
func (m *Migrator) Close() error {
	sourceErr, dbErr := m.migrate.Close()
	if sourceErr != nil {
		return fmt.Errorf("failed to close migration source: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close migration database: %w", dbErr)
	}
	return nil
}

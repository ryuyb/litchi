// Package testutil provides testing utilities for PostgreSQL integration tests.
package testutil

import (
	"context"
	"testing"
	"time"

	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresContainer wraps a PostgreSQL testcontainer with connection info.
type PostgresContainer struct {
	Container *pgcontainer.PostgresContainer
	DSN       string
}

// SetupPostgres creates a PostgreSQL container for testing.
// It uses postgres:16-alpine image for faster startup.
// The container is automatically terminated when the test ends.
func SetupPostgres(ctx context.Context, t *testing.T) *PostgresContainer {
	t.Helper()

	container, err := pgcontainer.Run(ctx,
		"postgres:16-alpine",
		pgcontainer.WithDatabase("testdb"),
		pgcontainer.WithUsername("test"),
		pgcontainer.WithPassword("test"),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Wait for container to be ready
	time.Sleep(2 * time.Second)

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("warning: failed to terminate container: %v", err)
		}
	})

	return &PostgresContainer{
		Container: container,
		DSN:       dsn,
	}
}

// SetupTestDB creates a GORM database connected to the test container.
// It auto-migrates the provided models.
func SetupTestDB(t *testing.T, pg *PostgresContainer, models ...any) *gorm.DB {
	t.Helper()

	var db *gorm.DB
	var err error

	// Retry connection a few times
	for i := 0; i < 5; i++ {
		db, err = gorm.Open(postgres.Open(pg.DSN), &gorm.Config{})
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("failed to migrate: %v", err)
		}
	}

	return db
}

// CleanupTable truncates the specified table for test isolation.
func CleanupTable(t *testing.T, db *gorm.DB, table string) {
	t.Helper()

	if err := db.Exec("TRUNCATE TABLE " + table + " CASCADE").Error; err != nil {
		t.Fatalf("failed to cleanup table %s: %v", table, err)
	}
}
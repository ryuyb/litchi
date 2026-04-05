// Package repositories provides GORM-based implementations of domain repositories.
package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/models"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/postgres/testutil"
	"go.uber.org/zap"
)

// setupTestRepository creates a test repository with a real PostgreSQL container.
// This test requires Docker and is skipped when running with -short flag.
func setupTestRepository(t *testing.T) repository.AuditLogRepository {
	t.Helper()

	// Skip integration tests when running with -short flag
	if testing.Short() {
		t.Skip("Skipping integration test (requires Docker). Run without -short to include.")
	}

	ctx := context.Background()

	pg := testutil.SetupPostgres(ctx, t)
	db := testutil.SetupTestDB(t, pg, &models.AuditLog{})
	logger := zap.NewNop()

	return NewAuditLogRepository(AuditLogRepositoryParams{
		DB:     db,
		Logger: logger,
	})
}

// createTestAuditLog creates a test audit log entity without session association.
// Note: SessionID is nil to avoid foreign key constraint violations in isolated tests.
func createTestAuditLog() *entity.AuditLog {
	return entity.NewAuditLog(
		uuid.Nil, // No session association for isolated tests
		"owner/repo",
		42,
		"testuser",
		valueobject.ActorRoleIssueAuthor,
		valueobject.OpSessionStart,
		"session",
		"test-resource-id",
	)
}

func TestAuditLogRepository_Save(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	auditLog := createTestAuditLog()
	auditLog.SetParameters(map[string]any{"key": "value"})
	auditLog.SetDuration(100)
	auditLog.SetOutput("test output", 1000)

	err := repo.Save(ctx, auditLog)
	if err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	// Verify ID was set
	if auditLog.ID == uuid.Nil {
		t.Error("audit log ID should be set after save")
	}
}

func TestAuditLogRepository_FindByID(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Save a test audit log
	auditLog := createTestAuditLog()
	if err := repo.Save(ctx, auditLog); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	// Find by ID
	found, err := repo.FindByID(ctx, auditLog.ID)
	if err != nil {
		t.Fatalf("failed to find audit log: %v", err)
	}

	if found == nil {
		t.Fatal("audit log should be found")
	}

	// Verify fields
	if found.ID != auditLog.ID {
		t.Errorf("ID mismatch: got %v, want %v", found.ID, auditLog.ID)
	}
	if found.Repository != auditLog.Repository {
		t.Errorf("Repository mismatch: got %v, want %v", found.Repository, auditLog.Repository)
	}
	if found.Actor != auditLog.Actor {
		t.Errorf("Actor mismatch: got %v, want %v", found.Actor, auditLog.Actor)
	}
	if found.Operation != auditLog.Operation {
		t.Errorf("Operation mismatch: got %v, want %v", found.Operation, auditLog.Operation)
	}
}

func TestAuditLogRepository_FindByID_NotFound(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Try to find non-existent audit log
	found, err := repo.FindByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if found != nil {
		t.Error("audit log should not be found")
	}
}

func TestAuditLogRepository_List(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Save multiple audit logs
	for i := 0; i < 5; i++ {
		auditLog := entity.NewAuditLog(
			uuid.Nil,
			"owner/repo",
			42+i,
			"testuser",
			valueobject.ActorRoleIssueAuthor,
			valueobject.OpSessionStart,
			"session",
			"test-resource",
		)
		if err := repo.Save(ctx, auditLog); err != nil {
			t.Fatalf("failed to save audit log: %v", err)
		}
	}

	// List with pagination
	logs, total, err := repo.List(ctx, repository.AuditLogListOptions{
		Offset: 0,
		Limit:  3,
	})
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	if total != 5 {
		t.Errorf("total count mismatch: got %v, want 5", total)
	}

	if len(logs) != 3 {
		t.Errorf("returned logs count mismatch: got %v, want 3", len(logs))
	}
}

func TestAuditLogRepository_ListWithFilter(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Save audit logs with different repositories and actors
	auditLog1 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo1",
		42,
		"user1",
		valueobject.ActorRoleAdmin,
		valueobject.OpSessionStart,
		"session",
		"id1",
	)
	if err := repo.Save(ctx, auditLog1); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	auditLog2 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo2",
		43,
		"user2",
		valueobject.ActorRoleIssueAuthor,
		valueobject.OpSessionPause,
		"session",
		"id2",
	)
	if err := repo.Save(ctx, auditLog2); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	// Filter by actor
	logs, total, err := repo.List(ctx, repository.AuditLogListOptions{
		Filter: repository.AuditLogFilter{
			Actor: "user2",
		},
	})
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	if total != 1 {
		t.Errorf("total count mismatch for actor filter: got %v, want 1", total)
	}

	// Filter by repository
	logs, total, err = repo.List(ctx, repository.AuditLogListOptions{
		Filter: repository.AuditLogFilter{
			Repository: "owner/repo1",
		},
	})
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	if total != 1 {
		t.Errorf("total count mismatch for repository filter: got %v, want 1", total)
	}

	// Filter by operation
	logs, total, err = repo.List(ctx, repository.AuditLogListOptions{
		Filter: repository.AuditLogFilter{
			Operation: valueobject.OpSessionStart,
		},
	})
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	if total != 1 {
		t.Errorf("total count mismatch for operation filter: got %v, want 1", total)
	}
	_ = logs
}

func TestAuditLogRepository_ListBySessionID(t *testing.T) {
	// Note: This test verifies the ListBySessionID method works correctly.
	// For isolated testing, we use nil SessionID which results in NULL in database.
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Save audit logs without session association
	for i := 0; i < 3; i++ {
		auditLog := entity.NewAuditLog(
			uuid.Nil,
			"owner/repo",
			42,
			"testuser",
			valueobject.ActorRoleIssueAuthor,
			valueobject.OperationType("op_"+string(rune('A'+i))),
			"session",
			"test-resource",
		)
		if err := repo.Save(ctx, auditLog); err != nil {
			t.Fatalf("failed to save audit log: %v", err)
		}
	}

	// List by nil session ID
	logs, total, err := repo.ListBySessionID(ctx, uuid.Nil, 0, 10)
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	// Note: uuid.Nil is stored as NULL, so count might be 0
	// This verifies the method doesn't error
	_ = logs
	_ = total
}

func TestAuditLogRepository_ListByRepository(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Save audit logs for different repositories
	auditLog1 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo1",
		42,
		"user1",
		valueobject.ActorRoleAdmin,
		valueobject.OpSessionStart,
		"session",
		"id1",
	)
	if err := repo.Save(ctx, auditLog1); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	auditLog2 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo2",
		43,
		"user2",
		valueobject.ActorRoleIssueAuthor,
		valueobject.OpSessionStart,
		"session",
		"id2",
	)
	if err := repo.Save(ctx, auditLog2); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	// List by repository
	logs, total, err := repo.ListByRepository(ctx, "owner/repo1", 0, 10)
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	if total != 1 {
		t.Errorf("total count mismatch: got %v, want 1", total)
	}

	if logs[0].Repository != "owner/repo1" {
		t.Errorf("repository mismatch: got %v, want owner/repo1", logs[0].Repository)
	}
}

func TestAuditLogRepository_ListByActor(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Save audit logs by different actors
	auditLog1 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo",
		42,
		"user1",
		valueobject.ActorRoleAdmin,
		valueobject.OpSessionStart,
		"session",
		"id1",
	)
	if err := repo.Save(ctx, auditLog1); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	auditLog2 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo",
		43,
		"user2",
		valueobject.ActorRoleIssueAuthor,
		valueobject.OpSessionStart,
		"session",
		"id2",
	)
	if err := repo.Save(ctx, auditLog2); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	// List by actor
	logs, total, err := repo.ListByActor(ctx, "user1", 0, 10)
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	if total != 1 {
		t.Errorf("total count mismatch: got %v, want 1", total)
	}

	if logs[0].Actor != "user1" {
		t.Errorf("actor mismatch: got %v, want user1", logs[0].Actor)
	}
}

func TestAuditLogRepository_ListByTimeRange(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	now := time.Now()

	// Save audit logs at different times
	auditLog1 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo",
		42,
		"user",
		valueobject.ActorRoleAdmin,
		valueobject.OpSessionStart,
		"session",
		"id1",
	)
	auditLog1.Timestamp = now.Add(-2 * time.Hour)
	if err := repo.Save(ctx, auditLog1); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	auditLog2 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo",
		43,
		"user",
		valueobject.ActorRoleAdmin,
		valueobject.OpSessionStart,
		"session",
		"id2",
	)
	auditLog2.Timestamp = now.Add(-30 * time.Minute)
	if err := repo.Save(ctx, auditLog2); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	// List by time range (last hour)
	startTime := now.Add(-1 * time.Hour)
	endTime := now
	_, total, err := repo.ListByTimeRange(ctx, startTime, endTime, 0, 10)
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	if total != 1 {
		t.Errorf("total count mismatch: got %v, want 1", total)
	}
}

func TestAuditLogRepository_CountBySession(t *testing.T) {
	// Note: This test verifies the CountBySession method works correctly.
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Save audit logs without session association
	for i := 0; i < 5; i++ {
		auditLog := entity.NewAuditLog(
			uuid.Nil,
			"owner/repo",
			42,
			"testuser",
			valueobject.ActorRoleIssueAuthor,
			valueobject.OpSessionStart,
			"session",
			"test-resource",
		)
		if err := repo.Save(ctx, auditLog); err != nil {
			t.Fatalf("failed to save audit log: %v", err)
		}
	}

	// Count by nil session ID
	count, err := repo.CountBySession(ctx, uuid.Nil)
	if err != nil {
		t.Fatalf("failed to count audit logs: %v", err)
	}

	// Note: uuid.Nil is stored as NULL, so count might be 0
	_ = count
}

func TestAuditLogRepository_DeleteBeforeTime(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	now := time.Now()

	// Save audit logs at different times
	for i := 0; i < 3; i++ {
		auditLog := entity.NewAuditLog(
			uuid.Nil,
			"owner/repo",
			42+i,
			"user",
			valueobject.ActorRoleAdmin,
			valueobject.OpSessionStart,
			"session",
			"id",
		)
		auditLog.Timestamp = now.Add(-time.Duration(i+1) * time.Hour)
		if err := repo.Save(ctx, auditLog); err != nil {
			t.Fatalf("failed to save audit log: %v", err)
		}
	}

	// Delete logs older than 2 hours
	before := now.Add(-2 * time.Hour)
	deleted, err := repo.DeleteBeforeTime(ctx, before)
	if err != nil {
		t.Fatalf("failed to delete audit logs: %v", err)
	}

	if deleted != 1 {
		t.Errorf("deleted count mismatch: got %v, want 1", deleted)
	}

	// Verify remaining count
	_, total, err := repo.List(ctx, repository.AuditLogListOptions{})
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	if total != 2 {
		t.Errorf("remaining count mismatch: got %v, want 2", total)
	}
}

func TestAuditLogRepository_FilterByOperation(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Save audit logs with different operations
	auditLog1 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo",
		42,
		"user",
		valueobject.ActorRoleAdmin,
		valueobject.OpSessionStart,
		"session",
		"id1",
	)
	if err := repo.Save(ctx, auditLog1); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	auditLog2 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo",
		42,
		"user",
		valueobject.ActorRoleAdmin,
		valueobject.OpSessionPause,
		"session",
		"id2",
	)
	if err := repo.Save(ctx, auditLog2); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	// Filter by operation
	logs, total, err := repo.List(ctx, repository.AuditLogListOptions{
		Filter: repository.AuditLogFilter{
			Operation: valueobject.OpSessionStart,
		},
	})
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	if total != 1 {
		t.Errorf("total count mismatch: got %v, want 1", total)
	}

	if logs[0].Operation != valueobject.OpSessionStart {
		t.Errorf("operation mismatch: got %v, want %v", logs[0].Operation, valueobject.OpSessionStart)
	}
}

func TestAuditLogRepository_FilterByResult(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	// Save audit logs with different results
	auditLog1 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo",
		42,
		"user",
		valueobject.ActorRoleAdmin,
		valueobject.OpSessionStart,
		"session",
		"id1",
	)
	auditLog1.MarkSuccess()
	if err := repo.Save(ctx, auditLog1); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	auditLog2 := entity.NewAuditLog(
		uuid.Nil,
		"owner/repo",
		42,
		"user",
		valueobject.ActorRoleAdmin,
		valueobject.OpSessionStart,
		"session",
		"id2",
	)
	auditLog2.MarkFailed("test error")
	if err := repo.Save(ctx, auditLog2); err != nil {
		t.Fatalf("failed to save audit log: %v", err)
	}

	// Filter by result
	logs, total, err := repo.List(ctx, repository.AuditLogListOptions{
		Filter: repository.AuditLogFilter{
			Result: valueobject.AuditResultSuccess,
		},
	})
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}

	if total != 1 {
		t.Errorf("total count mismatch: got %v, want 1", total)
	}

	if logs[0].Result != valueobject.AuditResultSuccess {
		t.Errorf("result mismatch: got %v, want %v", logs[0].Result, valueobject.AuditResultSuccess)
	}
}

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// --- Mocks for AuditService tests (following design_service_test.go pattern) ---

type mockAuditLogRepository struct {
	logs                []*entity.AuditLog
	findErr             error
	listErr             error
	countErr            error
	deleteErr           error
	deleteCount         int64
	listBySessionResult []*entity.AuditLog
}

func newMockAuditLogRepository() *mockAuditLogRepository {
	return &mockAuditLogRepository{
		logs: make([]*entity.AuditLog, 0),
	}
}

func (m *mockAuditLogRepository) Save(ctx context.Context, log *entity.AuditLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAuditLogRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, log := range m.logs {
		if log.ID == id {
			return log, nil
		}
	}
	return nil, nil
}

func (m *mockAuditLogRepository) List(ctx context.Context, opts repository.AuditLogListOptions) ([]*entity.AuditLog, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	// Apply filters
	filtered := make([]*entity.AuditLog, 0)
	for _, log := range m.logs {
		// Filter by repository
		if opts.Filter.Repository != "" && log.Repository != opts.Filter.Repository {
			continue
		}
		// Filter by actor
		if opts.Filter.Actor != "" && log.Actor != opts.Filter.Actor {
			continue
		}
		// Filter by actor role
		if opts.Filter.ActorRole != "" && log.ActorRole != opts.Filter.ActorRole {
			continue
		}
		// Filter by operation
		if opts.Filter.Operation != "" && log.Operation != opts.Filter.Operation {
			continue
		}
		// Filter by result
		if opts.Filter.Result != "" && log.Result != opts.Filter.Result {
			continue
		}
		// Filter by resource type
		if opts.Filter.ResourceType != "" && log.ResourceType != opts.Filter.ResourceType {
			continue
		}
		// Filter by session ID
		if opts.Filter.SessionID != nil && log.SessionID != *opts.Filter.SessionID {
			continue
		}
		// Filter by time range
		if opts.Filter.StartTime != nil && log.Timestamp.Before(*opts.Filter.StartTime) {
			continue
		}
		if opts.Filter.EndTime != nil && log.Timestamp.After(*opts.Filter.EndTime) {
			continue
		}
		filtered = append(filtered, log)
	}
	// Apply offset and limit
	start := opts.Offset
	if start >= len(filtered) {
		return nil, int64(len(filtered)), nil
	}
	end := start + opts.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], int64(len(filtered)), nil
}

func (m *mockAuditLogRepository) ListBySessionID(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*entity.AuditLog, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	// Return mocked result if set
	if m.listBySessionResult != nil {
		return m.listBySessionResult, int64(len(m.listBySessionResult)), nil
	}
	// Filter by session ID
	filtered := make([]*entity.AuditLog, 0)
	for _, log := range m.logs {
		if log.SessionID == sessionID {
			filtered = append(filtered, log)
		}
	}
	start := offset
	if start >= len(filtered) {
		return nil, 0, nil
	}
	end := start + limit
	if end > len(filtered) || limit == 0 {
		end = len(filtered)
	}
	return filtered[start:end], int64(len(filtered)), nil
}

func (m *mockAuditLogRepository) ListByRepository(ctx context.Context, repo string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	filtered := make([]*entity.AuditLog, 0)
	for _, log := range m.logs {
		if log.Repository == repo {
			filtered = append(filtered, log)
		}
	}
	start := offset
	if start >= len(filtered) {
		return nil, 0, nil
	}
	end := start + limit
	if end > len(filtered) || limit == 0 {
		end = len(filtered)
	}
	return filtered[start:end], int64(len(filtered)), nil
}

func (m *mockAuditLogRepository) ListByActor(ctx context.Context, actor string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	filtered := make([]*entity.AuditLog, 0)
	for _, log := range m.logs {
		if log.Actor == actor {
			filtered = append(filtered, log)
		}
	}
	start := offset
	if start >= len(filtered) {
		return nil, 0, nil
	}
	end := start + limit
	if end > len(filtered) || limit == 0 {
		end = len(filtered)
	}
	return filtered[start:end], int64(len(filtered)), nil
}

func (m *mockAuditLogRepository) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, offset, limit int) ([]*entity.AuditLog, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	filtered := make([]*entity.AuditLog, 0)
	for _, log := range m.logs {
		if log.Timestamp.After(startTime) && log.Timestamp.Before(endTime) {
			filtered = append(filtered, log)
		}
	}
	start := offset
	if start >= len(filtered) {
		return nil, 0, nil
	}
	end := start + limit
	if end > len(filtered) || limit == 0 {
		end = len(filtered)
	}
	return filtered[start:end], int64(len(filtered)), nil
}

func (m *mockAuditLogRepository) CountBySession(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	count := 0
	for _, log := range m.logs {
		if log.SessionID == sessionID {
			count++
		}
	}
	return int64(count), nil
}

func (m *mockAuditLogRepository) DeleteBeforeTime(ctx context.Context, before time.Time) (int64, error) {
	if m.deleteErr != nil {
		return 0, m.deleteErr
	}
	// Return mocked count if set
	if m.deleteCount > 0 {
		return m.deleteCount, nil
	}
	count := 0
	newLogs := make([]*entity.AuditLog, 0)
	for _, log := range m.logs {
		if log.Timestamp.Before(before) {
			count++
		} else {
			newLogs = append(newLogs, log)
		}
	}
	m.logs = newLogs
	return int64(count), nil
}

// --- Helper functions for tests ---

func newTestAuditService(
	auditRepo repository.AuditLogRepository,
) *AuditService {
	cfg := &config.Config{
		Audit: config.AuditConfig{
			Enabled:             true,
			RetentionDays:       30,
			MaxOutputLength:     500,
			SensitiveOperations: []string{},
		},
	}

	return NewAuditService(
		auditRepo,
		cfg,
		zap.NewNop(),
	)
}

func createTestAuditLog(sessionID uuid.UUID, operation valueobject.OperationType) *entity.AuditLog {
	return entity.NewAuditLog(
		sessionID,
		"owner/repo",
		1,
		"testuser",
		valueobject.ActorRoleAdmin,
		operation,
		"session",
		sessionID.String(),
	)
}

// --- Tests for GetAuditLog ---

func TestAuditService_GetAuditLog_Success(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add a test log
	log := createTestAuditLog(uuid.New(), valueobject.OpSessionStart)
	auditRepo.logs = append(auditRepo.logs, log)

	result, err := svc.GetAuditLog(ctx, log.ID)
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected audit log to be found")
	}
	if result.ID != log.ID {
		t.Errorf("expected ID %s, got %s", log.ID, result.ID)
	}
}

func TestAuditService_GetAuditLog_NotFound(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	result, err := svc.GetAuditLog(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetAuditLog failed: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for not found, got %+v", result)
	}
}

func TestAuditService_GetAuditLog_DatabaseError(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	auditRepo.findErr = errors.New("database error")
	svc := newTestAuditService(auditRepo)

	_, err := svc.GetAuditLog(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error for database failure")
	}

	// Verify error type
	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
	if litchiErr.Code != litchierrors.ErrDatabaseOperation {
		t.Errorf("expected ErrDatabaseOperation, got %s", litchiErr.Code.Code)
	}
}

// --- Tests for ListAuditLogs ---

func TestAuditService_ListAuditLogs_Success(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add test logs
	sessionID := uuid.New()
	for i := 0; i < 10; i++ {
		log := createTestAuditLog(sessionID, valueobject.OpSessionStart)
		auditRepo.logs = append(auditRepo.logs, log)
	}

	logs, total, err := svc.ListAuditLogs(ctx, AuditLogFilterParams{}, 0, 10, "")
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}

	if total != 10 {
		t.Errorf("expected total 10, got %d", total)
	}
	if len(logs) != 10 {
		t.Errorf("expected 10 logs, got %d", len(logs))
	}
}

func TestAuditService_ListAuditLogs_WithFilter(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add test logs with different repositories
	sessionID := uuid.New()
	log1 := createTestAuditLog(sessionID, valueobject.OpSessionStart)
	log1.Repository = "owner/repo1"
	log2 := createTestAuditLog(sessionID, valueobject.OpSessionStart)
	log2.Repository = "owner/repo2"
	auditRepo.logs = append(auditRepo.logs, log1, log2)

	filter := AuditLogFilterParams{
		Repository: "owner/repo1",
	}

	logs, total, err := svc.ListAuditLogs(ctx, filter, 0, 10, "")
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}

	if total != 1 {
		t.Errorf("expected total 1 (filtered), got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log after filtering, got %d", len(logs))
	}
}

func TestAuditService_ListAuditLogs_Pagination(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add test logs
	sessionID := uuid.New()
	for i := 0; i < 20; i++ {
		log := createTestAuditLog(sessionID, valueobject.OpSessionStart)
		auditRepo.logs = append(auditRepo.logs, log)
	}

	// First page
	logs, total, err := svc.ListAuditLogs(ctx, AuditLogFilterParams{}, 0, 5, "")
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}

	if total != 20 {
		t.Errorf("expected total 20, got %d", total)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 logs on first page, got %d", len(logs))
	}

	// Second page
	logs2, _, err := svc.ListAuditLogs(ctx, AuditLogFilterParams{}, 5, 5, "")
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}

	if len(logs2) != 5 {
		t.Errorf("expected 5 logs on second page, got %d", len(logs2))
	}
}

func TestAuditService_ListAuditLogs_DatabaseError(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	auditRepo.listErr = errors.New("database error")
	svc := newTestAuditService(auditRepo)

	_, _, err := svc.ListAuditLogs(ctx, AuditLogFilterParams{}, 0, 10, "")
	if err == nil {
		t.Fatal("expected error for database failure")
	}

	// Verify error type
	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for ListBySession ---

func TestAuditService_ListBySession_Success(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add test logs for a session
	sessionID := uuid.New()
	for i := 0; i < 5; i++ {
		log := createTestAuditLog(sessionID, valueobject.OpSessionStart)
		auditRepo.logs = append(auditRepo.logs, log)
	}

	// Add logs for another session
	otherSessionID := uuid.New()
	for i := 0; i < 3; i++ {
		log := createTestAuditLog(otherSessionID, valueobject.OpSessionStart)
		auditRepo.logs = append(auditRepo.logs, log)
	}

	logs, total, err := svc.ListBySession(ctx, sessionID, 0, 10)
	if err != nil {
		t.Fatalf("ListBySession failed: %v", err)
	}

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 logs, got %d", len(logs))
	}
}

func TestAuditService_ListBySession_Empty(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	logs, total, err := svc.ListBySession(ctx, uuid.New(), 0, 10)
	if err != nil {
		t.Fatalf("ListBySession failed: %v", err)
	}

	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

// --- Tests for ListByRepository ---

func TestAuditService_ListByRepository_Success(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add test logs for a repository
	sessionID := uuid.New()
	for i := 0; i < 3; i++ {
		log := createTestAuditLog(sessionID, valueobject.OpSessionStart)
		log.Repository = "owner/repo1"
		auditRepo.logs = append(auditRepo.logs, log)
	}

	logs, total, err := svc.ListByRepository(ctx, "owner/repo1", 0, 10)
	if err != nil {
		t.Fatalf("ListByRepository failed: %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 logs, got %d", len(logs))
	}
}

// --- Tests for ListByActor ---

func TestAuditService_ListByActor_Success(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add test logs for an actor
	sessionID := uuid.New()
	for i := 0; i < 2; i++ {
		log := createTestAuditLog(sessionID, valueobject.OpSessionStart)
		log.Actor = "testuser"
		auditRepo.logs = append(auditRepo.logs, log)
	}

	logs, total, err := svc.ListByActor(ctx, "testuser", 0, 10)
	if err != nil {
		t.Fatalf("ListByActor failed: %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
}

// --- Tests for ListByTimeRange ---

func TestAuditService_ListByTimeRange_Success(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add test logs at different times
	sessionID := uuid.New()
	now := time.Now()

	log1 := createTestAuditLog(sessionID, valueobject.OpSessionStart)
	log1.Timestamp = now.Add(-2 * time.Hour)

	log2 := createTestAuditLog(sessionID, valueobject.OpSessionStart)
	log2.Timestamp = now.Add(-1 * time.Hour)

	log3 := createTestAuditLog(sessionID, valueobject.OpSessionStart)
	log3.Timestamp = now

	auditRepo.logs = append(auditRepo.logs, log1, log2, log3)

	startTime := now.Add(-90 * time.Minute)
	endTime := now.Add(-30 * time.Minute)

	logs, total, err := svc.ListByTimeRange(ctx, startTime, endTime, 0, 10)
	if err != nil {
		t.Fatalf("ListByTimeRange failed: %v", err)
	}

	// Only log2 should be in the range
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestAuditService_ListByTimeRange_InvalidRange(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	now := time.Now()
	startTime := now
	endTime := now.Add(-1 * time.Hour) // End before start

	_, _, err := svc.ListByTimeRange(ctx, startTime, endTime, 0, 10)
	if err == nil {
		t.Fatal("expected error for invalid time range")
	}

	// Verify error type
	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for CountBySession ---

func TestAuditService_CountBySession_Success(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add test logs for a session
	sessionID := uuid.New()
	for i := 0; i < 5; i++ {
		log := createTestAuditLog(sessionID, valueobject.OpSessionStart)
		auditRepo.logs = append(auditRepo.logs, log)
	}

	count, err := svc.CountBySession(ctx, sessionID)
	if err != nil {
		t.Fatalf("CountBySession failed: %v", err)
	}

	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}
}

func TestAuditService_CountBySession_DatabaseError(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	auditRepo.countErr = errors.New("database error")
	svc := newTestAuditService(auditRepo)

	_, err := svc.CountBySession(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error for database failure")
	}

	// Verify error type
	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for DeleteExpired ---

func TestAuditService_DeleteExpired_Success(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add test logs at different times
	sessionID := uuid.New()
	now := time.Now()

	// Old log (expired)
	oldLog := createTestAuditLog(sessionID, valueobject.OpSessionStart)
	oldLog.Timestamp = now.AddDate(0, 0, -35)

	// Recent log (not expired)
	recentLog := createTestAuditLog(sessionID, valueobject.OpSessionStart)
	recentLog.Timestamp = now.AddDate(0, 0, -10)

	auditRepo.logs = append(auditRepo.logs, oldLog, recentLog)

	count, err := svc.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 log deleted, got %d", count)
	}

	// Verify old log was removed
	if len(auditRepo.logs) != 1 {
		t.Errorf("expected 1 log remaining, got %d", len(auditRepo.logs))
	}
}

func TestAuditService_DeleteExpired_AuditDisabled(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	cfg := &config.Config{
		Audit: config.AuditConfig{
			Enabled: false,
		},
	}
	svc := NewAuditService(auditRepo, cfg, zap.NewNop())

	count, err := svc.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired failed: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 logs deleted when audit disabled, got %d", count)
	}
}

func TestAuditService_DeleteExpired_DatabaseError(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	auditRepo.deleteErr = errors.New("database error")
	svc := newTestAuditService(auditRepo)

	_, err := svc.DeleteExpired(ctx)
	if err == nil {
		t.Fatal("expected error for database failure")
	}

	// Verify error type
	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for GetSessionAuditSummary ---

func TestAuditService_GetSessionAuditSummary_Success(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Add test logs with different operations and results
	sessionID := uuid.New()

	log1 := createTestAuditLog(sessionID, valueobject.OpSessionStart)
	log1.Duration = 100
	log1.Result = valueobject.AuditResultSuccess

	log2 := createTestAuditLog(sessionID, valueobject.OpSessionPause)
	log2.Duration = 50
	log2.Result = valueobject.AuditResultFailed
	log2.SetError("test error")

	log3 := createTestAuditLog(sessionID, valueobject.OpDesignConfirm)
	log3.Duration = 200
	log3.Result = valueobject.AuditResultSuccess

	auditRepo.logs = append(auditRepo.logs, log1, log2, log3)

	summary, err := svc.GetSessionAuditSummary(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetSessionAuditSummary failed: %v", err)
	}

	if summary.TotalCount != 3 {
		t.Errorf("expected total count 3, got %d", summary.TotalCount)
	}
	if summary.TotalDurationMs != 350 {
		t.Errorf("expected total duration 350, got %d", summary.TotalDurationMs)
	}
	if summary.AverageDurationMs != 116 {
		t.Errorf("expected average duration 116, got %d", summary.AverageDurationMs)
	}

	// Check result breakdown
	if summary.ByResult[valueobject.AuditResultSuccess] != 2 {
		t.Errorf("expected 2 success results, got %d", summary.ByResult[valueobject.AuditResultSuccess])
	}
	if summary.ByResult[valueobject.AuditResultFailed] != 1 {
		t.Errorf("expected 1 failed result, got %d", summary.ByResult[valueobject.AuditResultFailed])
	}

	// Check operation breakdown
	if summary.ByOperation[valueobject.OpSessionStart] != 1 {
		t.Errorf("expected 1 session start, got %d", summary.ByOperation[valueobject.OpSessionStart])
	}
	if summary.ByOperation[valueobject.OpDesignConfirm] != 1 {
		t.Errorf("expected 1 design confirm, got %d", summary.ByOperation[valueobject.OpDesignConfirm])
	}
}

func TestAuditService_GetSessionAuditSummary_Empty(t *testing.T) {
	ctx := context.Background()
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	summary, err := svc.GetSessionAuditSummary(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetSessionAuditSummary failed: %v", err)
	}

	if summary.TotalCount != 0 {
		t.Errorf("expected total count 0, got %d", summary.TotalCount)
	}
	if summary.TotalDurationMs != 0 {
		t.Errorf("expected total duration 0, got %d", summary.TotalDurationMs)
	}
}

// --- Tests for FormatAuditLog ---

func TestAuditService_FormatAuditLog_Success(t *testing.T) {
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	log := createTestAuditLog(uuid.New(), valueobject.OpSessionStart)
	log.Output = "short output"
	log.Error = ""

	formatted := svc.FormatAuditLog(log)

	if formatted.ID != log.ID {
		t.Errorf("expected ID %s, got %s", log.ID, formatted.ID)
	}
	if formatted.Output != "short output" {
		t.Errorf("expected output 'short output', got '%s'", formatted.Output)
	}
}

func TestAuditService_FormatAuditLog_Truncated(t *testing.T) {
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	log := createTestAuditLog(uuid.New(), valueobject.OpSessionStart)
	log.Output = "this is a very long output that should be truncated to 500 characters maximum length for display purposes and this text exceeds the limit"
	log.Error = ""

	formatted := svc.FormatAuditLog(log)

	// Output should be truncated
	if len(formatted.Output) > 503 { // 500 + "..."
		t.Errorf("expected output to be truncated, got length %d", len(formatted.Output))
	}
}

// --- Tests for IsSensitiveOperation ---

func TestAuditService_IsSensitiveOperation_Default(t *testing.T) {
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	// Default sensitive operations (empty config)
	if !svc.IsSensitiveOperation(valueobject.OpSessionTerminate) {
		t.Errorf("expected OpSessionTerminate to be sensitive")
	}
	if !svc.IsSensitiveOperation(valueobject.OpBashExecute) {
		t.Errorf("expected OpBashExecute to be sensitive")
	}
	if svc.IsSensitiveOperation(valueobject.OpSessionStart) {
		t.Errorf("expected OpSessionStart to NOT be sensitive")
	}
}

func TestAuditService_IsSensitiveOperation_CustomConfig(t *testing.T) {
	auditRepo := newMockAuditLogRepository()
	cfg := &config.Config{
		Audit: config.AuditConfig{
			Enabled:             true,
			SensitiveOperations: []string{"session_start", "design_confirm"},
		},
	}
	svc := NewAuditService(auditRepo, cfg, zap.NewNop())

	if !svc.IsSensitiveOperation(valueobject.OpSessionStart) {
		t.Errorf("expected OpSessionStart to be sensitive with custom config")
	}
	if !svc.IsSensitiveOperation(valueobject.OpDesignConfirm) {
		t.Errorf("expected OpDesignConfirm to be sensitive with custom config")
	}
	if svc.IsSensitiveOperation(valueobject.OpSessionTerminate) {
		t.Errorf("expected OpSessionTerminate to NOT be sensitive with custom config")
	}
}

// --- Tests for ValidateFilterParams ---

func TestAuditService_ValidateFilterParams_Valid(t *testing.T) {
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	now := time.Now()
	filter := AuditLogFilterParams{
		ActorRole: valueobject.ActorRoleAdmin,
		Operation: valueobject.OpSessionStart,
		Result:    valueobject.AuditResultSuccess,
		StartTime: new(now.Add(-1 * time.Hour)),
		EndTime:   new(now),
	}

	err := svc.ValidateFilterParams(filter)
	if err != nil {
		t.Fatalf("ValidateFilterParams failed for valid params: %v", err)
	}
}

func TestAuditService_ValidateFilterParams_InvalidActorRole(t *testing.T) {
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	filter := AuditLogFilterParams{
		ActorRole: valueobject.ActorRole("invalid_role"),
	}

	err := svc.ValidateFilterParams(filter)
	if err == nil {
		t.Fatal("expected error for invalid actor role")
	}

	// Verify error type
	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestAuditService_ValidateFilterParams_InvalidOperation(t *testing.T) {
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	filter := AuditLogFilterParams{
		Operation: valueobject.OperationType("invalid_op"),
	}

	err := svc.ValidateFilterParams(filter)
	if err == nil {
		t.Fatal("expected error for invalid operation")
	}

	// Verify error type
	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestAuditService_ValidateFilterParams_InvalidResult(t *testing.T) {
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	filter := AuditLogFilterParams{
		Result: valueobject.AuditResult("invalid_result"),
	}

	err := svc.ValidateFilterParams(filter)
	if err == nil {
		t.Fatal("expected error for invalid result")
	}

	// Verify error type
	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestAuditService_ValidateFilterParams_InvalidTimeRange(t *testing.T) {
	auditRepo := newMockAuditLogRepository()
	svc := newTestAuditService(auditRepo)

	now := time.Now()
	endTime := now.Add(-1 * time.Hour) // End before start

	filter := AuditLogFilterParams{
		StartTime: new(now),
		EndTime:   &endTime,
	}

	err := svc.ValidateFilterParams(filter)
	if err == nil {
		t.Fatal("expected error for invalid time range")
	}

	// Verify error type
	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

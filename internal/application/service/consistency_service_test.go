// Package service provides application service tests.
//
// NOTE: Several tests in this file directly modify aggregate internal fields
// (e.g., CurrentStage, PauseContext) to construct inconsistent states.
// This is intentional for testing the ConsistencyService's ability to detect
// and repair such states. In production code, all state changes should go
// through aggregate methods to maintain invariants.
package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"go.uber.org/zap"
)

// mockCacheRepository implements repository.CacheRepository for testing.
type mockCacheRepository struct {
	cache    *repository.ExecutionContextCache
	loadErr  error
	saveErr  error
	deleteErr error
}

func (m *mockCacheRepository) Save(ctx context.Context, worktreePath string, cache *repository.ExecutionContextCache) error {
	m.cache = cache
	return m.saveErr
}

func (m *mockCacheRepository) Load(ctx context.Context, worktreePath string) (*repository.ExecutionContextCache, error) {
	return m.cache, m.loadErr
}

func (m *mockCacheRepository) Delete(ctx context.Context, worktreePath string) error {
	return m.deleteErr
}

func TestConsistencyService_Check_NoIssues(t *testing.T) {
	// Create a valid session
	issue := entity.NewIssue(1, "Test Issue", "Body", "org/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)

	// Complete clarification
	session.Clarification.Complete()
	dims, err := valueobject.NewClarityDimensions(25, 20, 18, 12, 8) // Valid scores within max
	if err != nil {
		t.Fatalf("Failed to create clarity dimensions: %v", err)
	}
	session.SetClarityDimensions(dims)

	// Create service with mock cache
	cacheRepo := &mockCacheRepository{cache: nil}
	svc := NewConsistencyService(nil, cacheRepo, zap.NewNop())

	// Check consistency
	report, err := svc.Check(context.Background(), session, "/worktree")
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if report.HasIssues {
		t.Errorf("Expected no issues, got %d: %+v", len(report.Issues), report.Issues)
	}
}

func TestConsistencyService_Check_CacheMismatch(t *testing.T) {
	// Create a session
	issue := entity.NewIssue(1, "Test Issue", "Body", "org/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)

	// Create cache with different session ID
	cacheRepo := &mockCacheRepository{
		cache: &repository.ExecutionContextCache{
			SessionID:    uuid.New(), // Different from session
			CurrentStage: "clarification",
			Status:       "active",
		},
	}

	svc := NewConsistencyService(nil, cacheRepo, zap.NewNop())

	// Check consistency
	report, err := svc.Check(context.Background(), session, "/worktree")
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if !report.HasIssues {
		t.Error("Expected cache mismatch issue")
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Type == service.IssueTypeCacheMismatch {
			found = true
			if issue.Severity != service.SeverityHigh {
				t.Errorf("Expected high severity for session ID mismatch, got %s", issue.Severity)
			}
		}
	}
	if !found {
		t.Error("Expected IssueTypeCacheMismatch issue")
	}
}

func TestConsistencyService_Check_StageStatusMismatch(t *testing.T) {
	// Create a session in completed stage but not completed status
	issue := entity.NewIssue(1, "Test Issue", "Body", "org/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)

	// NOTE: Directly modifying internal state to create an inconsistent state for testing.
	// This simulates a scenario where stage and status are out of sync.
	// In production, this should never happen as Complete() method enforces consistency.
	session.CurrentStage = valueobject.StageCompleted
	// Status is still "active", which is inconsistent

	cacheRepo := &mockCacheRepository{cache: nil}
	svc := NewConsistencyService(nil, cacheRepo, zap.NewNop())

	// Check consistency
	report, err := svc.Check(context.Background(), session, "/worktree")
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if !report.HasIssues {
		t.Error("Expected status mismatch issue")
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Type == service.IssueTypeStatusMismatch {
			found = true
			if !issue.AutoRepair {
				t.Error("Expected status mismatch to be auto-repairable")
			}
		}
	}
	if !found {
		t.Error("Expected IssueTypeStatusMismatch issue")
	}
}

func TestConsistencyService_Check_StalePauseContext(t *testing.T) {
	// Create an active session with PauseContext
	issue := entity.NewIssue(1, "Test Issue", "Body", "org/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)

	// Add stale pause context
	pauseCtx := valueobject.NewPauseContext(valueobject.PauseReasonTaskFailed)
	session.PauseContext = &pauseCtx
	// Session is still active, which is inconsistent

	cacheRepo := &mockCacheRepository{cache: nil}
	svc := NewConsistencyService(nil, cacheRepo, zap.NewNop())

	// Check consistency
	report, err := svc.Check(context.Background(), session, "/worktree")
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if !report.HasIssues {
		t.Error("Expected stale pause context issue")
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Type == service.IssueTypePauseContextStale {
			found = true
			if !issue.AutoRepair {
				t.Error("Expected stale pause context to be auto-repairable")
			}
		}
	}
	if !found {
		t.Error("Expected IssueTypePauseContextStale issue")
	}
}

func TestConsistencyService_Check_TaskProgress(t *testing.T) {
	// Create a session in execution stage
	issue := entity.NewIssue(1, "Test Issue", "Body", "org/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)

	// Set to execution stage
	session.CurrentStage = valueobject.StageExecution
	session.Design = entity.NewDesign("design content")
	session.Design.Confirm()

	// Create tasks
	task1 := entity.NewTask("Task 1", nil, 1)
	task2 := entity.NewTask("Task 2", nil, 2)
	session.SetTasks([]*entity.Task{task1, task2})

	// Start execution
	session.StartExecution("/worktree", "issue-1")

	// Start task 1
	session.StartTask(task1.ID)

	// Mark task 1 completed in Execution but not in Task
	// This creates inconsistency
	session.Execution.MarkTaskCompleted(task1.ID)
	// But task1 status is still "in_progress"

	cacheRepo := &mockCacheRepository{cache: nil}
	svc := NewConsistencyService(nil, cacheRepo, zap.NewNop())

	// Check consistency
	report, err := svc.Check(context.Background(), session, "/worktree")
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should detect task progress inconsistency
	found := false
	for _, issue := range report.Issues {
		if issue.Type == service.IssueTypeTaskProgress {
			found = true
		}
	}
	if !found {
		t.Error("Expected IssueTypeTaskProgress issue for completed task mismatch")
	}
}

func TestConsistencyService_Repair_CacheMismatch(t *testing.T) {
	// Create a session with execution context
	issue := entity.NewIssue(1, "Test Issue", "Body", "org/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)
	session.CurrentStage = valueobject.StageExecution

	// Create execution with worktree path
	session.Execution = entity.NewExecution("/test/worktree", "issue-1")

	cacheRepo := &mockCacheRepository{cache: nil}
	svc := NewConsistencyService(nil, cacheRepo, zap.NewNop())

	// Create issue to repair
	issues := []service.ConsistencyIssue{
		{
			Type:       service.IssueTypeCacheMismatch,
			Severity:   service.SeverityMedium,
			AutoRepair: true,
		},
	}

	// Repair
	actions, failedRepairs := svc.Repair(context.Background(), session, issues)

	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	if !actions[0].Success {
		t.Errorf("Expected successful repair, got error: %s", actions[0].Error)
	}

	if len(failedRepairs) != 0 {
		t.Errorf("Expected no failed repairs, got %d", len(failedRepairs))
	}

	// Verify cache was saved
	if cacheRepo.cache == nil {
		t.Error("Expected cache to be saved")
	}
}

func TestConsistencyService_Repair_StatusMismatch(t *testing.T) {
	// Create a session in completed stage but active status
	issue := entity.NewIssue(1, "Test Issue", "Body", "org/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)
	session.CurrentStage = valueobject.StageCompleted
	session.SessionStatus = aggregate.SessionStatusActive // Inconsistent

	cacheRepo := &mockCacheRepository{cache: nil}
	svc := NewConsistencyService(nil, cacheRepo, zap.NewNop())

	// Create issue to repair
	issues := []service.ConsistencyIssue{
		{
			Type:       service.IssueTypeStatusMismatch,
			Severity:   service.SeverityHigh,
			FieldName:  "status",
			AutoRepair: true,
		},
	}

	// Repair
	actions, _ := svc.Repair(context.Background(), session, issues)

	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	if !actions[0].Success {
		t.Errorf("Expected successful repair, got error: %s", actions[0].Error)
	}

	if session.SessionStatus != aggregate.SessionStatusCompleted {
		t.Errorf("Expected status to be completed after repair, got %s", session.SessionStatus)
	}
}

func TestConsistencyService_Repair_StalePauseContext(t *testing.T) {
	// Create an active session with stale pause context
	issue := entity.NewIssue(1, "Test Issue", "Body", "org/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)
	session.SessionStatus = aggregate.SessionStatusActive
	pauseCtx := valueobject.NewPauseContext(valueobject.PauseReasonTaskFailed)
	session.PauseContext = &pauseCtx

	cacheRepo := &mockCacheRepository{cache: nil}
	svc := NewConsistencyService(nil, cacheRepo, zap.NewNop())

	// Create issue to repair
	issues := []service.ConsistencyIssue{
		{
			Type:       service.IssueTypePauseContextStale,
			Severity:   service.SeverityMedium,
			AutoRepair: true,
		},
	}

	// Repair
	actions, _ := svc.Repair(context.Background(), session, issues)

	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}

	if !actions[0].Success {
		t.Errorf("Expected successful repair, got error: %s", actions[0].Error)
	}

	if session.PauseContext != nil {
		t.Error("Expected PauseContext to be cleared after repair")
	}
}

func TestConsistencyService_CheckAndRepair(t *testing.T) {
	// Create a session with multiple issues
	issue := entity.NewIssue(1, "Test Issue", "Body", "org/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)
	session.CurrentStage = valueobject.StageCompleted
	session.SessionStatus = aggregate.SessionStatusActive // Issue 1: Status mismatch
	pauseCtx := valueobject.NewPauseContext(valueobject.PauseReasonTaskFailed)
	session.PauseContext = &pauseCtx // Issue 2: Stale pause context

	cacheRepo := &mockCacheRepository{cache: nil}
	svc := NewConsistencyService(nil, cacheRepo, zap.NewNop())

	// Check and repair
	report, err := svc.CheckAndRepair(context.Background(), session, "/worktree")
	if err != nil {
		t.Fatalf("CheckAndRepair failed: %v", err)
	}

	if !report.HasIssues {
		t.Error("Expected issues to be detected")
	}

	if report.RepairedCount == 0 {
		t.Error("Expected some issues to be repaired")
	}

	// Verify repairs
	if session.SessionStatus != aggregate.SessionStatusCompleted {
		t.Error("Expected status to be repaired to completed")
	}

	if session.PauseContext != nil {
		t.Error("Expected stale PauseContext to be cleared")
	}
}

func TestConsistencyService_Check_DesignMissing(t *testing.T) {
	// Create a session past clarification but no design
	issue := entity.NewIssue(1, "Test Issue", "Body", "org/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)
	session.CurrentStage = valueobject.StageTaskBreakdown
	// No design set - this is an issue

	cacheRepo := &mockCacheRepository{cache: nil}
	svc := NewConsistencyService(nil, cacheRepo, zap.NewNop())

	// Check consistency
	report, err := svc.Check(context.Background(), session, "/worktree")
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if !report.HasIssues {
		t.Error("Expected design missing issue")
	}

	found := false
	for _, issue := range report.Issues {
		if issue.Type == service.IssueTypeDesignMissing {
			found = true
			if issue.AutoRepair {
				t.Error("Expected design missing to NOT be auto-repairable")
			}
		}
	}
	if !found {
		t.Error("Expected IssueTypeDesignMissing issue")
	}
}
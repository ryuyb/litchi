package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"go.uber.org/zap"
)

func TestFileCacheRepository_Save(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "litchi-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := zap.NewNop()
	repo := NewFileCacheRepository(logger)
	ctx := context.Background()

	sessionID := uuid.New()
	complexityScore := 75
	pauseReason := "waiting for input"

	cache := &repository.ExecutionContextCache{
		SessionID:    sessionID,
		CurrentStage: "Execution",
		Status:       "paused",
		PauseReason:  &pauseReason,
		Clarification: &repository.ClarificationCache{
			Status:           "completed",
			ConfirmedPoints:  []string{"point1", "point2"},
			PendingQuestions: []string{},
		},
		Design: &repository.DesignCache{
			Status:              "confirmed",
			CurrentVersion:      2,
			ComplexityScore:     &complexityScore,
			RequireConfirmation: true,
			Confirmed:           true,
		},
		Execution: &repository.ExecutionCache{
			CurrentTaskID:    ptr(uuid.New()),
			CompletedTaskIDs: []uuid.UUID{uuid.New()},
			Branch:           "feature/test",
			BranchDeprecated: false,
			WorktreePath:     tempDir,
		},
		Tasks: []repository.TaskCache{
			{
				ID:         uuid.New(),
				Status:      "completed",
				RetryCount:  0,
			},
		},
		UpdatedAt: time.Now(),
	}

	err = repo.Save(ctx, tempDir, cache)
	if err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// Verify file was created
	contextPath := filepath.Join(tempDir, ".litchi", "context.json")
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		t.Error("context.json was not created")
	}
}

func TestFileCacheRepository_Save_NilCache(t *testing.T) {
	logger := zap.NewNop()
	repo := NewFileCacheRepository(logger)
	ctx := context.Background()

	err := repo.Save(ctx, "/tmp", nil)
	if err == nil {
		t.Error("expected error when saving nil cache")
	}
}

func TestFileCacheRepository_Load(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "litchi-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := zap.NewNop()
	repo := NewFileCacheRepository(logger)
	ctx := context.Background()

	sessionID := uuid.New()
	complexityScore := 80

	// Save a cache first
	originalCache := &repository.ExecutionContextCache{
		SessionID:    sessionID,
		CurrentStage: "Design",
		Status:       "running",
		Design: &repository.DesignCache{
			Status:              "in_progress",
			CurrentVersion:      1,
			ComplexityScore:     &complexityScore,
			RequireConfirmation: false,
			Confirmed:           false,
		},
		UpdatedAt: time.Now().UTC().Truncate(time.Second), // Truncate for comparison
	}

	if err := repo.Save(ctx, tempDir, originalCache); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// Load the cache
	loadedCache, err := repo.Load(ctx, tempDir)
	if err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}

	if loadedCache == nil {
		t.Fatal("loaded cache should not be nil")
	}

	// Verify loaded cache
	if loadedCache.SessionID != sessionID {
		t.Errorf("session ID mismatch: got %v, want %v", loadedCache.SessionID, sessionID)
	}

	if loadedCache.CurrentStage != "Design" {
		t.Errorf("current stage mismatch: got %v, want Design", loadedCache.CurrentStage)
	}

	if loadedCache.Status != "running" {
		t.Errorf("status mismatch: got %v, want running", loadedCache.Status)
	}

	if loadedCache.Design == nil {
		t.Fatal("design should not be nil")
	}

	if loadedCache.Design.CurrentVersion != 1 {
		t.Errorf("design version mismatch: got %v, want 1", loadedCache.Design.CurrentVersion)
	}

	if *loadedCache.Design.ComplexityScore != 80 {
		t.Errorf("complexity score mismatch: got %v, want 80", *loadedCache.Design.ComplexityScore)
	}
}

func TestFileCacheRepository_Load_NotFound(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "litchi-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := zap.NewNop()
	repo := NewFileCacheRepository(logger)
	ctx := context.Background()

	// Load from non-existent cache
	cache, err := repo.Load(ctx, tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cache != nil {
		t.Error("cache should be nil when file does not exist")
	}
}

func TestFileCacheRepository_Delete(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "litchi-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := zap.NewNop()
	repo := NewFileCacheRepository(logger)
	ctx := context.Background()

	// Save a cache first
	cache := &repository.ExecutionContextCache{
		SessionID:    uuid.New(),
		CurrentStage: "Execution",
		Status:       "running",
		UpdatedAt:    time.Now(),
	}

	if err := repo.Save(ctx, tempDir, cache); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// Verify .litchi directory exists
	litchiPath := filepath.Join(tempDir, ".litchi")
	if _, err := os.Stat(litchiPath); os.IsNotExist(err) {
		t.Fatal(".litchi directory should exist")
	}

	// Delete the cache
	if err := repo.Delete(ctx, tempDir); err != nil {
		t.Fatalf("failed to delete cache: %v", err)
	}

	// Verify .litchi directory is removed
	if _, err := os.Stat(litchiPath); !os.IsNotExist(err) {
		t.Error(".litchi directory should be deleted")
	}
}

func TestFileCacheRepository_Delete_NotExists(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "litchi-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := zap.NewNop()
	repo := NewFileCacheRepository(logger)
	ctx := context.Background()

	// Delete non-existent cache should not error
	err = repo.Delete(ctx, tempDir)
	if err != nil {
		t.Errorf("delete should not error when .litchi does not exist: %v", err)
	}
}

func TestFileCacheRepository_SaveAndLoad_WithTasks(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "litchi-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := zap.NewNop()
	repo := NewFileCacheRepository(logger)
	ctx := context.Background()

	taskID1 := uuid.New()
	taskID2 := uuid.New()
	completedTaskID := uuid.New()

	cache := &repository.ExecutionContextCache{
		SessionID:    uuid.New(),
		CurrentStage: "Execution",
		Status:       "running",
		Execution: &repository.ExecutionCache{
			CurrentTaskID:    &taskID1,
			CompletedTaskIDs: []uuid.UUID{completedTaskID},
			Branch:           "feature/test-branch",
			BranchDeprecated: false,
			WorktreePath:     tempDir,
		},
		Tasks: []repository.TaskCache{
			{
				ID:         taskID1,
				Status:     "in_progress",
				RetryCount: 0,
			},
			{
				ID:         taskID2,
				Status:     "pending",
				RetryCount: 0,
			},
		},
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
	}

	// Save
	if err := repo.Save(ctx, tempDir, cache); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// Load
	loadedCache, err := repo.Load(ctx, tempDir)
	if err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}

	// Verify tasks
	if len(loadedCache.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(loadedCache.Tasks))
	}

	if loadedCache.Tasks[0].ID != taskID1 {
		t.Errorf("task 1 ID mismatch: got %v, want %v", loadedCache.Tasks[0].ID, taskID1)
	}

	if loadedCache.Tasks[0].Status != "in_progress" {
		t.Errorf("task 1 status mismatch: got %v, want in_progress", loadedCache.Tasks[0].Status)
	}

	// Verify execution
	if loadedCache.Execution == nil {
		t.Fatal("execution should not be nil")
	}

	if *loadedCache.Execution.CurrentTaskID != taskID1 {
		t.Errorf("current task ID mismatch: got %v, want %v", loadedCache.Execution.CurrentTaskID, taskID1)
	}

	if len(loadedCache.Execution.CompletedTaskIDs) != 1 {
		t.Errorf("completed task IDs count mismatch: got %d, want 1", len(loadedCache.Execution.CompletedTaskIDs))
	}

	if loadedCache.Execution.Branch != "feature/test-branch" {
		t.Errorf("branch mismatch: got %v, want feature/test-branch", loadedCache.Execution.Branch)
	}
}

func TestFileCacheRepository_SaveAndLoad_EmptyCache(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "litchi-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := zap.NewNop()
	repo := NewFileCacheRepository(logger)
	ctx := context.Background()

	// Minimal cache
	cache := &repository.ExecutionContextCache{
		SessionID:    uuid.New(),
		CurrentStage: "Clarification",
		Status:       "running",
		Tasks:        []repository.TaskCache{},
		UpdatedAt:    time.Now().UTC().Truncate(time.Second),
	}

	// Save
	if err := repo.Save(ctx, tempDir, cache); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// Load
	loadedCache, err := repo.Load(ctx, tempDir)
	if err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}

	// Verify
	if loadedCache.SessionID != cache.SessionID {
		t.Errorf("session ID mismatch: got %v, want %v", loadedCache.SessionID, cache.SessionID)
	}

	if loadedCache.Clarification != nil {
		t.Error("clarification should be nil")
	}

	if loadedCache.Design != nil {
		t.Error("design should be nil")
	}

	if loadedCache.Execution != nil {
		t.Error("execution should be nil")
	}
}

// ptr is a helper function to create pointer to uuid.UUID.
func ptr(id uuid.UUID) *uuid.UUID {
	return &id
}
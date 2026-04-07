// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/git"
	"github.com/ryuyb/litchi/internal/infrastructure/github"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// --- Test Fixtures ---

func newTestPRService(
	sessionRepo repository.WorkSessionRepository,
	auditRepo repository.AuditLogRepository,
	conflictDetector git.ConflictDetector,
	branchService git.BranchService,
) *PRService {
	logger := zap.NewNop()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{
		Git: config.GitConfig{
			DefaultBaseBranch: "main",
		},
	}

	return &PRService{
		sessionRepo:      sessionRepo,
		auditRepo:        auditRepo,
		ghPRService:      nil, // Will use nil for unit tests
		ghIssueService:   nil,
		conflictDetector: conflictDetector,
		branchService:    branchService,
		eventDispatcher:  dispatcher,
		config:           cfg,
		logger:           logger.Named("pr_service"),
	}
}

func newTestSessionForPR(stage valueobject.Stage, withPR bool, allTasksCompleted bool) *aggregate.WorkSession {
	issue := entity.NewIssueFromGitHub(
		123, "Test Issue", "Issue body", "owner/repo", "testuser",
		[]string{"bug"}, "https://github.com/owner/repo/issues/123", time.Now(),
	)

	session, _ := aggregate.NewWorkSession(issue)
	session.CurrentStage = stage
	session.SessionStatus = aggregate.SessionStatusActive

	// Add design
	session.Design = entity.NewDesign("Test design content")
	session.Design.Confirm()

	// Add tasks
	task1 := entity.NewTask("Task 1", nil, 0)
	task2 := entity.NewTask("Task 2", []uuid.UUID{task1.ID}, 1)
	session.Tasks = []*entity.Task{task1, task2}

	if allTasksCompleted {
		task1.Status = valueobject.TaskStatusCompleted
		task2.Status = valueobject.TaskStatusCompleted
	}

	// Add execution
	session.Execution = entity.NewExecution("/path/to/worktree", "issue-123-test-issue")

	// Add PR if needed
	if withPR {
		session.PRNumber = new(456)
	}

	return session
}

// --- Tests ---

func TestPRService_CreatePR_SessionNotFound(t *testing.T) {
	ctx := context.Background()
	sessionID := uuid.New()

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, sessionID).Return(nil, nil)

	_, err := svc.CreatePR(ctx, sessionID)

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrSessionNotFound))
}

func TestPRService_CreatePR_WrongStage(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StageExecution, false, true)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	_, err := svc.CreatePR(ctx, session.ID)

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrInvalidStage))
}

func TestPRService_CreatePR_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StagePullRequest, true, true)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	_, err := svc.CreatePR(ctx, session.ID)

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrPRAlreadyExists))
}

func TestPRService_CreatePR_TasksNotCompleted(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StagePullRequest, false, false)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	_, err := svc.CreatePR(ctx, session.ID)

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrValidationFailed))
	assert.Contains(t, err.Error(), "not all tasks are completed")
}

func TestPRService_CreatePR_ConflictDetected(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StagePullRequest, false, true)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)
	conflictDetector.EXPECT().DetectConflicts(ctx, "/path/to/worktree", "issue-123-test-issue", "main").
		Return([]git.ConflictInfo{
			{FilePath: "file1.go", ConflictType: "modify/modify"},
			{FilePath: "file2.go", ConflictType: "delete/modify"},
		}, nil)
	auditRepo.EXPECT().Save(ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	_, err := svc.CreatePR(ctx, session.ID)

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrPRConflict))
	assert.Contains(t, err.Error(), "file1.go")
}

func TestPRService_CreatePR_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestPRService_CreatePR_GitHubAPIError(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestPRService_UpdatePR_NoPRExists(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StagePullRequest, false, true)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	err := svc.UpdatePR(ctx, session.ID, "test")

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrPRNotFound))
}

func TestPRService_UpdatePR_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestPRService_CheckConflicts_NoConflicts(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StageExecution, false, true)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)
	conflictDetector.EXPECT().DetectConflicts(ctx, "/path/to/worktree", "issue-123-test-issue", "main").
		Return(nil, nil)
	auditRepo.EXPECT().Save(ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	conflicts, err := svc.CheckConflicts(ctx, session.ID)

	assert.NoError(t, err)
	assert.Empty(t, conflicts)
}

func TestPRService_CheckConflicts_HasConflicts(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StageExecution, false, true)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)
	conflictDetector.EXPECT().DetectConflicts(ctx, "/path/to/worktree", "issue-123-test-issue", "main").
		Return([]git.ConflictInfo{
			{FilePath: "conflict.go", ConflictType: "modify/modify"},
		}, nil)
	auditRepo.EXPECT().Save(ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	conflicts, err := svc.CheckConflicts(ctx, session.ID)

	assert.NoError(t, err)
	assert.Len(t, conflicts, 1)
	assert.Equal(t, "conflict.go", conflicts[0])
}

func TestPRService_GetPRStatus_NoPR(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StageExecution, false, true)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	status, err := svc.GetPRStatus(ctx, session.ID)

	assert.NoError(t, err)
	assert.False(t, status.HasPR)
}

func TestPRService_GetPRStatus_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestPRService_ClosePR_NoPRExists(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StagePullRequest, false, true)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	err := svc.ClosePR(ctx, session.ID, "test")

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrPRNotFound))
}

func TestPRService_ClosePR_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestPRService_BuildPRTitle(t *testing.T) {
	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)
	session := newTestSessionForPR(valueobject.StagePullRequest, false, true)

	title := svc.buildPRTitle(session)

	assert.Contains(t, title, "#123")
	assert.Contains(t, title, "Test Issue")
}

func TestPRService_BuildPRBody(t *testing.T) {
	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)
	session := newTestSessionForPR(valueobject.StagePullRequest, false, true)
	session.Tasks[0].Status = valueobject.TaskStatusCompleted
	session.Tasks[1].Status = valueobject.TaskStatusSkipped

	body := svc.buildPRBody(session)

	assert.Contains(t, body, "Resolves #123")
	assert.Contains(t, body, "Test Issue")
	assert.Contains(t, body, "## Design")
	assert.Contains(t, body, "## Tasks Completed")
	assert.Contains(t, body, "Task 1")
	assert.Contains(t, body, "Task 2")
	assert.Contains(t, body, "Litchi")
}

// --- Integration-style tests with full PRService constructor ---

func TestPRService_Integration_CreatePR(t *testing.T) {
	t.Skip("Requires real GitHub API and Git repository - use integration test")
}

func TestPRService_Integration_UpdatePR(t *testing.T) {
	t.Skip("Requires real GitHub API and Git repository - use integration test")
}

func TestPRService_Integration_CheckConflicts(t *testing.T) {
	t.Skip("Requires real Git repository - use integration test")
}

// --- Type assertions for interfaces ---

func TestPRService_ImplementsInterfaces(t *testing.T) {
	// Ensure PRService has expected method signatures
	var _ *PRService = &PRService{}

	// Ensure git interfaces are satisfied
	var _ git.ConflictDetector = git.NewMockConflictDetector(t)
	var _ git.BranchService = git.NewMockBranchService(t)
}

// --- Additional mock tests for PRStatus ---

func TestPRService_PRStatus_StructFields(t *testing.T) {
	status := &PRStatus{
		SessionID:    uuid.New(),
		CurrentStage: "pull_request",
		HasPR:        true,
		PRNumber:     789,
		Title:        "Test PR",
		State:        "open",
		HeadBranch:   "feature-branch",
		BaseBranch:   "main",
		Merged:       false,
		Draft:        false,
		HTMLURL:      "https://github.com/owner/repo/pull/789",
		Commits:      5,
		Additions:    100,
		Deletions:    20,
		Changed:      10,
		Branch:       "feature-branch",
		WorktreePath: "/path/to/worktree",
	}

	assert.True(t, status.HasPR)
	assert.Equal(t, 789, status.PRNumber)
	assert.Equal(t, "Test PR", status.Title)
	assert.Equal(t, 5, status.Commits)
}

// --- Tests for error handling in helper methods ---

func TestPRService_BuildPRTitle_NilIssue(t *testing.T) {
	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	// Create session with nil issue
	session := &aggregate.WorkSession{
		ID:    uuid.New(),
		Issue: nil,
		Tasks: []*entity.Task{},
	}

	title := svc.buildPRTitle(session)

	assert.Equal(t, "Implement changes", title)
}

func TestPRService_CheckMergeConflicts_NilExecution(t *testing.T) {
	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	// Create session with nil execution
	session := &aggregate.WorkSession{
		ID:        uuid.New(),
		Execution: nil,
	}
	session.Issue = entity.NewIssueFromGitHub(123, "Test", "body", "owner/repo", "user", nil, "url", time.Now())

	conflicts, err := svc.checkMergeConflicts(context.Background(), session, "branch")

	assert.NoError(t, err)
	assert.Nil(t, conflicts)
}

func TestPRService_CheckMergeConflicts_EmptyWorktreePath(t *testing.T) {
	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	// Create session with empty worktree path
	session := &aggregate.WorkSession{
		ID: uuid.New(),
		Execution: &entity.Execution{
			WorktreePath: "",
			Branch:       valueobject.NewBranch("test-branch"),
		},
	}
	session.Issue = entity.NewIssueFromGitHub(123, "Test", "body", "owner/repo", "user", nil, "url", time.Now())

	conflicts, err := svc.checkMergeConflicts(context.Background(), session, "branch")

	assert.NoError(t, err)
	assert.Nil(t, conflicts)
}

// --- Tests for inactive sessions ---

func TestPRService_CreatePR_InactiveSession(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StagePullRequest, false, true)
	session.SessionStatus = aggregate.SessionStatusPaused

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	_, err := svc.CreatePR(ctx, session.ID)

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrValidationFailed))
	assert.Contains(t, err.Error(), "not active")
}

func TestPRService_UpdatePR_InactiveSession(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StagePullRequest, true, true)
	session.SessionStatus = aggregate.SessionStatusPaused

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	err := svc.UpdatePR(ctx, session.ID, "test")

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrValidationFailed))
	assert.Contains(t, err.Error(), "not active")
}

// --- Tests for nil execution context ---

func TestPRService_CreatePR_NilExecution(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StagePullRequest, false, true)
	session.Execution = nil

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	_, err := svc.CreatePR(ctx, session.ID)

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrValidationFailed))
	assert.Contains(t, err.Error(), "execution context not found")
}

func TestPRService_CheckConflicts_NilExecution(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForPR(valueobject.StageExecution, false, true)
	session.Execution = nil

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	conflictDetector := git.NewMockConflictDetector(t)
	branchService := git.NewMockBranchService(t)

	svc := newTestPRService(sessionRepo, auditRepo, conflictDetector, branchService)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	_, err := svc.CheckConflicts(ctx, session.ID)

	assert.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrValidationFailed))
	assert.Contains(t, err.Error(), "execution context not found")
}

// --- Verify github types are correct ---

func TestPRService_GitHubTypes(t *testing.T) {
	// Ensure github types exist and have expected fields
	var _ *github.PRCreateRequest = &github.PRCreateRequest{
		Title:      "test",
		Body:       "body",
		HeadBranch: "head",
		BaseBranch: "base",
		Draft:      false,
	}

	var _ *github.PRUpdateRequest = &github.PRUpdateRequest{
		Title: new("test"),
		Body:  new("body"),
	}

	var _ *github.PRInfo = &github.PRInfo{
		Number:     1,
		Title:      "test",
		State:      "open",
		HeadBranch: "head",
		BaseBranch: "base",
	}

	var _ *github.PullRequest = &github.PullRequest{
		PRInfo: github.PRInfo{
			Number: 1,
		},
		Commits:   1,
		Additions: 1,
		Deletions: 1,
		Changed:   1,
	}
}

func strPtr(s string) *string {
	return &s
}
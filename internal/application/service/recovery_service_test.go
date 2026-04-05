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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Mock Implementations ---

// MockWorkSessionRepository is a mock implementation of WorkSessionRepository.
type MockWorkSessionRepository struct {
	mock.Mock
}

func (m *MockWorkSessionRepository) Create(ctx context.Context, session *aggregate.WorkSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockWorkSessionRepository) Update(ctx context.Context, session *aggregate.WorkSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockWorkSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *MockWorkSessionRepository) FindByIssueID(ctx context.Context, issueID uuid.UUID) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, issueID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *MockWorkSessionRepository) FindByGitHubIssue(ctx context.Context, repository string, issueNumber int) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, repository, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *MockWorkSessionRepository) FindByStatus(ctx context.Context, status aggregate.SessionStatus) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *MockWorkSessionRepository) FindByStage(ctx context.Context, stage valueobject.Stage) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, stage)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *MockWorkSessionRepository) ListWithPagination(ctx context.Context, params repository.PaginationParams, filter *repository.WorkSessionFilter) ([]*aggregate.WorkSession, *repository.PaginationResult, error) {
	args := m.Called(ctx, params, filter)
	return args.Get(0).([]*aggregate.WorkSession), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

func (m *MockWorkSessionRepository) FindActiveByRepository(ctx context.Context, repo string) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, repo)
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *MockWorkSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWorkSessionRepository) ExistsByGitHubIssue(ctx context.Context, repository string, issueNumber int) (bool, error) {
	args := m.Called(ctx, repository, issueNumber)
	return args.Bool(0), args.Error(1)
}

// MockCacheRepository is a mock implementation of CacheRepository.
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Save(ctx context.Context, worktreePath string, cache *repository.ExecutionContextCache) error {
	args := m.Called(ctx, worktreePath, cache)
	return args.Error(0)
}

func (m *MockCacheRepository) Load(ctx context.Context, worktreePath string) (*repository.ExecutionContextCache, error) {
	args := m.Called(ctx, worktreePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.ExecutionContextCache), args.Error(1)
}

func (m *MockCacheRepository) Delete(ctx context.Context, worktreePath string) error {
	args := m.Called(ctx, worktreePath)
	return args.Error(0)
}

// MockSessionControlService is a mock implementation of SessionControlService.
type MockSessionControlService struct {
	mock.Mock
}

func (m *MockSessionControlService) PauseSession(session *aggregate.WorkSession, ctx valueobject.PauseContext) error {
	args := m.Called(session, ctx)
	return args.Error(0)
}

func (m *MockSessionControlService) ResumeSession(session *aggregate.WorkSession, action string) error {
	args := m.Called(session, action)
	return args.Error(0)
}

func (m *MockSessionControlService) AutoResumeSession(session *aggregate.WorkSession) (bool, error) {
	args := m.Called(session)
	return args.Bool(0), args.Error(1)
}

func (m *MockSessionControlService) TerminateSession(session *aggregate.WorkSession, reason string) error {
	args := m.Called(session, reason)
	return args.Error(0)
}

func (m *MockSessionControlService) CanResumeWithAction(session *aggregate.WorkSession, action string) bool {
	args := m.Called(session, action)
	return args.Bool(0)
}

func (m *MockSessionControlService) GetValidResumeActions(session *aggregate.WorkSession) []string {
	args := m.Called(session)
	return args.Get(0).([]string)
}

// MockEventDispatcher is a mock implementation of EventDispatcher.
type MockEventDispatcher struct {
	mock.Mock
}

func (m *MockEventDispatcher) DispatchAsync(ctx context.Context, event event.DomainEvent) {
	m.Called(ctx, event)
}

func (m *MockEventDispatcher) DispatchSync(ctx context.Context, event event.DomainEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// --- Test Fixtures ---

func createTestSessionForRecovery(t *testing.T, pauseReason valueobject.PauseReason) *aggregate.WorkSession {
	issue := entity.NewIssueFromGitHub(
		123,                            // number
		"Test Issue",                   // title
		"Test body",                    // body
		"owner/repo",                   // repository
		"testuser",                     // author
		[]string{"bug"},                // labels
		"https://github.com/owner/repo/issues/123", // url
		time.Now(),                     // createdAt
	)

	session, err := aggregate.NewWorkSession(issue)
	require.NoError(t, err)

	// Pause the session with the given reason
	pauseContext := valueobject.NewPauseContext(pauseReason)
	err = session.PauseWithContext(pauseContext)
	require.NoError(t, err)

	return session
}

func createTestRecoveryService(
	sessionRepo *MockWorkSessionRepository,
	cacheRepo *MockCacheRepository,
	consistencyService *ConsistencyService,
	sessionControlService *MockSessionControlService,
	taskService *TaskService,
) *RecoveryService {
	return &RecoveryService{
		sessionRepo:          sessionRepo,
		cacheRepo:            cacheRepo,
		consistencyService:   consistencyService,
		sessionControlService: sessionControlService,
		taskService:          taskService,
		logger:               zap.NewNop(),
		config:               &config.Config{},
	}
}

// --- Tests ---

func TestRecoveryService_RecoverOnStartup(t *testing.T) {
	t.Run("no recoverable sessions", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		// No paused sessions
		sessionRepo.On("FindByStatus", mock.Anything, aggregate.SessionStatusPaused).
			Return([]*aggregate.WorkSession{}, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		err := recoveryService.RecoverOnStartup(context.Background())
		assert.NoError(t, err)

		sessionRepo.AssertExpectations(t)
	})

	t.Run("recover service_restart session", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		session := createTestSessionForRecovery(t, valueobject.PauseReasonServiceRestart)

		// Find paused sessions
		sessionRepo.On("FindByStatus", mock.Anything, aggregate.SessionStatusPaused).
			Return([]*aggregate.WorkSession{session}, nil)

		// Auto-resume should succeed
		sessionControlService.On("AutoResumeSession", session).
			Return(true, nil).Run(func(args mock.Arguments) {
				// Simulate resume
				s := args.Get(0).(*aggregate.WorkSession)
				s.SessionStatus = aggregate.SessionStatusActive
				s.PauseContext = nil
			})

		// Save resumed session
		sessionRepo.On("Update", mock.Anything, session).
			Return(nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		err := recoveryService.RecoverOnStartup(context.Background())
		assert.NoError(t, err)

		sessionRepo.AssertExpectations(t)
		sessionControlService.AssertExpectations(t)
	})

	t.Run("skip non-auto-recoverable sessions", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		// Create session with manual recovery reason
		session := createTestSessionForRecovery(t, valueobject.PauseReasonUserRequest)

		// Find paused sessions
		sessionRepo.On("FindByStatus", mock.Anything, aggregate.SessionStatusPaused).
			Return([]*aggregate.WorkSession{session}, nil)

		// AutoResumeSession should not be called for non-auto-recoverable sessions

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		err := recoveryService.RecoverOnStartup(context.Background())
		assert.NoError(t, err)

		sessionRepo.AssertExpectations(t)
		// AutoResumeSession should not have been called
		sessionControlService.AssertNotCalled(t, "AutoResumeSession", mock.Anything)
	})
}

func TestRecoveryService_HandleUserResumeCommand(t *testing.T) {
	t.Run("successful resume by issue author", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		session := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)

		// Find session by GitHub issue
		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 123).
			Return(session, nil)

		// Get valid actions
		sessionControlService.On("GetValidResumeActions", session).
			Return([]string{"admin_continue", "admin_skip", "admin_rollback", "admin_force"})

		// Resume with action
		sessionControlService.On("ResumeSession", session, "admin_continue").
			Return(nil).Run(func(args mock.Arguments) {
				s := args.Get(0).(*aggregate.WorkSession)
				s.SessionStatus = aggregate.SessionStatusActive
				s.PauseContext = nil
			})

		// Save resumed session
		sessionRepo.On("Update", mock.Anything, session).
			Return(nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		status, err := recoveryService.HandleUserResumeCommand(
			context.Background(),
			"owner/repo",
			123,
			"testuser", // Issue author
			"admin_continue",
		)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, session.ID, status.SessionID)
		assert.Equal(t, "active", status.Status)

		sessionRepo.AssertExpectations(t)
		sessionControlService.AssertExpectations(t)
	})

	t.Run("session not found", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 999).
			Return(nil, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		status, err := recoveryService.HandleUserResumeCommand(
			context.Background(),
			"owner/repo",
			999,
			"testuser",
			"admin_continue",
		)

		assert.Nil(t, status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "L4DOM0001")

		sessionRepo.AssertExpectations(t)
	})

	t.Run("session not paused", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		// Create active (not paused) session
		issue := entity.NewIssueFromGitHub(123, "Test", "Body", "owner/repo", "testuser", nil, "", time.Now())
		session, err := aggregate.NewWorkSession(issue)
		require.NoError(t, err)

		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 123).
			Return(session, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		status, err := recoveryService.HandleUserResumeCommand(
			context.Background(),
			"owner/repo",
			123,
			"testuser",
			"admin_continue",
		)

		assert.Nil(t, status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "L4API0002")

		sessionRepo.AssertExpectations(t)
	})

	t.Run("invalid action", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		session := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)

		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 123).
			Return(session, nil)

		// Only these actions are valid for task_failed
		sessionControlService.On("GetValidResumeActions", session).
			Return([]string{"admin_continue", "admin_skip", "admin_rollback"})

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		status, err := recoveryService.HandleUserResumeCommand(
			context.Background(),
			"owner/repo",
			123,
			"testuser",
			"invalid_action", // Not in valid actions
		)

		assert.Nil(t, status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "L4API0001")

		sessionRepo.AssertExpectations(t)
		sessionControlService.AssertExpectations(t)
	})
}

func TestRecoveryService_GetRecoveryStatus(t *testing.T) {
	t.Run("get status for paused session", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		session := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)

		sessionRepo.On("FindByID", mock.Anything, session.ID).
			Return(session, nil)

		sessionControlService.On("GetValidResumeActions", session).
			Return([]string{"admin_continue", "admin_skip", "admin_rollback"})

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		status, err := recoveryService.GetRecoveryStatus(context.Background(), session.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, session.ID, status.SessionID)
		assert.Equal(t, "paused", status.Status)
		assert.Equal(t, "task_failed", status.PauseReason)
		assert.False(t, status.CanAutoRecover)
		assert.Contains(t, status.ValidActions, "admin_continue")

		sessionRepo.AssertExpectations(t)
		sessionControlService.AssertExpectations(t)
	})

	t.Run("session not found", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		sessionID := uuid.New()
		sessionRepo.On("FindByID", mock.Anything, sessionID).
			Return(nil, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		status, err := recoveryService.GetRecoveryStatus(context.Background(), sessionID)

		assert.Nil(t, status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "L4DOM0001")

		sessionRepo.AssertExpectations(t)
	})
}

func TestRecoveryService_ListRecoverableSessions(t *testing.T) {
	t.Run("list paused sessions", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		session1 := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)
		session2 := createTestSessionForRecovery(t, valueobject.PauseReasonUserRequest)

		sessionRepo.On("FindByStatus", mock.Anything, aggregate.SessionStatusPaused).
			Return([]*aggregate.WorkSession{session1, session2}, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		infos, err := recoveryService.ListRecoverableSessions(context.Background())

		assert.NoError(t, err)
		assert.Len(t, infos, 2)
		assert.Equal(t, session1.ID, infos[0].SessionID)
		assert.Equal(t, session2.ID, infos[1].SessionID)

		sessionRepo.AssertExpectations(t)
	})

	t.Run("empty list when no paused sessions", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		sessionRepo.On("FindByStatus", mock.Anything, aggregate.SessionStatusPaused).
			Return([]*aggregate.WorkSession{}, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		infos, err := recoveryService.ListRecoverableSessions(context.Background())

		assert.NoError(t, err)
		assert.Empty(t, infos)

		sessionRepo.AssertExpectations(t)
	})
}

func TestRecoveryService_CanAutoRecoverSession(t *testing.T) {
	recoveryService := &RecoveryService{
		logger: zap.NewNop(),
	}

	t.Run("nil session", func(t *testing.T) {
		assert.False(t, recoveryService.canAutoRecoverSession(nil))
	})

	t.Run("service_restart - auto recoverable", func(t *testing.T) {
		session := createTestSessionForRecovery(t, valueobject.PauseReasonServiceRestart)
		assert.True(t, recoveryService.canAutoRecoverSession(session))
	})

	t.Run("rate_limited - auto recoverable", func(t *testing.T) {
		session := createTestSessionForRecovery(t, valueobject.PauseReasonRateLimited)
		assert.True(t, recoveryService.canAutoRecoverSession(session))
	})

	t.Run("resource_exhausted - auto recoverable", func(t *testing.T) {
		session := createTestSessionForRecovery(t, valueobject.PauseReasonResourceExhausted)
		assert.True(t, recoveryService.canAutoRecoverSession(session))
	})

	t.Run("user_request - not auto recoverable", func(t *testing.T) {
		session := createTestSessionForRecovery(t, valueobject.PauseReasonUserRequest)
		assert.False(t, recoveryService.canAutoRecoverSession(session))
	})

	t.Run("task_failed - not auto recoverable", func(t *testing.T) {
		session := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)
		assert.False(t, recoveryService.canAutoRecoverSession(session))
	})
}

func TestRecoveryService_BuildRecoveryStatus(t *testing.T) {
	t.Run("build status with task progress", func(t *testing.T) {
		sessionRepo := new(MockWorkSessionRepository)
		cacheRepo := new(MockCacheRepository)
		sessionControlService := new(MockSessionControlService)

		// Create session in Execution stage
		issue := entity.NewIssueFromGitHub(123, "Test", "Body", "owner/repo", "testuser", nil, "", time.Now())
		session, err := aggregate.NewWorkSession(issue)
		require.NoError(t, err)

		// Complete clarification to move to design
		session.Clarification.Complete()
		err = session.TransitionTo(valueobject.StageDesign)
		require.NoError(t, err)

		// Set up design
		session.Design = entity.NewDesign("test design")
		session.Design.Confirm()

		// Create tasks
		task1 := entity.NewTask("Task 1", nil, 0)
		task1.Status = valueobject.TaskStatusCompleted
		task2 := entity.NewTask("Task 2", nil, 1)
		task2.Status = valueobject.TaskStatusPending
		task3 := entity.NewTask("Task 3", nil, 2)
		task3.Status = valueobject.TaskStatusFailed
		session.SetTasks([]*entity.Task{task1, task2, task3})

		// Transition to TaskBreakdown then Execution
		err = session.TransitionTo(valueobject.StageTaskBreakdown)
		require.NoError(t, err)
		err = session.TransitionTo(valueobject.StageExecution)
		require.NoError(t, err)

		// Pause
		pauseContext := valueobject.NewPauseContext(valueobject.PauseReasonTaskFailed).
			WithRelatedTask(task3.ID.String()).
			WithErrorDetails("Test error")
		err = session.PauseWithContext(pauseContext)
		require.NoError(t, err)

		sessionControlService.On("GetValidResumeActions", session).
			Return([]string{"admin_continue", "admin_skip", "admin_rollback"})

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil)

		status := recoveryService.buildRecoveryStatus(session)

		assert.NotNil(t, status)
		assert.Equal(t, "execution", status.CurrentStage)
		assert.Equal(t, "paused", status.Status)
		assert.Equal(t, "task_failed", status.PauseReason)
		assert.NotNil(t, status.TaskProgress)
		assert.Equal(t, 3, status.TaskProgress.Total)
		assert.Equal(t, 1, status.TaskProgress.Completed)
		assert.Equal(t, 1, status.TaskProgress.Failed)
		assert.Equal(t, 1, status.TaskProgress.Pending)

		sessionControlService.AssertExpectations(t)
	})
}
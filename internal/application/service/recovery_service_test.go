// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	domainService "github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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
	sessionRepo *repository.MockWorkSessionRepository,
	cacheRepo *repository.MockCacheRepository,
	consistencyService *ConsistencyService,
	sessionControlService *domainService.MockSessionControlService,
	taskService *TaskService,
	authService *AuthService,
) *RecoveryService {
	return &RecoveryService{
		sessionRepo:           sessionRepo,
		cacheRepo:             cacheRepo,
		consistencyService:    consistencyService,
		sessionControlService: sessionControlService,
		taskService:           taskService,
		authService:           authService,
		logger:                zap.NewNop(),
		config:                &config.Config{},
	}
}

// --- Tests ---

func TestRecoveryService_RecoverOnStartup(t *testing.T) {
	t.Run("no recoverable sessions", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		// No paused sessions
		sessionRepo.On("FindByStatus", mock.Anything, aggregate.SessionStatusPaused).
			Return([]*aggregate.WorkSession{}, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

		err := recoveryService.RecoverOnStartup(context.Background())
		assert.NoError(t, err)
	})

	t.Run("recover service_restart session", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

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
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

		err := recoveryService.RecoverOnStartup(context.Background())
		assert.NoError(t, err)
	})

	t.Run("skip non-auto-recoverable sessions", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		// Create session with manual recovery reason
		session := createTestSessionForRecovery(t, valueobject.PauseReasonUserRequest)

		// Find paused sessions
		sessionRepo.On("FindByStatus", mock.Anything, aggregate.SessionStatusPaused).
			Return([]*aggregate.WorkSession{session}, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

		err := recoveryService.RecoverOnStartup(context.Background())
		assert.NoError(t, err)
	})
}

func TestRecoveryService_HandleUserResumeCommand(t *testing.T) {
	t.Run("successful resume by issue author", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

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
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

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
	})

	t.Run("session not found", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 999).
			Return(nil, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

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
	})

	t.Run("session not paused", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		// Create active (not paused) session
		issue := entity.NewIssueFromGitHub(123, "Test", "Body", "owner/repo", "testuser", nil, "", time.Now())
		session, err := aggregate.NewWorkSession(issue)
		require.NoError(t, err)

		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 123).
			Return(session, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

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
	})

	t.Run("invalid action", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		session := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)

		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 123).
			Return(session, nil)

		// Only these actions are valid for task_failed
		sessionControlService.On("GetValidResumeActions", session).
			Return([]string{"admin_continue", "admin_skip", "admin_rollback"})

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

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
	})

	t.Run("admin_force denied for non-admin user", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		session := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)

		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 123).
			Return(session, nil)

		sessionControlService.On("GetValidResumeActions", session).
			Return([]string{"admin_continue", "admin_skip", "admin_rollback", "admin_force"})

		// Mock AuthService - user has write permission (not admin/maintain)
		mockAPI := NewMockPermissionAPI(t)
		mockAPI.EXPECT().GetPermissionLevel(mock.Anything, "owner", "repo", "nonadmin").
			Return(&PermissionResult{Permission: "write"}, nil)
		authService := newTestAuthService(mockAPI)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, authService)

		status, err := recoveryService.HandleUserResumeCommand(
			context.Background(),
			"owner/repo",
			123,
			"nonadmin", // Not issue author, no admin permission
			"admin_force",
		)

		assert.Nil(t, status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "L4API0001")
		assert.Contains(t, err.Error(), "admin or maintain permission")
	})

	t.Run("admin_force allowed for admin user", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		session := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)

		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 123).
			Return(session, nil)

		sessionControlService.On("GetValidResumeActions", session).
			Return([]string{"admin_continue", "admin_skip", "admin_rollback", "admin_force"})

		// Mock AuthService - user has admin permission
		mockAPI := NewMockPermissionAPI(t)
		mockAPI.EXPECT().GetPermissionLevel(mock.Anything, "owner", "repo", "adminuser").
			Return(&PermissionResult{Permission: "admin"}, nil)
		authService := newTestAuthService(mockAPI)

		sessionControlService.On("ResumeSession", session, "admin_force").
			Return(nil).Run(func(args mock.Arguments) {
				s := args.Get(0).(*aggregate.WorkSession)
				s.SessionStatus = aggregate.SessionStatusActive
				s.PauseContext = nil
			})

		sessionRepo.On("Update", mock.Anything, session).
			Return(nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, authService)

		status, err := recoveryService.HandleUserResumeCommand(
			context.Background(),
			"owner/repo",
			123,
			"adminuser",
			"admin_force",
		)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, "active", status.Status)
	})

	t.Run("admin_force allowed for maintain user", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		session := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)

		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 123).
			Return(session, nil)

		sessionControlService.On("GetValidResumeActions", session).
			Return([]string{"admin_continue", "admin_skip", "admin_rollback", "admin_force"})

		// Mock AuthService - user has maintain permission
		mockAPI := NewMockPermissionAPI(t)
		mockAPI.EXPECT().GetPermissionLevel(mock.Anything, "owner", "repo", "maintainuser").
			Return(&PermissionResult{Permission: "maintain"}, nil)
		authService := newTestAuthService(mockAPI)

		sessionControlService.On("ResumeSession", session, "admin_force").
			Return(nil).Run(func(args mock.Arguments) {
				s := args.Get(0).(*aggregate.WorkSession)
				s.SessionStatus = aggregate.SessionStatusActive
				s.PauseContext = nil
			})

		sessionRepo.On("Update", mock.Anything, session).
			Return(nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, authService)

		status, err := recoveryService.HandleUserResumeCommand(
			context.Background(),
			"owner/repo",
			123,
			"maintainuser",
			"admin_force",
		)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, "active", status.Status)
	})

	t.Run("admin_force denied on permission API error", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		session := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)

		sessionRepo.On("FindByGitHubIssue", mock.Anything, "owner/repo", 123).
			Return(session, nil)

		sessionControlService.On("GetValidResumeActions", session).
			Return([]string{"admin_continue", "admin_skip", "admin_rollback", "admin_force"})

		// Mock AuthService - API error
		mockAPI := NewMockPermissionAPI(t)
		mockAPI.EXPECT().GetPermissionLevel(mock.Anything, "owner", "repo", "someuser").
			Return(nil, errors.New("GitHub API error"))
		authService := newTestAuthService(mockAPI)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, authService)

		status, err := recoveryService.HandleUserResumeCommand(
			context.Background(),
			"owner/repo",
			123,
			"someuser",
			"admin_force",
		)

		assert.Nil(t, status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "L4API0001")
	})
}

func TestRecoveryService_GetRecoveryStatus(t *testing.T) {
	t.Run("get status for paused session", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		session := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)

		sessionRepo.On("FindByID", mock.Anything, session.ID).
			Return(session, nil)

		sessionControlService.On("GetValidResumeActions", session).
			Return([]string{"admin_continue", "admin_skip", "admin_rollback"})

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

		status, err := recoveryService.GetRecoveryStatus(context.Background(), session.ID)

		assert.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, session.ID, status.SessionID)
		assert.Equal(t, "paused", status.Status)
		assert.Equal(t, "task_failed", status.PauseReason)
		assert.False(t, status.CanAutoRecover)
		assert.Contains(t, status.ValidActions, "admin_continue")
	})

	t.Run("session not found", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		sessionID := uuid.New()
		sessionRepo.On("FindByID", mock.Anything, sessionID).
			Return(nil, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

		status, err := recoveryService.GetRecoveryStatus(context.Background(), sessionID)

		assert.Nil(t, status)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "L4DOM0001")
	})
}

func TestRecoveryService_ListRecoverableSessions(t *testing.T) {
	t.Run("list paused sessions", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		session1 := createTestSessionForRecovery(t, valueobject.PauseReasonTaskFailed)
		session2 := createTestSessionForRecovery(t, valueobject.PauseReasonUserRequest)

		sessionRepo.On("FindByStatus", mock.Anything, aggregate.SessionStatusPaused).
			Return([]*aggregate.WorkSession{session1, session2}, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

		infos, err := recoveryService.ListRecoverableSessions(context.Background())

		assert.NoError(t, err)
		assert.Len(t, infos, 2)
		assert.Equal(t, session1.ID, infos[0].SessionID)
		assert.Equal(t, session2.ID, infos[1].SessionID)
	})

	t.Run("empty list when no paused sessions", func(t *testing.T) {
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

		sessionRepo.On("FindByStatus", mock.Anything, aggregate.SessionStatusPaused).
			Return([]*aggregate.WorkSession{}, nil)

		consistencyService := NewConsistencyService(nil, nil, zap.NewNop())
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

		infos, err := recoveryService.ListRecoverableSessions(context.Background())

		assert.NoError(t, err)
		assert.Empty(t, infos)
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
		sessionRepo := repository.NewMockWorkSessionRepository(t)
		cacheRepo := repository.NewMockCacheRepository(t)
		sessionControlService := domainService.NewMockSessionControlService(t)

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
		recoveryService := createTestRecoveryService(sessionRepo, cacheRepo, consistencyService, sessionControlService, nil, nil)

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
	})
}
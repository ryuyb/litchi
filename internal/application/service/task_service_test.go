// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// --- Local Mocks for Testing ---

// LocalMockSessionRepoForTask is a local mock for WorkSessionRepository.
type LocalMockSessionRepoForTask struct {
	mock.Mock
}

func (m *LocalMockSessionRepoForTask) FindByID(ctx context.Context, id uuid.UUID) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForTask) FindByGitHubIssue(ctx context.Context, repoName string, issueNumber int) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, repoName, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForTask) FindByIssueID(ctx context.Context, issueID uuid.UUID) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, issueID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForTask) FindByStatus(ctx context.Context, status aggregate.SessionStatus) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForTask) FindByStage(ctx context.Context, stage valueobject.Stage) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, stage)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForTask) Create(ctx context.Context, session *aggregate.WorkSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *LocalMockSessionRepoForTask) Update(ctx context.Context, session *aggregate.WorkSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *LocalMockSessionRepoForTask) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *LocalMockSessionRepoForTask) ExistsByGitHubIssue(ctx context.Context, repoName string, issueNumber int) (bool, error) {
	args := m.Called(ctx, repoName, issueNumber)
	return args.Bool(0), args.Error(1)
}

func (m *LocalMockSessionRepoForTask) FindActiveByRepository(ctx context.Context, repository string) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, repository)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForTask) ListWithPagination(ctx context.Context, params repository.PaginationParams, filter *repository.WorkSessionFilter) ([]*aggregate.WorkSession, *repository.PaginationResult, error) {
	args := m.Called(ctx, params, filter)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

// LocalMockAuditRepoForTask is a local mock for AuditLogRepository.
type LocalMockAuditRepoForTask struct {
	mock.Mock
}

func (m *LocalMockAuditRepoForTask) Save(ctx context.Context, log *entity.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *LocalMockAuditRepoForTask) FindByID(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AuditLog), args.Error(1)
}

func (m *LocalMockAuditRepoForTask) List(ctx context.Context, opts repository.AuditLogListOptions) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepoForTask) ListBySessionID(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, sessionID, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepoForTask) ListByRepository(ctx context.Context, repoName string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, repoName, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepoForTask) ListByActor(ctx context.Context, actor string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, actor, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepoForTask) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, startTime, endTime, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepoForTask) CountBySession(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *LocalMockAuditRepoForTask) DeleteBeforeTime(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

// LocalMockAgentRunnerForTask is a mock implementation of AgentRunner.
type LocalMockAgentRunnerForTask struct {
	mock.Mock
}

func (m *LocalMockAgentRunnerForTask) Execute(ctx context.Context, req *service.AgentRequest) (*service.AgentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentResponse), args.Error(1)
}

func (m *LocalMockAgentRunnerForTask) ExecuteWithRetry(ctx context.Context, req *service.AgentRequest, policy valueobject.RetryPolicy) (*service.AgentResponse, error) {
	args := m.Called(ctx, req, policy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentResponse), args.Error(1)
}

func (m *LocalMockAgentRunnerForTask) ValidateRequest(req *service.AgentRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *LocalMockAgentRunnerForTask) PrepareContext(ctx context.Context, sessionID uuid.UUID, worktreePath string) (*service.AgentContext, error) {
	args := m.Called(ctx, sessionID, worktreePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentContext), args.Error(1)
}

func (m *LocalMockAgentRunnerForTask) SaveContext(ctx context.Context, worktreePath string, cache *service.AgentContextCache) error {
	args := m.Called(ctx, worktreePath, cache)
	return args.Error(0)
}

func (m *LocalMockAgentRunnerForTask) Cancel(sessionID uuid.UUID) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *LocalMockAgentRunnerForTask) GetStatus(sessionID uuid.UUID) (*service.AgentStatus, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentStatus), args.Error(1)
}

func (m *LocalMockAgentRunnerForTask) IsRunning(sessionID uuid.UUID) bool {
	args := m.Called(sessionID)
	return args.Bool(0)
}

func (m *LocalMockAgentRunnerForTask) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// --- Test Fixtures ---

func newTestTaskService(
	sessionRepo repository.WorkSessionRepository,
	auditRepo repository.AuditLogRepository,
	agentRunner service.AgentRunner,
) *TaskService {
	logger := zap.NewNop()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{
		Agent: config.AgentConfig{
			TaskRetryLimit: 3,
		},
		Failure: config.FailureConfig{
			Timeout: config.TimeoutConfig{
				TaskBreakdown:  "10m",
				TaskExecution:  "15m",
			},
		},
	}

	return &TaskService{
		sessionRepo:     sessionRepo,
		auditRepo:       auditRepo,
		agentRunner:     agentRunner,
		ghIssueService:  nil, // Will use mock via interface
		eventDispatcher: dispatcher,
		config:          cfg,
		logger:          logger.Named("task_service"),
	}
}

func newTestSessionForTaskBreakdown() *aggregate.WorkSession {
	issue := entity.NewIssueFromGitHub(
		123,
		"Test Issue",
		"Test body content",
		"owner/repo",
		"testuser",
		[]string{"bug"},
		"https://github.com/owner/repo/issues/123",
		time.Now(),
	)

	session, _ := aggregate.NewWorkSession(issue)
	// Complete clarification
	session.ConfirmClarificationPoint("Requirement 1")
	session.ConfirmClarificationPoint("Requirement 2")
	_ = session.CompleteClarification()
	_ = session.TransitionTo(valueobject.StageDesign)

	// Create and confirm design
	design := entity.NewDesign("# Design Document\n\nTest content")
	session.SetDesign(design)
	_ = session.ConfirmDesign()
	_ = session.TransitionTo(valueobject.StageTaskBreakdown)

	return session
}

func newTestSessionForExecution() *aggregate.WorkSession {
	session := newTestSessionForTaskBreakdown()

	// Add tasks
	task1 := entity.NewTask("Task 1: Setup database schema", nil, 0)
	task2 := entity.NewTask("Task 2: Implement repository", []uuid.UUID{task1.ID}, 1)
	task3 := entity.NewTask("Task 3: Add service layer", []uuid.UUID{task2.ID}, 2)

	session.SetTasks([]*entity.Task{task1, task2, task3})

	// Transition to Execution stage first (StartExecution requires Execution stage)
	_ = session.TransitionTo(valueobject.StageExecution)

	// Initialize execution (now that we're in Execution stage)
	_ = session.StartExecution("/tmp/worktree", "feature-123")

	return session
}

// --- Tests ---

func TestTaskService_StartTaskBreakdown_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestTaskService_StartTaskBreakdown_SessionNotFound(t *testing.T) {
	ctx := context.Background()
	sessionID := uuid.New()

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, sessionID).Return(nil, nil)

	// Execute
	_, err := svc.StartTaskBreakdown(ctx, sessionID)

	// Assert
	assert.Error(t, err)

	sessionRepo.AssertExpectations(t)
}

func TestTaskService_StartTaskBreakdown_WrongStage(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForTaskBreakdown()
	// Roll back to Design stage
	_ = session.RollbackTo(valueobject.StageDesign, "test", false)

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	_, err := svc.StartTaskBreakdown(ctx, session.ID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected TaskBreakdown")

	sessionRepo.AssertExpectations(t)
}

func TestTaskService_ParseTaskBreakdown_Success(t *testing.T) {
	svc := newTestTaskService(nil, nil, nil)

	// Test JSON parsing
	jsonOutput := `[
		{"description": "Create database schema", "dependencies": [], "order": 0},
		{"description": "Implement repository", "dependencies": [0], "order": 1},
		{"description": "Add service layer", "dependencies": [1], "order": 2}
	]`

	tasks, err := svc.parseTaskBreakdown(jsonOutput)

	assert.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Verify first task (no dependencies)
	assert.Equal(t, "Create database schema", tasks[0].Description)
	assert.Equal(t, 0, tasks[0].Order)
	assert.Empty(t, tasks[0].Dependencies)

	// Verify second task (depends on first)
	assert.Equal(t, "Implement repository", tasks[1].Description)
	assert.Equal(t, 1, tasks[1].Order)
	assert.Len(t, tasks[1].Dependencies, 1)
	assert.Equal(t, tasks[0].ID, tasks[1].Dependencies[0])

	// Verify third task (depends on second)
	assert.Equal(t, "Add service layer", tasks[2].Description)
	assert.Equal(t, 2, tasks[2].Order)
	assert.Len(t, tasks[2].Dependencies, 1)
	assert.Equal(t, tasks[1].ID, tasks[2].Dependencies[0])
}

func TestTaskService_ParseTaskBreakdown_EmptyOutput(t *testing.T) {
	svc := newTestTaskService(nil, nil, nil)

	_, err := svc.parseTaskBreakdown("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty output")
}

func TestTaskService_ParseTaskBreakdown_NoJSON(t *testing.T) {
	svc := newTestTaskService(nil, nil, nil)

	_, err := svc.parseTaskBreakdown("This is just plain text without JSON")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no JSON array")
}

func TestTaskService_ParseTaskBreakdown_EmptyArray(t *testing.T) {
	svc := newTestTaskService(nil, nil, nil)

	_, err := svc.parseTaskBreakdown("[]")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no tasks defined")
}

func TestTaskService_ParseTaskBreakdown_InvalidDependency(t *testing.T) {
	svc := newTestTaskService(nil, nil, nil)

	// Task 1 depends on task index 99 which doesn't exist
	jsonOutput := `[
		{"description": "First task", "dependencies": [], "order": 0},
		{"description": "Second task", "dependencies": [99], "order": 1}
	]`

	_, err := svc.parseTaskBreakdown(jsonOutput)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dependency index")
}

func TestTaskService_GetNextExecutableTask_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForExecution()

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	taskInfo, err := svc.GetNextExecutableTask(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, taskInfo)
	assert.Equal(t, "Task 1: Setup database schema", taskInfo.Description)
	assert.Equal(t, "pending", taskInfo.Status)
	assert.True(t, taskInfo.DependenciesSatisfied)

	sessionRepo.AssertExpectations(t)
}

func TestTaskTaskService_GetNextExecutableTask_AllCompleted(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForExecution()

	// Mark all tasks as completed
	for _, task := range session.Tasks {
		_ = session.StartTask(task.ID)
		_ = session.CompleteTask(task.ID, valueobject.NewExecutionResult("done", true, 100))
	}

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	taskInfo, err := svc.GetNextExecutableTask(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.Nil(t, taskInfo) // No executable task when all completed

	sessionRepo.AssertExpectations(t)
}

func TestTaskService_GetTaskList_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForExecution()

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	listStatus, err := svc.GetTaskList(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, listStatus)
	assert.Equal(t, 3, listStatus.TotalTasks)
	assert.Equal(t, 0, listStatus.Completed)
	assert.Equal(t, 0, listStatus.InProgress)
	assert.Equal(t, 3, listStatus.Pending)
	assert.False(t, listStatus.AllCompleted)
	assert.False(t, listStatus.HasFailedTask)

	// Check first task can execute (no dependencies)
	assert.True(t, listStatus.Tasks[0].CanExecute)
	assert.True(t, listStatus.Tasks[0].DependenciesSatisfied)

	// Check second task cannot execute (depends on first)
	assert.False(t, listStatus.Tasks[1].CanExecute)
	assert.False(t, listStatus.Tasks[1].DependenciesSatisfied)

	sessionRepo.AssertExpectations(t)
}

func TestTaskService_GetTaskStatus_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForExecution()
	taskID := session.Tasks[0].ID

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	status, err := svc.GetTaskStatus(ctx, session.ID, taskID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, taskID, status.TaskID)
	assert.Equal(t, "Task 1: Setup database schema", status.Description)
	assert.Equal(t, "pending", status.Status)
	assert.Equal(t, "待执行", status.StatusDisplayName)
	assert.Equal(t, 0, status.Order)

	sessionRepo.AssertExpectations(t)
}

func TestTaskService_GetTaskStatus_TaskNotFound(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForExecution()
	nonExistentTaskID := uuid.New()

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	_, err := svc.GetTaskStatus(ctx, session.ID, nonExistentTaskID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	sessionRepo.AssertExpectations(t)
}

func TestTaskService_ExecuteNextTask_SessionNotFound(t *testing.T) {
	ctx := context.Background()
	sessionID := uuid.New()

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, sessionID).Return(nil, nil)

	// Execute
	_, _, err := svc.ExecuteNextTask(ctx, sessionID)

	// Assert
	assert.Error(t, err)

	sessionRepo.AssertExpectations(t)
}

func TestTaskService_ExecuteNextTask_WrongStage(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForTaskBreakdown()
	// Session is in TaskBreakdown, not Execution stage

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	_, _, err := svc.ExecuteNextTask(ctx, session.ID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected Execution")

	sessionRepo.AssertExpectations(t)
}

func TestTaskService_CompleteTask_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForExecution()
	taskID := session.Tasks[0].ID

	// Start the task first
	_ = session.StartTask(taskID)

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	result := valueobject.NewExecutionResult("completed successfully", true, 500)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)
	sessionRepo.On("Update", ctx, session).Return(nil)
	auditRepo.On("Save", ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	// Execute
	err := svc.CompleteTask(ctx, session.ID, taskID, result)

	// Assert
	assert.NoError(t, err)

	// Verify task is completed
	task := session.GetTask(taskID)
	assert.True(t, task.IsCompleted())

	sessionRepo.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
}

func TestTaskService_FailTask_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForExecution()
	taskID := session.Tasks[0].ID

	// Start the task first
	_ = session.StartTask(taskID)

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)
	sessionRepo.On("Update", ctx, session).Return(nil)
	auditRepo.On("Save", ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	// Execute
	err := svc.FailTask(ctx, session.ID, taskID, "Test failure", "Try again")

	// Assert
	assert.NoError(t, err)

	// Verify task is failed
	task := session.GetTask(taskID)
	assert.True(t, task.IsFailed())
	assert.Equal(t, "Test failure", task.FailureReason)
	assert.Equal(t, "Try again", task.Suggestion)

	sessionRepo.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
}

func TestTaskService_RetryTask_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForExecution()
	taskID := session.Tasks[0].ID

	// Start and fail the task first
	_ = session.StartTask(taskID)
	_ = session.FailTask(taskID, "Initial failure", "Suggestion")

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)
	sessionRepo.On("Update", ctx, session).Return(nil)
	auditRepo.On("Save", ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	// Execute
	err := svc.RetryTask(ctx, session.ID, taskID)

	// Assert
	assert.NoError(t, err)

	// Verify task is back in InProgress
	task := session.GetTask(taskID)
	assert.True(t, task.IsInProgress())
	assert.Equal(t, 1, task.RetryCount)

	sessionRepo.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
}

func TestTaskService_RetryTask_MaxRetryLimit(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForExecution()
	taskID := session.Tasks[0].ID

	// Start and fail the task multiple times (exceed limit)
	_ = session.StartTask(taskID)
	_ = session.FailTask(taskID, "Failure 1", "")
	_ = session.RetryTask(taskID, 3)
	_ = session.FailTask(taskID, "Failure 2", "")
	_ = session.RetryTask(taskID, 3)
	_ = session.FailTask(taskID, "Failure 3", "")
	_ = session.RetryTask(taskID, 3)
	_ = session.FailTask(taskID, "Failure 4", "")

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute - should fail because retry limit reached
	err := svc.RetryTask(ctx, session.ID, taskID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum retry limit")

	sessionRepo.AssertExpectations(t)
}

func TestTaskService_SkipTask_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionForExecution()
	taskID := session.Tasks[0].ID

	sessionRepo := new(LocalMockSessionRepoForTask)
	auditRepo := new(LocalMockAuditRepoForTask)
	agentRunner := new(LocalMockAgentRunnerForTask)

	svc := newTestTaskService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)
	sessionRepo.On("Update", ctx, session).Return(nil)
	auditRepo.On("Save", ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	// Execute
	err := svc.SkipTask(ctx, session.ID, taskID, "Not needed for this implementation")

	// Assert
	assert.NoError(t, err)

	// Verify task is skipped
	task := session.GetTask(taskID)
	assert.True(t, task.IsSkipped())
	assert.Equal(t, "Not needed for this implementation", task.FailureReason)

	sessionRepo.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
}

func TestTaskService_DependencyResolution(t *testing.T) {
	// Test that dependencies are properly resolved
	session := newTestSessionForExecution()

	// Complete first task
	_ = session.StartTask(session.Tasks[0].ID)
	_ = session.CompleteTask(session.Tasks[0].ID, valueobject.NewExecutionResult("done", true, 100))

	// Now second task should have dependencies satisfied
	maxRetryLimit := 3
	nextTask := session.GetNextExecutableTask(maxRetryLimit)

	assert.NotNil(t, nextTask)
	assert.Equal(t, session.Tasks[1].ID, nextTask.ID)
	assert.Equal(t, "Task 2: Implement repository", nextTask.Description)

	// Third task still blocked
	_ = session.StartTask(session.Tasks[1].ID)
	nextTask = session.GetNextExecutableTask(maxRetryLimit)
	assert.Nil(t, nextTask) // Third task blocked because second is in progress
}

func TestTaskService_BuildTaskBreakdownPrompt(t *testing.T) {
	svc := newTestTaskService(nil, nil, nil)
	session := newTestSessionForTaskBreakdown()

	prompt := svc.buildTaskBreakdownPrompt(session)

	assert.Contains(t, prompt, "Design Document:")
	assert.Contains(t, prompt, "Issue Title:")
	assert.Contains(t, prompt, "JSON array")
	assert.Contains(t, prompt, "dependencies")
}

func TestTaskService_BuildTaskExecutionPrompt(t *testing.T) {
	svc := newTestTaskService(nil, nil, nil)
	session := newTestSessionForExecution()
	task := session.Tasks[0]

	prompt := svc.buildTaskExecutionPrompt(session, task)

	assert.Contains(t, prompt, "Task Description:")
	assert.Contains(t, prompt, task.Description)
	assert.Contains(t, prompt, "Design Document:")
	assert.Contains(t, prompt, "Repository:")
}

func TestTaskService_ExtractSuggestion(t *testing.T) {
	svc := newTestTaskService(nil, nil, nil)

	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "with suggestion prefix",
			output:   "Suggestion: Try increasing timeout",
			expected: "Suggestion: Try increasing timeout",
		},
		{
			name:     "with fix prefix",
			output:   "Fix: Add missing import",
			expected: "Fix: Add missing import",
		},
		{
			name:     "with next step prefix",
			output:   "Next step: Update configuration",
			expected: "Next step: Update configuration",
		},
		{
			name:     "fallback to first line",
			output:   "Something went wrong\nMore details here",
			expected: "Something went wrong",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "Check Agent logs for details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := svc.extractSuggestion(tt.output)
			assert.Equal(t, tt.expected, suggestion)
		})
	}
}

func TestTaskService_TaskBreakdownJSONWithExtraText(t *testing.T) {
	svc := newTestTaskService(nil, nil, nil)

	// Output with text before and after JSON
	output := `Here is the task breakdown based on the design:

[
  {"description": "First task", "dependencies": [], "order": 0},
  {"description": "Second task", "dependencies": [0], "order": 1}
]

Let me know if you need any modifications.`

	tasks, err := svc.parseTaskBreakdown(output)

	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "First task", tasks[0].Description)
	assert.Equal(t, "Second task", tasks[1].Description)
}

// Test that parseTaskBreakdown correctly handles complex dependency chains
func TestTaskService_ParseTaskBreakdown_ComplexDependencies(t *testing.T) {
	svc := newTestTaskService(nil, nil, nil)

	// Create a complex task breakdown with multiple dependencies
	taskData := []struct {
		Description  string `json:"description"`
		Dependencies []int  `json:"dependencies"`
		Order        int    `json:"order"`
	}{
		{"Setup database schema", []int{}, 0},
		{"Create user entity", []int{0}, 1},
		{"Create auth entity", []int{0}, 2},
		{"Implement UserRepository", []int{1}, 3},
		{"Implement AuthRepository", []int{2}, 4},
		{"Add UserService", []int{3}, 5},
		{"Add AuthService", []int{4}, 6},
		{"Integration tests", []int{5, 6}, 7}, // Depends on both UserService and AuthService
	}

	jsonBytes, _ := json.Marshal(taskData)
	tasks, err := svc.parseTaskBreakdown(string(jsonBytes))

	assert.NoError(t, err)
	assert.Len(t, tasks, 8)

	// Verify task 7 (Integration tests) depends on both task 5 and 6
	assert.Len(t, tasks[7].Dependencies, 2)
	assert.Contains(t, tasks[7].Dependencies, tasks[5].ID)
	assert.Contains(t, tasks[7].Dependencies, tasks[6].ID)
}
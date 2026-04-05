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
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// --- Local Mocks for Testing ---

// LocalMockSessionRepo is a local mock for WorkSessionRepository.
type LocalMockSessionRepo struct {
	mock.Mock
}

func (m *LocalMockSessionRepo) FindByID(ctx context.Context, id uuid.UUID) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepo) FindByGitHubIssue(ctx context.Context, repoName string, issueNumber int) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, repoName, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepo) FindByIssueID(ctx context.Context, issueID uuid.UUID) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, issueID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepo) FindByStatus(ctx context.Context, status aggregate.SessionStatus) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepo) FindByStage(ctx context.Context, stage valueobject.Stage) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, stage)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepo) Create(ctx context.Context, session *aggregate.WorkSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *LocalMockSessionRepo) Update(ctx context.Context, session *aggregate.WorkSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *LocalMockSessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *LocalMockSessionRepo) ExistsByGitHubIssue(ctx context.Context, repoName string, issueNumber int) (bool, error) {
	args := m.Called(ctx, repoName, issueNumber)
	return args.Bool(0), args.Error(1)
}

func (m *LocalMockSessionRepo) FindActiveByRepository(ctx context.Context, repository string) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, repository)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepo) ListWithPagination(ctx context.Context, params repository.PaginationParams, filter *repository.WorkSessionFilter) ([]*aggregate.WorkSession, *repository.PaginationResult, error) {
	args := m.Called(ctx, params, filter)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

// LocalMockAuditRepo is a local mock for AuditLogRepository.
type LocalMockAuditRepo struct {
	mock.Mock
}

func (m *LocalMockAuditRepo) Save(ctx context.Context, log *entity.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *LocalMockAuditRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AuditLog), args.Error(1)
}

func (m *LocalMockAuditRepo) List(ctx context.Context, opts repository.AuditLogListOptions) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepo) ListBySessionID(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, sessionID, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepo) ListByRepository(ctx context.Context, repoName string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, repoName, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepo) ListByActor(ctx context.Context, actor string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, actor, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepo) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, startTime, endTime, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepo) CountBySession(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *LocalMockAuditRepo) DeleteBeforeTime(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

// LocalMockAgentRunner is a mock implementation of AgentRunner.
type LocalMockAgentRunner struct {
	mock.Mock
}

func (m *LocalMockAgentRunner) Execute(ctx context.Context, req *service.AgentRequest) (*service.AgentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentResponse), args.Error(1)
}

func (m *LocalMockAgentRunner) ExecuteWithRetry(ctx context.Context, req *service.AgentRequest, policy valueobject.RetryPolicy) (*service.AgentResponse, error) {
	args := m.Called(ctx, req, policy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentResponse), args.Error(1)
}

func (m *LocalMockAgentRunner) ValidateRequest(req *service.AgentRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *LocalMockAgentRunner) PrepareContext(ctx context.Context, sessionID uuid.UUID, worktreePath string) (*service.AgentContext, error) {
	args := m.Called(ctx, sessionID, worktreePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentContext), args.Error(1)
}

func (m *LocalMockAgentRunner) SaveContext(ctx context.Context, worktreePath string, cache *service.AgentContextCache) error {
	args := m.Called(ctx, worktreePath, cache)
	return args.Error(0)
}

func (m *LocalMockAgentRunner) Cancel(sessionID uuid.UUID) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *LocalMockAgentRunner) GetStatus(sessionID uuid.UUID) (*service.AgentStatus, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentStatus), args.Error(1)
}

func (m *LocalMockAgentRunner) IsRunning(sessionID uuid.UUID) bool {
	args := m.Called(sessionID)
	return args.Bool(0)
}

func (m *LocalMockAgentRunner) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// --- Test Fixtures ---

func newTestClarificationService(
	sessionRepo repository.WorkSessionRepository,
	auditRepo repository.AuditLogRepository,
	agentRunner service.AgentRunner,
) *ClarificationService {
	logger := zap.NewNop()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{
		Clarity: config.ClarityConfig{
			Threshold:             60,
			AutoProceedThreshold:  80,
			ForceClarifyThreshold: 40,
		},
		Failure: config.FailureConfig{
			Timeout: config.TimeoutConfig{
				ClarificationAgent: "5m",
			},
		},
	}

	return &ClarificationService{
		sessionRepo:     sessionRepo,
		auditRepo:       auditRepo,
		agentRunner:     agentRunner,
		ghIssueService:  nil, // Will use mock via interface
		eventDispatcher: dispatcher,
		config:          cfg,
		logger:          logger.Named("clarification_service"),
	}
}

func newTestSessionWithClarification() *aggregate.WorkSession {
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
	return session
}

// --- Tests ---

func TestClarificationService_StartClarification_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestClarificationService_StartClarification_SessionNotFound(t *testing.T) {
	ctx := context.Background()
	sessionID := uuid.New()

	sessionRepo := new(LocalMockSessionRepo)
	auditRepo := new(LocalMockAuditRepo)
	agentRunner := new(LocalMockAgentRunner)

	svc := newTestClarificationService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, sessionID).Return(nil, nil)

	// Execute
	_, err := svc.StartClarification(ctx, sessionID)

	// Assert
	assert.Error(t, err)

	sessionRepo.AssertExpectations(t)
}

func TestClarificationService_StartClarification_WrongStage(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithClarification()
	// Manually transition to Design stage
	_ = session.CompleteClarification()
	_ = session.TransitionTo(valueobject.StageDesign)

	sessionRepo := new(LocalMockSessionRepo)
	auditRepo := new(LocalMockAuditRepo)
	agentRunner := new(LocalMockAgentRunner)

	svc := newTestClarificationService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	_, err := svc.StartClarification(ctx, session.ID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected Clarification")

	sessionRepo.AssertExpectations(t)
}

func TestClarificationService_ProcessAnswer_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestClarificationService_ProcessAnswer_NotAuthor(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestClarificationService_ForceStartDesign_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestClarificationService_ForceStartDesign_PendingQuestions(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestClarificationService_GetClarityStatus_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithClarification()
	dimensions, _ := valueobject.NewClarityDimensions(25, 20, 18, 14, 9) // Total: 86
	session.SetClarityDimensions(dimensions)

	sessionRepo := new(LocalMockSessionRepo)
	auditRepo := new(LocalMockAuditRepo)
	agentRunner := new(LocalMockAgentRunner)

	svc := newTestClarificationService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	status, err := svc.GetClarityStatus(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, session.ID, status.SessionID)
	assert.Equal(t, "clarification", status.CurrentStage)
	assert.Equal(t, 86, status.ClarityScore)

	sessionRepo.AssertExpectations(t)
}

func TestClarificationService_EvaluateClarity_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithClarification()

	sessionRepo := new(LocalMockSessionRepo)
	auditRepo := new(LocalMockAuditRepo)
	agentRunner := new(LocalMockAgentRunner)

	svc := newTestClarificationService(sessionRepo, auditRepo, agentRunner)

	// Mock expectations
	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)
	agentRunner.On("Execute", ctx, mock.AnythingOfType("*service.AgentRequest")).Return(
		&service.AgentResponse{
			Success: true,
			Output: `{"completeness":{"score":25},"clarity":{"score":20},"consistency":{"score":18},"feasibility":{"score":14},"testability":{"score":9}}`,
		}, nil,
	)
	sessionRepo.On("Update", ctx, session).Return(nil)
	auditRepo.On("Save", ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	// Execute
	score, err := svc.EvaluateClarity(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 86, score)

	sessionRepo.AssertExpectations(t)
	agentRunner.AssertExpectations(t)
}
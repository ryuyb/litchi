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

// --- Local Mocks for Testing (reused from clarification_service_test.go) ---

// LocalMockSessionRepoForDesign is a local mock for WorkSessionRepository.
type LocalMockSessionRepoForDesign struct {
	mock.Mock
}

func (m *LocalMockSessionRepoForDesign) FindByID(ctx context.Context, id uuid.UUID) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForDesign) FindByGitHubIssue(ctx context.Context, repoName string, issueNumber int) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, repoName, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForDesign) FindByIssueID(ctx context.Context, issueID uuid.UUID) (*aggregate.WorkSession, error) {
	args := m.Called(ctx, issueID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForDesign) FindByStatus(ctx context.Context, status aggregate.SessionStatus) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForDesign) FindByStage(ctx context.Context, stage valueobject.Stage) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, stage)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForDesign) Create(ctx context.Context, session *aggregate.WorkSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *LocalMockSessionRepoForDesign) Update(ctx context.Context, session *aggregate.WorkSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *LocalMockSessionRepoForDesign) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *LocalMockSessionRepoForDesign) ExistsByGitHubIssue(ctx context.Context, repoName string, issueNumber int) (bool, error) {
	args := m.Called(ctx, repoName, issueNumber)
	return args.Bool(0), args.Error(1)
}

func (m *LocalMockSessionRepoForDesign) FindActiveByRepository(ctx context.Context, repository string) ([]*aggregate.WorkSession, error) {
	args := m.Called(ctx, repository)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Error(1)
}

func (m *LocalMockSessionRepoForDesign) ListWithPagination(ctx context.Context, params repository.PaginationParams, filter *repository.WorkSessionFilter) ([]*aggregate.WorkSession, *repository.PaginationResult, error) {
	args := m.Called(ctx, params, filter)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]*aggregate.WorkSession), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

// LocalMockAuditRepoForDesign is a local mock for AuditLogRepository.
type LocalMockAuditRepoForDesign struct {
	mock.Mock
}

func (m *LocalMockAuditRepoForDesign) Save(ctx context.Context, log *entity.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *LocalMockAuditRepoForDesign) FindByID(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AuditLog), args.Error(1)
}

func (m *LocalMockAuditRepoForDesign) List(ctx context.Context, opts repository.AuditLogListOptions) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepoForDesign) ListBySessionID(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, sessionID, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepoForDesign) ListByRepository(ctx context.Context, repoName string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, repoName, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepoForDesign) ListByActor(ctx context.Context, actor string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, actor, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepoForDesign) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, offset, limit int) ([]*entity.AuditLog, int64, error) {
	args := m.Called(ctx, startTime, endTime, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*entity.AuditLog), args.Get(1).(int64), args.Error(2)
}

func (m *LocalMockAuditRepoForDesign) CountBySession(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *LocalMockAuditRepoForDesign) DeleteBeforeTime(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

// LocalMockAgentRunnerForDesign is a mock implementation of AgentRunner.
type LocalMockAgentRunnerForDesign struct {
	mock.Mock
}

func (m *LocalMockAgentRunnerForDesign) Execute(ctx context.Context, req *service.AgentRequest) (*service.AgentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentResponse), args.Error(1)
}

func (m *LocalMockAgentRunnerForDesign) ExecuteWithRetry(ctx context.Context, req *service.AgentRequest, policy valueobject.RetryPolicy) (*service.AgentResponse, error) {
	args := m.Called(ctx, req, policy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentResponse), args.Error(1)
}

func (m *LocalMockAgentRunnerForDesign) ValidateRequest(req *service.AgentRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *LocalMockAgentRunnerForDesign) PrepareContext(ctx context.Context, sessionID uuid.UUID, worktreePath string) (*service.AgentContext, error) {
	args := m.Called(ctx, sessionID, worktreePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentContext), args.Error(1)
}

func (m *LocalMockAgentRunnerForDesign) SaveContext(ctx context.Context, worktreePath string, cache *service.AgentContextCache) error {
	args := m.Called(ctx, worktreePath, cache)
	return args.Error(0)
}

func (m *LocalMockAgentRunnerForDesign) Cancel(sessionID uuid.UUID) error {
	args := m.Called(sessionID)
	return args.Error(0)
}

func (m *LocalMockAgentRunnerForDesign) GetStatus(sessionID uuid.UUID) (*service.AgentStatus, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AgentStatus), args.Error(1)
}

func (m *LocalMockAgentRunnerForDesign) IsRunning(sessionID uuid.UUID) bool {
	args := m.Called(sessionID)
	return args.Bool(0)
}

func (m *LocalMockAgentRunnerForDesign) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// --- Test Fixtures ---

func newTestDesignService(
	sessionRepo repository.WorkSessionRepository,
	auditRepo repository.AuditLogRepository,
	agentRunner service.AgentRunner,
	complexityEvaluator *service.DefaultComplexityEvaluator,
) *DesignService {
	logger := zap.NewNop()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{
		Complexity: config.ComplexityConfig{
			Threshold:          70,
			ForceDesignConfirm: false,
		},
		Failure: config.FailureConfig{
			Timeout: config.TimeoutConfig{
				DesignGeneration: "10m",
				DesignAnalysis:   "5m",
			},
		},
	}

	return &DesignService{
		sessionRepo:        sessionRepo,
		auditRepo:          auditRepo,
		agentRunner:        agentRunner,
		ghIssueService:     nil, // Will use mock via interface
		complexityEvaluator: complexityEvaluator,
		eventDispatcher:    dispatcher,
		config:             cfg,
		logger:             logger.Named("design_service"),
	}
}

func newTestSessionWithDesign() *aggregate.WorkSession {
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
	return session
}

// --- Tests ---

func TestDesignService_StartDesign_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestDesignService_StartDesign_SessionNotFound(t *testing.T) {
	ctx := context.Background()
	sessionID := uuid.New()

	sessionRepo := new(LocalMockSessionRepoForDesign)
	auditRepo := new(LocalMockAuditRepoForDesign)
	agentRunner := new(LocalMockAgentRunnerForDesign)

	svc := newTestDesignService(sessionRepo, auditRepo, agentRunner, nil)

	sessionRepo.On("FindByID", ctx, sessionID).Return(nil, nil)

	// Execute
	_, err := svc.StartDesign(ctx, sessionID)

	// Assert
	assert.Error(t, err)

	sessionRepo.AssertExpectations(t)
}

func TestDesignService_StartDesign_WrongStage(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithDesign()
	// Roll back to clarification stage
	_ = session.RollbackTo(valueobject.StageClarification, "test", false)

	sessionRepo := new(LocalMockSessionRepoForDesign)
	auditRepo := new(LocalMockAuditRepoForDesign)
	agentRunner := new(LocalMockAgentRunnerForDesign)

	svc := newTestDesignService(sessionRepo, auditRepo, agentRunner, nil)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	_, err := svc.StartDesign(ctx, session.ID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected Design")

	sessionRepo.AssertExpectations(t)
}

func TestDesignService_ConfirmDesign_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestDesignService_ConfirmDesign_NotAdmin(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestDesignService_RejectDesign_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestDesignService_UpdateDesign_Success(t *testing.T) {
	t.Skip("Requires GitHub API integration - use integration test")
}

func TestDesignService_EvaluateComplexity_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithDesign()

	// Create design
	design := entity.NewDesign("# Design Document\n\nTest content")
	session.SetDesign(design)

	sessionRepo := new(LocalMockSessionRepoForDesign)
	auditRepo := new(LocalMockAuditRepoForDesign)
	agentRunner := new(LocalMockAgentRunnerForDesign)

	// Create complexity evaluator with static analyzer
	dimensions := valueobject.ComplexityDimensions{
		EstimatedCodeChange:    60,
		AffectedModules:        50,
		BreakingChanges:        40,
		TestCoverageDifficulty: 30,
	}
	analyzer := service.StaticComplexityAnalyzer(dimensions)
	evaluator := service.NewDefaultComplexityEvaluator(analyzer)

	svc := newTestDesignService(sessionRepo, auditRepo, agentRunner, evaluator)

	// Mock expectations
	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)
	agentRunner.On("Execute", ctx, mock.AnythingOfType("*service.AgentRequest")).Return(
		&service.AgentResponse{
			Success: true,
			Output: `{"estimatedCodeChange":60,"affectedModules":50,"breakingChanges":40,"testCoverageDifficulty":30}`,
		}, nil,
	)
	sessionRepo.On("Update", ctx, session).Return(nil)
	auditRepo.On("Save", ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	// Execute
	score, dims, err := svc.EvaluateComplexity(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	// Expected: (60*30 + 50*25 + 40*25 + 30*20) / 100 = (1800 + 1250 + 1000 + 600) / 100 = 46
	assert.Equal(t, 46, score)
	assert.Equal(t, 60, dims.EstimatedCodeChange)

	sessionRepo.AssertExpectations(t)
	agentRunner.AssertExpectations(t)
}

func TestDesignService_GetDesignStatus_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithDesign()

	// Create design
	design := entity.NewDesign("# Design Document\n\nTest content")
	score, _ := valueobject.NewComplexityScore(55)
	design.SetComplexityScore(score, 70)
	session.SetDesign(design)

	sessionRepo := new(LocalMockSessionRepoForDesign)
	auditRepo := new(LocalMockAuditRepoForDesign)
	agentRunner := new(LocalMockAgentRunnerForDesign)

	svc := newTestDesignService(sessionRepo, auditRepo, agentRunner, nil)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	status, err := svc.GetDesignStatus(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, session.ID, status.SessionID)
	assert.Equal(t, "design", status.CurrentStage)
	assert.True(t, status.HasDesign)
	assert.Equal(t, 1, status.CurrentVersion)
	assert.Equal(t, 55, status.ComplexityScore)

	sessionRepo.AssertExpectations(t)
}

func TestDesignService_GetDesignStatus_NoDesign(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithDesign()
	// Don't set design

	sessionRepo := new(LocalMockSessionRepoForDesign)
	auditRepo := new(LocalMockAuditRepoForDesign)
	agentRunner := new(LocalMockAgentRunnerForDesign)

	svc := newTestDesignService(sessionRepo, auditRepo, agentRunner, nil)

	sessionRepo.On("FindByID", ctx, session.ID).Return(session, nil)

	// Execute
	status, err := svc.GetDesignStatus(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, session.ID, status.SessionID)
	assert.False(t, status.HasDesign)

	sessionRepo.AssertExpectations(t)
}
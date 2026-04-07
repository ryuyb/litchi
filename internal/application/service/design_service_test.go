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
	domainService "github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// --- Test Fixtures ---

func newTestDesignService(
	sessionRepo repository.WorkSessionRepository,
	auditRepo repository.AuditLogRepository,
	agentRunner domainService.AgentRunner,
	complexityEvaluator *domainService.DefaultComplexityEvaluator,
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
		sessionRepo:         sessionRepo,
		auditRepo:           auditRepo,
		agentRunner:         agentRunner,
		ghIssueService:      nil, // Will use mock via interface
		complexityEvaluator: complexityEvaluator,
		eventDispatcher:     dispatcher,
		config:              cfg,
		logger:              logger.Named("design_service"),
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

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	agentRunner := domainService.NewMockAgentRunner(t)

	svc := newTestDesignService(sessionRepo, auditRepo, agentRunner, nil)

	sessionRepo.EXPECT().FindByID(ctx, sessionID).Return(nil, nil)

	// Execute
	_, err := svc.StartDesign(ctx, sessionID)

	// Assert
	assert.Error(t, err)
}

func TestDesignService_StartDesign_WrongStage(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithDesign()
	// Roll back to clarification stage
	_ = session.RollbackTo(valueobject.StageClarification, "test", false)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	agentRunner := domainService.NewMockAgentRunner(t)

	svc := newTestDesignService(sessionRepo, auditRepo, agentRunner, nil)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	// Execute
	_, err := svc.StartDesign(ctx, session.ID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected Design")
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

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	agentRunner := domainService.NewMockAgentRunner(t)

	// Create complexity evaluator with static analyzer
	dimensions := valueobject.ComplexityDimensions{
		EstimatedCodeChange:    60,
		AffectedModules:        50,
		BreakingChanges:        40,
		TestCoverageDifficulty: 30,
	}
	analyzer := domainService.StaticComplexityAnalyzer(dimensions)
	evaluator := domainService.NewDefaultComplexityEvaluator(analyzer)

	svc := newTestDesignService(sessionRepo, auditRepo, agentRunner, evaluator)

	// Mock expectations
	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)
	agentRunner.EXPECT().Execute(ctx, mock.AnythingOfType("*service.AgentRequest")).Return(
		&domainService.AgentResponse{
			Success: true,
			Output:  `{"estimatedCodeChange":60,"affectedModules":50,"breakingChanges":40,"testCoverageDifficulty":30}`,
		}, nil,
	)
	sessionRepo.EXPECT().Update(ctx, session).Return(nil)
	auditRepo.EXPECT().Save(ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	// Execute
	score, dims, err := svc.EvaluateComplexity(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	// Expected: (60*30 + 50*25 + 40*25 + 30*20) / 100 = (1800 + 1250 + 1000 + 600) / 100 = 46
	assert.Equal(t, 46, score)
	assert.Equal(t, 60, dims.EstimatedCodeChange)
}

func TestDesignService_GetDesignStatus_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithDesign()

	// Create design
	design := entity.NewDesign("# Design Document\n\nTest content")
	score, _ := valueobject.NewComplexityScore(55)
	design.SetComplexityScore(score, 70)
	session.SetDesign(design)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	agentRunner := domainService.NewMockAgentRunner(t)

	svc := newTestDesignService(sessionRepo, auditRepo, agentRunner, nil)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	// Execute
	status, err := svc.GetDesignStatus(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, session.ID, status.SessionID)
	assert.Equal(t, "design", status.CurrentStage)
	assert.True(t, status.HasDesign)
	assert.Equal(t, 1, status.CurrentVersion)
	assert.Equal(t, 55, status.ComplexityScore)
}

func TestDesignService_GetDesignStatus_NoDesign(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithDesign()
	// Don't set design

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	agentRunner := domainService.NewMockAgentRunner(t)

	svc := newTestDesignService(sessionRepo, auditRepo, agentRunner, nil)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	// Execute
	status, err := svc.GetDesignStatus(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, session.ID, status.SessionID)
	assert.False(t, status.HasDesign)
}
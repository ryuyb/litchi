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

func newTestClarificationService(
	sessionRepo repository.WorkSessionRepository,
	auditRepo repository.AuditLogRepository,
	agentRunner domainService.AgentRunner,
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

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	agentRunner := domainService.NewMockAgentRunner(t)

	svc := newTestClarificationService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.EXPECT().FindByID(ctx, sessionID).Return(nil, nil)

	// Execute
	_, err := svc.StartClarification(ctx, sessionID)

	// Assert
	assert.Error(t, err)
}

func TestClarificationService_StartClarification_WrongStage(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithClarification()
	// Manually transition to Design stage
	_ = session.CompleteClarification()
	_ = session.TransitionTo(valueobject.StageDesign)

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	agentRunner := domainService.NewMockAgentRunner(t)

	svc := newTestClarificationService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	// Execute
	_, err := svc.StartClarification(ctx, session.ID)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected Clarification")
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

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	agentRunner := domainService.NewMockAgentRunner(t)

	svc := newTestClarificationService(sessionRepo, auditRepo, agentRunner)

	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)

	// Execute
	status, err := svc.GetClarityStatus(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, session.ID, status.SessionID)
	assert.Equal(t, "clarification", status.CurrentStage)
	assert.Equal(t, 86, status.ClarityScore)
}

func TestClarificationService_EvaluateClarity_Success(t *testing.T) {
	ctx := context.Background()
	session := newTestSessionWithClarification()

	sessionRepo := repository.NewMockWorkSessionRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	agentRunner := domainService.NewMockAgentRunner(t)

	svc := newTestClarificationService(sessionRepo, auditRepo, agentRunner)

	// Mock expectations
	sessionRepo.EXPECT().FindByID(ctx, session.ID).Return(session, nil)
	agentRunner.EXPECT().Execute(ctx, mock.AnythingOfType("*service.AgentRequest")).Return(
		&domainService.AgentResponse{
			Success: true,
			Output: `{"completeness":{"score":25},"clarity":{"score":20},"consistency":{"score":18},"feasibility":{"score":14},"testability":{"score":9}}`,
		}, nil,
	)
	sessionRepo.EXPECT().Update(ctx, session).Return(nil)
	auditRepo.EXPECT().Save(ctx, mock.AnythingOfType("*entity.AuditLog")).Return(nil)

	// Execute
	score, err := svc.EvaluateClarity(ctx, session.ID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 86, score)
}
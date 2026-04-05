// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/github"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/ryuyb/litchi/internal/pkg/utils"
	"go.uber.org/zap"
)

// DefaultComplexityScore is the default complexity score used when evaluation fails.
// Value 50 is a safe middle ground within the valid 0-100 range.
const DefaultComplexityScore = 50

// DesignService handles the design phase of WorkSession.
// It manages design generation, version management, and complexity evaluation.
type DesignService struct {
	sessionRepo        repository.WorkSessionRepository
	auditRepo          repository.AuditLogRepository
	agentRunner        service.AgentRunner
	ghIssueService     *github.IssueService
	complexityEvaluator *service.DefaultComplexityEvaluator
	eventDispatcher    *event.Dispatcher
	config             *config.Config
	logger             *zap.Logger
}

// NewDesignService creates a new DesignService.
func NewDesignService(
	sessionRepo repository.WorkSessionRepository,
	auditRepo repository.AuditLogRepository,
	agentRunner service.AgentRunner,
	ghIssueService *github.IssueService,
	complexityEvaluator *service.DefaultComplexityEvaluator,
	eventDispatcher *event.Dispatcher,
	config *config.Config,
	logger *zap.Logger,
) *DesignService {
	return &DesignService{
		sessionRepo:        sessionRepo,
		auditRepo:          auditRepo,
		agentRunner:        agentRunner,
		ghIssueService:     ghIssueService,
		complexityEvaluator: complexityEvaluator,
		eventDispatcher:    eventDispatcher,
		config:             config,
		logger:             logger.Named("design_service"),
	}
}

// StartDesign starts the design process for a session.
// This method generates the initial design document based on clarified requirements.
//
// Steps:
// 1. Validate session is in Design stage
// 2. Prepare Agent context with issue and clarification info
// 3. Execute Agent to generate design
// 4. Evaluate design complexity
// 5. Post design as GitHub comment
// 6. Update session with design
//
// Returns the generated design content.
func (s *DesignService) StartDesign(
	ctx context.Context,
	sessionID uuid.UUID,
) (designContent string, err error) {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return "", litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return "", litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate session is in Design stage
	if session.GetCurrentStage() != valueobject.StageDesign {
		return "", litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected Design", session.GetCurrentStage()),
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return "", litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Check if design already exists
	if session.Design != nil {
		return "", litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"design already exists, use update to create new version",
		)
	}

	// 5. Prepare Agent request
	agentReq := &service.AgentRequest{
		SessionID: session.ID,
		Stage:     service.AgentStageDesign,
		Prompt:    s.buildDesignPrompt(session),
		Context: &service.AgentContext{
			IssueTitle:      session.Issue.Title,
			IssueBody:       session.Issue.Body,
			Repository:      session.Issue.Repository,
			ClarifiedPoints: session.Clarification.ConfirmedPoints,
		},
		Timeout: s.parseTimeout(s.config.Failure.Timeout.DesignGeneration),
	}

	// 6. Execute Agent to generate design
	response, err := s.agentRunner.Execute(ctx, agentReq)
	if err != nil {
		s.logger.Error("failed to execute agent for design generation",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpDesignStart, startTime, false, err.Error())
		return "", litchierrors.Wrap(litchierrors.ErrAgentExecutionFail, err)
	}

	designContent = response.Output

	// 7. Create design entity
	design := entity.NewDesign(designContent)

	// 8. Evaluate complexity
	threshold := s.config.Complexity.Threshold
	if s.complexityEvaluator != nil {
		score, dimensions, err := s.evaluateComplexity(ctx, session, design)
		if err != nil {
			s.logger.Warn("failed to evaluate complexity, using default",
				zap.String("session_id", sessionID.String()),
				zap.Error(err),
			)
			// Set default complexity if evaluation fails
			defaultScore := getDefaultComplexityScore()
			design.SetComplexityScore(defaultScore, threshold)
		} else {
			design.SetComplexityScore(score, threshold)
			s.logger.Info("complexity evaluated",
				zap.String("session_id", sessionID.String()),
				zap.Int("score", score.Value()),
				zap.Any("dimensions", dimensions),
			)
		}
	} else {
		// No evaluator configured, use default
		defaultScore, _ := valueobject.NewComplexityScore(50)
		design.SetComplexityScore(defaultScore, threshold)
	}

	// 9. Check if force design confirm is enabled
	if s.config.Complexity.ForceDesignConfirm {
		design.RequireConfirmation = true
	}

	// 10. Set design to session
	session.SetDesign(design)

	// 11. Post design to GitHub issue
	if err := s.postDesignToIssue(ctx, session, designContent); err != nil {
		s.logger.Warn("failed to post design to issue",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		// Continue even if posting fails - design is saved in session
	}

	// 12. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return "", litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 13. Record audit log
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpDesignStart, startTime, true, "")

	// 14. Publish events
	s.publishEvents(ctx, session)

	s.logger.Info("design started",
		zap.String("session_id", sessionID.String()),
		zap.Int("complexity_score", design.ComplexityScore.Value()),
		zap.Bool("requires_confirmation", design.RequireConfirmation),
	)

	return designContent, nil
}

// ConfirmDesign confirms the design for a session.
// Only admins can confirm designs.
//
// Steps:
// 1. Validate actor is admin
// 2. Validate design exists and requires confirmation
// 3. Mark design as confirmed
// 4. If all conditions met, transition to TaskBreakdown stage
//
// Returns true if transitioned to TaskBreakdown stage.
func (s *DesignService) ConfirmDesign(
	ctx context.Context,
	sessionID uuid.UUID,
	actor string,
) (transitioned bool, err error) {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return false, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return false, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate session is in Design stage
	if session.GetCurrentStage() != valueobject.StageDesign {
		return false, litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected Design", session.GetCurrentStage()),
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return false, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Validate design exists
	if session.Design == nil {
		return false, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"design not initialized",
		)
	}

	// 5. Check actor permission (must be admin)
	actorRole, err := s.checkActorPermission(ctx, session, actor)
	if err != nil {
		return false, err
	}

	// Only admin can confirm design
	if actorRole != valueobject.ActorRoleAdmin {
		return false, litchierrors.New(litchierrors.ErrPermissionDenied).WithDetail(
			"only admins can confirm designs",
		)
	}

	// 6. Confirm design
	if err := session.ConfirmDesign(); err != nil {
		return false, err
	}

	// 7. Check if can transition to TaskBreakdown
	if session.Design.CanProceedToTaskBreakdown() {
		// Transition to TaskBreakdown stage
		if err := session.TransitionTo(valueobject.StageTaskBreakdown); err != nil {
			return false, err
		}
		transitioned = true
	}

	// 8. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return false, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 9. Record audit log
	s.recordAuditLog(ctx, session, actor, actorRole,
		valueobject.OpDesignConfirm, startTime, true, "")

	// 10. Publish events
	s.publishEvents(ctx, session)

	s.logger.Info("design confirmed",
		zap.String("session_id", sessionID.String()),
		zap.String("actor", actor),
		zap.Bool("transitioned", transitioned),
	)

	return transitioned, nil
}

// RejectDesign rejects the design for a session.
// Only admins can reject designs.
//
// Steps:
// 1. Validate actor is admin
// 2. Validate design exists
// 3. Mark design as rejected
// 4. Post rejection reason to GitHub issue
func (s *DesignService) RejectDesign(
	ctx context.Context,
	sessionID uuid.UUID,
	actor string,
	reason string,
) error {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate session is in Design stage
	if session.GetCurrentStage() != valueobject.StageDesign {
		return litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected Design", session.GetCurrentStage()),
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Validate design exists
	if session.Design == nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"design not initialized",
		)
	}

	// 5. Check actor permission (must be admin)
	actorRole, err := s.checkActorPermission(ctx, session, actor)
	if err != nil {
		return err
	}

	// Only admin can reject design
	if actorRole != valueobject.ActorRoleAdmin {
		return litchierrors.New(litchierrors.ErrPermissionDenied).WithDetail(
			"only admins can reject designs",
		)
	}

	// 6. Reject design
	if err := session.RejectDesign(reason); err != nil {
		return err
	}

	// 7. Post rejection to GitHub issue
	if err := s.postRejectionToIssue(ctx, session, reason); err != nil {
		s.logger.Warn("failed to post rejection to issue",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
	}

	// 8. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 9. Record audit log
	s.recordAuditLog(ctx, session, actor, actorRole,
		valueobject.OpDesignReject, startTime, true, reason)

	// 10. Publish events
	s.publishEvents(ctx, session)

	s.logger.Info("design rejected",
		zap.String("session_id", sessionID.String()),
		zap.String("actor", actor),
		zap.String("reason", reason),
	)

	return nil
}

// UpdateDesign creates a new version of the design.
// This is used when design needs revision after feedback.
//
// Steps:
// 1. Validate design exists
// 2. Generate new design version using Agent
// 3. Add new version to design
// 4. Re-evaluate complexity
// 5. Post update to GitHub issue
//
// Returns the new version number.
func (s *DesignService) UpdateDesign(
	ctx context.Context,
	sessionID uuid.UUID,
	reason string,
	feedback string,
) (newVersion int, err error) {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return 0, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate session is in Design stage
	if session.GetCurrentStage() != valueobject.StageDesign {
		return 0, litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected Design", session.GetCurrentStage()),
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return 0, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Validate design exists
	if session.Design == nil {
		return 0, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"design not initialized",
		)
	}

	// 5. Prepare Agent request for updated design
	agentReq := &service.AgentRequest{
		SessionID: session.ID,
		Stage:     service.AgentStageDesign,
		Prompt:    s.buildDesignUpdatePrompt(session, reason, feedback),
		Context: &service.AgentContext{
			IssueTitle:      session.Issue.Title,
			IssueBody:       session.Issue.Body,
			Repository:      session.Issue.Repository,
			ClarifiedPoints: session.Clarification.ConfirmedPoints,
			DesignContent:   session.Design.GetCurrentContent(),
		},
		Timeout: s.parseTimeout(s.config.Failure.Timeout.DesignGeneration),
	}

	// 6. Execute Agent to generate updated design
	response, err := s.agentRunner.Execute(ctx, agentReq)
	if err != nil {
		s.logger.Error("failed to execute agent for design update",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpDesignUpdate, startTime, false, err.Error())
		return 0, litchierrors.Wrap(litchierrors.ErrAgentExecutionFail, err)
	}

	newContent := response.Output

	// 7. Add new version
	session.AddDesignVersion(newContent, reason)
	newVersion = session.Design.CurrentVersion

	// 8. Re-evaluate complexity
	threshold := s.config.Complexity.Threshold
	if s.complexityEvaluator != nil {
		score, _, err := s.evaluateComplexity(ctx, session, session.Design)
		if err != nil {
			s.logger.Warn("failed to re-evaluate complexity",
				zap.String("session_id", sessionID.String()),
				zap.Error(err),
			)
		} else {
			session.Design.SetComplexityScore(score, threshold)
		}
	}

	// 9. Post update to GitHub issue
	if err := s.postDesignUpdateToIssue(ctx, session, newContent, reason); err != nil {
		s.logger.Warn("failed to post design update to issue",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
	}

	// 10. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 11. Record audit log
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpDesignUpdate, startTime, true, "")

	// 12. Publish events
	s.publishEvents(ctx, session)

	s.logger.Info("design updated",
		zap.String("session_id", sessionID.String()),
		zap.Int("new_version", newVersion),
		zap.String("reason", reason),
	)

	return newVersion, nil
}

// EvaluateComplexity evaluates the complexity of the current design.
// Returns the complexity score and dimensions.
func (s *DesignService) EvaluateComplexity(
	ctx context.Context,
	sessionID uuid.UUID,
) (score int, dimensions valueobject.ComplexityDimensions, err error) {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return 0, valueobject.ComplexityDimensions{}, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return 0, valueobject.ComplexityDimensions{}, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate design exists
	if session.Design == nil {
		return 0, valueobject.ComplexityDimensions{}, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"design not initialized",
		)
	}

	// 3. Evaluate complexity
	if s.complexityEvaluator == nil {
		return 0, valueobject.ComplexityDimensions{}, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"complexity evaluator not configured",
		)
	}

	complexityScore, complexityDims, err := s.evaluateComplexity(ctx, session, session.Design)
	if err != nil {
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpComplexityEvaluate, startTime, false, err.Error())
		return 0, valueobject.ComplexityDimensions{}, err
	}

	// 4. Update design with new complexity score
	threshold := s.config.Complexity.Threshold
	session.Design.SetComplexityScore(complexityScore, threshold)

	// 5. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return 0, valueobject.ComplexityDimensions{}, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 6. Record audit log
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpComplexityEvaluate, startTime, true, "")

	// 7. Publish events
	s.publishEvents(ctx, session)

	return complexityScore.Value(), complexityDims, nil
}

// GetDesignStatus returns the current design status.
func (s *DesignService) GetDesignStatus(
	ctx context.Context,
	sessionID uuid.UUID,
) (*DesignStatus, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	status := &DesignStatus{
		SessionID:    sessionID,
		CurrentStage: string(session.GetCurrentStage()),
	}

	if session.Design != nil {
		status.HasDesign = true
		status.CurrentVersion = session.Design.CurrentVersion
		status.ComplexityScore = session.Design.ComplexityScore.Value()
		status.ComplexityGrade = session.Design.ComplexityScore.Grade()
		status.RequireConfirmation = session.Design.RequireConfirmation
		status.Confirmed = session.Design.Confirmed
		status.CanProceed = session.Design.CanProceedToTaskBreakdown()
		status.Content = session.Design.GetCurrentContent()
		status.VersionCount = len(session.Design.Versions)
	}

	status.Threshold = s.config.Complexity.Threshold
	status.ForceConfirm = s.config.Complexity.ForceDesignConfirm

	return status, nil
}

// DesignStatus represents the current status of design.
type DesignStatus struct {
	SessionID          uuid.UUID `json:"sessionId"`
	CurrentStage       string    `json:"currentStage"`
	HasDesign          bool      `json:"hasDesign"`
	CurrentVersion     int       `json:"currentVersion"`
	ComplexityScore    int       `json:"complexityScore"`
	ComplexityGrade    string    `json:"complexityGrade"`
	RequireConfirmation bool     `json:"requireConfirmation"`
	Confirmed          bool      `json:"confirmed"`
	CanProceed         bool      `json:"canProceed"`
	Content            string    `json:"content,omitempty"`
	VersionCount       int       `json:"versionCount"`
	Threshold          int       `json:"threshold"`
	ForceConfirm       bool      `json:"forceConfirm"`
}

// --- Internal Helper Methods ---

// buildDesignPrompt builds the prompt for Agent to generate design.
func (s *DesignService) buildDesignPrompt(session *aggregate.WorkSession) string {
	confirmedPointsStr := ""
	for i, point := range session.Clarification.ConfirmedPoints {
		confirmedPointsStr += fmt.Sprintf("%d. %s\n", i+1, point)
	}

	return fmt.Sprintf(`Please create a detailed design document for the following GitHub issue.

Issue Title: %s

Issue Body:
%s

Confirmed Requirements:
%s

Requirements:
1. Create a comprehensive design document following the template below
2. Include architecture decisions with rationale
3. List all affected files and modules
4. Describe implementation steps in logical order
5. Identify potential risks and mitigations
6. Define test strategy

Design Document Template:
# Design Document

## Overview
[High-level summary of the change]

## Architecture Decisions
- [Decision 1]: [Rationale]
- [Decision 2]: [Rationale]

## Affected Components
- [Component 1]: [Description of changes]
- [Component 2]: [Description of changes]

## Implementation Plan
1. [Step 1]
2. [Step 2]
3. [Step 3]

## API Changes
[Describe any API changes if applicable]

## Database Changes
[Describe any database schema changes if applicable]

## Test Strategy
- Unit tests: [Description]
- Integration tests: [Description]
- E2E tests: [Description if needed]

## Risks and Mitigations
- Risk: [Description] -> Mitigation: [Strategy]

## Estimated Effort
- Estimated lines of code: [approximate]
- Number of affected modules: [count]
- Breaking changes: [yes/no with explanation]
- Test coverage difficulty: [low/medium/high]

Output: Provide the complete design document in markdown format.`,
		session.Issue.Title,
		session.Issue.Body,
		confirmedPointsStr,
	)
}

// buildDesignUpdatePrompt builds the prompt for Agent to update design.
func (s *DesignService) buildDesignUpdatePrompt(session *aggregate.WorkSession, reason, feedback string) string {
	return fmt.Sprintf(`Please update the design document based on the following feedback.

Current Design:
%s

Update Reason: %s

Feedback:
%s

Requirements:
1. Update the design to address the feedback
2. Maintain consistency with the original structure
3. Update affected sections as needed
4. Preserve sections that don't need changes

Output: Provide the complete updated design document in markdown format.`,
		session.Design.GetCurrentContent(),
		reason,
		feedback,
	)
}

// evaluateComplexity evaluates design complexity using Agent.
func (s *DesignService) evaluateComplexity(
	ctx context.Context,
	session *aggregate.WorkSession,
	design *entity.Design,
) (valueobject.ComplexityScore, valueobject.ComplexityDimensions, error) {
	// Build prompt for complexity evaluation
	prompt := s.buildComplexityPrompt(session, design)

	agentReq := &service.AgentRequest{
		SessionID: session.ID,
		Stage:     service.AgentStageDesign,
		Prompt:    prompt,
		Context: &service.AgentContext{
			IssueTitle:    session.Issue.Title,
			Repository:    session.Issue.Repository,
			DesignContent: design.GetCurrentContent(),
		},
		Timeout: s.parseTimeout(s.config.Failure.Timeout.DesignAnalysis),
	}

	response, err := s.agentRunner.Execute(ctx, agentReq)
	if err != nil {
		return valueobject.ComplexityScore{}, valueobject.ComplexityDimensions{},
			litchierrors.Wrap(litchierrors.ErrAgentExecutionFail, err)
	}

	// Parse complexity dimensions from response
	dimensions, err := s.parseComplexityDimensions(response)
	if err != nil {
		s.logger.Warn("failed to parse complexity dimensions, using default",
			zap.String("session_id", session.ID.String()),
			zap.Error(err),
		)
		// Return default dimensions based on design content
		dimensions = s.defaultComplexityDimensions(design)
	}

	// Calculate score from dimensions using evaluator
	if s.complexityEvaluator != nil {
		return s.complexityEvaluator.Evaluate(design, nil, nil)
	}

	// Fallback: calculate directly from dimensions
	score, err := valueobject.NewComplexityScoreFromDimensions(dimensions)
	if err != nil {
		return valueobject.ComplexityScore{}, dimensions, err
	}

	return score, dimensions, nil
}

// buildComplexityPrompt builds the prompt for complexity evaluation.
func (s *DesignService) buildComplexityPrompt(session *aggregate.WorkSession, design *entity.Design) string {
	return fmt.Sprintf(`Please evaluate the complexity of the following design.

Design Document:
%s

Repository: %s
Issue Title: %s

Evaluation Dimensions (score each 0-100):
1. EstimatedCodeChange (0-100): How much code needs to be changed?
   - 0-20: Minor changes, < 100 lines
   - 21-40: Small changes, 100-300 lines
   - 41-60: Medium changes, 300-500 lines
   - 61-80: Large changes, 500-1000 lines
   - 81-100: Major refactoring, > 1000 lines

2. AffectedModules (0-100): How many modules/components are affected?
   - 0-20: Single module
   - 21-40: 2-3 modules
   - 41-60: 4-6 modules
   - 61-80: 7-10 modules
   - 81-100: > 10 modules or system-wide

3. BreakingChanges (0-100): Risk of breaking existing functionality?
   - 0-20: No breaking changes, backward compatible
   - 21-40: Minor breaking changes in internal APIs
   - 41-60: Breaking changes in public APIs with migration path
   - 61-80: Significant breaking changes requiring updates
   - 81-100: Major breaking changes, difficult migration

4. TestCoverageDifficulty (0-100): How difficult to test thoroughly?
   - 0-20: Easy to test, straightforward unit tests
   - 21-40: Moderate, some integration tests needed
   - 41-60: Requires mocking and integration tests
   - 61-80: Complex test setup, multiple test types needed
   - 81-100: Very difficult to test, requires special infrastructure

Output format (JSON):
{
  "estimatedCodeChange": <0-100>,
  "affectedModules": <0-100>,
  "breakingChanges": <0-100>,
  "testCoverageDifficulty": <0-100>
}`,
		design.GetCurrentContent(),
		session.Issue.Repository,
		session.Issue.Title,
	)
}

// parseComplexityDimensions parses complexity dimensions from Agent response.
func (s *DesignService) parseComplexityDimensions(response *service.AgentResponse) (valueobject.ComplexityDimensions, error) {
	if response == nil || response.Output == "" {
		return valueobject.ComplexityDimensions{}, fmt.Errorf("empty response")
	}

	// Try to parse JSON from response
	output := response.Output

	// Find JSON block in response
	jsonStart := strings.Index(output, "{")
	jsonEnd := strings.LastIndex(output, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return valueobject.ComplexityDimensions{}, fmt.Errorf("no JSON found in response")
	}

	jsonStr := output[jsonStart : jsonEnd+1]

	// Parse into intermediate struct
	var evalResult struct {
		EstimatedCodeChange    int `json:"estimatedCodeChange"`
		AffectedModules        int `json:"affectedModules"`
		BreakingChanges        int `json:"breakingChanges"`
		TestCoverageDifficulty int `json:"testCoverageDifficulty"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &evalResult); err != nil {
		return valueobject.ComplexityDimensions{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Build ComplexityDimensions
	dimensions := valueobject.ComplexityDimensions{
		EstimatedCodeChange:    evalResult.EstimatedCodeChange,
		AffectedModules:        evalResult.AffectedModules,
		BreakingChanges:        evalResult.BreakingChanges,
		TestCoverageDifficulty: evalResult.TestCoverageDifficulty,
	}

	return dimensions, nil
}

// defaultComplexityDimensions returns default dimensions based on design content.
func (s *DesignService) defaultComplexityDimensions(design *entity.Design) valueobject.ComplexityDimensions {
	// Simple heuristic based on content length
	content := design.GetCurrentContent()
	lineCount := len(strings.Split(content, "\n"))

	// Map line count to complexity
	estimatedCode := 30 // Default low-medium
	if lineCount > 200 {
		estimatedCode = 70
	} else if lineCount > 100 {
		estimatedCode = 50
	}

	return valueobject.ComplexityDimensions{
		EstimatedCodeChange:    estimatedCode,
		AffectedModules:        40,
		BreakingChanges:        30,
		TestCoverageDifficulty: 40,
	}
}

// postDesignToIssue posts the design as a comment on the GitHub issue.
func (s *DesignService) postDesignToIssue(
	ctx context.Context,
	session *aggregate.WorkSession,
	designContent string,
) error {
	// Build comment body
	commentBody := "## 设计方案\n\n"
	commentBody += designContent
	commentBody += "\n\n---\n"

	// Add complexity info
	if session.Design != nil {
		commentBody += fmt.Sprintf("**复杂度评分**: %d (%s)\n\n",
			session.Design.ComplexityScore.Value(),
			session.Design.ComplexityScore.Grade(),
		)

		if session.Design.RequireConfirmation {
			commentBody += "*设计方案需要管理员确认。请使用 `@bot confirm_design` 确认或 `@bot reject_design <原因>` 拒绝。*"
		} else {
			commentBody += "*设计方案已自动确认，即将进入任务拆解阶段。*"
		}
	}

	// Post comment
	owner := utils.ExtractOwner(session.Issue.Repository)
	repo := utils.ExtractRepo(session.Issue.Repository)

	if _, err := s.ghIssueService.CreateComment(ctx, owner, repo, session.Issue.Number, commentBody); err != nil {
		return err
	}

	return nil
}

// postRejectionToIssue posts the rejection reason as a comment on the GitHub issue.
func (s *DesignService) postRejectionToIssue(
	ctx context.Context,
	session *aggregate.WorkSession,
	reason string,
) error {
	commentBody := "## 设计方案已拒绝\n\n"
	commentBody += fmt.Sprintf("**拒绝原因**: %s\n\n", reason)
	commentBody += "请使用 `@bot update_design <反馈>` 提供修改建议，系统将生成新的设计方案版本。"

	owner := utils.ExtractOwner(session.Issue.Repository)
	repo := utils.ExtractRepo(session.Issue.Repository)

	if _, err := s.ghIssueService.CreateComment(ctx, owner, repo, session.Issue.Number, commentBody); err != nil {
		return err
	}

	return nil
}

// postDesignUpdateToIssue posts the updated design as a comment on the GitHub issue.
func (s *DesignService) postDesignUpdateToIssue(
	ctx context.Context,
	session *aggregate.WorkSession,
	newContent string,
	reason string,
) error {
	commentBody := fmt.Sprintf("## 设计方案更新 (v%d)\n\n", session.Design.CurrentVersion)
	commentBody += fmt.Sprintf("**更新原因**: %s\n\n", reason)
	commentBody += newContent
	commentBody += "\n\n---\n"

	// Add complexity info
	if session.Design != nil {
		commentBody += fmt.Sprintf("**复杂度评分**: %d (%s)\n\n",
			session.Design.ComplexityScore.Value(),
			session.Design.ComplexityScore.Grade(),
		)

		if session.Design.RequireConfirmation {
			commentBody += "*更新后的设计方案需要管理员确认。*"
		}
	}

	owner := utils.ExtractOwner(session.Issue.Repository)
	repo := utils.ExtractRepo(session.Issue.Repository)

	if _, err := s.ghIssueService.CreateComment(ctx, owner, repo, session.Issue.Number, commentBody); err != nil {
		return err
	}

	return nil
}

// checkActorPermission checks if the actor has permission to confirm/reject designs.
func (s *DesignService) checkActorPermission(
	ctx context.Context,
	session *aggregate.WorkSession,
	actor string,
) (valueobject.ActorRole, error) {
	// Check if actor is repo admin
	owner := utils.ExtractOwner(session.Issue.Repository)
	repo := utils.ExtractRepo(session.Issue.Repository)

	isAdmin, err := s.ghIssueService.IsRepoAdmin(ctx, owner, repo, actor)
	if err != nil {
		s.logger.Warn("failed to check admin permission",
			zap.String("actor", actor),
			zap.Error(err),
		)
		return "", litchierrors.New(litchierrors.ErrPermissionDenied).WithDetail(
			"failed to verify admin permission",
		)
	}

	if isAdmin {
		return valueobject.ActorRoleAdmin, nil
	}

	// Check if actor is issue author (has limited permissions)
	if session.Issue.Author == actor {
		return valueobject.ActorRoleIssueAuthor, nil
	}

	return "", litchierrors.New(litchierrors.ErrPermissionDenied).WithDetail(
		fmt.Sprintf("actor %s is not authorized (must be admin)", actor),
	)
}

// recordAuditLog records an audit log entry.
func (s *DesignService) recordAuditLog(
	ctx context.Context,
	session *aggregate.WorkSession,
	actor string,
	actorRole valueobject.ActorRole,
	operation valueobject.OperationType,
	startTime time.Time,
	success bool,
	errMsg string,
) {
	if session == nil {
		return
	}

	auditLog := entity.NewAuditLog(
		session.ID,
		session.Issue.Repository,
		session.Issue.Number,
		actor,
		actorRole,
		operation,
		"design",
		session.ID.String(),
	)

	auditLog.SetDuration(int(time.Since(startTime).Milliseconds()))

	if success {
		auditLog.MarkSuccess()
	} else if errMsg != "" {
		auditLog.MarkFailed(errMsg)
	}

	if err := s.auditRepo.Save(ctx, auditLog); err != nil {
		s.logger.Warn("failed to save audit log",
			zap.String("session_id", session.ID.String()),
			zap.Error(err),
		)
	}
}

// publishEvents publishes domain events from the session.
func (s *DesignService) publishEvents(ctx context.Context, session *aggregate.WorkSession) {
	events := session.GetEvents()
	if len(events) == 0 {
		return
	}

	if err := s.eventDispatcher.DispatchBatch(ctx, events); err != nil {
		s.logger.Warn("failed to dispatch events",
			zap.String("session_id", session.ID.String()),
			zap.Int("event_count", len(events)),
			zap.Error(err),
		)
	}

	session.ClearEvents()
}

// parseTimeout parses timeout string to Duration.
func (s *DesignService) parseTimeout(timeoutStr string) time.Duration {
	if timeoutStr == "" {
		return 10 * time.Minute // Default for design
	}
	d, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 10 * time.Minute
	}
	return d
}

// getDefaultComplexityScore returns a safe default complexity score.
// The default value (50) is guaranteed to be valid within the 0-100 range.
func getDefaultComplexityScore() valueobject.ComplexityScore {
	// DefaultComplexityScore (50) is always valid, so we can safely ignore the error
	score, _ := valueobject.NewComplexityScore(DefaultComplexityScore)
	return score
}
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

// ClarificationService handles the clarification phase of WorkSession.
// It manages question generation, answer processing, and clarity evaluation.
type ClarificationService struct {
	sessionRepo     repository.WorkSessionRepository
	auditRepo       repository.AuditLogRepository
	agentRunner     service.AgentRunner
	ghIssueService  *github.IssueService
	eventDispatcher *event.Dispatcher
	config          *config.Config
	logger          *zap.Logger
}

// NewClarificationService creates a new ClarificationService.
func NewClarificationService(
	sessionRepo repository.WorkSessionRepository,
	auditRepo repository.AuditLogRepository,
	agentRunner service.AgentRunner,
	ghIssueService *github.IssueService,
	eventDispatcher *event.Dispatcher,
	config *config.Config,
	logger *zap.Logger,
) *ClarificationService {
	return &ClarificationService{
		sessionRepo:     sessionRepo,
		auditRepo:       auditRepo,
		agentRunner:     agentRunner,
		ghIssueService:  ghIssueService,
		eventDispatcher: eventDispatcher,
		config:          config,
		logger:          logger.Named("clarification_service"),
	}
}

// StartClarification starts the clarification process for a session.
// This method generates initial questions based on the issue content.
//
// Steps:
// 1. Validate session is in Clarification stage
// 2. Prepare Agent context with issue information
// 3. Execute Agent to generate questions
// 4. Post questions as GitHub comments
// 5. Update session with generated questions
//
// Returns the list of generated questions.
func (s *ClarificationService) StartClarification(
	ctx context.Context,
	sessionID uuid.UUID,
) (questions []string, err error) {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate session is in Clarification stage
	if session.GetCurrentStage() != valueobject.StageClarification {
		return nil, litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected Clarification", session.GetCurrentStage()),
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Prepare Agent request
	agentReq := &service.AgentRequest{
		SessionID: session.ID,
		Stage:     service.AgentStageClarification,
		Prompt:    s.buildClarificationPrompt(session),
		Context: &service.AgentContext{
			IssueTitle: session.Issue.Title,
			IssueBody:  session.Issue.Body,
			Repository: session.Issue.Repository,
		},
		Timeout: s.parseTimeout(s.config.Failure.Timeout.ClarificationAgent),
	}

	// 5. Execute Agent to generate questions
	response, err := s.agentRunner.Execute(ctx, agentReq)
	if err != nil {
		s.logger.Error("failed to execute agent for clarification",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpClarificationStart, startTime, false, err.Error())
		return nil, litchierrors.Wrap(litchierrors.ErrAgentExecutionFail, err)
	}

	// 6. Parse questions from Agent response
	questions = s.parseQuestionsFromResponse(response)
	if len(questions) == 0 {
		// No questions generated means issue is clear enough
		s.logger.Info("no clarification questions generated, issue is clear",
			zap.String("session_id", sessionID.String()),
		)
	}

	// 7. Add questions to session
	for _, q := range questions {
		session.AddClarificationQuestion(q)
	}

	// 8. Post questions to GitHub issue
	if len(questions) > 0 {
		if err := s.postQuestionsToIssue(ctx, session, questions); err != nil {
			s.logger.Warn("failed to post questions to issue",
				zap.String("session_id", sessionID.String()),
				zap.Error(err),
			)
			// Continue even if posting fails - questions are saved in session
		}
	}

	// 9. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 10. Record audit log
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpClarificationStart, startTime, true, "")

	// 11. Publish events
	s.publishEvents(ctx, session)

	s.logger.Info("clarification started",
		zap.String("session_id", sessionID.String()),
		zap.Int("question_count", len(questions)),
	)

	return questions, nil
}

// ProcessAnswer processes a user's answer to a clarification question.
// This method is triggered when the issue author or admin replies to a question.
//
// Steps:
// 1. Validate actor permission (issue author or admin)
// 2. Find the matching question
// 3. Record answer in session
// 4. Evaluate if more questions needed
// 5. If all questions answered, evaluate clarity
//
// Returns true if clarification is complete after this answer.
func (s *ClarificationService) ProcessAnswer(
	ctx context.Context,
	sessionID uuid.UUID,
	question string,
	answer string,
	actor string,
) (complete bool, err error) {
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

	// 2. Validate session is in Clarification stage
	if session.GetCurrentStage() != valueobject.StageClarification {
		return false, litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected Clarification", session.GetCurrentStage()),
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return false, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Check actor permission
	actorRole, err := s.checkActorPermission(ctx, session, actor)
	if err != nil {
		return false, err
	}

	// 5. Record answer
	if err := session.AnswerClarificationQuestion(question, answer, actor); err != nil {
		return false, err
	}

	// 6. Extract confirmed points from answer
	points := s.extractConfirmedPoints(question, answer)
	for _, point := range points {
		session.ConfirmClarificationPoint(point)
	}

	// 7. Check if all questions answered
	if !session.Clarification.HasPendingQuestions() {
		// All questions answered, evaluate clarity
		if err := s.evaluateClarity(ctx, session); err != nil {
			s.logger.Warn("failed to evaluate clarity",
				zap.String("session_id", sessionID.String()),
				zap.Error(err),
			)
			// Save session anyway
			if err := s.sessionRepo.Update(ctx, session); err != nil {
				return false, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
			}
			return false, nil
		}

		// Check if can complete clarification
		threshold := s.config.Clarity.Threshold
		if session.CanCompleteClarification(threshold) {
			// Complete clarification
			if err := session.CompleteClarification(); err != nil {
				return false, err
			}

			// Transition to Design stage
			if err := session.TransitionTo(valueobject.StageDesign); err != nil {
				return false, err
			}

			complete = true
		}
	}

	// 8. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return false, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 9. Record audit log
	s.recordAuditLog(ctx, session, actor, actorRole,
		valueobject.OpClarificationAnswer, startTime, true, "")

	// 10. Publish events
	s.publishEvents(ctx, session)

	s.logger.Info("clarification answer processed",
		zap.String("session_id", sessionID.String()),
		zap.String("actor", actor),
		zap.Bool("complete", complete),
	)

	return complete, nil
}

// ForceStartDesign forces the transition to Design stage regardless of clarity score.
// This is used when user explicitly confirms to proceed with low clarity.
// User must use "@bot start_design" command to trigger this.
//
// Steps:
// 1. Validate actor permission
// 2. Validate no pending questions
// 3. Mark clarification as complete
// 4. Transition to Design stage
func (s *ClarificationService) ForceStartDesign(
	ctx context.Context,
	sessionID uuid.UUID,
	actor string,
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

	// 2. Validate session is in Clarification stage
	if session.GetCurrentStage() != valueobject.StageClarification {
		return litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected Clarification", session.GetCurrentStage()),
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Check actor permission
	actorRole, err := s.checkActorPermission(ctx, session, actor)
	if err != nil {
		return err
	}

	// 5. Validate no pending questions (hard constraint)
	if session.Clarification.HasPendingQuestions() {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"cannot force start design: pending questions must be answered",
		)
	}

	// 6. Validate at least one confirmed point (hard constraint)
	if len(session.Clarification.ConfirmedPoints) == 0 {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"cannot force start design: at least one requirement point must be confirmed",
		)
	}

	// 7. Complete clarification
	if err := session.CompleteClarification(); err != nil {
		return err
	}

	// 8. Transition to Design stage
	if err := session.TransitionTo(valueobject.StageDesign); err != nil {
		return err
	}

	// 9. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 10. Record audit log
	s.recordAuditLog(ctx, session, actor, actorRole,
		valueobject.OpClarificationForceStartDesign, startTime, true, "")

	// 11. Publish events
	s.publishEvents(ctx, session)

	s.logger.Info("force start design completed",
		zap.String("session_id", sessionID.String()),
		zap.String("actor", actor),
		zap.Int("clarity_score", session.Clarification.GetClarityScore()),
	)

	return nil
}

// EvaluateClarity evaluates the clarity score for a session.
// This is called when all questions are answered.
// Returns the calculated clarity score.
func (s *ClarificationService) EvaluateClarity(
	ctx context.Context,
	sessionID uuid.UUID,
) (score int, err error) {
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

	// 2. Validate session is in Clarification stage
	if session.GetCurrentStage() != valueobject.StageClarification {
		return 0, litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected Clarification", session.GetCurrentStage()),
		)
	}

	// 3. Evaluate clarity
	if err := s.evaluateClarity(ctx, session); err != nil {
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpClarityEvaluate, startTime, false, err.Error())
		return 0, err
	}

	// 4. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 5. Record audit log
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpClarityEvaluate, startTime, true, "")

	// 6. Publish events
	s.publishEvents(ctx, session)

	return session.Clarification.GetClarityScore(), nil
}

// GetClarityStatus returns the current clarity evaluation status.
func (s *ClarificationService) GetClarityStatus(
	ctx context.Context,
	sessionID uuid.UUID,
) (*ClarityStatus, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	status := &ClarityStatus{
		SessionID:         sessionID,
		CurrentStage:      string(session.GetCurrentStage()),
		HasPendingQuestions: session.Clarification.HasPendingQuestions(),
		PendingQuestions:  session.Clarification.PendingQuestions,
		ConfirmedPoints:   session.Clarification.ConfirmedPoints,
		ClarityScore:      session.Clarification.GetClarityScore(),
		ClarityGrade:      session.Clarification.ClarityDimensions.Grade(),
		CanComplete:       session.CanCompleteClarification(s.config.Clarity.Threshold),
		Threshold:         s.config.Clarity.Threshold,
	}

	return status, nil
}

// ClarityStatus represents the current status of clarification.
type ClarityStatus struct {
	SessionID          uuid.UUID `json:"sessionId"`
	CurrentStage       string    `json:"currentStage"`
	HasPendingQuestions bool     `json:"hasPendingQuestions"`
	PendingQuestions   []string  `json:"pendingQuestions"`
	ConfirmedPoints    []string  `json:"confirmedPoints"`
	ClarityScore       int       `json:"clarityScore"`
	ClarityGrade       string    `json:"clarityGrade"`
	CanComplete        bool      `json:"canComplete"`
	Threshold          int       `json:"threshold"`
}

// --- Internal Helper Methods ---

// buildClarificationPrompt builds the prompt for Agent to generate questions.
func (s *ClarificationService) buildClarificationPrompt(session *aggregate.WorkSession) string {
	return fmt.Sprintf(`Please analyze the following GitHub issue and generate clarification questions to understand the requirements better.

Issue Title: %s

Issue Body:
%s

Requirements:
1. Identify any unclear or ambiguous aspects of the requirements
2. Ask specific questions to clarify scope, constraints, and expected behavior
3. Focus on technical details, edge cases, and implementation considerations
4. Generate 3-5 questions that will help create a clear design

Output format: Provide questions as a numbered list, one question per line.
Example:
1. What is the expected behavior when X happens?
2. Are there any specific constraints on Y?
3. How should Z be handled in edge cases?`,
		session.Issue.Title,
		session.Issue.Body,
	)
}

// parseQuestionsFromResponse parses questions from Agent response.
func (s *ClarificationService) parseQuestionsFromResponse(response *service.AgentResponse) []string {
	if response == nil || response.Output == "" {
		return nil
	}

	// Parse numbered list format
	lines := strings.Split(response.Output, "\n")
	questions := []string{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for numbered list format (1. Question, 2. Question, etc.)
		if strings.HasPrefix(line, "1.") ||
			strings.HasPrefix(line, "2.") ||
			strings.HasPrefix(line, "3.") ||
			strings.HasPrefix(line, "4.") ||
			strings.HasPrefix(line, "5.") ||
			strings.HasPrefix(line, "6.") ||
			strings.HasPrefix(line, "7.") ||
			strings.HasPrefix(line, "8.") ||
			strings.HasPrefix(line, "9.") {
			// Remove the number prefix
			question := strings.TrimSpace(line[2:])
			if question != "" {
				questions = append(questions, question)
			}
		}
	}

	return questions
}

// postQuestionsToIssue posts questions as a comment on the GitHub issue.
func (s *ClarificationService) postQuestionsToIssue(
	ctx context.Context,
	session *aggregate.WorkSession,
	questions []string,
) error {
	// Build comment body
	commentBody := "## 澄清问题\n\n"
	commentBody += "为了更好地理解需求，请回答以下问题：\n\n"

	for i, q := range questions {
		commentBody += fmt.Sprintf("%d. %s\n\n", i+1, q)
	}

	commentBody += "\n---\n"
	commentBody += "*请直接回复每个问题，或使用 `@bot start_design` 强制进入设计阶段（需先回答所有问题）。*"

	// Post comment
	owner := utils.ExtractOwner(session.Issue.Repository)
	repo := utils.ExtractRepo(session.Issue.Repository)

	if _, err := s.ghIssueService.CreateComment(ctx, owner, repo, session.Issue.Number, commentBody); err != nil {
		return err
	}

	return nil
}

// extractConfirmedPoints extracts confirmed requirement points from answer.
func (s *ClarificationService) extractConfirmedPoints(question, answer string) []string {
	// Simple extraction: treat the answer as a confirmed point if it's substantive
	// More sophisticated parsing could be added later

	answer = strings.TrimSpace(answer)
	if answer == "" || strings.ToLower(answer) == "n/a" ||
		strings.ToLower(answer) == "no" || strings.ToLower(answer) == "none" {
		return nil
	}

	// For now, just return the answer as a confirmed point
	// In production, this could use NLP or Agent to extract structured points
	return []string{fmt.Sprintf("Q: %s -> A: %s", question, answer)}
}

// evaluateClarity evaluates clarity dimensions using Agent.
func (s *ClarificationService) evaluateClarity(ctx context.Context, session *aggregate.WorkSession) error {
	// Build prompt for clarity evaluation
	prompt := s.buildClarityEvaluationPrompt(session)

	agentReq := &service.AgentRequest{
		SessionID: session.ID,
		Stage:     service.AgentStageClarification,
		Prompt:    prompt,
		Context: &service.AgentContext{
			IssueTitle:      session.Issue.Title,
			IssueBody:       session.Issue.Body,
			ClarifiedPoints: session.Clarification.ConfirmedPoints,
		},
		Timeout: s.parseTimeout(s.config.Failure.Timeout.ClarificationAgent),
	}

	response, err := s.agentRunner.Execute(ctx, agentReq)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrAgentExecutionFail, err)
	}

	// Parse clarity dimensions from response
	dimensions, err := s.parseClarityDimensions(response)
	if err != nil {
		s.logger.Warn("failed to parse clarity dimensions, using default",
			zap.String("session_id", session.ID.String()),
			zap.Error(err),
		)
		// Use default dimensions based on confirmed points
		dimensions = s.defaultClarityDimensions(session)
	}

	session.SetClarityDimensions(dimensions)

	return nil
}

// buildClarityEvaluationPrompt builds the prompt for clarity evaluation.
func (s *ClarificationService) buildClarityEvaluationPrompt(session *aggregate.WorkSession) string {
	confirmedPointsStr := ""
	for i, point := range session.Clarification.ConfirmedPoints {
		confirmedPointsStr += fmt.Sprintf("%d. %s\n", i+1, point)
	}

	return fmt.Sprintf(`Please evaluate the clarity of the following requirements based on the issue and confirmed points.

Issue Title: %s

Issue Body:
%s

Confirmed Points:
%s

Evaluation Dimensions (score each 0-maxScore):
1. Completeness (0-30): Are all requirements clearly defined? Are edge cases covered?
2. Clarity (0-25): Is the language clear and unambiguous? Are technical terms defined?
3. Consistency (0-20): Are requirements internally consistent? No contradictions?
4. Feasibility (0-15): Can this be implemented with current tech stack? No impossible requirements?
5. Testability (0-10): Can acceptance criteria be tested? Clear success criteria?

Output format (JSON):
{
  "completeness": {"score": <0-30>, "checks": {"<check_name>": {"score": <points>, "passed": <true/false>, "detail": "<reason>"}}},
  "clarity": {"score": <0-25>, "checks": {...}},
  "consistency": {"score": <0-20>, "checks": {...}},
  "feasibility": {"score": <0-15>, "checks": {...}},
  "testability": {"score": <0-10>, "checks": {...}}
}`,
		session.Issue.Title,
		session.Issue.Body,
		confirmedPointsStr,
	)
}

// parseClarityDimensions parses clarity dimensions from Agent response.
func (s *ClarificationService) parseClarityDimensions(response *service.AgentResponse) (valueobject.ClarityDimensions, error) {
	if response == nil || response.Output == "" {
		return valueobject.ClarityDimensions{}, fmt.Errorf("empty response")
	}

	// Try to parse JSON from response
	output := response.Output

	// Find JSON block in response
	jsonStart := strings.Index(output, "{")
	jsonEnd := strings.LastIndex(output, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return valueobject.ClarityDimensions{}, fmt.Errorf("no JSON found in response")
	}

	jsonStr := output[jsonStart : jsonEnd+1]

	// Parse into intermediate struct
	var evalResult struct {
		Completeness struct {
			Score  int              `json:"score"`
			Checks map[string]struct {
				Score  int    `json:"score"`
				Passed bool   `json:"passed"`
				Detail string `json:"detail"`
			} `json:"checks"`
		} `json:"completeness"`
		Clarity struct {
			Score  int              `json:"score"`
			Checks map[string]struct {
				Score  int    `json:"score"`
				Passed bool   `json:"passed"`
				Detail string `json:"detail"`
			} `json:"checks"`
		} `json:"clarity"`
		Consistency struct {
			Score  int              `json:"score"`
			Checks map[string]struct {
				Score  int    `json:"score"`
				Passed bool   `json:"passed"`
				Detail string `json:"detail"`
			} `json:"checks"`
		} `json:"consistency"`
		Feasibility struct {
			Score  int              `json:"score"`
			Checks map[string]struct {
				Score  int    `json:"score"`
				Passed bool   `json:"passed"`
				Detail string `json:"detail"`
			} `json:"checks"`
		} `json:"feasibility"`
		Testability struct {
			Score  int              `json:"score"`
			Checks map[string]struct {
				Score  int    `json:"score"`
				Passed bool   `json:"passed"`
				Detail string `json:"detail"`
			} `json:"checks"`
		} `json:"testability"`
	}

	if err := parseJSON(jsonStr, &evalResult); err != nil {
		return valueobject.ClarityDimensions{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Build ClarityDimensions
	dimensions, err := valueobject.NewClarityDimensions(
		evalResult.Completeness.Score,
		evalResult.Clarity.Score,
		evalResult.Consistency.Score,
		evalResult.Feasibility.Score,
		evalResult.Testability.Score,
	)
	if err != nil {
		return valueobject.ClarityDimensions{}, err
	}

	// Set checks for each dimension
	s.setChecksFromEval(&dimensions, "completeness", evalResult.Completeness.Checks)
	s.setChecksFromEval(&dimensions, "clarity", evalResult.Clarity.Checks)
	s.setChecksFromEval(&dimensions, "consistency", evalResult.Consistency.Checks)
	s.setChecksFromEval(&dimensions, "feasibility", evalResult.Feasibility.Checks)
	s.setChecksFromEval(&dimensions, "testability", evalResult.Testability.Checks)

	return dimensions, nil
}

// setChecksFromEval sets checks for a dimension from evaluation result.
func (s *ClarificationService) setChecksFromEval(
	dimensions *valueobject.ClarityDimensions,
	dimensionName string,
	checks map[string]struct {
		Score  int    `json:"score"`
		Passed bool   `json:"passed"`
		Detail string `json:"detail"`
	},
) {
	for checkName, check := range checks {
		dimensions.SetCheck(dimensionName, checkName, check.Score, check.Passed, check.Detail)
	}
}

// defaultClarityDimensions returns default dimensions based on session state.
// If dimension calculation fails, returns safe default values.
func (s *ClarificationService) defaultClarityDimensions(session *aggregate.WorkSession) valueobject.ClarityDimensions {
	// Calculate base score from confirmed points count
	pointsCount := len(session.Clarification.ConfirmedPoints)

	// More confirmed points = higher clarity
	// Use min to ensure scores don't exceed max values
	baseCompleteness := min(30, pointsCount * 6) // 0-30 (max: 30)
	baseClarity := min(25, pointsCount * 5)      // 0-25 (max: 25)
	baseConsistency := 20                        // Assume consistent if no contradictions found (max: 20)
	baseFeasibility := 15                        // Assume feasible (max: 15)
	baseTestability := min(10, pointsCount * 2)  // 0-10 (max: 10)

	dimensions, err := valueobject.NewClarityDimensions(
		baseCompleteness,
		baseClarity,
		baseConsistency,
		baseFeasibility,
		baseTestability,
	)
	if err != nil {
		s.logger.Warn("failed to create clarity dimensions, using safe defaults",
			zap.Int("points_count", pointsCount),
			zap.Error(err),
		)
		// Return safe default dimensions that are guaranteed to be valid
		dimensions, _ = valueobject.NewClarityDimensions(20, 15, 10, 8, 5)
	}

	return dimensions
}

// checkActorPermission checks if the actor has permission to answer questions.
func (s *ClarificationService) checkActorPermission(
	ctx context.Context,
	session *aggregate.WorkSession,
	actor string,
) (valueobject.ActorRole, error) {
	// Issue author can always answer
	if session.Issue.Author == actor {
		return valueobject.ActorRoleIssueAuthor, nil
	}

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

	return "", litchierrors.New(litchierrors.ErrPermissionDenied).WithDetail(
		fmt.Sprintf("actor %s is not authorized (must be issue author or admin)", actor),
	)
}

// recordAuditLog records an audit log entry.
func (s *ClarificationService) recordAuditLog(
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
		"clarification",
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
func (s *ClarificationService) publishEvents(ctx context.Context, session *aggregate.WorkSession) {
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
func (s *ClarificationService) parseTimeout(timeoutStr string) time.Duration {
	if timeoutStr == "" {
		return 5 * time.Minute // Default
	}
	d, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 5 * time.Minute
	}
	return d
}

// parseJSON is a helper to parse JSON string.
func parseJSON(jsonStr string, target any) error {
	return json.Unmarshal([]byte(jsonStr), target)
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
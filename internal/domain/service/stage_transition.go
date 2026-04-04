package service

import (
	"fmt"

	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// DefaultStageTransitionService provides default implementation of StageTransitionService.
// It validates transition preconditions using domain rules and configuration context.
type DefaultStageTransitionService struct {
	scheduler TaskScheduler
}

// NewDefaultStageTransitionService creates a new DefaultStageTransitionService instance.
// If scheduler is nil, a default TaskScheduler will be used.
func NewDefaultStageTransitionService(scheduler TaskScheduler) *DefaultStageTransitionService {
	if scheduler == nil {
		scheduler = NewDefaultTaskScheduler()
	}
	return &DefaultStageTransitionService{scheduler: scheduler}
}

// CanTransition checks if forward transition is allowed.
func (s *DefaultStageTransitionService) CanTransition(
	session *aggregate.WorkSession,
	target valueobject.Stage,
	ctx TransitionContext,
) bool {
	return s.GetTransitionError(session, target, ctx) == nil
}

// GetTransitionError returns detailed error if transition cannot proceed.
func (s *DefaultStageTransitionService) GetTransitionError(
	session *aggregate.WorkSession,
	target valueobject.Stage,
	ctx TransitionContext,
) error {
	// Check basic transition rules (stage sequence)
	if !session.CanTransitionTo(target) {
		return errors.New(errors.ErrInvalidStageTransition).WithDetail(
			fmt.Sprintf("invalid transition from %s to %s (must be sequential)", session.GetCurrentStage(), target),
		)
	}

	// Check stage-specific preconditions
	return s.ValidateTransitionPreconditions(session, target, ctx)
}

// CanRollback checks if rollback is allowed.
func (s *DefaultStageTransitionService) CanRollback(
	session *aggregate.WorkSession,
	target valueobject.Stage,
	ctx TransitionContext,
) bool {
	return s.GetRollbackError(session, target, ctx) == nil
}

// GetRollbackError returns detailed error if rollback cannot proceed.
func (s *DefaultStageTransitionService) GetRollbackError(
	session *aggregate.WorkSession,
	target valueobject.Stage,
	ctx TransitionContext,
) error {
	// Check basic rollback rules
	if !session.CanRollbackTo(target) {
		return errors.New(errors.ErrInvalidStageTransition).WithDetail(
			fmt.Sprintf("invalid rollback from %s to %s", session.GetCurrentStage(), target),
		)
	}

	// Check rollback-specific preconditions
	return s.ValidateRollbackPreconditions(session, target, ctx)
}

// ValidateTransitionPreconditions validates stage-specific preconditions for forward transition.
func (s *DefaultStageTransitionService) ValidateTransitionPreconditions(
	session *aggregate.WorkSession,
	target valueobject.Stage,
	ctx TransitionContext,
) error {
	switch target {
	case valueobject.StageDesign:
		return s.validateClarificationToDesign(session, ctx)

	case valueobject.StageTaskBreakdown:
		return s.validateDesignToTaskBreakdown(session, ctx)

	case valueobject.StageExecution:
		return s.validateTaskBreakdownToExecution(session)

	case valueobject.StagePullRequest:
		return s.validateExecutionToPullRequest(session)

	case valueobject.StageCompleted:
		return s.validatePullRequestToCompleted(session)
	}

	return nil
}

// ValidateRollbackPreconditions validates rollback-specific preconditions.
func (s *DefaultStageTransitionService) ValidateRollbackPreconditions(
	session *aggregate.WorkSession,
	target valueobject.Stage,
	ctx TransitionContext,
) error {
	currentStage := session.GetCurrentStage()

	// PR stage specific rollback constraints
	if currentStage == valueobject.StagePullRequest {
		// Check if PR rollback is allowed
		if !ctx.AllowPRRollback {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"PR stage rollback is disabled by configuration",
			)
		}

		// Check PR rollback count limit
		if session.PRRollbackCount >= ctx.MaxPRRollbackCount {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				fmt.Sprintf("PR rollback count (%d) exceeds maximum limit (%d)",
					session.PRRollbackCount, ctx.MaxPRRollbackCount),
			)
		}
	}

	// Clarification stage cannot rollback
	if currentStage == valueobject.StageClarification {
		return errors.New(errors.ErrInvalidStageTransition).WithDetail(
			"cannot rollback from clarification stage",
		)
	}

	// Completed stage cannot rollback
	if currentStage == valueobject.StageCompleted {
		return errors.New(errors.ErrInvalidStageTransition).WithDetail(
			"cannot rollback from completed stage",
		)
	}

	return nil
}

// GetAllowedRollbackTargets returns all valid rollback targets.
func (s *DefaultStageTransitionService) GetAllowedRollbackTargets(
	session *aggregate.WorkSession,
	ctx TransitionContext,
) []valueobject.Stage {
	currentStage := session.GetCurrentStage()
	targets := []valueobject.Stage{}

	// Check each possible rollback target
	for _, stage := range valueobject.AllStages() {
		// Skip current and later stages
		if valueobject.StageOrder(stage) >= valueobject.StageOrder(currentStage) {
			continue
		}

		if s.CanRollback(session, stage, ctx) {
			targets = append(targets, stage)
		}
	}

	return targets
}

// --- Stage-specific validation methods ---

// validateClarificationToDesign validates transition from Clarification to Design.
func (s *DefaultStageTransitionService) validateClarificationToDesign(
	session *aggregate.WorkSession,
	ctx TransitionContext,
) error {
	clarification := session.Clarification
	if clarification == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"clarification not initialized",
		)
	}

	// Must have no pending questions
	if clarification.HasPendingQuestions() {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"cannot proceed to design: pending questions must be answered",
		)
	}

	// Must have at least one confirmed point
	if len(clarification.ConfirmedPoints) == 0 {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"cannot proceed to design: at least one requirement point must be confirmed",
		)
	}

	// Check clarity score threshold
	clarityScore := clarification.GetClarityScore()
	if clarityScore < ctx.ClarityThreshold {
		// Low clarity requires confirmation
		// Note: This check is informational; the actual confirmation
		// is handled at the application layer based on user response
		return errors.New(errors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("clarity score (%d) below threshold (%d), requires manual confirmation",
				clarityScore, ctx.ClarityThreshold),
		)
	}

	return nil
}

// validateDesignToTaskBreakdown validates transition from Design to TaskBreakdown.
func (s *DefaultStageTransitionService) validateDesignToTaskBreakdown(
	session *aggregate.WorkSession,
	ctx TransitionContext,
) error {
	design := session.GetDesign()
	if design == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"design not initialized",
		)
	}

	// If force confirm is enabled, design must be confirmed
	if ctx.ForceDesignConfirm && !design.IsConfirmed() {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"design must be confirmed (force confirm enabled)",
		)
	}

	// If confirmation is required (based on complexity), check status
	if design.NeedsConfirmation() && !design.IsConfirmed() {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("design requires confirmation (complexity: %s)", design.ComplexityScore.DisplayName()),
		)
	}

	return nil
}

// validateTaskBreakdownToExecution validates transition from TaskBreakdown to Execution.
func (s *DefaultStageTransitionService) validateTaskBreakdownToExecution(
	session *aggregate.WorkSession,
) error {
	tasks := session.GetTasks()
	if len(tasks) == 0 {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"cannot proceed to execution: no tasks defined",
		)
	}

	// Validate task dependencies using injected scheduler
	if err := s.scheduler.ValidateDependencies(tasks); err != nil {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"task dependencies validation failed: " + err.Error(),
		)
	}

	return nil
}

// validateExecutionToPullRequest validates transition from Execution to PullRequest.
func (s *DefaultStageTransitionService) validateExecutionToPullRequest(
	session *aggregate.WorkSession,
) error {
	// All tasks must be completed or skipped
	if !session.AreAllTasksCompleted() {
		failedTask := session.GetFailedTask()
		if failedTask != nil {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				fmt.Sprintf("cannot create PR: task %s failed (%s)",
					failedTask.TaskID, failedTask.Reason),
			)
		}
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"cannot create PR: some tasks are not completed",
		)
	}

	return nil
}

// validatePullRequestToCompleted validates transition from PullRequest to Completed.
func (s *DefaultStageTransitionService) validatePullRequestToCompleted(
	session *aggregate.WorkSession,
) error {
	// PR must be created
	if session.GetPRNumber() == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"cannot complete: PR not created",
		)
	}

	return nil
}
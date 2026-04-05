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

// EvaluateTransition evaluates transition decision based on clarity score rules.
func (s *DefaultStageTransitionService) EvaluateTransition(
	session *aggregate.WorkSession,
	target valueobject.Stage,
	ctx TransitionContext,
) TransitionResult {
	// 1. Check basic transition rules (stage sequence)
	if !session.CanTransitionTo(target) {
		return TransitionResult{
			Decision: DecisionDenied,
			Reason:   fmt.Sprintf("invalid transition from %s to %s", session.GetCurrentStage(), target),
			CanForce: false,
		}
	}

	// 2. Evaluate based on target stage
	switch target {
	case valueobject.StageDesign:
		return s.evaluateClarificationToDesign(session, ctx)
	case valueobject.StageTaskBreakdown:
		return s.evaluateDesignToTaskBreakdown(session, ctx)
	case valueobject.StageExecution:
		return s.evaluateTaskBreakdownToExecutionResult(session)
	case valueobject.StagePullRequest:
		return s.evaluateExecutionToPullRequestResult(session)
	case valueobject.StageCompleted:
		return s.evaluatePullRequestToCompletedResult(session)
	default:
		return TransitionResult{Decision: DecisionAllow}
	}
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

// --- Evaluation methods for TransitionResult --

// evaluateClarificationToDesign evaluates transition from Clarification to Design.
// This is the core method implementing clarity score-based decision logic.
func (s *DefaultStageTransitionService) evaluateClarificationToDesign(
	session *aggregate.WorkSession,
	ctx TransitionContext,
) TransitionResult {
	clarification := session.Clarification
	if clarification == nil {
		return TransitionResult{
			Decision: DecisionDenied,
			Reason:   "clarification not initialized",
			CanForce: false,
		}
	}

	// Check pending questions (hard constraint, cannot force)
	if clarification.HasPendingQuestions() {
		return TransitionResult{
			Decision:       DecisionDenied,
			Reason:         "pending questions must be answered",
			RequiredAction: "Please answer all pending clarification questions",
			CanForce:       false,
		}
	}

	// Check confirmed points (hard constraint, cannot force)
	if len(clarification.ConfirmedPoints) == 0 {
		return TransitionResult{
			Decision:       DecisionDenied,
			Reason:         "at least one requirement point must be confirmed",
			RequiredAction: "Please confirm at least one requirement point",
			CanForce:       false,
		}
	}

	clarityScore := clarification.GetClarityScore()

	// User command "start_design" bypasses clarity check
	if ctx.SkipClarityCheck {
		return TransitionResult{
			Decision:     DecisionAllow,
			Reason:       "user command bypasses clarity check",
			ClarityScore: clarityScore,
			CanForce:     false, // Already allowed, no need to force
			SkipClarity:  true,  // Indicates user used force command
		}
	}

	// Evaluate based on clarity score thresholds
	if clarityScore >= ctx.AutoProceedThreshold {
		// >= 80: Auto proceed without confirmation
		return TransitionResult{
			Decision:     DecisionAllow,
			Reason:       "high clarity score, auto proceed without confirmation",
			ClarityScore: clarityScore,
			CanForce:     false, // Already allowed, no need to force
		}
	} else if clarityScore >= ctx.ClarityThreshold {
		// 60-79: Auto proceed, but design needs confirmation
		return TransitionResult{
			Decision:       DecisionAllow,
			Reason:         "medium clarity score, auto proceed but design confirmation required",
			ClarityScore:   clarityScore,
			RequiredAction: "Design needs manual confirmation",
			CanForce:       false,
		}
	} else if clarityScore >= ctx.ForceClarifyThreshold {
		// 40-59: Need user confirmation
		return TransitionResult{
			Decision:       DecisionNeedConfirmation,
			Reason:         fmt.Sprintf("clarity score %d below threshold %d, needs confirmation", clarityScore, ctx.ClarityThreshold),
			ClarityScore:   clarityScore,
			RequiredAction: "Clarity score is low, please confirm to start design (reply 'start_design' to confirm)",
			CanForce:       true,
		}
	} else {
		// < 40: Denied, must continue clarification (but can force)
		return TransitionResult{
			Decision:       DecisionDenied,
			Reason:         fmt.Sprintf("clarity score %d too low, must continue clarification", clarityScore),
			ClarityScore:   clarityScore,
			RequiredAction: "Clarity score is too low, please continue clarifying requirements",
			CanForce:       true, // Allow user to force proceed with "start_design"
		}
	}
}

// evaluateDesignToTaskBreakdown evaluates transition from Design to TaskBreakdown.
func (s *DefaultStageTransitionService) evaluateDesignToTaskBreakdown(
	session *aggregate.WorkSession,
	ctx TransitionContext,
) TransitionResult {
	design := session.GetDesign()
	if design == nil {
		return TransitionResult{
			Decision: DecisionDenied,
			Reason:   "design not initialized",
			CanForce: false,
		}
	}

	// Check if design has content
	if design.CurrentVersion == 0 || len(design.Versions) == 0 {
		return TransitionResult{
			Decision:       DecisionDenied,
			Reason:         "design has no versions",
			RequiredAction: "Please create a design",
			CanForce:       false,
		}
	}

	// Check confirmation requirements
	if ctx.ForceDesignConfirm && !design.IsConfirmed() {
		return TransitionResult{
			Decision:       DecisionNeedConfirmation,
			Reason:         "design must be confirmed (force confirm enabled)",
			RequiredAction: "Please confirm the design",
			CanForce:       false,
		}
	}

	if design.NeedsConfirmation() && !design.IsConfirmed() {
		return TransitionResult{
			Decision:       DecisionNeedConfirmation,
			Reason:         fmt.Sprintf("design requires confirmation (complexity: %s)", design.ComplexityScore.DisplayName()),
			RequiredAction: "Please confirm the design",
			CanForce:       false,
		}
	}

	return TransitionResult{
		Decision: DecisionAllow,
		Reason:   "design confirmed or no confirmation required",
	}
}

// evaluateTaskBreakdownToExecutionResult evaluates transition from TaskBreakdown to Execution.
func (s *DefaultStageTransitionService) evaluateTaskBreakdownToExecutionResult(
	session *aggregate.WorkSession,
) TransitionResult {
	tasks := session.GetTasks()
	if len(tasks) == 0 {
		return TransitionResult{
			Decision:       DecisionDenied,
			Reason:         "no tasks defined",
			RequiredAction: "Please wait for task breakdown to complete",
			CanForce:       false,
		}
	}

	// Validate task dependencies
	if err := s.scheduler.ValidateDependencies(tasks); err != nil {
		return TransitionResult{
			Decision:       DecisionDenied,
			Reason:         "task dependencies validation failed: " + err.Error(),
			RequiredAction: "Please check task dependencies",
			CanForce:       false,
		}
	}

	return TransitionResult{
		Decision: DecisionAllow,
		Reason:   "tasks defined and dependencies valid",
	}
}

// evaluateExecutionToPullRequestResult evaluates transition from Execution to PullRequest.
func (s *DefaultStageTransitionService) evaluateExecutionToPullRequestResult(
	session *aggregate.WorkSession,
) TransitionResult {
	// All tasks must be completed or skipped
	if !session.AreAllTasksCompleted() {
		failedTask := session.GetFailedTask()
		if failedTask != nil {
			return TransitionResult{
				Decision:       DecisionDenied,
				Reason:         fmt.Sprintf("task %s failed (%s)", failedTask.TaskID, failedTask.Reason),
				RequiredAction: "Please handle failed tasks",
				CanForce:       false,
			}
		}
		return TransitionResult{
			Decision:       DecisionDenied,
			Reason:         "some tasks are not completed",
			RequiredAction: "Please wait for all tasks to complete",
			CanForce:       false,
		}
	}

	return TransitionResult{
		Decision: DecisionAllow,
		Reason:   "all tasks completed or skipped",
	}
}

// evaluatePullRequestToCompletedResult evaluates transition from PullRequest to Completed.
func (s *DefaultStageTransitionService) evaluatePullRequestToCompletedResult(
	session *aggregate.WorkSession,
) TransitionResult {
	// PR must be created
	if session.GetPRNumber() == nil {
		return TransitionResult{
			Decision:       DecisionDenied,
			Reason:         "PR not created",
			RequiredAction: "Please wait for PR creation",
			CanForce:       false,
		}
	}

	return TransitionResult{
		Decision: DecisionAllow,
		Reason:   "PR created and ready for completion",
	}
}

// --- Rollback Evaluation Methods (T3.1.2) ---

// EvaluateRollback evaluates rollback decision based on R1-R6 rules.
func (s *DefaultStageTransitionService) EvaluateRollback(
	session *aggregate.WorkSession,
	target valueobject.Stage,
	ctx TransitionContext,
) RollbackResult {
	currentStage := session.GetCurrentStage()

	// 1. Check basic rollback rules (stage sequence)
	if !session.CanRollbackTo(target) {
		return RollbackResult{
			Decision:         RollbackDenied,
			Reason:           fmt.Sprintf("invalid rollback from %s to %s", currentStage, target),
			PreconditionsMet: false,
		}
	}

	// 2. Check rollback-specific conditions
	if err := s.ValidateRollbackConditions(session, target, ctx); err != nil {
		return RollbackResult{
			Decision:         RollbackDenied,
			Reason:           err.Error(),
			PreconditionsMet: false,
		}
	}

	// 3. Determine rollback rule and type
	rule := s.GetRollbackRule(currentStage, target)
	rollbackType := s.determineRollbackType(rule, currentStage, target)

	// 4. Build rollback result with context
	result := RollbackResult{
		Decision:         RollbackAllowed,
		RollbackType:     rollbackType,
		RollbackRule:     rule,
		PreconditionsMet: true,
	}

	// 5. Set rollback effects based on rule
	s.populateRollbackEffects(&result, currentStage, target, session)

	return result
}

// GetRollbackRule returns the rollback rule identifier (R1-R6).
func (s *DefaultStageTransitionService) GetRollbackRule(
	currentStage, targetStage valueobject.Stage,
) string {
	switch currentStage {
	case valueobject.StageExecution:
		if targetStage == valueobject.StageDesign {
			return "R1" // Execution -> Design
		}
		if targetStage == valueobject.StageClarification {
			return "R3" // Execution -> Clarification
		}
	case valueobject.StageDesign:
		if targetStage == valueobject.StageClarification {
			return "R2" // Design -> Clarification
		}
	case valueobject.StageTaskBreakdown:
		// TaskBreakdown -> Clarification follows R2 rules (same as Design -> Clarification)
		if targetStage == valueobject.StageClarification {
			return "R2"
		}
		// TaskBreakdown -> Design is not a standard rollback rule (not defined in state-machine.md)
		// Per state-machine.md line 224: "保留任务列表，等待设计更新后重新拆解"
		// We return a special identifier for this non-standard path
		if targetStage == valueobject.StageDesign {
			return "R2b" // Non-standard: TaskBreakdown -> Design (keep tasks)
		}
	case valueobject.StagePullRequest:
		if targetStage == valueobject.StageExecution {
			return "R4" // PR -> Execution (shallow)
		}
		if targetStage == valueobject.StageDesign {
			return "R5" // PR -> Design (deep)
		}
		if targetStage == valueobject.StageClarification {
			return "R6" // PR -> Clarification (full)
		}
	}
	return ""
}

// ValidateRollbackConditions validates detailed rollback conditions.
// This checks conditions beyond basic stage sequence rules.
func (s *DefaultStageTransitionService) ValidateRollbackConditions(
	session *aggregate.WorkSession,
	target valueobject.Stage,
	ctx TransitionContext,
) error {
	currentStage := session.GetCurrentStage()

	// PR stage specific conditions (R4, R5, R6)
	if currentStage == valueobject.StagePullRequest {
		// Check if PR rollback is enabled
		if !ctx.AllowPRRollback {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"PR stage rollback is disabled by configuration",
			)
		}

		// Check rollback count limit
		if session.PRRollbackCount >= ctx.MaxPRRollbackCount {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				fmt.Sprintf("PR rollback count (%d) exceeds maximum (%d)",
					session.PRRollbackCount, ctx.MaxPRRollbackCount),
			)
		}

		// PR must exist for rollback
		if session.PRNumber == nil {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"cannot rollback: PR not created",
			)
		}

		// Note: PR status (open/merged/closed) check requires GitHub API call
		// This is handled at application layer. Domain layer assumes PR is open
		// if PRNumber is set and session is not completed.
	}

	// Completed stage cannot rollback (already checked by CanRollbackTo)
	if currentStage == valueobject.StageCompleted {
		return errors.New(errors.ErrInvalidStageTransition).WithDetail(
			"cannot rollback from completed stage",
		)
	}

	// Clarification stage cannot rollback (already checked by CanRollbackTo)
	if currentStage == valueobject.StageClarification {
		return errors.New(errors.ErrInvalidStageTransition).WithDetail(
			"cannot rollback from clarification stage",
		)
	}

	return nil
}

// determineRollbackType determines the rollback depth type based on the rollback rule.
// Rollback depth is defined per rule in state-machine.md:
//   - R1 (Execution -> Design): Deep (design version +1, branch deprecated)
//   - R2 (Design -> Clarification): Shallow (keep requirements, clear design)
//   - R3 (Execution -> Clarification): Full (clear design, deprecate branch)
//   - R4 (PR -> Execution): Shallow (PR remains open, branch preserved)
//   - R5 (PR -> Design): Deep (PR closed, branch deprecated)
//   - R6 (PR -> Clarification): Full (PR closed, all cleared)
func (s *DefaultStageTransitionService) determineRollbackType(
	rollbackRule string,
	currentStage, targetStage valueobject.Stage,
) RollbackType {
	// Map rollback rules directly to types as per design document
	switch rollbackRule {
	case "R1", "R5":
		return RollbackTypeDeep
	case "R2", "R4", "R2b":
		return RollbackTypeShallow
	case "R3", "R6":
		return RollbackTypeFull
	}

	// Fallback: for non-standard rollback paths, use stage order distance.
	// Note: Non-standard paths should not occur in normal operation as
	// GetRollbackRule returns empty string for undefined paths and
	// EvaluateRollback denies such rollbacks. This fallback is provided
	// for defensive programming only.
	//
	// The RollbackType returned here is primarily for logging/UI display.
	// The actual rollback effects (WillDeprecateBranch, WillClearTasks, etc.)
	// are calculated independently in populateRollbackEffects based on
	// stage order, ensuring correct behavior regardless of RollbackType.
	currentOrder := valueobject.StageOrder(currentStage)
	targetOrder := valueobject.StageOrder(targetStage)

	diff := currentOrder - targetOrder

	if diff == 1 {
		return RollbackTypeShallow
	} else if diff == 2 {
		return RollbackTypeDeep
	}
	return RollbackTypeFull
}

// populateRollbackEffects sets rollback effect flags based on the rollback rule.
func (s *DefaultStageTransitionService) populateRollbackEffects(
	result *RollbackResult,
	currentStage, targetStage valueobject.Stage,
	session *aggregate.WorkSession,
) {
	targetOrder := valueobject.StageOrder(targetStage)
	designOrder := valueobject.StageOrder(valueobject.StageDesign)
	clarificationOrder := valueobject.StageOrder(valueobject.StageClarification)

	// Branch deprecation: for rollbacks to Design or earlier
	result.WillDeprecateBranch = targetOrder <= designOrder

	// PR closure: for rollbacks from PR stage to Design or earlier
	result.WillClosePR = currentStage == valueobject.StagePullRequest &&
		targetOrder <= designOrder

	// Task clearing: for rollbacks to Design or earlier
	// Exception: R2b (TaskBreakdown -> Design) keeps tasks per design document
	result.WillClearTasks = targetOrder <= designOrder && result.RollbackRule != "R2b"

	// Design clearing: for rollbacks to Clarification
	result.WillClearDesign = targetOrder <= clarificationOrder

	// PR rollback count: for any rollback from PR stage
	result.WillIncrementPRRollbackCount = currentStage == valueobject.StagePullRequest

	// Recovery actions based on rollback rule
	switch result.RollbackRule {
	case "R1":
		result.RecoveryActions = []string{
			"Design version will be incremented",
			"New design content must be created",
			"Tasks will be regenerated after design confirmation",
		}
	case "R2":
		result.RecoveryActions = []string{
			"Clarification questions can be added",
			"Design must be recreated after clarification",
		}
	case "R2b":
		// TaskBreakdown -> Design: non-standard path, keeps tasks
		result.RecoveryActions = []string{
			"Design can be updated",
			"Task list is preserved for re-breakdown",
			"Re-confirm design to proceed with task breakdown",
		}
	case "R3":
		result.RecoveryActions = []string{
			"Branch marked as deprecated",
			"New branch required for re-execution",
			"Design must be recreated",
		}
	case "R4":
		result.RecoveryActions = []string{
			"Fix tasks can be added to current branch",
			"PR remains open",
			"Continue with fixes and re-submit",
		}
	case "R5":
		result.RecoveryActions = []string{
			"PR will be closed",
			"Branch marked as deprecated",
			"New design version required",
			"New branch required for implementation",
		}
	case "R6":
		result.RecoveryActions = []string{
			"PR will be closed",
			"Branch marked as deprecated",
			"Clarification must be re-done",
			"New branch required after re-clarification",
		}
	}
}

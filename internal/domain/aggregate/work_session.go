package aggregate

import (
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// SessionStatus represents the operational status of a WorkSession.
type SessionStatus string

const (
	SessionStatusActive     SessionStatus = "active"     // Session is actively processing
	SessionStatusPaused     SessionStatus = "paused"     // Session is paused (user-initiated)
	SessionStatusCompleted  SessionStatus = "completed"  // Session completed successfully
	SessionStatusTerminated SessionStatus = "terminated" // Session terminated by user
)

// IsValid checks if the session status is valid.
func (ss SessionStatus) IsValid() bool {
	return ss == SessionStatusActive || ss == SessionStatusPaused ||
		ss == SessionStatusCompleted || ss == SessionStatusTerminated
}

// IsTerminal checks if the session is in a terminal state.
// Completed and Terminated are terminal states where no further operations are allowed.
func (ss SessionStatus) IsTerminal() bool {
	return ss == SessionStatusCompleted || ss == SessionStatusTerminated
}

// CanPause checks if the session can be paused.
// Only Active sessions can be paused, as per state machine design.
// Paused sessions must be resumed first before being paused again.
func (ss SessionStatus) CanPause() bool {
	return ss == SessionStatusActive
}

// CanResume checks if the session can be resumed.
// Only Paused sessions can be resumed.
func (ss SessionStatus) CanResume() bool {
	return ss == SessionStatusPaused
}

// CanTerminate checks if the session can be terminated.
// Session can be terminated from Active or Paused status.
// This aligns with the state machine design: terminate can be invoked
// from any non-terminal stage (Clarification through PullRequest)
// or from Paused status. Terminal states (Completed, Terminated) cannot be terminated again.
func (ss SessionStatus) CanTerminate() bool {
	return ss == SessionStatusActive || ss == SessionStatusPaused
}

// WorkSession is the core aggregate root for the Litchi domain.
// It coordinates Issue, Clarification, Design, Tasks, and Execution
// throughout the workflow stages: Clarification → Design → TaskBreakdown → Execution → PullRequest → Completed.
//
// Concurrency Safety:
// WorkSession is NOT thread-safe. All methods must be called from a single goroutine.
// The Application layer is responsible for ensuring serial access to each WorkSession instance.
// Concurrent access should be coordinated through the repository or application service layer.
type WorkSession struct {
	// Identity
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Core entities within the aggregate
	Issue         *entity.Issue         `json:"issue"`
	Clarification *entity.Clarification `json:"clarification"`
	Design        *entity.Design        `json:"design"`
	Tasks         []*entity.Task        `json:"tasks"`
	Execution     *entity.Execution     `json:"execution"`

	// State tracking
	CurrentStage    valueobject.Stage `json:"currentStage"`
	SessionStatus   SessionStatus     `json:"sessionStatus"`
	PRNumber        *int              `json:"prNumber,omitempty"` // PR number after creation
	PRRollbackCount int               `json:"prRollbackCount"`    // Number of PR stage rollbacks

	// Pause context tracking (T3.1.3)
	PauseContext *valueobject.PauseContext `json:"pauseContext,omitempty"`
	PauseHistory []valueobject.PauseRecord `json:"pauseHistory"`

	// Domain events (collected for publishing)
	events []event.DomainEvent
}

// NewWorkSession creates a new WorkSession from a GitHub issue.
// This is the factory method for creating a new session.
func NewWorkSession(issue *entity.Issue) (*WorkSession, error) {
	if err := issue.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()
	sessionID := uuid.New()

	ws := &WorkSession{
		ID:              sessionID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Issue:           issue,
		Clarification:   entity.NewClarification(),
		CurrentStage:    valueobject.StageClarification,
		SessionStatus:   SessionStatusActive,
		Tasks:           []*entity.Task{},
		PRRollbackCount: 0,
		events:          []event.DomainEvent{},
	}

	// Emit WorkSessionStarted event
	ws.recordEvent(event.NewWorkSessionStarted(
		sessionID,
		issue.Number,
		issue.Repository,
		issue.Title,
	))

	return ws, nil
}

// Validate validates the WorkSession aggregate invariants.
// Invariants:
// - One WorkSession corresponds to one Issue
// - Stage transitions must be sequential forward or allowed rollback
// - Tasks execution requires completed design
// - PR creation requires all tasks completed
func (ws *WorkSession) Validate() error {
	// Issue must be valid
	if ws.Issue == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("issue cannot be nil")
	}
	if err := ws.Issue.Validate(); err != nil {
		return err
	}

	// Stage must be valid
	if !ws.CurrentStage.IsValid() {
		return errors.New(errors.ErrInvalidStage).WithDetail(
			"current stage is invalid: " + ws.CurrentStage.String(),
		)
	}

	// Session status must be valid
	if !ws.SessionStatus.IsValid() {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"session status is invalid: " + string(ws.SessionStatus),
		)
	}

	// Design must exist after Clarification stage
	if valueobject.StageOrder(ws.CurrentStage) > valueobject.StageOrder(valueobject.StageClarification) {
		if ws.Design == nil {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"design must exist after clarification stage",
			)
		}
	}

	// Tasks must exist after TaskBreakdown stage
	if valueobject.StageOrder(ws.CurrentStage) > valueobject.StageOrder(valueobject.StageTaskBreakdown) {
		if len(ws.Tasks) == 0 {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"tasks must exist after task breakdown stage",
			)
		}
	}

	// Execution must exist after TaskBreakdown stage
	if valueobject.StageOrder(ws.CurrentStage) > valueobject.StageOrder(valueobject.StageTaskBreakdown) {
		if ws.Execution == nil {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"execution must exist after task breakdown stage",
			)
		}
	}

	// All tasks must be completed before PR creation
	if ws.CurrentStage == valueobject.StagePullRequest || ws.CurrentStage == valueobject.StageCompleted {
		if !ws.AreAllTasksCompleted() {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"all tasks must be completed before PR creation",
			)
		}
	}

	return nil
}

// --- Stage Transition Methods ---

// CanTransitionTo checks if the session can transition to the target stage.
// Forward transitions must be sequential (one stage at a time).
func (ws *WorkSession) CanTransitionTo(target valueobject.Stage) bool {
	// Session must be active
	if ws.SessionStatus != SessionStatusActive {
		return false
	}

	// Use Stage value object's transition validation
	return ws.CurrentStage.CanTransitionTo(target)
}

// TransitionTo performs a forward stage transition.
// This method validates preconditions and updates the session state.
func (ws *WorkSession) TransitionTo(target valueobject.Stage) error {
	if !ws.CanTransitionTo(target) {
		return errors.New(errors.ErrInvalidStageTransition).WithDetail(
			"cannot transition from " + ws.CurrentStage.String() + " to " + target.String(),
		)
	}

	// Check stage-specific preconditions
	switch target {
	case valueobject.StageDesign:
		// Clarification must be completed
		if ws.Clarification == nil || !ws.Clarification.IsCompleted() {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"clarification must be completed before entering design stage",
			)
		}

	case valueobject.StageTaskBreakdown:
		// Design must be confirmed (if required)
		if ws.Design == nil || !ws.Design.CanProceedToTaskBreakdown() {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"design must be confirmed before task breakdown",
			)
		}

	case valueobject.StageExecution:
		// Tasks must be defined
		if len(ws.Tasks) == 0 {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"tasks must be defined before execution",
			)
		}

	case valueobject.StagePullRequest:
		// All tasks must be completed
		if !ws.AreAllTasksCompleted() {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"all tasks must be completed before creating PR",
			)
		}

	case valueobject.StageCompleted:
		// PR must be created (PRNumber set)
		if ws.PRNumber == nil {
			return errors.New(errors.ErrValidationFailed).WithDetail(
				"PR must be created before completion",
			)
		}
	}

	// Perform transition
	fromStage := ws.CurrentStage
	ws.CurrentStage = target
	ws.UpdatedAt = time.Now()

	// Record event
	ws.recordEvent(event.NewStageTransitioned(ws.ID, fromStage, target))

	return nil
}

// CanRollbackTo checks if the session can rollback to the target stage.
func (ws *WorkSession) CanRollbackTo(target valueobject.Stage) bool {
	// Session must be active
	if ws.SessionStatus != SessionStatusActive {
		return false
	}

	// Completed sessions cannot rollback
	if ws.CurrentStage == valueobject.StageCompleted {
		return false
	}

	// Use Stage value object's rollback validation
	return ws.CurrentStage.CanRollbackTo(target)
}

// RollbackTo performs a stage rollback.
// This method handles branch deprecation and preserves context as needed.
func (ws *WorkSession) RollbackTo(target valueobject.Stage, reason string, userInitiated bool) error {
	if !ws.CanRollbackTo(target) {
		return errors.New(errors.ErrInvalidStageTransition).WithDetail(
			"cannot rollback from " + ws.CurrentStage.String() + " to " + target.String(),
		)
	}

	// Handle rollback-specific logic
	fromStage := ws.CurrentStage

	switch fromStage {
	case valueobject.StagePullRequest:
		// PR stage rollback - handle PR and branch based on depth
		ws.PRRollbackCount++

		// Deep rollback (to Design or Clarification) - close PR and deprecate branch
		if valueobject.StageOrder(target) <= valueobject.StageOrder(valueobject.StageDesign) {
			// Deprecate current branch
			if ws.Execution != nil {
				ws.Execution.DeprecateBranch(reason, ws.PRNumber, target.String())
			}
			// PR will be closed by application layer
		}
		// Shallow rollback (to Execution) - keep PR open, keep branch

	case valueobject.StageExecution:
		// Execution rollback - deprecate branch for deep rollback
		if valueobject.StageOrder(target) <= valueobject.StageOrder(valueobject.StageDesign) {
			if ws.Execution != nil {
				ws.Execution.DeprecateBranch(reason, nil, target.String())
			}
		}

	case valueobject.StageTaskBreakdown:
		// TaskBreakdown rollback - clear tasks for deep rollback
		if valueobject.StageOrder(target) <= valueobject.StageOrder(valueobject.StageDesign) {
			ws.Tasks = []*entity.Task{}
		}

	case valueobject.StageDesign:
		// Design rollback to Clarification - create new version when re-entering
		// Application layer will handle creating new design version
	}

	// Record rollback in execution history
	if ws.Execution != nil {
		ws.Execution.RecordRollback(fromStage, target, reason, userInitiated)
	}

	// Update stage
	ws.CurrentStage = target
	ws.UpdatedAt = time.Now()

	// Reset stage-specific state
	switch target {
	case valueobject.StageClarification:
		// Keep confirmed points, clear pending questions
		if ws.Clarification != nil {
			ws.Clarification.ClearPendingQuestions()
		}
		// Clear design
		ws.Design = nil
		// Clear tasks
		ws.Tasks = []*entity.Task{}
		// Clear execution
		ws.Execution = nil

	case valueobject.StageDesign:
		// Design will be updated/recreated by application layer
		// Clear tasks
		ws.Tasks = []*entity.Task{}
		// Clear execution
		ws.Execution = nil
	}

	// Record event
	ws.recordEvent(event.NewStageRolledBack(ws.ID, fromStage, target, reason, userInitiated))

	// Record specialized PR rollback events for fine-grained tracking
	if fromStage == valueobject.StagePullRequest && ws.PRNumber != nil {
		deprecatedBranch := ""
		if ws.Execution != nil {
			deprecatedBranch = ws.Execution.Branch.Name
		}

		switch target {
		case valueobject.StageExecution:
			ws.recordEvent(event.NewPRRolledBackToExecution(ws.ID, *ws.PRNumber, reason))
		case valueobject.StageDesign:
			ws.recordEvent(event.NewPRRolledBackToDesign(ws.ID, *ws.PRNumber, reason, deprecatedBranch))
		case valueobject.StageClarification:
			ws.recordEvent(event.NewPRRolledBackToClarification(ws.ID, *ws.PRNumber, reason, deprecatedBranch))
		}
	}

	return nil
}

// --- Session Control Methods ---

// Pause pauses the active session.
// The reason parameter records why the session was paused (e.g., "user_request", "system_maintenance").
// For backward compatibility, this method accepts a string reason.
// Use PauseWithContext for detailed pause context.
func (ws *WorkSession) Pause(reason string) error {
	pauseReason, err := valueobject.ParsePauseReason(reason)
	if err != nil {
		// Backward compatibility: unknown reasons are classified as "other"
		// This preserves the original reason string for audit purposes
		pauseReason = valueobject.PauseReasonOther
		return ws.PauseWithContext(
			valueobject.NewPauseContext(pauseReason).WithErrorDetails(reason),
		)
	}
	return ws.PauseWithContext(valueobject.NewPauseContext(pauseReason))
}

// PauseWithContext pauses the session with detailed context.
// This is the preferred method for pausing as it captures full context for recovery.
func (ws *WorkSession) PauseWithContext(ctx valueobject.PauseContext) error {
	if !ws.SessionStatus.CanPause() {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"cannot pause session with status: " + string(ws.SessionStatus),
		)
	}

	ws.SessionStatus = SessionStatusPaused
	ws.PauseContext = &ctx
	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewWorkSessionPausedWithContext(ws.ID, ctx))

	return nil
}

// Resume resumes a paused session.
// For backward compatibility, this method resumes without tracking the action.
// Use ResumeWithAction for proper action tracking.
func (ws *WorkSession) Resume() error {
	return ws.ResumeWithAction("manual_resume")
}

// ResumeWithAction resumes a paused session with the specified action.
// The action parameter records how the session was resumed (e.g., "admin_continue", "auto_resume").
func (ws *WorkSession) ResumeWithAction(action string) error {
	if !ws.SessionStatus.CanResume() {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"cannot resume session with status: " + string(ws.SessionStatus),
		)
	}

	// Record pause history before clearing context
	if ws.PauseContext != nil {
		record := valueobject.NewPauseRecord(*ws.PauseContext)
		record.Complete(action)
		ws.PauseHistory = append(ws.PauseHistory, record)
	}

	previousStage := ws.CurrentStage
	var pauseReason string
	if ws.PauseContext != nil {
		pauseReason = ws.PauseContext.Reason.DisplayName()
	}

	ws.SessionStatus = SessionStatusActive
	ws.PauseContext = nil
	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewWorkSessionResumedWithAction(ws.ID, previousStage, pauseReason, action))

	return nil
}

// CanAutoResume checks if the session can be automatically resumed.
// Returns true if:
// - Session is in Paused status
// - PauseContext exists
// - PauseReason supports auto-recovery
// - Auto-resume time has passed (if specified)
func (ws *WorkSession) CanAutoResume() bool {
	if ws.SessionStatus != SessionStatusPaused {
		return false
	}
	if ws.PauseContext == nil {
		return false
	}
	return ws.PauseContext.CanAutoResumeNow()
}

// GetPauseContext returns the current pause context.
// Returns nil if the session is not paused.
func (ws *WorkSession) GetPauseContext() *valueobject.PauseContext {
	return ws.PauseContext
}

// GetPauseHistory returns the pause history for audit purposes.
func (ws *WorkSession) GetPauseHistory() []valueobject.PauseRecord {
	return ws.PauseHistory
}

// Terminate terminates the session.
func (ws *WorkSession) Terminate(reason string) error {
	if !ws.SessionStatus.CanTerminate() {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"cannot terminate session with status: " + string(ws.SessionStatus),
		)
	}

	ws.SessionStatus = SessionStatusTerminated
	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewWorkSessionTerminated(ws.ID, reason))

	return nil
}

// Complete marks the session as completed.
func (ws *WorkSession) Complete() error {
	if ws.CurrentStage != valueobject.StageCompleted {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"session must be in completed stage to mark as completed",
		)
	}

	ws.SessionStatus = SessionStatusCompleted
	ws.UpdatedAt = time.Now()

	// PRNumber should be set at this point
	prNumber := 0
	if ws.PRNumber != nil {
		prNumber = *ws.PRNumber
	}
	ws.recordEvent(event.NewWorkSessionCompleted(ws.ID, prNumber))

	return nil
}

// --- Entity Management Methods ---

// SetDesign sets the design entity.
// Complexity evaluation is handled by ComplexityEvaluator domain service (T2.4.1).
func (ws *WorkSession) SetDesign(design *entity.Design) {
	ws.Design = design
	ws.UpdatedAt = time.Now()
}

// SetTasks sets the task list for execution.
func (ws *WorkSession) SetTasks(tasks []*entity.Task) {
	ws.Tasks = tasks
	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewTaskListCreated(ws.ID, len(tasks)))
}

// StartExecution initializes the execution phase.
func (ws *WorkSession) StartExecution(worktreePath, branchName string) error {
	if ws.CurrentStage != valueobject.StageExecution {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"can only start execution in execution stage",
		)
	}

	ws.Execution = entity.NewExecution(worktreePath, branchName)
	ws.UpdatedAt = time.Now()

	return nil
}

// SetPRNumber records the PR number after PR creation.
func (ws *WorkSession) SetPRNumber(prNumber int) {
	ws.PRNumber = &prNumber
	ws.UpdatedAt = time.Now()

	// Get branch and title from execution context
	branch := ""
	if ws.Execution != nil {
		branch = ws.Execution.Branch.Name
	}
	title := ws.Issue.Title

	ws.recordEvent(event.NewPullRequestCreated(ws.ID, prNumber, branch, title))
}

// --- Task Status Methods ---

// GetNextExecutableTask returns the next task that can be executed.
// Tasks with satisfied dependencies and pending/in-retry status are candidates.
func (ws *WorkSession) GetNextExecutableTask(maxRetryLimit int) *entity.Task {
	completedSet := ws.getCompletedTaskSet()

	for _, task := range ws.Tasks {
		// Check if task can be started
		if task.IsPending() && ws.areDependenciesSatisfied(task, completedSet) {
			return task
		}
		// Check if failed task can be retried
		if task.IsFailed() && task.CanRetry(maxRetryLimit) && ws.areDependenciesSatisfied(task, completedSet) {
			return task
		}
	}

	return nil
}

// StartTask marks a task as started.
func (ws *WorkSession) StartTask(taskID uuid.UUID) error {
	task := ws.findTask(taskID)
	if task == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("task not found")
	}

	// Check if dependencies are satisfied
	completedSet := ws.getCompletedTaskSet()
	if !ws.areDependenciesSatisfied(task, completedSet) {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"task dependencies not satisfied",
		)
	}

	if err := task.Start(); err != nil {
		return err
	}

	if ws.Execution != nil {
		ws.Execution.SetCurrentTask(taskID)
	}
	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewTaskStarted(ws.ID, taskID, task.Description))

	return nil
}

// CompleteTask marks a task as completed.
func (ws *WorkSession) CompleteTask(taskID uuid.UUID, result valueobject.ExecutionResult) error {
	task := ws.findTask(taskID)
	if task == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("task not found")
	}

	if err := task.Complete(result); err != nil {
		return err
	}

	if ws.Execution != nil {
		ws.Execution.MarkTaskCompleted(taskID)
	}
	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewTaskCompleted(ws.ID, taskID))

	return nil
}

// FailTask marks a task as failed.
func (ws *WorkSession) FailTask(taskID uuid.UUID, reason, suggestion string) error {
	task := ws.findTask(taskID)
	if task == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("task not found")
	}

	if err := task.Fail(reason, suggestion); err != nil {
		return err
	}

	if ws.Execution != nil {
		ws.Execution.SetFailedTask(taskID, reason, suggestion)
	}
	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewTaskFailed(ws.ID, taskID, reason, suggestion))

	return nil
}

// SkipTask marks a task as skipped.
func (ws *WorkSession) SkipTask(taskID uuid.UUID, reason string) error {
	task := ws.findTask(taskID)
	if task == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("task not found")
	}

	if err := task.Skip(reason); err != nil {
		return err
	}

	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewTaskSkipped(ws.ID, taskID, reason))

	return nil
}

// RetryTask retries a failed task.
func (ws *WorkSession) RetryTask(taskID uuid.UUID, maxRetryLimit int) error {
	task := ws.findTask(taskID)
	if task == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("task not found")
	}

	// Record retry count before increment
	retryCount := task.RetryCount

	if err := task.Retry(maxRetryLimit); err != nil {
		return err
	}

	if ws.Execution != nil {
		ws.Execution.ClearFailedTask()
	}
	ws.UpdatedAt = time.Now()

	// Record TaskRetryStarted event
	ws.recordEvent(event.NewTaskRetryStarted(ws.ID, taskID, retryCount+1))

	return nil
}

// AreAllTasksCompleted checks if all tasks are completed or skipped.
func (ws *WorkSession) AreAllTasksCompleted() bool {
	for _, task := range ws.Tasks {
		if !task.IsCompleted() && !task.IsSkipped() {
			return false
		}
	}
	return true
}

// getCompletedTaskSet returns a set of completed task IDs for efficient lookup.
// This is an internal method used by dependency checking.
func (ws *WorkSession) getCompletedTaskSet() map[uuid.UUID]bool {
	idSet := make(map[uuid.UUID]bool)

	for _, task := range ws.Tasks {
		if task.IsCompleted() || task.IsSkipped() {
			idSet[task.ID] = true
		}
	}

	// Also include tasks from execution's completed list (for recovery)
	if ws.Execution != nil {
		for _, id := range ws.Execution.CompletedTasks {
			idSet[id] = true
		}
	}

	return idSet
}

// GetCompletedTaskIDs returns the IDs of all completed tasks.
// Uses map for O(n) deduplication instead of O(n*m) nested loop.
func (ws *WorkSession) GetCompletedTaskIDs() []uuid.UUID {
	idSet := ws.getCompletedTaskSet()

	ids := make([]uuid.UUID, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	return ids
}

// HasFailedTask checks if there's a failed task blocking execution.
func (ws *WorkSession) HasFailedTask() bool {
	for _, task := range ws.Tasks {
		if task.IsFailed() {
			return true
		}
	}
	return false
}

// GetFailedTask returns the current failed task information.
func (ws *WorkSession) GetFailedTask() *valueobject.FailedTask {
	if ws.Execution != nil && ws.Execution.FailedTask != nil {
		return ws.Execution.FailedTask
	}
	for _, task := range ws.Tasks {
		if task.IsFailed() {
			return &valueobject.FailedTask{
				TaskID:     task.ID.String(),
				Reason:     task.FailureReason,
				Suggestion: task.Suggestion,
			}
		}
	}
	return nil
}

// --- Clarification Methods ---

// AddClarificationQuestion adds a question to the clarification.
func (ws *WorkSession) AddClarificationQuestion(question string) {
	if ws.Clarification != nil {
		ws.Clarification.AddQuestion(question)
		ws.UpdatedAt = time.Now()

		ws.recordEvent(event.NewQuestionAsked(ws.ID, question))
	}
}

// AnswerClarificationQuestion records an answer to a pending question.
// The actor parameter identifies who answered the question (Issue author or admin).
func (ws *WorkSession) AnswerClarificationQuestion(question, answer, actor string) error {
	if ws.Clarification == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("clarification not initialized")
	}

	err := ws.Clarification.AnswerQuestion(question, answer)
	if err != nil {
		return err
	}

	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewQuestionAnswered(ws.ID, question, answer, actor))

	return nil
}

// ConfirmClarificationPoint adds a confirmed requirement point.
func (ws *WorkSession) ConfirmClarificationPoint(point string) {
	if ws.Clarification != nil {
		ws.Clarification.ConfirmPoint(point)
		ws.UpdatedAt = time.Now()
	}
}

// SetClarityDimensions sets the clarity evaluation result.
func (ws *WorkSession) SetClarityDimensions(dimensions valueobject.ClarityDimensions) {
	if ws.Clarification != nil {
		ws.Clarification.SetClarityDimensions(dimensions)
		ws.UpdatedAt = time.Now()
	}
}

// CanCompleteClarification checks if clarification can be completed.
func (ws *WorkSession) CanCompleteClarification(threshold int) bool {
	return ws.Clarification != nil && ws.Clarification.CanComplete(threshold)
}

// CompleteClarification marks clarification as completed.
func (ws *WorkSession) CompleteClarification() error {
	if ws.Clarification == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("clarification not initialized")
	}

	ws.Clarification.Complete()
	ws.UpdatedAt = time.Now()

	// Get clarity score for event
	clarityScore := ws.Clarification.GetClarityScore()

	ws.recordEvent(event.NewClarificationCompleted(ws.ID, clarityScore))

	return nil
}

// --- Design Methods ---

// ConfirmDesign marks the design as confirmed.
func (ws *WorkSession) ConfirmDesign() error {
	if ws.Design == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("design not initialized")
	}

	version := ws.Design.CurrentVersion
	ws.Design.Confirm()
	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewDesignApproved(ws.ID, version))

	return nil
}

// RejectDesign marks the design as rejected.
func (ws *WorkSession) RejectDesign(reason string) error {
	if ws.Design == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("design not initialized")
	}

	version := ws.Design.CurrentVersion
	ws.Design.Reject()
	ws.UpdatedAt = time.Now()

	ws.recordEvent(event.NewDesignRejected(ws.ID, version, reason))

	return nil
}

// AddDesignVersion adds a new design version (for rollback or update).
func (ws *WorkSession) AddDesignVersion(content, reason string) {
	if ws.Design != nil {
		ws.Design.AddVersion(content, reason)
		ws.UpdatedAt = time.Now()

		ws.recordEvent(event.NewDesignCreated(ws.ID, ws.Design.CurrentVersion, reason))
	}
}

// --- Query Methods ---

// IsActive returns true if the session is active.
func (ws *WorkSession) IsActive() bool {
	return ws.SessionStatus == SessionStatusActive
}

// IsPaused returns true if the session is paused.
func (ws *WorkSession) IsPaused() bool {
	return ws.SessionStatus == SessionStatusPaused
}

// IsTerminated returns true if the session is terminated.
func (ws *WorkSession) IsTerminated() bool {
	return ws.SessionStatus == SessionStatusTerminated
}

// IsCompleted returns true if the session is completed.
func (ws *WorkSession) IsCompletedSession() bool {
	return ws.SessionStatus == SessionStatusCompleted
}

// GetCurrentStage returns the current stage.
func (ws *WorkSession) GetCurrentStage() valueobject.Stage {
	return ws.CurrentStage
}

// GetIssue returns the issue entity.
func (ws *WorkSession) GetIssue() *entity.Issue {
	return ws.Issue
}

// GetDesign returns the design entity.
func (ws *WorkSession) GetDesign() *entity.Design {
	return ws.Design
}

// GetTasks returns all tasks.
func (ws *WorkSession) GetTasks() []*entity.Task {
	return ws.Tasks
}

// GetTask returns a specific task by ID.
func (ws *WorkSession) GetTask(taskID uuid.UUID) *entity.Task {
	return ws.findTask(taskID)
}

// GetExecution returns the execution entity.
func (ws *WorkSession) GetExecution() *entity.Execution {
	return ws.Execution
}

// GetPRNumber returns the PR number (if created).
func (ws *WorkSession) GetPRNumber() *int {
	return ws.PRNumber
}

// --- Domain Events ---

// GetEvents returns all collected domain events.
func (ws *WorkSession) GetEvents() []event.DomainEvent {
	return ws.events
}

// ClearEvents clears all collected events (after publishing).
func (ws *WorkSession) ClearEvents() {
	ws.events = []event.DomainEvent{}
}

// recordEvent records a domain event (internal helper).
func (ws *WorkSession) recordEvent(e event.DomainEvent) {
	ws.events = append(ws.events, e)
}

// --- Helper Methods ---

// findTask finds a task by ID.
func (ws *WorkSession) findTask(taskID uuid.UUID) *entity.Task {
	for _, task := range ws.Tasks {
		if task.ID == taskID {
			return task
		}
	}
	return nil
}

// areDependenciesSatisfied checks if all dependencies are completed.
// Uses map for O(1) lookup per dependency instead of O(n) linear search.
func (ws *WorkSession) areDependenciesSatisfied(task *entity.Task, completedSet map[uuid.UUID]bool) bool {
	if !task.HasDependencies() {
		return true
	}

	for _, depID := range task.Dependencies {
		if !completedSet[depID] {
			return false
		}
	}

	return true
}

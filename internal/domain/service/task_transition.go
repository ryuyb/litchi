package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// taskValidTransitions defines all valid state transitions for tasks.
// This is a static lookup table to avoid repeated map allocation.
// Key: current status, Value: list of valid target statuses.
var taskValidTransitions = map[valueobject.TaskStatus][]valueobject.TaskStatus{
	valueobject.TaskStatusPending:    {valueobject.TaskStatusInProgress, valueobject.TaskStatusSkipped},
	valueobject.TaskStatusInProgress: {valueobject.TaskStatusCompleted, valueobject.TaskStatusFailed, valueobject.TaskStatusSkipped},
	valueobject.TaskStatusFailed:     {valueobject.TaskStatusInProgress}, // Retry
	valueobject.TaskStatusCompleted:  {},                                 // Terminal
	valueobject.TaskStatusSkipped:    {},                                 // Terminal
}

// TaskTransitionResult represents the result of a task state transition.
type TaskTransitionResult struct {
	Task           *entity.Task            `json:"task"`           // The task after transition
	PreviousStatus valueobject.TaskStatus  `json:"previousStatus"` // Status before transition
	NewStatus      valueobject.TaskStatus  `json:"newStatus"`      // Status after transition
	Event          event.DomainEvent       `json:"event"`          // Event to publish (if any)
	Success        bool                    `json:"success"`        // Whether transition succeeded
	Error          string                  `json:"error"`          // Error message (if failed)
	RetryContext   *valueobject.RetryContext `json:"retryContext,omitempty"` // Retry context (only for RetryTask)
}

// TaskTransitionService handles task state transitions and event publishing.
// It implements the state machine pattern with proper validation and event emission.
type TaskTransitionService struct {
	eventDispatcher EventDispatcher
}

// NewTaskTransitionService creates a new TaskTransitionService.
func NewTaskTransitionService(dispatcher EventDispatcher) *TaskTransitionService {
	return &TaskTransitionService{
		eventDispatcher: dispatcher,
	}
}

// StartTask transitions a task from Pending to InProgress.
// Returns an error if the transition is invalid.
func (s *TaskTransitionService) StartTask(sessionID uuid.UUID, task *entity.Task) (*TaskTransitionResult, error) {
	if task == nil {
		return nil, errors.New(errors.ErrValidationFailed).WithDetail("task cannot be nil")
	}

	previousStatus := task.Status

	// Attempt the transition
	if err := task.Start(); err != nil {
		return &TaskTransitionResult{
			Task:           task,
			PreviousStatus: previousStatus,
			NewStatus:      previousStatus, // No change
			Success:        false,
			Error:          err.Error(),
		}, err
	}

	// Create and publish event
	evt := event.NewTaskStarted(sessionID, task.ID, task.Description)
	if s.eventDispatcher != nil {
		s.eventDispatcher.Dispatch(evt)
	}

	return &TaskTransitionResult{
		Task:           task,
		PreviousStatus: previousStatus,
		NewStatus:      task.Status,
		Event:          evt,
		Success:        true,
	}, nil
}

// CompleteTask transitions a task from InProgress to Completed.
func (s *TaskTransitionService) CompleteTask(sessionID uuid.UUID, task *entity.Task, result valueobject.ExecutionResult) (*TaskTransitionResult, error) {
	if task == nil {
		return nil, errors.New(errors.ErrValidationFailed).WithDetail("task cannot be nil")
	}

	previousStatus := task.Status

	// Attempt the transition
	if err := task.Complete(result); err != nil {
		return &TaskTransitionResult{
			Task:           task,
			PreviousStatus: previousStatus,
			NewStatus:      previousStatus,
			Success:        false,
			Error:          err.Error(),
		}, err
	}

	// Create and publish event
	evt := event.NewTaskCompleted(sessionID, task.ID)
	if s.eventDispatcher != nil {
		s.eventDispatcher.Dispatch(evt)
	}

	return &TaskTransitionResult{
		Task:           task,
		PreviousStatus: previousStatus,
		NewStatus:      task.Status,
		Event:          evt,
		Success:        true,
	}, nil
}

// FailTask transitions a task from InProgress to Failed.
func (s *TaskTransitionService) FailTask(sessionID uuid.UUID, task *entity.Task, reason, suggestion string) (*TaskTransitionResult, error) {
	if task == nil {
		return nil, errors.New(errors.ErrValidationFailed).WithDetail("task cannot be nil")
	}

	previousStatus := task.Status

	// Attempt the transition
	if err := task.Fail(reason, suggestion); err != nil {
		return &TaskTransitionResult{
			Task:           task,
			PreviousStatus: previousStatus,
			NewStatus:      previousStatus,
			Success:        false,
			Error:          err.Error(),
		}, err
	}

	// Create and publish event
	evt := event.NewTaskFailed(sessionID, task.ID, reason, suggestion)
	if s.eventDispatcher != nil {
		s.eventDispatcher.Dispatch(evt)
	}

	return &TaskTransitionResult{
		Task:           task,
		PreviousStatus: previousStatus,
		NewStatus:      task.Status,
		Event:          evt,
		Success:        true,
	}, nil
}

// SkipTask transitions a task from Pending/InProgress to Skipped.
func (s *TaskTransitionService) SkipTask(sessionID uuid.UUID, task *entity.Task, reason string) (*TaskTransitionResult, error) {
	if task == nil {
		return nil, errors.New(errors.ErrValidationFailed).WithDetail("task cannot be nil")
	}

	previousStatus := task.Status

	// Attempt the transition
	if err := task.Skip(reason); err != nil {
		return &TaskTransitionResult{
			Task:           task,
			PreviousStatus: previousStatus,
			NewStatus:      previousStatus,
			Success:        false,
			Error:          err.Error(),
		}, err
	}

	// Create and publish event
	evt := event.NewTaskSkipped(sessionID, task.ID, reason)
	if s.eventDispatcher != nil {
		s.eventDispatcher.Dispatch(evt)
	}

	return &TaskTransitionResult{
		Task:           task,
		PreviousStatus: previousStatus,
		NewStatus:      task.Status,
		Event:          evt,
		Success:        true,
	}, nil
}

// RetryTask transitions a failed task back to InProgress for retry.
// The retry context with delay information is included in the result.
func (s *TaskTransitionService) RetryTask(sessionID uuid.UUID, task *entity.Task, maxRetryLimit int, retryContext *valueobject.RetryContext) (*TaskTransitionResult, error) {
	if task == nil {
		return nil, errors.New(errors.ErrValidationFailed).WithDetail("task cannot be nil")
	}

	previousStatus := task.Status

	// Check if retry is allowed
	if !task.CanRetry(maxRetryLimit) {
		errMsg := fmt.Sprintf("task cannot be retried: status=%s, retryCount=%d, maxLimit=%d",
			task.Status.String(), task.RetryCount, maxRetryLimit)
		return &TaskTransitionResult{
			Task:           task,
			PreviousStatus: previousStatus,
			NewStatus:      previousStatus,
			Success:        false,
			Error:          errMsg,
			RetryContext:   retryContext,
		}, errors.New(errors.ErrValidationFailed).WithDetail(errMsg)
	}

	// Attempt the transition
	if err := task.Retry(maxRetryLimit); err != nil {
		return &TaskTransitionResult{
			Task:           task,
			PreviousStatus: previousStatus,
			NewStatus:      previousStatus,
			Success:        false,
			Error:          err.Error(),
			RetryContext:   retryContext,
		}, err
	}

	// Record the retry attempt in context (success=false because retry outcome is not yet determined)
	if retryContext != nil {
		retryContext.RecordAttempt(false, "retry initiated")
	}

	// Create and publish event
	newRetryCount := task.RetryCount
	evt := event.NewTaskRetryStarted(sessionID, task.ID, newRetryCount)
	if s.eventDispatcher != nil {
		s.eventDispatcher.Dispatch(evt)
	}

	return &TaskTransitionResult{
		Task:           task,
		PreviousStatus: previousStatus,
		NewStatus:      task.Status,
		Event:          evt,
		Success:        true,
		RetryContext:   retryContext,
	}, nil
}

// HandleFinalFailure handles a task that has exhausted all retry attempts.
// Returns the appropriate action based on the failure handling policy.
func (s *TaskTransitionService) HandleFinalFailure(
	sessionID uuid.UUID,
	task *entity.Task,
	failureHandling valueobject.FinalFailureHandling,
	retryContext *valueobject.RetryContext,
) (*FinalFailureResult, error) {
	if task == nil {
		return nil, errors.New(errors.ErrValidationFailed).WithDetail("task cannot be nil")
	}

	if !task.IsFailed() {
		return nil, errors.New(errors.ErrValidationFailed).WithDetail("task must be in failed status to handle final failure")
	}

	if retryContext != nil && !retryContext.IsExhausted() {
		return nil, errors.New(errors.ErrValidationFailed).WithDetail("retry attempts not yet exhausted")
	}

	result := &FinalFailureResult{
		Task:            task,
		SessionID:       sessionID,
		Action:          failureHandling.Action,
		RollbackStage:   failureHandling.RollbackStage,
		NotifyUser:      failureHandling.NotifyUser,
		Reason:          failureHandling.Reason,
		TotalRetryTime:  0,
		RetryAttempts:   task.RetryCount,
	}

	if retryContext != nil {
		result.TotalRetryTime = retryContext.GetTotalWaitDuration()
	}

	return result, nil
}

// ValidateTransition checks if a transition from current status to target status is valid.
func (s *TaskTransitionService) ValidateTransition(task *entity.Task, targetStatus valueobject.TaskStatus) error {
	if task == nil {
		return errors.New(errors.ErrValidationFailed).WithDetail("task cannot be nil")
	}

	if !task.Status.CanTransitionTo(targetStatus) {
		return errors.New(errors.ErrInvalidTaskStatus).WithDetail(
			fmt.Sprintf("invalid transition: %s -> %s", task.Status.String(), targetStatus.String()),
		)
	}

	return nil
}

// GetValidTransitions returns all valid target statuses for the current task status.
func (s *TaskTransitionService) GetValidTransitions(task *entity.Task) []valueobject.TaskStatus {
	if task == nil {
		return []valueobject.TaskStatus{}
	}
	return taskValidTransitions[task.Status]
}

// FinalFailureResult represents the result of handling a final failure.
type FinalFailureResult struct {
	Task            *entity.Task              `json:"task"`
	SessionID       uuid.UUID                 `json:"sessionId"`
	Action          valueobject.FinalFailureAction `json:"action"`
	RollbackStage   valueobject.Stage         `json:"rollbackStage"`
	NotifyUser      bool                      `json:"notifyUser"`
	Reason          string                    `json:"reason"`
	TotalRetryTime  time.Duration             `json:"totalRetryTime"`
	RetryAttempts   int                       `json:"retryAttempts"`
}
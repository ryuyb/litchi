package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/stretchr/testify/mock"
)

// --- TaskTransitionService Tests ---

func TestTaskTransitionServiceStartTask(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)
	sessionID := uuid.New()

	task := entity.NewTask("Test task", nil, 1)
	if task.Status != valueobject.TaskStatusPending {
		t.Fatalf("Initial status should be pending, got %s", task.Status)
	}

	// Expect Dispatch to be called
	dispatcher.EXPECT().Dispatch(mock.AnythingOfType("*event.TaskStarted")).Run(func(e event.DomainEvent) {
		if e.EventType() != "TaskStarted" {
			t.Errorf("Expected TaskStarted event, got %s", e.EventType())
		}
	})

	result, err := service.StartTask(sessionID, task)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	// Verify result
	if !result.Success {
		t.Errorf("Result.Success should be true")
	}
	if result.PreviousStatus != valueobject.TaskStatusPending {
		t.Errorf("PreviousStatus = %s, expected pending", result.PreviousStatus)
	}
	if result.NewStatus != valueobject.TaskStatusInProgress {
		t.Errorf("NewStatus = %s, expected in_progress", result.NewStatus)
	}

	// Verify task state
	if task.Status != valueobject.TaskStatusInProgress {
		t.Errorf("Task status = %s, expected in_progress", task.Status)
	}

	// Verify event
	if result.Event == nil {
		t.Errorf("Event should not be nil")
	}
	if result.Event.EventType() != "TaskStarted" {
		t.Errorf("Event type = %s, expected TaskStarted", result.Event.EventType())
	}
}

func TestTaskTransitionServiceStartTaskInvalid(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)
	sessionID := uuid.New()

	// Create task already in progress
	task := entity.NewTask("Test task", nil, 1)
	task.Status = valueobject.TaskStatusInProgress

	result, err := service.StartTask(sessionID, task)
	if err == nil {
		t.Fatalf("StartTask should fail for non-pending task")
	}

	if result.Success {
		t.Errorf("Result.Success should be false")
	}
	if result.NewStatus != valueobject.TaskStatusInProgress {
		t.Errorf("Status should remain in_progress")
	}
	// No event should be dispatched - mock will assert expectations
}

func TestTaskTransitionServiceCompleteTask(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)
	sessionID := uuid.New()

	task := entity.NewTask("Test task", nil, 1)
	task.Status = valueobject.TaskStatusInProgress // Must be in progress to complete

	result := valueobject.ExecutionResult{
		Output:   "Task output",
		Success:  true,
		Duration: 300000, // 5 minutes in milliseconds
	}

	dispatcher.EXPECT().Dispatch(mock.AnythingOfType("*event.TaskCompleted"))

	transitionResult, err := service.CompleteTask(sessionID, task, result)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	if !transitionResult.Success {
		t.Errorf("Result.Success should be true")
	}
	if transitionResult.NewStatus != valueobject.TaskStatusCompleted {
		t.Errorf("NewStatus = %s, expected completed", transitionResult.NewStatus)
	}
	if transitionResult.Event.EventType() != "TaskCompleted" {
		t.Errorf("Event type = %s, expected TaskCompleted", transitionResult.Event.EventType())
	}
}

func TestTaskTransitionServiceFailTask(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)
	sessionID := uuid.New()

	task := entity.NewTask("Test task", nil, 1)
	task.Status = valueobject.TaskStatusInProgress

	dispatcher.EXPECT().Dispatch(mock.AnythingOfType("*event.TaskFailed"))

	transitionResult, err := service.FailTask(sessionID, task, "Connection timeout", "Check network settings")
	if err != nil {
		t.Fatalf("FailTask failed: %v", err)
	}

	if !transitionResult.Success {
		t.Errorf("Result.Success should be true")
	}
	if transitionResult.NewStatus != valueobject.TaskStatusFailed {
		t.Errorf("NewStatus = %s, expected failed", transitionResult.NewStatus)
	}
	if task.FailureReason != "Connection timeout" {
		t.Errorf("FailureReason = %s, expected Connection timeout", task.FailureReason)
	}
	if task.Suggestion != "Check network settings" {
		t.Errorf("Suggestion = %s, expected Check network settings", task.Suggestion)
	}
	if task.RetryCount != 1 {
		t.Errorf("RetryCount = %d, expected 1", task.RetryCount)
	}
	if transitionResult.Event.EventType() != "TaskFailed" {
		t.Errorf("Event type = %s, expected TaskFailed", transitionResult.Event.EventType())
	}
}

func TestTaskTransitionServiceSkipTask(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus valueobject.TaskStatus
	}{
		{"skip_from_pending", valueobject.TaskStatusPending},
		{"skip_from_in_progress", valueobject.TaskStatusInProgress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher := NewMockEventDispatcher(t)
			service := NewTaskTransitionService(dispatcher)
			sessionID := uuid.New()

			task := entity.NewTask("Test task", nil, 1)
			task.Status = tt.initialStatus

			dispatcher.EXPECT().Dispatch(mock.AnythingOfType("*event.TaskSkipped"))

			result, err := service.SkipTask(sessionID, task, "Not needed")
			if err != nil {
				t.Fatalf("SkipTask failed: %v", err)
			}

			if !result.Success {
				t.Errorf("Result.Success should be true")
			}
			if result.NewStatus != valueobject.TaskStatusSkipped {
				t.Errorf("NewStatus = %s, expected skipped", result.NewStatus)
			}
			if result.Event.EventType() != "TaskSkipped" {
				t.Errorf("Event type = %s, expected TaskSkipped", result.Event.EventType())
			}
		})
	}
}

func TestTaskTransitionServiceSkipTaskInvalid(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)
	sessionID := uuid.New()

	task := entity.NewTask("Test task", nil, 1)
	task.Status = valueobject.TaskStatusCompleted // Cannot skip completed task

	result, err := service.SkipTask(sessionID, task, "Not needed")
	if err == nil {
		t.Fatalf("SkipTask should fail for completed task")
	}

	if result.Success {
		t.Errorf("Result.Success should be false")
	}
}

func TestTaskTransitionServiceRetryTask(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)
	sessionID := uuid.New()

	task := entity.NewTask("Test task", nil, 1)
	task.Status = valueobject.TaskStatusFailed
	task.RetryCount = 1

	policy := valueobject.DefaultRetryPolicy
	retryCtx := valueobject.NewRetryContext(policy)

	dispatcher.EXPECT().Dispatch(mock.AnythingOfType("*event.TaskRetryStarted"))

	result, err := service.RetryTask(sessionID, task, 3, retryCtx)
	if err != nil {
		t.Fatalf("RetryTask failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Result.Success should be true")
	}
	if result.NewStatus != valueobject.TaskStatusInProgress {
		t.Errorf("NewStatus = %s, expected in_progress", result.NewStatus)
	}
	if result.Event.EventType() != "TaskRetryStarted" {
		t.Errorf("Event type = %s, expected TaskRetryStarted", result.Event.EventType())
	}

	// Verify retry context is included and was updated
	if result.RetryContext == nil {
		t.Fatalf("RetryContext should not be nil")
	}
	if result.RetryContext.CurrentAttempt < 1 {
		t.Errorf("CurrentAttempt should be >= 1 after retry")
	}
}

func TestTaskTransitionServiceRetryTaskMaxLimit(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)
	sessionID := uuid.New()

	task := entity.NewTask("Test task", nil, 1)
	task.Status = valueobject.TaskStatusFailed
	task.RetryCount = 3 // Already at max

	policy := valueobject.DefaultRetryPolicy
	retryCtx := valueobject.NewRetryContext(policy)

	result, err := service.RetryTask(sessionID, task, 3, retryCtx)
	if err == nil {
		t.Fatalf("RetryTask should fail when max limit reached")
	}

	if result.Success {
		t.Errorf("Result.Success should be false")
	}
	if result.NewStatus != valueobject.TaskStatusFailed {
		t.Errorf("Status should remain failed")
	}
}

func TestTaskTransitionServiceHandleFinalFailure(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)
	sessionID := uuid.New()

	task := entity.NewTask("Test task", nil, 1)
	task.Status = valueobject.TaskStatusFailed
	task.RetryCount = 3

	policy := valueobject.DefaultRetryPolicy
	retryCtx := valueobject.NewRetryContext(policy)
	retryCtx.CurrentAttempt = 3 // Exhausted

	tests := []struct {
		name        string
		handling    valueobject.FinalFailureHandling
		expectAction valueobject.FinalFailureAction
	}{
		{
			name:        "pause_session",
			handling:    valueobject.DefaultFinalFailureHandling,
			expectAction: valueobject.FinalFailureActionPauseSession,
		},
		{
			name:        "rollback_to_design",
			handling:    valueobject.FinalFailureHandlingWithRollback(valueobject.StageDesign),
			expectAction: valueobject.FinalFailureActionRollback,
		},
		{
			name:        "skip_task",
			handling:    valueobject.NewFinalFailureHandling(valueobject.FinalFailureActionSkipTask),
			expectAction: valueobject.FinalFailureActionSkipTask,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.HandleFinalFailure(sessionID, task, tt.handling, retryCtx)
			if err != nil {
				t.Fatalf("HandleFinalFailure failed: %v", err)
			}

			if result.Action != tt.expectAction {
				t.Errorf("Action = %s, expected %s", result.Action, tt.expectAction)
			}
			if !result.NotifyUser {
				t.Errorf("NotifyUser should be true")
			}
			if result.RetryAttempts != 3 {
				t.Errorf("RetryAttempts = %d, expected 3", result.RetryAttempts)
			}
		})
	}
}

func TestTaskTransitionServiceValidateTransition(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)

	tests := []struct {
		name          string
		currentStatus valueobject.TaskStatus
		targetStatus  valueobject.TaskStatus
		shouldPass    bool
	}{
		{"pending_to_in_progress", valueobject.TaskStatusPending, valueobject.TaskStatusInProgress, true},
		{"pending_to_completed", valueobject.TaskStatusPending, valueobject.TaskStatusCompleted, false},
		{"in_progress_to_completed", valueobject.TaskStatusInProgress, valueobject.TaskStatusCompleted, true},
		{"in_progress_to_failed", valueobject.TaskStatusInProgress, valueobject.TaskStatusFailed, true},
		{"failed_to_in_progress", valueobject.TaskStatusFailed, valueobject.TaskStatusInProgress, true},
		{"completed_to_in_progress", valueobject.TaskStatusCompleted, valueobject.TaskStatusInProgress, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := entity.NewTask("Test task", nil, 1)
			task.Status = tt.currentStatus

			err := service.ValidateTransition(task, tt.targetStatus)
			if tt.shouldPass && err != nil {
				t.Errorf("ValidateTransition should pass, got error: %v", err)
			}
			if !tt.shouldPass && err == nil {
				t.Errorf("ValidateTransition should fail for %s -> %s", tt.currentStatus, tt.targetStatus)
			}
		})
	}
}

func TestTaskTransitionServiceGetValidTransitions(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)

	tests := []struct {
		name              string
		currentStatus     valueobject.TaskStatus
		expectedCount     int
	}{
		{"pending_has_2_transitions", valueobject.TaskStatusPending, 2},
		{"in_progress_has_3_transitions", valueobject.TaskStatusInProgress, 3},
		{"failed_has_1_transition", valueobject.TaskStatusFailed, 1},
		{"completed_has_0_transitions", valueobject.TaskStatusCompleted, 0},
		{"skipped_has_0_transitions", valueobject.TaskStatusSkipped, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := entity.NewTask("Test task", nil, 1)
			task.Status = tt.currentStatus

			transitions := service.GetValidTransitions(task)
			if len(transitions) != tt.expectedCount {
				t.Errorf("GetValidTransitions for %s returned %d transitions, expected %d",
					tt.currentStatus, len(transitions), tt.expectedCount)
			}
		})
	}
}

func TestTaskTransitionServiceNilTask(t *testing.T) {
	dispatcher := NewMockEventDispatcher(t)
	service := NewTaskTransitionService(dispatcher)
	sessionID := uuid.New()

	_, err := service.StartTask(sessionID, nil)
	if err == nil {
		t.Error("StartTask with nil task should return error")
	}

	_, err = service.CompleteTask(sessionID, nil, valueobject.ExecutionResult{})
	if err == nil {
		t.Error("CompleteTask with nil task should return error")
	}

	_, err = service.FailTask(sessionID, nil, "reason", "suggestion")
	if err == nil {
		t.Error("FailTask with nil task should return error")
	}

	_, err = service.SkipTask(sessionID, nil, "reason")
	if err == nil {
		t.Error("SkipTask with nil task should return error")
	}

	_, err = service.RetryTask(sessionID, nil, 3, nil)
	if err == nil {
		t.Error("RetryTask with nil task should return error")
	}
}

func TestTaskTransitionServiceWithoutDispatcher(t *testing.T) {
	// Test that service works without dispatcher (nil)
	service := NewTaskTransitionService(nil)
	sessionID := uuid.New()

	task := entity.NewTask("Test task", nil, 1)

	result, err := service.StartTask(sessionID, task)
	if err != nil {
		t.Fatalf("StartTask failed without dispatcher: %v", err)
	}

	if !result.Success {
		t.Errorf("Result.Success should be true")
	}
	// Event should still be created even if not dispatched
	if result.Event == nil {
		t.Errorf("Event should still be created")
	}
}
package event

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"go.uber.org/zap"
)

func TestDomainEventBasics(t *testing.T) {
	sessionID := uuid.New()
	now := time.Now()

	// Test WorkSessionStarted
	started := NewWorkSessionStarted(sessionID, 123, "owner/repo", "Test Issue")
	if started.EventType() != "WorkSessionStarted" {
		t.Errorf("EventType mismatch: got %s", started.EventType())
	}
	if started.SessionID() != sessionID {
		t.Errorf("SessionID mismatch")
	}
	if started.OccurredAt().Before(now) {
		t.Errorf("OccurredAt should be after now")
	}
	if started.IssueNumber != 123 {
		t.Errorf("IssueNumber mismatch: got %d", started.IssueNumber)
	}
	if started.Repository != "owner/repo" {
		t.Errorf("Repository mismatch: got %s", started.Repository)
	}

	// Test ToMap
	m := started.ToMap()
	if m["type"] != "WorkSessionStarted" {
		t.Errorf("ToMap type mismatch")
	}
	if m["issueNumber"] != 123 {
		t.Errorf("ToMap issueNumber mismatch")
	}
}

func TestWorkSessionResumedEvent(t *testing.T) {
	sessionID := uuid.New()

	// Test WorkSessionResumed with previousStage
	resumed := NewWorkSessionResumed(sessionID, valueobject.StageExecution)
	if resumed.EventType() != "WorkSessionResumed" {
		t.Errorf("EventType mismatch: got %s", resumed.EventType())
	}
	if resumed.SessionID() != sessionID {
		t.Errorf("SessionID mismatch")
	}
	if resumed.PreviousStage != valueobject.StageExecution {
		t.Errorf("PreviousStage mismatch: got %s", resumed.PreviousStage)
	}

	// Test ToMap
	m := resumed.ToMap()
	if m["previousStage"] != "execution" {
		t.Errorf("ToMap previousStage mismatch: got %s", m["previousStage"])
	}
}

func TestStageTransitionedEvent(t *testing.T) {
	sessionID := uuid.New()

	e := NewStageTransitioned(
		sessionID,
		valueobject.StageClarification,
		valueobject.StageDesign,
	)

	if e.EventType() != "StageTransitioned" {
		t.Errorf("EventType mismatch: got %s", e.EventType())
	}
	if e.FromStage != valueobject.StageClarification {
		t.Errorf("FromStage mismatch: got %s", e.FromStage)
	}
	if e.ToStage != valueobject.StageDesign {
		t.Errorf("ToStage mismatch: got %s", e.ToStage)
	}

	// Test ToMap
	m := e.ToMap()
	if m["fromStage"] != "clarification" {
		t.Errorf("ToMap fromStage mismatch: got %s", m["fromStage"])
	}
	if m["toStage"] != "design" {
		t.Errorf("ToMap toStage mismatch: got %s", m["toStage"])
	}
}

func TestStageRolledBackEvent(t *testing.T) {
	sessionID := uuid.New()

	e := NewStageRolledBack(
		sessionID,
		valueobject.StageExecution,
		valueobject.StageDesign,
		"User requested rollback",
		true,
	)

	if e.EventType() != "StageRolledBack" {
		t.Errorf("EventType mismatch: got %s", e.EventType())
	}
	if e.FromStage != valueobject.StageExecution {
		t.Errorf("FromStage mismatch: got %s", e.FromStage)
	}
	if e.ToStage != valueobject.StageDesign {
		t.Errorf("ToStage mismatch: got %s", e.ToStage)
	}
	if e.Reason != "User requested rollback" {
		t.Errorf("Reason mismatch: got %s", e.Reason)
	}
	if !e.UserInitiated {
		t.Errorf("UserInitiated should be true")
	}
}

func TestTaskEvents(t *testing.T) {
	sessionID := uuid.New()
	taskID := uuid.New()

	// Test TaskStarted
	started := NewTaskStarted(sessionID, taskID, "Implement feature X")
	if started.EventType() != "TaskStarted" {
		t.Errorf("EventType mismatch: got %s", started.EventType())
	}
	if started.TaskID != taskID {
		t.Errorf("TaskID mismatch")
	}
	if started.TaskDescription != "Implement feature X" {
		t.Errorf("TaskDescription mismatch: got %s", started.TaskDescription)
	}

	// Test TaskCompleted
	completed := NewTaskCompleted(sessionID, taskID)
	if completed.EventType() != "TaskCompleted" {
		t.Errorf("EventType mismatch: got %s", completed.EventType())
	}
	if completed.TaskID != taskID {
		t.Errorf("TaskID mismatch")
	}

	// Test TaskFailed
	failed := NewTaskFailed(sessionID, taskID, "Test failed", "Fix the test")
	if failed.EventType() != "TaskFailed" {
		t.Errorf("EventType mismatch: got %s", failed.EventType())
	}
	if failed.Reason != "Test failed" {
		t.Errorf("Reason mismatch: got %s", failed.Reason)
	}
	if failed.Suggestion != "Fix the test" {
		t.Errorf("Suggestion mismatch: got %s", failed.Suggestion)
	}

	// Test TaskSkipped
	skipped := NewTaskSkipped(sessionID, taskID, "No longer needed")
	if skipped.EventType() != "TaskSkipped" {
		t.Errorf("EventType mismatch: got %s", skipped.EventType())
	}
	if skipped.Reason != "No longer needed" {
		t.Errorf("Reason mismatch: got %s", skipped.Reason)
	}

	// Test TaskRetryStarted
	retry := NewTaskRetryStarted(sessionID, taskID, 2)
	if retry.EventType() != "TaskRetryStarted" {
		t.Errorf("EventType mismatch: got %s", retry.EventType())
	}
	if retry.TaskID != taskID {
		t.Errorf("TaskID mismatch")
	}
	if retry.RetryCount != 2 {
		t.Errorf("RetryCount mismatch: got %d", retry.RetryCount)
	}
}

func TestClarificationEvents(t *testing.T) {
	sessionID := uuid.New()

	// Test QuestionAsked
	asked := NewQuestionAsked(sessionID, "What is the expected behavior?")
	if asked.EventType() != "QuestionAsked" {
		t.Errorf("EventType mismatch: got %s", asked.EventType())
	}
	if asked.Question != "What is the expected behavior?" {
		t.Errorf("Question mismatch: got %s", asked.Question)
	}

	// Test QuestionAnswered
	answered := NewQuestionAnswered(sessionID, "What is the expected behavior?", "It should work", "test-user")
	if answered.EventType() != "QuestionAnswered" {
		t.Errorf("EventType mismatch: got %s", answered.EventType())
	}
	if answered.Question != "What is the expected behavior?" {
		t.Errorf("Question mismatch: got %s", answered.Question)
	}
	if answered.Answer != "It should work" {
		t.Errorf("Answer mismatch: got %s", answered.Answer)
	}
	if answered.Actor != "test-user" {
		t.Errorf("Actor mismatch: got %s", answered.Actor)
	}

	// Test ClarificationCompleted
	completed := NewClarificationCompleted(sessionID, 75)
	if completed.EventType() != "ClarificationCompleted" {
		t.Errorf("EventType mismatch: got %s", completed.EventType())
	}
	if completed.ClarityScore != 75 {
		t.Errorf("ClarityScore mismatch: got %d", completed.ClarityScore)
	}
}

func TestDesignEvents(t *testing.T) {
	sessionID := uuid.New()

	// Test DesignCreated
	created := NewDesignCreated(sessionID, 1, "Initial design")
	if created.EventType() != "DesignCreated" {
		t.Errorf("EventType mismatch: got %s", created.EventType())
	}
	if created.Version != 1 {
		t.Errorf("Version mismatch: got %d", created.Version)
	}
	if created.Reason != "Initial design" {
		t.Errorf("Reason mismatch: got %s", created.Reason)
	}

	// Test DesignApproved
	approved := NewDesignApproved(sessionID, 2)
	if approved.EventType() != "DesignApproved" {
		t.Errorf("EventType mismatch: got %s", approved.EventType())
	}
	if approved.Version != 2 {
		t.Errorf("Version mismatch: got %d", approved.Version)
	}

	// Test DesignRejected
	rejected := NewDesignRejected(sessionID, 2, "Needs revision")
	if rejected.EventType() != "DesignRejected" {
		t.Errorf("EventType mismatch: got %s", rejected.EventType())
	}
	if rejected.Reason != "Needs revision" {
		t.Errorf("Reason mismatch: got %s", rejected.Reason)
	}
}

func TestPullRequestCreatedEvent(t *testing.T) {
	sessionID := uuid.New()

	e := NewPullRequestCreated(sessionID, 42, "feature-branch", "Add new feature")
	if e.EventType() != "PullRequestCreated" {
		t.Errorf("EventType mismatch: got %s", e.EventType())
	}
	if e.PRNumber != 42 {
		t.Errorf("PRNumber mismatch: got %d", e.PRNumber)
	}
	if e.Branch != "feature-branch" {
		t.Errorf("Branch mismatch: got %s", e.Branch)
	}
	if e.PRTitle != "Add new feature" {
		t.Errorf("PRTitle mismatch: got %s", e.PRTitle)
	}
}

func TestPullRequestMergedEvent(t *testing.T) {
	sessionID := uuid.New()

	e := NewPullRequestMerged(sessionID, 42, "reviewer", "abc123sha")
	if e.EventType() != "PullRequestMerged" {
		t.Errorf("EventType mismatch: got %s", e.EventType())
	}
	if e.PRNumber != 42 {
		t.Errorf("PRNumber mismatch: got %d", e.PRNumber)
	}
	if e.MergedBy != "reviewer" {
		t.Errorf("MergedBy mismatch: got %s", e.MergedBy)
	}
	if e.MergeSHA != "abc123sha" {
		t.Errorf("MergeSHA mismatch: got %s", e.MergeSHA)
	}
}

func TestPRRollbackEvents(t *testing.T) {
	sessionID := uuid.New()

	// Test PRRolledBackToExecution (R4: shallow rollback)
	r4 := NewPRRolledBackToExecution(sessionID, 42, "CI failure")
	if r4.EventType() != "PRRolledBackToExecution" {
		t.Errorf("EventType mismatch: got %s", r4.EventType())
	}
	if r4.PRNumber != 42 {
		t.Errorf("PRNumber mismatch: got %d", r4.PRNumber)
	}
	if r4.Reason != "CI failure" {
		t.Errorf("Reason mismatch: got %s", r4.Reason)
	}

	// Test PRRolledBackToDesign (R5: deep rollback)
	r5 := NewPRRolledBackToDesign(sessionID, 42, "Design issue", "feature-branch")
	if r5.EventType() != "PRRolledBackToDesign" {
		t.Errorf("EventType mismatch: got %s", r5.EventType())
	}
	if r5.PRNumber != 42 {
		t.Errorf("PRNumber mismatch: got %d", r5.PRNumber)
	}
	if r5.DeprecatedBranch != "feature-branch" {
		t.Errorf("DeprecatedBranch mismatch: got %s", r5.DeprecatedBranch)
	}

	// Test PRRolledBackToClarification (R6: deepest rollback)
	r6 := NewPRRolledBackToClarification(sessionID, 42, "Requirement change", "feature-branch")
	if r6.EventType() != "PRRolledBackToClarification" {
		t.Errorf("EventType mismatch: got %s", r6.EventType())
	}
	if r6.PRNumber != 42 {
		t.Errorf("PRNumber mismatch: got %d", r6.PRNumber)
	}
	if r6.DeprecatedBranch != "feature-branch" {
		t.Errorf("DeprecatedBranch mismatch: got %s", r6.DeprecatedBranch)
	}
}

func TestRepositoryEvents(t *testing.T) {
	// Test RepositoryAdded
	added := NewRepositoryAdded("owner/repo")
	if added.EventType() != "RepositoryAdded" {
		t.Errorf("EventType mismatch: got %s", added.EventType())
	}
	if added.RepositoryName != "owner/repo" {
		t.Errorf("RepositoryName mismatch: got %s", added.RepositoryName)
	}
	// Repository events are system-level, should have nil SessionID
	if added.SessionID() != uuid.Nil {
		t.Errorf("Repository event should have nil SessionID")
	}
	// Test IsSystemEvent helper
	if !IsSystemEvent(added) {
		t.Errorf("RepositoryAdded should be a system event")
	}

	// Test RepositoryUpdated
	updated := NewRepositoryUpdated("owner/repo", []string{"maxConcurrency", "taskRetryLimit"})
	if updated.EventType() != "RepositoryUpdated" {
		t.Errorf("EventType mismatch: got %s", updated.EventType())
	}
	if len(updated.Changes) != 2 {
		t.Errorf("Changes length mismatch: got %d", len(updated.Changes))
	}
	if updated.SessionID() != uuid.Nil {
		t.Errorf("Repository event should have nil SessionID")
	}
	if !IsSystemEvent(updated) {
		t.Errorf("RepositoryUpdated should be a system event")
	}
}

func TestIsSystemEvent(t *testing.T) {
	sessionID := uuid.New()

	// Session-related event is not system event
	started := NewWorkSessionStarted(sessionID, 123, "owner/repo", "Test")
	if IsSystemEvent(started) {
		t.Errorf("WorkSessionStarted should not be a system event")
	}

	// Repository event is system event
	repoAdded := NewRepositoryAdded("owner/repo")
	if !IsSystemEvent(repoAdded) {
		t.Errorf("RepositoryAdded should be a system event")
	}
}

// --- Dispatcher Tests ---

func TestDispatcherRegister(t *testing.T) {
	d := NewDispatcher()

	d.Register("WorkSessionStarted", func(ctx context.Context, event DomainEvent) error {
		return nil
	})

	if !d.HasHandlers("WorkSessionStarted") {
		t.Error("Should have handlers for WorkSessionStarted")
	}
	if d.HandlerCount("WorkSessionStarted") != 1 {
		t.Errorf("HandlerCount mismatch: got %d", d.HandlerCount("WorkSessionStarted"))
	}
}

func TestDispatcherRegisterAll(t *testing.T) {
	d := NewDispatcher()

	count := 0
	d.RegisterAll(func(ctx context.Context, event DomainEvent) error {
		count++
		return nil
	})

	// RegisterAll should register for wildcard "*"
	if !d.HasHandlers("*") {
		t.Error("Should have wildcard handlers")
	}
}

func TestDispatcherDispatchSync(t *testing.T) {
	d := NewDispatcher()

	sessionID := uuid.New()
	handlerCalled := false
	var receivedEvent DomainEvent

	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		handlerCalled = true
		receivedEvent = e
		return nil
	})

	event := NewWorkSessionStarted(sessionID, 123, "owner/repo", "Test")
	err := d.DispatchSync(context.Background(), event)

	if err != nil {
		t.Errorf("DispatchSync failed: %v", err)
	}
	if !handlerCalled {
		t.Error("Handler should have been called")
	}
	if receivedEvent == nil {
		t.Error("Event should have been received")
	}
}

func TestDispatcherDispatchAsync(t *testing.T) {
	d := NewDispatcher(WithAsyncWorkers(5))

	sessionID := uuid.New()
	var mu sync.Mutex
	handlerCalled := false

	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		mu.Lock()
		handlerCalled = true
		mu.Unlock()
		return nil
	})

	event := NewWorkSessionStarted(sessionID, 123, "owner/repo", "Test")
	d.DispatchAsync(context.Background(), event)

	// Wait for async handler to complete
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if !handlerCalled {
		t.Error("Handler should have been called asynchronously")
	}
	mu.Unlock()
}

func TestDispatcherDispatchBatch(t *testing.T) {
	d := NewDispatcher()

	sessionID := uuid.New()
	count := 0

	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		count++
		return nil
	})

	events := []DomainEvent{
		NewWorkSessionStarted(sessionID, 1, "owner/repo", "Test 1"),
		NewWorkSessionStarted(sessionID, 2, "owner/repo", "Test 2"),
		NewWorkSessionStarted(sessionID, 3, "owner/repo", "Test 3"),
	}

	err := d.DispatchBatch(context.Background(), events)
	if err != nil {
		t.Errorf("DispatchBatch failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Handler should have been called 3 times, got %d", count)
	}
}

func TestDispatcherTypedHandlers(t *testing.T) {
	d := NewDispatcher()

	sessionID := uuid.New()
	taskID := uuid.New()
	receivedTaskID := uuid.Nil

	d.RegisterTaskStarted(func(ctx context.Context, e *TaskStarted) error {
		receivedTaskID = e.TaskID
		return nil
	})

	event := NewTaskStarted(sessionID, taskID, "Test task")
	err := d.DispatchSync(context.Background(), event)

	if err != nil {
		t.Errorf("DispatchSync failed: %v", err)
	}
	if receivedTaskID != taskID {
		t.Errorf("TaskID mismatch: got %s, want %s", receivedTaskID, taskID)
	}
}

func TestDispatcherErrorHandler(t *testing.T) {
	d := NewDispatcher()

	sessionID := uuid.New()

	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		return &testError{msg: "handler error"}
	})

	event := NewWorkSessionStarted(sessionID, 123, "owner/repo", "Test")
	err := d.DispatchSync(context.Background(), event)

	if err == nil {
		t.Error("Should return error from handler")
	}
}

func TestDispatcherMultipleHandlers(t *testing.T) {
	d := NewDispatcher()

	sessionID := uuid.New()
	count := 0

	// Register multiple handlers for same event type
	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		count++
		return nil
	})
	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		count++
		return nil
	})
	d.RegisterAll(func(ctx context.Context, e DomainEvent) error {
		count++
		return nil
	})

	event := NewWorkSessionStarted(sessionID, 123, "owner/repo", "Test")
	err := d.DispatchSync(context.Background(), event)

	if err != nil {
		t.Errorf("DispatchSync failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Should call 3 handlers (2 specific + 1 wildcard), got %d", count)
	}
}

func TestDispatcherDispatchSyncContinuesOnError(t *testing.T) {
	// Test that DispatchSync continues executing handlers even after error
	d := NewDispatcher()

	sessionID := uuid.New()
	executionOrder := []string{}

	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		executionOrder = append(executionOrder, "handler1")
		return &testError{msg: "handler1 error"}
	})
	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		executionOrder = append(executionOrder, "handler2")
		return nil
	})
	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		executionOrder = append(executionOrder, "handler3")
		return nil
	})

	event := NewWorkSessionStarted(sessionID, 123, "owner/repo", "Test")
	err := d.DispatchSync(context.Background(), event)

	// Should return first error
	if err == nil {
		t.Error("Should return error from first handler")
	}

	// But all handlers should have been executed
	if len(executionOrder) != 3 {
		t.Errorf("Should execute all 3 handlers, got %d: %v", len(executionOrder), executionOrder)
	}
}

func TestDispatcherDispatchSyncStopOnError(t *testing.T) {
	// Test that DispatchSyncStopOnError stops on first error
	d := NewDispatcher()

	sessionID := uuid.New()
	executionOrder := []string{}

	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		executionOrder = append(executionOrder, "handler1")
		return &testError{msg: "handler1 error"}
	})
	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		executionOrder = append(executionOrder, "handler2")
		return nil
	})
	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		executionOrder = append(executionOrder, "handler3")
		return nil
	})

	event := NewWorkSessionStarted(sessionID, 123, "owner/repo", "Test")
	err := d.DispatchSyncStopOnError(context.Background(), event)

	// Should return first error
	if err == nil {
		t.Error("Should return error from first handler")
	}

	// Should NOT execute remaining handlers
	if len(executionOrder) != 1 {
		t.Errorf("Should stop after first error, got %d handlers executed: %v", len(executionOrder), executionOrder)
	}
	if executionOrder[0] != "handler1" {
		t.Errorf("Should only execute handler1, got: %v", executionOrder)
	}
}

func TestDispatcherClear(t *testing.T) {
	d := NewDispatcher()

	d.Register("WorkSessionStarted", func(ctx context.Context, e DomainEvent) error {
		return nil
	})

	d.Clear()

	if d.HasHandlers("WorkSessionStarted") {
		t.Error("Should have no handlers after Clear")
	}
}

func TestDispatcherWithOptions(t *testing.T) {
	logger := zap.NewNop()
	d := NewDispatcher(
		WithLogger(logger),
		WithAsyncWorkers(20),
	)

	if d.logger != logger {
		t.Error("Logger should be set")
	}
	if d.asyncWorkers != 20 {
		t.Errorf("AsyncWorkers mismatch: got %d", d.asyncWorkers)
	}
}

func TestDispatcherWithNilLogger(t *testing.T) {
	// Passing nil logger should retain the default no-op logger
	d := NewDispatcher(
		WithLogger(nil),
	)

	// Logger should not be nil (should be zap.NewNop())
	if d.logger == nil {
		t.Error("Logger should not be nil, should use default no-op logger")
	}

	// Verify the dispatcher still works without panic
	sessionID := uuid.New()
	event := NewWorkSessionStarted(sessionID, 123, "owner/repo", "Test")
	err := d.DispatchSync(context.Background(), event)
	if err != nil {
		t.Errorf("DispatchSync should succeed: %v", err)
	}
}

func TestDispatcherNoHandlers(t *testing.T) {
	d := NewDispatcher()

	sessionID := uuid.New()
	event := NewWorkSessionStarted(sessionID, 123, "owner/repo", "Test")

	// Dispatch should succeed even with no handlers
	err := d.DispatchSync(context.Background(), event)
	if err != nil {
		t.Errorf("DispatchSync should succeed with no handlers: %v", err)
	}
}

// Test error type
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
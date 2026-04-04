package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

func TestStageTransitionService_CanTransition(t *testing.T) {
	service := NewDefaultStageTransitionService(nil) // Use default scheduler
	ctx := DefaultTransitionContext()

	// Create a valid session in Clarification stage
	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Cannot transition to Design without completing clarification
	if service.CanTransition(session, valueobject.StageDesign, ctx) {
		t.Error("should not be able to transition to Design without completing clarification")
	}

	// Add confirmed point (required for transition)
	session.ConfirmClarificationPoint("Feature X required")

	// Complete clarification
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(70))

	// Now can transition
	if !service.CanTransition(session, valueobject.StageDesign, ctx) {
		t.Error("should be able to transition to Design after clarification completed")
	}
}

func TestStageTransitionService_CanRollback(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Clarification cannot rollback
	if service.CanRollback(session, valueobject.StageClarification, ctx) {
		t.Error("should not be able to rollback from Clarification")
	}

	// Move to Design stage
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(70))
	session.TransitionTo(valueobject.StageDesign)

	// Can rollback to Clarification from Design
	if !service.CanRollback(session, valueobject.StageClarification, ctx) {
		t.Error("should be able to rollback from Design to Clarification")
	}

	// Cannot rollback to Execution (wrong direction)
	if service.CanRollback(session, valueobject.StageExecution, ctx) {
		t.Error("should not be able to rollback forward")
	}
}

func TestStageTransitionService_PRRollbackConstraints(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)

	// Test AllowPRRollback = false
	ctx := TransitionContext{
		ClarityThreshold:    60,
		ComplexityThreshold: 70,
		AllowPRRollback:     false,
		MaxPRRollbackCount:  3,
	}

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Advance to PR stage
	advanceSessionToPRStage(session)

	// Should not allow rollback when disabled
	if service.CanRollback(session, valueobject.StageExecution, ctx) {
		t.Error("should not allow PR rollback when disabled")
	}

	// Test MaxPRRollbackCount constraint
	ctx = TransitionContext{
		ClarityThreshold:    60,
		ComplexityThreshold: 70,
		AllowPRRollback:     true,
		MaxPRRollbackCount:  1,
	}

	session.PRRollbackCount = 1 // Already rolled back once

	if service.CanRollback(session, valueobject.StageExecution, ctx) {
		t.Error("should not allow rollback when count exceeds limit")
	}
}

func TestStageTransitionService_GetAllowedRollbackTargets(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Clarification stage - no rollback targets
	targets := service.GetAllowedRollbackTargets(session, ctx)
	if len(targets) != 0 {
		t.Errorf("expected 0 rollback targets from Clarification, got %d", len(targets))
	}

	// Design stage - can rollback to Clarification
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(70))
	session.TransitionTo(valueobject.StageDesign)

	targets = service.GetAllowedRollbackTargets(session, ctx)
	if len(targets) != 1 || targets[0] != valueobject.StageClarification {
		t.Errorf("expected Clarification as rollback target from Design")
	}

	// Execution stage - can rollback to Design or Clarification
	advanceSessionToExecutionStage(session)
	session.CurrentStage = valueobject.StageExecution

	targets = service.GetAllowedRollbackTargets(session, ctx)
	if len(targets) != 2 {
		t.Errorf("expected 2 rollback targets from Execution, got %d", len(targets))
	}
}

func TestStageTransitionService_ValidateTransitionPreconditions(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	// Test Clarification → Design preconditions
	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Missing: no pending questions, at least one confirmed point, clarity >= threshold
	err = service.ValidateTransitionPreconditions(session, valueobject.StageDesign, ctx)
	if err == nil {
		t.Error("should fail validation without completing clarification")
	}

	// Add confirmed point
	session.ConfirmClarificationPoint("Feature X required")
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(70))

	err = service.ValidateTransitionPreconditions(session, valueobject.StageDesign, ctx)
	if err != nil {
		t.Errorf("should pass validation after clarification completed: %v", err)
	}
}

func TestStageTransitionService_LowClarityScore(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	session.ConfirmClarificationPoint("Feature X required")
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(50)) // Below threshold

	// Should fail with low clarity score
	err = service.ValidateTransitionPreconditions(session, valueobject.StageDesign, ctx)
	if err == nil {
		t.Error("should fail validation with low clarity score")
	}
}

func TestStageTransitionService_DesignConfirmation(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Advance to Design stage
	session.ConfirmClarificationPoint("Feature X")
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(70))
	session.TransitionTo(valueobject.StageDesign)

	// Create design with high complexity (needs confirmation)
	design := entity.NewDesign("Design content")
	design.SetComplexityScore(createMockComplexityScore(80), 70)
	session.SetDesign(design)

	// Should fail without confirmation
	ctx := DefaultTransitionContext()
	err = service.ValidateTransitionPreconditions(session, valueobject.StageTaskBreakdown, ctx)
	if err == nil {
		t.Error("should fail validation without design confirmation")
	}

	// Confirm design
	session.ConfirmDesign()

	err = service.ValidateTransitionPreconditions(session, valueobject.StageTaskBreakdown, ctx)
	if err != nil {
		t.Errorf("should pass validation after design confirmed: %v", err)
	}

	// Test ForceDesignConfirm
	ctx = TransitionContext{
		ClarityThreshold:    60,
		ComplexityThreshold: 70,
		ForceDesignConfirm:  true,
	}

	// Reset design confirmation
	design2 := entity.NewDesign("Design content v2")
	design2.SetComplexityScore(createMockComplexityScore(30), 70) // Low complexity
	session.SetDesign(design2)

	err = service.ValidateTransitionPreconditions(session, valueobject.StageTaskBreakdown, ctx)
	if err == nil {
		t.Error("should fail validation when ForceDesignConfirm is true")
	}
}

func TestStageTransitionService_TaskValidation(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Advance to TaskBreakdown stage
	advanceSessionToTaskBreakdownStage(session)

	// No tasks defined
	err = service.ValidateTransitionPreconditions(session, valueobject.StageExecution, ctx)
	if err == nil {
		t.Error("should fail validation without tasks")
	}

	// Add tasks
	task := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	session.SetTasks([]*entity.Task{task})

	err = service.ValidateTransitionPreconditions(session, valueobject.StageExecution, ctx)
	if err != nil {
		t.Errorf("should pass validation with tasks: %v", err)
	}
}

func TestStageTransitionService_PRCompletion(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Advance to Execution stage
	advanceSessionToExecutionStage(session)

	// Some tasks not completed
	task := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	session.SetTasks([]*entity.Task{task})

	err = service.ValidateTransitionPreconditions(session, valueobject.StagePullRequest, ctx)
	if err == nil {
		t.Error("should fail validation with incomplete tasks")
	}

	// Complete task
	task.Start()
	task.Complete(valueobject.ExecutionResult{})

	err = service.ValidateTransitionPreconditions(session, valueobject.StagePullRequest, ctx)
	if err != nil {
		t.Errorf("should pass validation with completed tasks: %v", err)
	}

	// Test Completed stage - needs PR number
	session.TransitionTo(valueobject.StagePullRequest)
	err = service.ValidateTransitionPreconditions(session, valueobject.StageCompleted, ctx)
	if err == nil {
		t.Error("should fail validation without PR number")
	}

	session.SetPRNumber(123)
	err = service.ValidateTransitionPreconditions(session, valueobject.StageCompleted, ctx)
	if err != nil {
		t.Errorf("should pass validation with PR number: %v", err)
	}
}

func TestStageTransitionService_WithCustomScheduler(t *testing.T) {
	// Test that custom scheduler can be injected
	mockScheduler := &mockTaskScheduler{}
	service := NewDefaultStageTransitionService(mockScheduler)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test", "Body", "owner/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)
	advanceSessionToTaskBreakdownStage(session)

	// Create a task with circular dependency (mock will report error)
	task := entity.NewTask("Task", []uuid.UUID{}, 1)
	session.SetTasks([]*entity.Task{task})

	// The mock scheduler should be used
	err := service.ValidateTransitionPreconditions(session, valueobject.StageExecution, ctx)
	if err == nil {
		t.Error("expected error from mock scheduler")
	}
}

// mockTaskScheduler is a mock implementation for testing dependency injection
type mockTaskScheduler struct{}

func (m *mockTaskScheduler) GetExecutionOrder(tasks []*entity.Task) ([]*entity.Task, error) {
	return nil, nil
}

func (m *mockTaskScheduler) GetNextExecutable(tasks []*entity.Task, completedIDs []uuid.UUID, maxRetryLimit int) *entity.Task {
	return nil
}

func (m *mockTaskScheduler) GetParallelTasks(tasks []*entity.Task, completedIDs []uuid.UUID) []*entity.Task {
	return nil
}

func (m *mockTaskScheduler) GetBlockedTasks(tasks []*entity.Task, completedIDs []uuid.UUID) []*entity.Task {
	return nil
}

func (m *mockTaskScheduler) GetDependencyGraph(tasks []*entity.Task) map[uuid.UUID][]uuid.UUID {
	return nil
}

func (m *mockTaskScheduler) CanRetryTask(task *entity.Task, completedIDs []uuid.UUID, maxRetryLimit int) bool {
	return false
}

func (m *mockTaskScheduler) GetExecutionPlan(tasks []*entity.Task) ([][]*entity.Task, error) {
	return nil, nil
}

func (m *mockTaskScheduler) ValidateDependencies(tasks []*entity.Task) error {
	return errors.New(errors.ErrValidationFailed).WithDetail("mock scheduler error")
}

// Helper functions

func createMockClarityDimensions(score int) valueobject.ClarityDimensions {
	// Create clarity dimensions with a specific total score.
	// We need to distribute the score across dimensions according to their max scores.
	// Max total = 30 + 25 + 20 + 15 + 10 = 100
	//
	// For simplicity, we'll set each dimension to a proportion of its max score
	// that achieves the desired total.
	//
	// score 70 = 70% of 100, so each dimension gets 70% of its max
	// This gives: 21 + 17.5 + 14 + 10.5 + 7 = 70

	proportion := float64(score) / 100.0
	completeness := int(proportion * float64(valueobject.CompletenessMaxScore))
	clarity := int(proportion * float64(valueobject.ClarityMaxScore))
	consistency := int(proportion * float64(valueobject.ConsistencyMaxScore))
	feasibility := int(proportion * float64(valueobject.FeasibilityMaxScore))
	testability := int(proportion * float64(valueobject.TestabilityMaxScore))

	cd, _ := valueobject.NewClarityDimensions(completeness, clarity, consistency, feasibility, testability)
	return cd
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func createMockComplexityScore(value int) valueobject.ComplexityScore {
	score, _ := valueobject.NewComplexityScore(value)
	return score
}

func advanceSessionToPRStage(session *aggregate.WorkSession) {
	session.ConfirmClarificationPoint("Feature")
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(70))
	session.TransitionTo(valueobject.StageDesign)

	design := entity.NewDesign("Design")
	design.Confirm()
	session.SetDesign(design)
	session.TransitionTo(valueobject.StageTaskBreakdown)

	task := entity.NewTask("Task", []uuid.UUID{}, 1)
	task.Start()
	task.Complete(valueobject.ExecutionResult{})
	session.SetTasks([]*entity.Task{task})
	session.TransitionTo(valueobject.StageExecution)
	session.TransitionTo(valueobject.StagePullRequest)
}

func advanceSessionToTaskBreakdownStage(session *aggregate.WorkSession) {
	session.ConfirmClarificationPoint("Feature")
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(70))
	session.TransitionTo(valueobject.StageDesign)

	design := entity.NewDesign("Design")
	design.Confirm()
	session.SetDesign(design)
	session.TransitionTo(valueobject.StageTaskBreakdown)
}

func advanceSessionToExecutionStage(session *aggregate.WorkSession) {
	advanceSessionToTaskBreakdownStage(session)
	task := entity.NewTask("Task", []uuid.UUID{}, 1)
	session.SetTasks([]*entity.Task{task})
	session.TransitionTo(valueobject.StageExecution)
}
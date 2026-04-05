package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/stretchr/testify/mock"
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

	// Direct field modification to test rollback count limit boundary condition.
	// Note: In production, PRRollbackCount is incremented only through RollbackTo().
	// Using direct modification here to isolate the count limit test without
	// the complexity of performing a full rollback and re-advancing to PR stage.
	session.PRRollbackCount = 1

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
	// Test that custom scheduler can be injected using generated mock
	mockScheduler := NewMockTaskScheduler(t)
	service := NewDefaultStageTransitionService(mockScheduler)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test", "Body", "owner/repo", "author")
	session, _ := aggregate.NewWorkSession(issue)
	advanceSessionToTaskBreakdownStage(session)

	// Create a task with circular dependency (mock will report error)
	task := entity.NewTask("Task", []uuid.UUID{}, 1)
	session.SetTasks([]*entity.Task{task})

	// Setup mock expectation - ValidateDependencies will return an error
	mockScheduler.EXPECT().
		ValidateDependencies(mock.Anything).
		Return(errors.New(errors.ErrValidationFailed).WithDetail("mock scheduler error"))

	// The mock scheduler should be used
	err := service.ValidateTransitionPreconditions(session, valueobject.StageExecution, ctx)
	if err == nil {
		t.Error("expected error from mock scheduler")
	}
}

// Helper functions

// createMockClarityDimensions creates ClarityDimensions with a specific total score.
//
// Precision Note: This function uses proportional calculation which may have
// integer rounding errors. For example:
//   - score=80 with CompletenessMaxScore=30 → 24 (exact: 80% of 30)
//   - score=70 with ClarityMaxScore=25 → 17 (actual: 17.5, truncated)
//
// The final total score may differ by 1-2 points from the requested score.
// Tests should check score ranges rather than exact values when using this helper.
// For exact score requirements, consider using ClarityDimensions directly with
// manually calculated dimension scores.
func createMockClarityDimensions(score int) valueobject.ClarityDimensions {
	// Create clarity dimensions with a specific total score.
	// We need to distribute the score across dimensions according to their max scores.
	// Max total = 30 + 25 + 20 + 15 + 10 = 100
	//
	// For simplicity, we'll set each dimension to a proportion of its max score
	// that achieves the desired total.

	proportion := float64(score) / 100.0
	completeness := int(proportion * float64(valueobject.CompletenessMaxScore))
	clarity := int(proportion * float64(valueobject.ClarityMaxScore))
	consistency := int(proportion * float64(valueobject.ConsistencyMaxScore))
	feasibility := int(proportion * float64(valueobject.FeasibilityMaxScore))
	testability := int(proportion * float64(valueobject.TestabilityMaxScore))

	cd, _ := valueobject.NewClarityDimensions(completeness, clarity, consistency, feasibility, testability)
	return cd
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

// --- EvaluateTransition Tests (T3.1.1) ---

func TestTransitionContext_Validate(t *testing.T) {
	tests := []struct {
		name      string
		ctx       TransitionContext
		wantError bool
	}{
		{
			name:      "valid default context",
			ctx:       DefaultTransitionContext(),
			wantError: false,
		},
		{
			name: "valid custom context",
			ctx: TransitionContext{
				ForceClarifyThreshold: 30,
				ClarityThreshold:      50,
				AutoProceedThreshold:  70,
			},
			wantError: false,
		},
		{
			name: "ForceClarifyThreshold >= ClarityThreshold",
			ctx: TransitionContext{
				ForceClarifyThreshold: 60,
				ClarityThreshold:      60,
				AutoProceedThreshold:  80,
			},
			wantError: true,
		},
		{
			name: "ClarityThreshold >= AutoProceedThreshold",
			ctx: TransitionContext{
				ForceClarifyThreshold: 40,
				ClarityThreshold:      80,
				AutoProceedThreshold:  80,
			},
			wantError: true,
		},
		{
			name: "negative threshold",
			ctx: TransitionContext{
				ForceClarifyThreshold: -1,
				ClarityThreshold:      60,
				AutoProceedThreshold:  80,
			},
			wantError: true,
		},
		{
			name: "AutoProceedThreshold > 100",
			ctx: TransitionContext{
				ForceClarifyThreshold: 40,
				ClarityThreshold:      60,
				AutoProceedThreshold:  101,
			},
			wantError: true,
		},
		{
			name: "negative TaskRetryLimit",
			ctx: TransitionContext{
				ForceClarifyThreshold: 40,
				ClarityThreshold:      60,
				AutoProceedThreshold:  80,
				TaskRetryLimit:        -1,
			},
			wantError: true,
		},
		{
			name: "negative MaxPRRollbackCount",
			ctx: TransitionContext{
				ForceClarifyThreshold: 40,
				ClarityThreshold:      60,
				AutoProceedThreshold:  80,
				MaxPRRollbackCount:    -1,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ctx.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestEvaluateTransition_HighClarity(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Setup: no pending questions, has confirmed points, clarity score >= 80
	// Use score 80 (exactly at AutoProceedThreshold) for precise calculation
	session.ConfirmClarificationPoint("Feature X required")
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(80))

	result := service.EvaluateTransition(session, valueobject.StageDesign, ctx)

	if !result.IsAllowed() {
		t.Errorf("expected DecisionAllow for high clarity, got %s", result.Decision)
	}
	// Note: createMockClarityDimensions may have slight rounding differences
	// We check that the score is in the correct range (>= 80)
	if result.ClarityScore < 80 {
		t.Errorf("expected clarity score >= 80, got %d", result.ClarityScore)
	}
	if result.CanForce {
		t.Error("high clarity should not allow force (already allowed)")
	}
}

func TestEvaluateTransition_MediumClarity(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Setup: clarity score in 60-79 range
	// Use score 60 (exactly at ClarityThreshold) for precise calculation
	session.ConfirmClarificationPoint("Feature X required")
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(60))

	result := service.EvaluateTransition(session, valueobject.StageDesign, ctx)

	if !result.IsAllowed() {
		t.Errorf("expected DecisionAllow for medium clarity, got %s", result.Decision)
	}
	// Check that the score is in the correct range (60-79)
	if result.ClarityScore < 60 || result.ClarityScore >= 80 {
		t.Errorf("expected clarity score in range [60, 79], got %d", result.ClarityScore)
	}
	if !result.HasRequiredAction() {
		t.Error("medium clarity should have required action (design confirmation)")
	}
	if result.RequiredAction != "Design needs manual confirmation" {
		t.Errorf("unexpected required action: %s", result.RequiredAction)
	}
}

func TestEvaluateTransition_LowClarity_NeedConfirmation(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Setup: clarity score in 40-59 range
	// Use score 50 (middle of range)
	session.ConfirmClarificationPoint("Feature X required")
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(50))

	result := service.EvaluateTransition(session, valueobject.StageDesign, ctx)

	if !result.NeedsConfirmation() {
		t.Errorf("expected DecisionNeedConfirmation for low clarity, got %s", result.Decision)
	}
	// Check that the score is in the correct range (40-59)
	if result.ClarityScore < 40 || result.ClarityScore >= 60 {
		t.Errorf("expected clarity score in range [40, 59], got %d", result.ClarityScore)
	}
	if !result.CanForce {
		t.Error("low clarity should allow force with user command")
	}
	if !result.HasRequiredAction() {
		t.Error("low clarity should have required action")
	}
}

func TestEvaluateTransition_VeryLowClarity_Denied(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Setup: clarity score < 40
	// Use score 0 for precise calculation (all dimensions get 0)
	session.ConfirmClarificationPoint("Feature X required")
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(0))

	result := service.EvaluateTransition(session, valueobject.StageDesign, ctx)

	if !result.IsDenied() {
		t.Errorf("expected DecisionDenied for very low clarity, got %s", result.Decision)
	}
	// Check that the score is in the correct range (< 40)
	if result.ClarityScore >= 40 {
		t.Errorf("expected clarity score < 40, got %d", result.ClarityScore)
	}
	if !result.CanForce {
		t.Error("very low clarity should allow force with user command")
	}
	if !result.HasRequiredAction() {
		t.Error("very low clarity should have required action")
	}
}

func TestEvaluateTransition_SkipClarityCheck(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := TransitionContext{
		ClarityThreshold:      60,
		AutoProceedThreshold:  80,
		ForceClarifyThreshold: 40,
		SkipClarityCheck:      true, // User command "start_design"
	}

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Setup: very low clarity but skip check enabled
	session.ConfirmClarificationPoint("Feature X required")
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(30))

	result := service.EvaluateTransition(session, valueobject.StageDesign, ctx)

	if !result.IsAllowed() {
		t.Errorf("expected DecisionAllow with SkipClarityCheck, got %s", result.Decision)
	}
	if result.CanForce {
		t.Error("skip clarity check result should NOT have CanForce (already allowed)")
	}
	if !result.IsSkippedClarity() {
		t.Error("skip clarity check result should have SkipClarity=true")
	}
}

func TestEvaluateTransition_PendingQuestions(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Setup: high clarity but has pending questions
	session.ConfirmClarificationPoint("Feature X required")
	session.AddClarificationQuestion("What about Y?")
	// Do NOT complete clarification
	session.SetClarityDimensions(createMockClarityDimensions(85))

	result := service.EvaluateTransition(session, valueobject.StageDesign, ctx)

	if !result.IsDenied() {
		t.Errorf("expected DecisionDenied with pending questions, got %s", result.Decision)
	}
	if result.CanForce {
		t.Error("pending questions should not allow force")
	}
	if !result.HasRequiredAction() {
		t.Error("should have required action")
	}
}

func TestEvaluateTransition_NoConfirmedPoints(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Setup: high clarity but no confirmed points
	session.CompleteClarification()
	session.SetClarityDimensions(createMockClarityDimensions(85))

	result := service.EvaluateTransition(session, valueobject.StageDesign, ctx)

	if !result.IsDenied() {
		t.Errorf("expected DecisionDenied without confirmed points, got %s", result.Decision)
	}
	if result.CanForce {
		t.Error("no confirmed points should not allow force")
	}
}

func TestEvaluateTransition_DesignNeedsConfirm(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

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

	result := service.EvaluateTransition(session, valueobject.StageTaskBreakdown, ctx)

	if !result.NeedsConfirmation() {
		t.Errorf("expected DecisionNeedConfirmation, got %s", result.Decision)
	}
	if !result.HasRequiredAction() {
		t.Error("should have required action")
	}

	// Confirm design and re-evaluate
	session.ConfirmDesign()
	result = service.EvaluateTransition(session, valueobject.StageTaskBreakdown, ctx)

	if !result.IsAllowed() {
		t.Errorf("expected DecisionAllow after confirmation, got %s", result.Decision)
	}
}

func TestEvaluateTransition_InvalidTransition(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Try to jump from Clarification directly to Execution (invalid)
	result := service.EvaluateTransition(session, valueobject.StageExecution, ctx)

	if !result.IsDenied() {
		t.Errorf("expected DecisionDenied for invalid transition, got %s", result.Decision)
	}
	if result.CanForce {
		t.Error("invalid transition should not allow force")
	}
}

func TestEvaluateTransition_NoTasks(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	advanceSessionToTaskBreakdownStage(session)

	// No tasks defined
	result := service.EvaluateTransition(session, valueobject.StageExecution, ctx)

	if !result.IsDenied() {
		t.Errorf("expected DecisionDenied without tasks, got %s", result.Decision)
	}
}

func TestEvaluateTransition_IncompleteTasks(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	advanceSessionToExecutionStage(session)

	// Task not completed
	task := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	session.SetTasks([]*entity.Task{task})

	result := service.EvaluateTransition(session, valueobject.StagePullRequest, ctx)

	if !result.IsDenied() {
		t.Errorf("expected DecisionDenied with incomplete tasks, got %s", result.Decision)
	}
}

func TestEvaluateTransition_NoPR(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := DefaultTransitionContext()

	issue := entity.NewIssue(1, "Test Issue", "Body", "owner/repo", "author")
	session, err := aggregate.NewWorkSession(issue)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	advanceSessionToPRStage(session)

	// Clear PR number to test "PR stage without PR" boundary condition.
	// Note: Direct field modification is acceptable here for testing edge cases.
	// In production, this state should not occur (PR must be created before entering PR stage).
	session.PRNumber = nil

	result := service.EvaluateTransition(session, valueobject.StageCompleted, ctx)

	if !result.IsDenied() {
		t.Errorf("expected DecisionDenied without PR, got %s", result.Decision)
	}
}

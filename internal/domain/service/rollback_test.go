package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

func TestEvaluateRollback_R1_ExecutionToDesign(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := TransitionContext{
		AllowPRRollback:    true,
		MaxPRRollbackCount: 3,
	}

	// Create session in Execution stage
	session := createTestSessionAtStage(valueobject.StageExecution)

	result := service.EvaluateRollback(session, valueobject.StageDesign, ctx)

	if result.Decision != RollbackAllowed {
		t.Errorf("Expected RollbackAllowed, got %v", result.Decision)
	}
	if result.RollbackRule != "R1" {
		t.Errorf("Expected R1, got %s", result.RollbackRule)
	}
	// R1 is Deep as per state-machine.md (design version +1, branch deprecated)
	if result.RollbackType != RollbackTypeDeep {
		t.Errorf("Expected RollbackTypeDeep for R1, got %v", result.RollbackType)
	}
	if !result.WillDeprecateBranch {
		t.Error("Expected WillDeprecateBranch to be true for R1")
	}
	if result.WillClosePR {
		t.Error("Expected WillClosePR to be false for R1 (not from PR stage)")
	}
}

func TestEvaluateRollback_R2_DesignToClarification(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := TransitionContext{}

	// Create session in Design stage
	session := createTestSessionAtStage(valueobject.StageDesign)

	result := service.EvaluateRollback(session, valueobject.StageClarification, ctx)

	if result.Decision != RollbackAllowed {
		t.Errorf("Expected RollbackAllowed, got %v", result.Decision)
	}
	if result.RollbackRule != "R2" {
		t.Errorf("Expected R2, got %s", result.RollbackRule)
	}
	// R2 is Shallow (keep requirements, clear design)
	if result.RollbackType != RollbackTypeShallow {
		t.Errorf("Expected RollbackTypeShallow for R2, got %v", result.RollbackType)
	}
	if !result.WillClearDesign {
		t.Error("Expected WillClearDesign to be true for R2")
	}
}

func TestEvaluateRollback_R3_ExecutionToClarification(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := TransitionContext{}

	// Create session in Execution stage
	session := createTestSessionAtStage(valueobject.StageExecution)

	result := service.EvaluateRollback(session, valueobject.StageClarification, ctx)

	if result.Decision != RollbackAllowed {
		t.Errorf("Expected RollbackAllowed, got %v", result.Decision)
	}
	if result.RollbackRule != "R3" {
		t.Errorf("Expected R3, got %s", result.RollbackRule)
	}
	if result.RollbackType != RollbackTypeFull {
		t.Errorf("Expected RollbackTypeFull, got %v", result.RollbackType)
	}
	if !result.WillDeprecateBranch {
		t.Error("Expected WillDeprecateBranch to be true for R3")
	}
	if !result.WillClearDesign {
		t.Error("Expected WillClearDesign to be true for R3")
	}
}

func TestEvaluateRollback_R4_PullRequestToExecution(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := TransitionContext{
		AllowPRRollback:    true,
		MaxPRRollbackCount: 3,
	}

	// Create session in PullRequest stage with PR number
	session := createTestSessionAtStage(valueobject.StagePullRequest)
	session.PRNumber = new(42)

	result := service.EvaluateRollback(session, valueobject.StageExecution, ctx)

	if result.Decision != RollbackAllowed {
		t.Errorf("Expected RollbackAllowed, got %v", result.Decision)
	}
	if result.RollbackRule != "R4" {
		t.Errorf("Expected R4, got %s", result.RollbackRule)
	}
	if result.RollbackType != RollbackTypeShallow {
		t.Errorf("Expected RollbackTypeShallow, got %v", result.RollbackType)
	}
	if result.WillClosePR {
		t.Error("Expected WillClosePR to be false for R4 (shallow rollback)")
	}
	if !result.WillIncrementPRRollbackCount {
		t.Error("Expected WillIncrementPRRollbackCount to be true for R4")
	}
}

func TestEvaluateRollback_R5_PullRequestToDesign(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := TransitionContext{
		AllowPRRollback:    true,
		MaxPRRollbackCount: 3,
	}

	// Create session in PullRequest stage with PR number
	session := createTestSessionAtStage(valueobject.StagePullRequest)
	session.PRNumber = new(42)

	result := service.EvaluateRollback(session, valueobject.StageDesign, ctx)

	if result.Decision != RollbackAllowed {
		t.Errorf("Expected RollbackAllowed, got %v", result.Decision)
	}
	if result.RollbackRule != "R5" {
		t.Errorf("Expected R5, got %s", result.RollbackRule)
	}
	// R5 is Deep (PR closed, branch deprecated) as per state-machine.md
	if result.RollbackType != RollbackTypeDeep {
		t.Errorf("Expected RollbackTypeDeep for R5, got %v", result.RollbackType)
	}
	if !result.WillClosePR {
		t.Error("Expected WillClosePR to be true for R5 (deep rollback)")
	}
	if !result.WillDeprecateBranch {
		t.Error("Expected WillDeprecateBranch to be true for R5")
	}
}

func TestEvaluateRollback_R6_PullRequestToClarification(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := TransitionContext{
		AllowPRRollback:    true,
		MaxPRRollbackCount: 3,
	}

	// Create session in PullRequest stage with PR number
	session := createTestSessionAtStage(valueobject.StagePullRequest)
	session.PRNumber = new(42)

	result := service.EvaluateRollback(session, valueobject.StageClarification, ctx)

	if result.Decision != RollbackAllowed {
		t.Errorf("Expected RollbackAllowed, got %v", result.Decision)
	}
	if result.RollbackRule != "R6" {
		t.Errorf("Expected R6, got %s", result.RollbackRule)
	}
	if result.RollbackType != RollbackTypeFull {
		t.Errorf("Expected RollbackTypeFull, got %v", result.RollbackType)
	}
	if !result.WillClosePR {
		t.Error("Expected WillClosePR to be true for R6 (full rollback)")
	}
	if !result.WillClearDesign {
		t.Error("Expected WillClearDesign to be true for R6")
	}
}

func TestEvaluateRollback_PRRollbackCountLimit(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := TransitionContext{
		AllowPRRollback:    true,
		MaxPRRollbackCount: 2,
	}

	// Create session in PullRequest stage with PR number and max rollback count
	session := createTestSessionAtStage(valueobject.StagePullRequest)
	session.PRNumber = new(42)
	session.PRRollbackCount = 2 // Already at max

	result := service.EvaluateRollback(session, valueobject.StageExecution, ctx)

	if result.Decision != RollbackDenied {
		t.Errorf("Expected RollbackDenied, got %v", result.Decision)
	}
}

func TestEvaluateRollback_PRRollbackDisabled(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := TransitionContext{
		AllowPRRollback:    false,
		MaxPRRollbackCount: 3,
	}

	// Create session in PullRequest stage
	session := createTestSessionAtStage(valueobject.StagePullRequest)
	session.PRNumber = new(42)

	result := service.EvaluateRollback(session, valueobject.StageExecution, ctx)

	if result.Decision != RollbackDenied {
		t.Errorf("Expected RollbackDenied, got %v", result.Decision)
	}
}

func TestEvaluateRollback_InvalidRollbackPath(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)
	ctx := TransitionContext{}

	// Create session in Clarification stage (cannot rollback)
	session := createTestSessionAtStage(valueobject.StageClarification)

	result := service.EvaluateRollback(session, valueobject.StageDesign, ctx)

	if result.Decision != RollbackDenied {
		t.Errorf("Expected RollbackDenied for invalid rollback path, got %v", result.Decision)
	}
}

func TestGetRollbackRule_AllRules(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)

	tests := []struct {
		current  valueobject.Stage
		target   valueobject.Stage
		expected string
	}{
		{valueobject.StageExecution, valueobject.StageDesign, "R1"},
		{valueobject.StageDesign, valueobject.StageClarification, "R2"},
		{valueobject.StageTaskBreakdown, valueobject.StageClarification, "R2"},
		{valueobject.StageExecution, valueobject.StageClarification, "R3"},
		{valueobject.StagePullRequest, valueobject.StageExecution, "R4"},
		{valueobject.StagePullRequest, valueobject.StageDesign, "R5"},
		{valueobject.StagePullRequest, valueobject.StageClarification, "R6"},
		// TaskBreakdown -> Design is a non-standard path (R2b: keep tasks)
		{valueobject.StageTaskBreakdown, valueobject.StageDesign, "R2b"},
	}

	for _, tt := range tests {
		result := service.GetRollbackRule(tt.current, tt.target)
		if result != tt.expected {
			t.Errorf("GetRollbackRule(%s, %s) = %s, expected %s",
				tt.current, tt.target, result, tt.expected)
		}
	}
}

func TestDetermineRollbackType(t *testing.T) {
	service := NewDefaultStageTransitionService(nil)

	// RollbackType is determined by rollback rule as per state-machine.md:
	//   R1, R5 -> Deep
	//   R2, R4 -> Shallow
	//   R3, R6 -> Full
	tests := []struct {
		current  valueobject.Stage
		target   valueobject.Stage
		expected RollbackType
		rule     string
	}{
		// R1: Execution -> Design -> Deep
		{valueobject.StageExecution, valueobject.StageDesign, RollbackTypeDeep, "R1"},
		// R2: Design -> Clarification -> Shallow
		{valueobject.StageDesign, valueobject.StageClarification, RollbackTypeShallow, "R2"},
		// R2: TaskBreakdown -> Clarification (uses R2 rules) -> Shallow
		{valueobject.StageTaskBreakdown, valueobject.StageClarification, RollbackTypeShallow, "R2"},
		// R3: Execution -> Clarification -> Full
		{valueobject.StageExecution, valueobject.StageClarification, RollbackTypeFull, "R3"},
		// R4: PullRequest -> Execution -> Shallow
		{valueobject.StagePullRequest, valueobject.StageExecution, RollbackTypeShallow, "R4"},
		// R5: PullRequest -> Design -> Deep
		{valueobject.StagePullRequest, valueobject.StageDesign, RollbackTypeDeep, "R5"},
		// R6: PullRequest -> Clarification -> Full
		{valueobject.StagePullRequest, valueobject.StageClarification, RollbackTypeFull, "R6"},
	}

	for _, tt := range tests {
		// Test rollback type indirectly through EvaluateRollback
		session := createTestSessionAtStage(tt.current)
		if tt.current == valueobject.StagePullRequest {
			session.PRNumber = new(42)
		}

		ctx := TransitionContext{
			AllowPRRollback:    true,
			MaxPRRollbackCount: 3,
		}

		result := service.EvaluateRollback(session, tt.target, ctx)
		if result.RollbackType != tt.expected {
			t.Errorf("RollbackType for %s -> %s (rule %s) = %v, expected %v",
				tt.current, tt.target, tt.rule, result.RollbackType, tt.expected)
		}
		if result.RollbackRule != tt.rule {
			t.Errorf("RollbackRule for %s -> %s = %s, expected %s",
				tt.current, tt.target, result.RollbackRule, tt.rule)
		}
	}
}

func TestRollbackResult_Methods(t *testing.T) {
	// Test IsAllowed
	allowed := RollbackResult{Decision: RollbackAllowed}
	if !allowed.IsAllowed() {
		t.Error("IsAllowed() should return true for RollbackAllowed")
	}
	if allowed.NeedsConfirmation() {
		t.Error("NeedsConfirmation() should return false for RollbackAllowed")
	}
	if allowed.IsDenied() {
		t.Error("IsDenied() should return false for RollbackAllowed")
	}

	// Test NeedsConfirmation
	needConfirm := RollbackResult{Decision: RollbackNeedConfirmation}
	if needConfirm.IsAllowed() {
		t.Error("IsAllowed() should return false for RollbackNeedConfirmation")
	}
	if !needConfirm.NeedsConfirmation() {
		t.Error("NeedsConfirmation() should return true for RollbackNeedConfirmation")
	}

	// Test IsDenied
	denied := RollbackResult{Decision: RollbackDenied}
	if denied.IsAllowed() {
		t.Error("IsAllowed() should return false for RollbackDenied")
	}
	if !denied.IsDenied() {
		t.Error("IsDenied() should return true for RollbackDenied")
	}
}

// Helper function to create a test session at a specific stage
func createTestSessionAtStage(stage valueobject.Stage) *aggregate.WorkSession {
	issue := entity.NewIssue(1, "Test Issue", "Test Body", "owner/repo", "test-author")

	session := &aggregate.WorkSession{
		ID:            uuid.New(),
		Issue:         issue,
		CurrentStage:  stage,
		SessionStatus: aggregate.SessionStatusActive,
	}

	// Add necessary entities based on stage
	if stage != valueobject.StageClarification {
		session.Clarification = entity.NewClarification()
	}
	if stage == valueobject.StageDesign || stage == valueobject.StageTaskBreakdown ||
		stage == valueobject.StageExecution || stage == valueobject.StagePullRequest {
		session.Design = entity.NewDesign("test design content")
	}
	if stage == valueobject.StageExecution || stage == valueobject.StagePullRequest {
		session.Execution = entity.NewExecution("/tmp/worktree", "test-branch")
	}

	return session
}

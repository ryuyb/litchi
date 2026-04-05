package valueobject

import (
	"testing"
	"time"
)

func TestPauseReason_AllReasonsValid(t *testing.T) {
	reasons := AllPauseReasons()
	if len(reasons) != 15 {
		t.Errorf("Expected 15 pause reasons, got %d", len(reasons))
	}

	for _, reason := range reasons {
		if !reason.IsValid() {
			t.Errorf("PauseReason %s should be valid", reason)
		}
	}
}

func TestPauseReason_RecoveryCategory(t *testing.T) {
	tests := []struct {
		reason   PauseReason
		expected RecoveryCategory
	}{
		// Auto recovery
		{PauseReasonRateLimited, RecoveryAuto},
		{PauseReasonResourceExhausted, RecoveryAuto},
		{PauseReasonServiceRestart, RecoveryAuto}, // Service auto-recovers on startup

		// Semi-auto recovery
		{PauseReasonAgentCrashed, RecoverySemiAuto},
		{PauseReasonTestEnvUnavailable, RecoverySemiAuto},
		{PauseReasonBudgetExceeded, RecoverySemiAuto},

		// Manual intervention required
		{PauseReasonUserRequest, RecoveryManual},
		{PauseReasonTaskFailed, RecoveryManual},
		{PauseReasonApprovalPending, RecoveryManual},
		{PauseReasonExternalError, RecoveryManual},
		{PauseReasonPRReviewPending, RecoveryManual},
		{PauseReasonCIFailure, RecoveryManual},
		{PauseReasonTimeout, RecoveryManual},
		{PauseReasonSessionLost, RecoveryManual},
		{PauseReasonOther, RecoveryManual}, // Unknown reasons require manual intervention
	}

	for _, tt := range tests {
		result := tt.reason.RecoveryCategory()
		if result != tt.expected {
			t.Errorf("%s.RecoveryCategory() = %v, expected %v", tt.reason, result, tt.expected)
		}
	}
}

func TestPauseReason_CanAutoRecover(t *testing.T) {
	// Auto recovery reasons
	if !PauseReasonRateLimited.CanAutoRecover() {
		t.Error("PauseReasonRateLimited should support auto-recovery")
	}
	if !PauseReasonResourceExhausted.CanAutoRecover() {
		t.Error("PauseReasonResourceExhausted should support auto-recovery")
	}
	if !PauseReasonServiceRestart.CanAutoRecover() {
		t.Error("PauseReasonServiceRestart should support auto-recovery")
	}

	// Manual intervention reasons
	if PauseReasonUserRequest.CanAutoRecover() {
		t.Error("PauseReasonUserRequest should NOT support auto-recovery")
	}
	if PauseReasonTaskFailed.CanAutoRecover() {
		t.Error("PauseReasonTaskFailed should NOT support auto-recovery")
	}
}

func TestPauseReason_RecoveryActions(t *testing.T) {
	tests := []struct {
		reason          PauseReason
		expectedActions []string
	}{
		{PauseReasonUserRequest, []string{"admin_continue"}},
		{PauseReasonTaskFailed, []string{"admin_continue", "admin_skip", "admin_rollback"}},
		{PauseReasonApprovalPending, []string{"admin_approve", "admin_reject"}},
		{PauseReasonRateLimited, []string{"auto_wait", "admin_force"}},
	}

	for _, tt := range tests {
		actions := tt.reason.RecoveryActions()
		if len(actions) != len(tt.expectedActions) {
			t.Errorf("%s.RecoveryActions() returned %d actions, expected %d",
				tt.reason, len(actions), len(tt.expectedActions))
			continue
		}
		for i, expected := range tt.expectedActions {
			if actions[i] != expected {
				t.Errorf("%s.RecoveryActions()[%d] = %s, expected %s",
					tt.reason, i, actions[i], expected)
			}
		}
	}
}

func TestPauseReason_DisplayName(t *testing.T) {
	tests := []struct {
		reason   PauseReason
		expected string
	}{
		{PauseReasonUserRequest, "User Request"},
		{PauseReasonTaskFailed, "Task Failed"},
		{PauseReasonRateLimited, "Rate Limited"},
		{PauseReasonAgentCrashed, "Agent Crashed"},
	}

	for _, tt := range tests {
		result := tt.reason.DisplayName()
		if result != tt.expected {
			t.Errorf("%s.DisplayName() = %s, expected %s", tt.reason, result, tt.expected)
		}
	}
}

func TestPauseReason_Description(t *testing.T) {
	// Just verify it returns non-empty strings for known reasons
	reasons := []PauseReason{
		PauseReasonUserRequest,
		PauseReasonTaskFailed,
		PauseReasonRateLimited,
	}

	for _, reason := range reasons {
		desc := reason.Description()
		if desc == "" {
			t.Errorf("%s.Description() should not be empty", reason)
		}
	}
}

func TestParsePauseReason(t *testing.T) {
	tests := []struct {
		input    string
		expected PauseReason
		hasError bool
	}{
		{"user_request", PauseReasonUserRequest, false},
		{"task_failed", PauseReasonTaskFailed, false},
		{"rate_limited", PauseReasonRateLimited, false},
		{"invalid_reason", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		result, err := ParsePauseReason(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParsePauseReason(%s) should return error", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParsePauseReason(%s) returned unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ParsePauseReason(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		}
	}
}

func TestPauseContext_WithMethods(t *testing.T) {
	ctx := NewPauseContext(PauseReasonTaskFailed)

	// Test WithPausedBy
	ctx = ctx.WithPausedBy("admin-user")
	if ctx.PausedBy != "admin-user" {
		t.Errorf("WithPausedBy failed: got %s", ctx.PausedBy)
	}

	// Test WithRelatedTask
	ctx = ctx.WithRelatedTask("task-123")
	if ctx.RelatedTaskID != "task-123" {
		t.Errorf("WithRelatedTask failed: got %s", ctx.RelatedTaskID)
	}

	// Test WithRelatedPR
	ctx = ctx.WithRelatedPR(42)
	if ctx.RelatedPRNumber != 42 {
		t.Errorf("WithRelatedPR failed: got %d", ctx.RelatedPRNumber)
	}

	// Test WithErrorDetails
	ctx = ctx.WithErrorDetails("test error")
	if ctx.ErrorDetails != "test error" {
		t.Errorf("WithErrorDetails failed: got %s", ctx.ErrorDetails)
	}
}

func TestPauseContext_WithAutoResume(t *testing.T) {
	ctx := NewPauseContext(PauseReasonRateLimited)
	future := time.Now().Add(30 * time.Minute)

	ctx = ctx.WithAutoResume(future, "API reset")

	if ctx.AutoResumeAfter == nil {
		t.Error("WithAutoResume should set AutoResumeAfter")
	}
	if ctx.ResumeCondition != "API reset" {
		t.Errorf("ResumeCondition = %s, expected 'API reset'", ctx.ResumeCondition)
	}
}

func TestPauseContext_CanAutoResumeNow(t *testing.T) {
	// Test with auto-recovery reason and future time
	future := time.Now().Add(30 * time.Minute)
	ctx := NewPauseContext(PauseReasonRateLimited).WithAutoResume(future, "API reset")

	if ctx.CanAutoResumeNow() {
		t.Error("Should not auto-resume before scheduled time")
	}

	// Test with past time
	past := time.Now().Add(-1 * time.Minute)
	ctx = NewPauseContext(PauseReasonRateLimited).WithAutoResume(past, "API reset")

	if !ctx.CanAutoResumeNow() {
		t.Error("Should auto-resume after scheduled time")
	}

	// Test with non-auto-recovery reason
	manualCtx := NewPauseContext(PauseReasonUserRequest)
	if manualCtx.CanAutoResumeNow() {
		t.Error("Manual recovery reason should not allow auto-resume")
	}
}

func TestPauseRecord_Complete(t *testing.T) {
	ctx := NewPauseContext(PauseReasonUserRequest)
	record := NewPauseRecord(ctx)

	// Before completion
	if record.ResumedAt != nil {
		t.Error("ResumedAt should be nil before completion")
	}
	if record.Duration != 0 {
		t.Error("Duration should be 0 before completion")
	}

	// Complete the record
	record.Complete("admin_continue")

	// After completion
	if record.ResumedAt == nil {
		t.Error("ResumedAt should be set after completion")
	}
	// Duration can be 0 if the test runs very fast, so check >= 0
	if record.Duration < 0 {
		t.Errorf("Duration should be >= 0 after completion, got %d", record.Duration)
	}
	if record.ResumeAction != "admin_continue" {
		t.Errorf("ResumeAction = %s, expected 'admin_continue'", record.ResumeAction)
	}
}

func TestPauseReason_GORM_Value(t *testing.T) {
	reason := PauseReasonUserRequest
	value, err := reason.Value()
	if err != nil {
		t.Errorf("Value() returned error: %v", err)
	}
	if value != "user_request" {
		t.Errorf("Value() = %v, expected 'user_request'", value)
	}

	// Test invalid reason
	invalidReason := PauseReason("invalid")
	_, err = invalidReason.Value()
	if err == nil {
		t.Error("Value() should return error for invalid reason")
	}
}

func TestPauseReason_GORM_Scan(t *testing.T) {
	var reason PauseReason

	// Test string
	err := reason.Scan("user_request")
	if err != nil {
		t.Errorf("Scan(string) returned error: %v", err)
	}
	if reason != PauseReasonUserRequest {
		t.Errorf("Scan result = %s, expected user_request", reason)
	}

	// Test []byte
	err = reason.Scan([]byte("task_failed"))
	if err != nil {
		t.Errorf("Scan([]byte) returned error: %v", err)
	}
	if reason != PauseReasonTaskFailed {
		t.Errorf("Scan result = %s, expected task_failed", reason)
	}

	// Test nil
	err = reason.Scan(nil)
	if err == nil {
		t.Error("Scan(nil) should return error")
	}

	// Test invalid value
	err = reason.Scan("invalid_reason")
	if err == nil {
		t.Error("Scan(invalid) should return error")
	}
}

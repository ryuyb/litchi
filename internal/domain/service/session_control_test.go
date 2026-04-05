package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

func TestSessionControlService_PauseWithContext(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	ctx := valueobject.NewPauseContext(valueobject.PauseReasonUserRequest).
		WithPausedBy("admin-user")

	err := service.PauseSession(session, ctx)
	if err != nil {
		t.Errorf("PauseSession returned error: %v", err)
	}

	if session.SessionStatus != aggregate.SessionStatusPaused {
		t.Error("Session should be paused")
	}

	if session.GetPauseContext() == nil {
		t.Error("PauseContext should be set")
	}

	if session.GetPauseContext().Reason != valueobject.PauseReasonUserRequest {
		t.Errorf("PauseReason = %s, expected user_request", session.GetPauseContext().Reason)
	}
}

func TestSessionControlService_PauseSession_InvalidStatus(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Pause the session
	ctx := valueobject.NewPauseContext(valueobject.PauseReasonUserRequest)
	_ = service.PauseSession(session, ctx)

	// Try to pause again
	err := service.PauseSession(session, ctx)
	if err == nil {
		t.Error("Should not be able to pause an already paused session")
	}
}

func TestSessionControlService_ResumeWithAction(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Pause first
	ctx := valueobject.NewPauseContext(valueobject.PauseReasonUserRequest)
	_ = service.PauseSession(session, ctx)

	// Resume with valid action
	err := service.ResumeSession(session, "admin_continue")
	if err != nil {
		t.Errorf("ResumeSession returned error: %v", err)
	}

	if session.SessionStatus != aggregate.SessionStatusActive {
		t.Error("Session should be active after resume")
	}

	if session.GetPauseContext() != nil {
		t.Error("PauseContext should be cleared after resume")
	}

	// Check pause history
	history := session.GetPauseHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 pause record, got %d", len(history))
	}
}

func TestSessionControlService_ResumeWithAction_InvalidAction(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Pause with task_failed reason (requires specific actions)
	ctx := valueobject.NewPauseContext(valueobject.PauseReasonTaskFailed)
	_ = service.PauseSession(session, ctx)

	// Try to resume with invalid action
	err := service.ResumeSession(session, "invalid_action")
	if err == nil {
		t.Error("Should not be able to resume with invalid action")
	}
}

func TestSessionControlService_AutoResume(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Pause with auto-recovery reason
	pastTime := time.Now().Add(-1 * time.Minute)
	ctx := valueobject.NewPauseContext(valueobject.PauseReasonRateLimited).
		WithAutoResume(pastTime, "API reset")
	_ = service.PauseSession(session, ctx)

	// Auto-resume should work
	resumed, err := service.AutoResumeSession(session)
	if err != nil {
		t.Errorf("AutoResumeSession returned error: %v", err)
	}
	if !resumed {
		t.Error("AutoResumeSession should have resumed the session")
	}

	if session.SessionStatus != aggregate.SessionStatusActive {
		t.Error("Session should be active after auto-resume")
	}
}

func TestSessionControlService_AutoResume_NotReady(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Pause with auto-recovery reason but future time
	futureTime := time.Now().Add(30 * time.Minute)
	ctx := valueobject.NewPauseContext(valueobject.PauseReasonRateLimited).
		WithAutoResume(futureTime, "API reset")
	_ = service.PauseSession(session, ctx)

	// Auto-resume should not work yet
	resumed, err := service.AutoResumeSession(session)
	if err != nil {
		t.Errorf("AutoResumeSession returned error: %v", err)
	}
	if resumed {
		t.Error("AutoResumeSession should not have resumed (time not reached)")
	}

	if session.SessionStatus != aggregate.SessionStatusPaused {
		t.Error("Session should still be paused")
	}
}

func TestSessionControlService_AutoResume_ManualReason(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Pause with manual recovery reason
	ctx := valueobject.NewPauseContext(valueobject.PauseReasonUserRequest)
	_ = service.PauseSession(session, ctx)

	// Auto-resume should not work for manual reasons
	resumed, err := service.AutoResumeSession(session)
	if err != nil {
		t.Errorf("AutoResumeSession returned error: %v", err)
	}
	if resumed {
		t.Error("AutoResumeSession should not have resumed for manual reason")
	}
}

func TestSessionControlService_TerminateSession(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	err := service.TerminateSession(session, "user requested termination")
	if err != nil {
		t.Errorf("TerminateSession returned error: %v", err)
	}

	if session.SessionStatus != aggregate.SessionStatusTerminated {
		t.Error("Session should be terminated")
	}
}

func TestSessionControlService_CanResumeWithAction(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Test with task_failed pause reason
	ctx := valueobject.NewPauseContext(valueobject.PauseReasonTaskFailed)
	_ = service.PauseSession(session, ctx)

	tests := []struct {
		action   string
		expected bool
	}{
		{"admin_continue", true},
		{"admin_skip", true},
		{"admin_rollback", true},
		{"admin_force", true}, // admin_force is always allowed
		{"invalid_action", false},
	}

	for _, tt := range tests {
		result := service.CanResumeWithAction(session, tt.action)
		if result != tt.expected {
			t.Errorf("CanResumeWithAction(%s) = %v, expected %v", tt.action, result, tt.expected)
		}
	}
}

func TestSessionControlService_GetValidResumeActions(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Test with approval_pending reason
	ctx := valueobject.NewPauseContext(valueobject.PauseReasonApprovalPending)
	_ = service.PauseSession(session, ctx)

	actions := service.GetValidResumeActions(session)

	expectedActions := []string{"admin_approve", "admin_reject", "admin_force"}
	if len(actions) != len(expectedActions) {
		t.Errorf("Expected %d actions, got %d", len(expectedActions), len(actions))
	}

	// Verify admin_force is always included
	hasAdminForce := false
	for _, action := range actions {
		if action == "admin_force" {
			hasAdminForce = true
			break
		}
	}
	if !hasAdminForce {
		t.Error("admin_force should always be in valid actions")
	}
}

func TestSessionControlService_CanResumeWithAction_NotPaused(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Session is active, not paused
	if service.CanResumeWithAction(session, "admin_continue") {
		t.Error("Should not be able to resume a non-paused session")
	}
}

func TestSessionControlService_CanResumeWithAction_NilPauseContext(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Manually set session to paused without PauseContext (abnormal state)
	session.SessionStatus = aggregate.SessionStatusPaused
	session.PauseContext = nil

	// Only admin_force should be allowed when PauseContext is nil
	if service.CanResumeWithAction(session, "admin_continue") {
		t.Error("Should not allow admin_continue when PauseContext is nil")
	}
	if service.CanResumeWithAction(session, "admin_skip") {
		t.Error("Should not allow admin_skip when PauseContext is nil")
	}
	if !service.CanResumeWithAction(session, "admin_force") {
		t.Error("Should allow admin_force when PauseContext is nil")
	}
}

func TestSessionControlService_GetValidResumeActions_NilPauseContext(t *testing.T) {
	service := NewDefaultSessionControlService()
	session := createTestSessionForControl()

	// Manually set session to paused without PauseContext (abnormal state)
	session.SessionStatus = aggregate.SessionStatusPaused
	session.PauseContext = nil

	actions := service.GetValidResumeActions(session)

	// Should only return admin_force
	if len(actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions))
	}
	if len(actions) > 0 && actions[0] != "admin_force" {
		t.Errorf("Expected admin_force, got %s", actions[0])
	}
}

// Helper function to create a test session for session control tests
func createTestSessionForControl() *aggregate.WorkSession {
	issue := entity.NewIssue(1, "Test Issue", "Test Body", "owner/repo", "test-author")

	return &aggregate.WorkSession{
		ID:            uuid.New(),
		Issue:         issue,
		CurrentStage:  valueobject.StageExecution,
		SessionStatus: aggregate.SessionStatusActive,
		Execution:     entity.NewExecution("/tmp/worktree", "test-branch"),
	}
}

package service

import (
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// SessionControlService handles pause/resume/terminate operations for WorkSessions.
// It provides intelligent pause/recovery management including auto-recovery support.
//
// This is a domain service that coordinates session control operations with
// validation of recovery actions and automatic recovery conditions.
type SessionControlService interface {
	// PauseSession pauses a session with detailed context.
	PauseSession(session *aggregate.WorkSession, ctx valueobject.PauseContext) error

	// ResumeSession resumes a paused session with the specified action.
	// Returns error if the action is not valid for the current pause reason.
	ResumeSession(session *aggregate.WorkSession, action string) error

	// AutoResumeSession attempts automatic resume if conditions are met.
	// Returns (true, nil) if auto-resume was performed.
	// Returns (false, nil) if auto-resume conditions are not met.
	// Returns (false, error) if auto-resume failed.
	AutoResumeSession(session *aggregate.WorkSession) (bool, error)

	// TerminateSession terminates a session.
	// Cleanup is handled at application layer (closing PRs, archiving branches).
	TerminateSession(session *aggregate.WorkSession, reason string) error

	// CanResumeWithAction checks if a specific action can resume the session.
	CanResumeWithAction(session *aggregate.WorkSession, action string) bool

	// GetValidResumeActions returns valid resume actions for the current pause context.
	GetValidResumeActions(session *aggregate.WorkSession) []string
}

// DefaultSessionControlService implements SessionControlService.
type DefaultSessionControlService struct{}

// NewDefaultSessionControlService creates a new DefaultSessionControlService.
func NewDefaultSessionControlService() *DefaultSessionControlService {
	return &DefaultSessionControlService{}
}

// PauseSession pauses a session with detailed context.
func (s *DefaultSessionControlService) PauseSession(
	session *aggregate.WorkSession,
	ctx valueobject.PauseContext,
) error {
	return session.PauseWithContext(ctx)
}

// ResumeSession resumes a paused session with the specified action.
func (s *DefaultSessionControlService) ResumeSession(
	session *aggregate.WorkSession,
	action string,
) error {
	if !s.CanResumeWithAction(session, action) {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"invalid resume action for pause reason: " + action,
		)
	}
	return session.ResumeWithAction(action)
}

// AutoResumeSession attempts automatic resume if conditions are met.
func (s *DefaultSessionControlService) AutoResumeSession(
	session *aggregate.WorkSession,
) (bool, error) {
	if !session.CanAutoResume() {
		return false, nil
	}

	// Perform auto-resume with "auto_resume" action
	// The ResumeWithAction method will record the appropriate event
	err := session.ResumeWithAction("auto_resume")
	if err != nil {
		return false, err
	}

	return true, nil
}

// TerminateSession terminates a session.
func (s *DefaultSessionControlService) TerminateSession(
	session *aggregate.WorkSession,
	reason string,
) error {
	return session.Terminate(reason)
}

// CanResumeWithAction checks if a specific action can resume the session.
func (s *DefaultSessionControlService) CanResumeWithAction(
	session *aggregate.WorkSession,
	action string,
) bool {
	if session.SessionStatus != aggregate.SessionStatusPaused {
		return false
	}

	pauseContext := session.GetPauseContext()
	if pauseContext == nil {
		// PauseContext is empty, which is an abnormal state (data incomplete or legacy pause).
		// Only allow admin_force to resume in this case for safety.
		return action == "admin_force"
	}

	// Check if action is valid for this pause reason
	validActions := pauseContext.Reason.RecoveryActions()
	for _, validAction := range validActions {
		if action == validAction {
			return true
		}
	}

	// Admin can always force resume
	if action == "admin_force" {
		return true
	}

	return false
}

// GetValidResumeActions returns valid resume actions for the current pause context.
func (s *DefaultSessionControlService) GetValidResumeActions(
	session *aggregate.WorkSession,
) []string {
	pauseContext := session.GetPauseContext()
	if pauseContext == nil {
		// PauseContext is empty (abnormal state), only admin_force is allowed
		return []string{"admin_force"}
	}

	actions := pauseContext.Reason.RecoveryActions()
	// Always add admin_force as fallback option for admins
	actions = append(actions, "admin_force")
	return actions
}

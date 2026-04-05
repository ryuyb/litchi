package valueobject

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// PauseReason represents the reason for pausing a work session.
// There are 14 defined pause reasons categorized by recovery mechanism.
type PauseReason string

// Pause reason constants as defined in state-machine design document.
const (
	// --- Manual Intervention Required ---
	PauseReasonUserRequest     PauseReason = "user_request"      // Repository admin manually paused
	PauseReasonTaskFailed      PauseReason = "task_failed"       // Task execution failed, waiting for instruction
	PauseReasonApprovalPending PauseReason = "approval_pending"  // Waiting for approval on dangerous operation
	PauseReasonExternalError   PauseReason = "external_error"    // Non-rate-limit external errors (service unavailable, DNS failure, network timeout)
	PauseReasonServiceRestart  PauseReason = "service_restart"   // Session interrupted during service restart
	PauseReasonPRReviewPending PauseReason = "pr_review_pending" // Waiting for PR review feedback
	PauseReasonCIFailure       PauseReason = "ci_failure"        // CI check failed, waiting for resolution
	PauseReasonTimeout         PauseReason = "timeout"           // Operation exceeded time limit
	PauseReasonSessionLost     PauseReason = "session_lost"      // Session context lost, needs restart
	PauseReasonOther           PauseReason = "other"             // Unknown or custom pause reason (backward compatibility)

	// --- Semi-Automatic Recovery ---
	PauseReasonAgentCrashed       PauseReason = "agent_crashed"        // Agent process crashed unexpectedly
	PauseReasonTestEnvUnavailable PauseReason = "test_env_unavailable" // Test environment is not available
	PauseReasonBudgetExceeded     PauseReason = "budget_exceeded"      // Token/budget limit exceeded

	// --- Automatic Recovery ---
	PauseReasonRateLimited       PauseReason = "rate_limited"       // GitHub API rate limit reached (auto-wait for reset)
	PauseReasonResourceExhausted PauseReason = "resource_exhausted" // System concurrent resources exhausted (auto-queue)
)

// AllPauseReasons returns all valid pause reasons.
func AllPauseReasons() []PauseReason {
	return []PauseReason{
		PauseReasonUserRequest,
		PauseReasonTaskFailed,
		PauseReasonApprovalPending,
		PauseReasonExternalError,
		PauseReasonServiceRestart,
		PauseReasonPRReviewPending,
		PauseReasonCIFailure,
		PauseReasonAgentCrashed,
		PauseReasonRateLimited,
		PauseReasonTestEnvUnavailable,
		PauseReasonTimeout,
		PauseReasonResourceExhausted,
		PauseReasonBudgetExceeded,
		PauseReasonSessionLost,
		PauseReasonOther,
	}
}

// RecoveryCategory classifies how a pause can be recovered.
type RecoveryCategory int

const (
	// RecoveryAuto indicates automatic recovery without human intervention.
	// System will automatically resume when conditions are met.
	RecoveryAuto RecoveryCategory = iota

	// RecoverySemiAuto indicates semi-automatic recovery.
	// System performs automatic checks but may require user confirmation.
	RecoverySemiAuto

	// RecoveryManual indicates manual intervention is required.
	// An admin must explicitly resume the session.
	RecoveryManual
)

// ParsePauseReason parses a string into a PauseReason.
func ParsePauseReason(reasonStr string) (PauseReason, error) {
	for _, reason := range AllPauseReasons() {
		if string(reason) == reasonStr {
			return reason, nil
		}
	}
	return "", fmt.Errorf("invalid pause reason: %s", reasonStr)
}

// IsValid checks if the pause reason is valid.
func (pr PauseReason) IsValid() bool {
	for _, reason := range AllPauseReasons() {
		if pr == reason {
			return true
		}
	}
	return false
}

// RecoveryCategory returns the recovery category for this pause reason.
func (pr PauseReason) RecoveryCategory() RecoveryCategory {
	switch pr {
	case PauseReasonRateLimited, PauseReasonResourceExhausted, PauseReasonServiceRestart:
		return RecoveryAuto
	case PauseReasonAgentCrashed, PauseReasonTestEnvUnavailable, PauseReasonBudgetExceeded:
		return RecoverySemiAuto
	default:
		return RecoveryManual
	}
}

// CanAutoRecover returns true if this pause reason supports automatic recovery.
func (pr PauseReason) CanAutoRecover() bool {
	return pr.RecoveryCategory() == RecoveryAuto
}

// NeedsManualIntervention returns true if manual intervention is required.
func (pr PauseReason) NeedsManualIntervention() bool {
	return pr.RecoveryCategory() == RecoveryManual
}

// RecoveryActions returns the possible recovery actions for this pause reason.
// These actions are used to guide the user on available options.
func (pr PauseReason) RecoveryActions() []string {
	switch pr {
	case PauseReasonUserRequest:
		return []string{"admin_continue"}
	case PauseReasonTaskFailed:
		return []string{"admin_continue", "admin_skip", "admin_rollback"}
	case PauseReasonApprovalPending:
		return []string{"admin_approve", "admin_reject"}
	case PauseReasonExternalError:
		return []string{"admin_retry", "admin_force"}
	case PauseReasonServiceRestart:
		return []string{"auto_resume"} // Service auto-recovers on startup
	case PauseReasonPRReviewPending:
		return []string{"admin_continue", "admin_rollback"}
	case PauseReasonCIFailure:
		return []string{"admin_fix", "admin_rollback"}
	case PauseReasonAgentCrashed:
		return []string{"admin_continue_with_recovery"}
	case PauseReasonRateLimited:
		return []string{"auto_wait", "admin_force"}
	case PauseReasonTestEnvUnavailable:
		return []string{"env_restore", "admin_force"}
	case PauseReasonTimeout:
		return []string{"admin_continue", "admin_cancel"}
	case PauseReasonResourceExhausted:
		return []string{"auto_queue", "admin_cancel"}
	case PauseReasonBudgetExceeded:
		return []string{"admin_increase_budget", "admin_switch_model"}
	case PauseReasonSessionLost:
		return []string{"admin_restart_session"}
	case PauseReasonOther:
		return []string{"admin_force"} // Unknown reasons require admin intervention
	default:
		return []string{}
	}
}

// DisplayName returns the user-friendly display name for UI.
func (pr PauseReason) DisplayName() string {
	switch pr {
	case PauseReasonUserRequest:
		return "User Request"
	case PauseReasonTaskFailed:
		return "Task Failed"
	case PauseReasonApprovalPending:
		return "Approval Pending"
	case PauseReasonExternalError:
		return "External Error"
	case PauseReasonServiceRestart:
		return "Service Restart"
	case PauseReasonPRReviewPending:
		return "PR Review Pending"
	case PauseReasonCIFailure:
		return "CI Failure"
	case PauseReasonAgentCrashed:
		return "Agent Crashed"
	case PauseReasonRateLimited:
		return "Rate Limited"
	case PauseReasonTestEnvUnavailable:
		return "Test Environment Unavailable"
	case PauseReasonTimeout:
		return "Timeout"
	case PauseReasonResourceExhausted:
		return "Resource Exhausted"
	case PauseReasonBudgetExceeded:
		return "Budget Exceeded"
	case PauseReasonSessionLost:
		return "Session Lost"
	case PauseReasonOther:
		return "Other"
	default:
		return "Unknown"
	}
}

// Description returns a detailed description of the pause reason.
func (pr PauseReason) Description() string {
	switch pr {
	case PauseReasonUserRequest:
		return "Repository admin manually paused the session"
	case PauseReasonTaskFailed:
		return "Task execution failed, waiting for admin instruction (continue/skip/rollback)"
	case PauseReasonApprovalPending:
		return "Waiting for approval on dangerous operation"
	case PauseReasonExternalError:
		return "External service error (service unavailable, DNS failure, network timeout). Note: For API rate limits, use rate_limited instead."
	case PauseReasonServiceRestart:
		return "Session interrupted during service restart, will auto-resume on startup"
	case PauseReasonPRReviewPending:
		return "Waiting for PR review feedback"
	case PauseReasonCIFailure:
		return "CI check failed, waiting for resolution (fix or rollback)"
	case PauseReasonAgentCrashed:
		return "Agent process crashed unexpectedly, needs recovery attempt"
	case PauseReasonRateLimited:
		return "GitHub API rate limit reached, will auto-resume after reset time"
	case PauseReasonTestEnvUnavailable:
		return "Test environment is not available"
	case PauseReasonTimeout:
		return "Operation exceeded time limit"
	case PauseReasonResourceExhausted:
		return "System concurrent resources exhausted (queue full)"
	case PauseReasonBudgetExceeded:
		return "Token/budget limit exceeded"
	case PauseReasonSessionLost:
		return "Session context lost, needs restart"
	case PauseReasonOther:
		return "Unknown or custom pause reason, requires admin intervention. Original reason stored in ErrorDetails."
	default:
		return "Unknown pause reason"
	}
}

// GORM serialization methods for database persistence.

// Value converts PauseReason to a database value.
func (pr PauseReason) Value() (driver.Value, error) {
	if !pr.IsValid() {
		return nil, fmt.Errorf("cannot serialize invalid pause reason: %s", pr)
	}
	return string(pr), nil
}

// Scan converts a database value to PauseReason.
func (pr *PauseReason) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("pause reason cannot be null")
	}

	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("cannot scan pause reason from type: %T", value)
	}

	reason, err := ParsePauseReason(str)
	if err != nil {
		return err
	}
	*pr = reason
	return nil
}

// PauseContext provides detailed context for a paused session.
// It captures why the session was paused and relevant context for recovery.
type PauseContext struct {
	// Reason is the primary pause reason.
	Reason PauseReason `json:"reason"`

	// PausedAt is the timestamp when the session was paused.
	PausedAt time.Time `json:"pausedAt"`

	// PausedBy is the actor who initiated the pause (for user_request).
	PausedBy string `json:"pausedBy,omitempty"`

	// RelatedTaskID is the task that caused the pause (for task_failed).
	RelatedTaskID string `json:"relatedTaskId,omitempty"`

	// RelatedPRNumber is the PR number involved (for pr_review_pending, ci_failure).
	RelatedPRNumber int `json:"relatedPrNumber,omitempty"`

	// ErrorDetails contains additional error information (for external_error, agent_crashed).
	ErrorDetails string `json:"errorDetails,omitempty"`

	// AutoResumeAfter is the estimated time for auto-resume (for rate_limited, resource_exhausted).
	AutoResumeAfter *time.Time `json:"autoResumeAfter,omitempty"`

	// ResumeCondition describes the condition for auto-resume.
	ResumeCondition string `json:"resumeCondition,omitempty"`
}

// NewPauseContext creates a new PauseContext with the given reason.
func NewPauseContext(reason PauseReason) PauseContext {
	return PauseContext{
		Reason:   reason,
		PausedAt: time.Now(),
	}
}

// WithPausedBy sets the actor who paused the session.
func (pc PauseContext) WithPausedBy(actor string) PauseContext {
	pc.PausedBy = actor
	return pc
}

// WithRelatedTask sets the related task ID.
func (pc PauseContext) WithRelatedTask(taskID string) PauseContext {
	pc.RelatedTaskID = taskID
	return pc
}

// WithRelatedPR sets the related PR number.
func (pc PauseContext) WithRelatedPR(prNumber int) PauseContext {
	pc.RelatedPRNumber = prNumber
	return pc
}

// WithErrorDetails sets the error details.
func (pc PauseContext) WithErrorDetails(details string) PauseContext {
	pc.ErrorDetails = details
	return pc
}

// WithAutoResume sets the auto-resume time and condition.
func (pc PauseContext) WithAutoResume(after time.Time, condition string) PauseContext {
	pc.AutoResumeAfter = &after
	pc.ResumeCondition = condition
	return pc
}

// CanAutoResumeNow checks if the session can be auto-resumed at this moment.
func (pc PauseContext) CanAutoResumeNow() bool {
	if !pc.Reason.CanAutoRecover() {
		return false
	}
	if pc.AutoResumeAfter == nil {
		return true // No specific time set, can resume immediately
	}
	return time.Now().After(*pc.AutoResumeAfter)
}

// PauseRecord represents a historical record of pause/resume for audit trail.
type PauseRecord struct {
	// Reason is why the session was paused.
	Reason PauseReason `json:"reason"`

	// PausedAt is when the session was paused.
	PausedAt time.Time `json:"pausedAt"`

	// ResumedAt is when the session was resumed (nil if still paused).
	ResumedAt *time.Time `json:"resumedAt,omitempty"`

	// ResumeAction is how the session was resumed.
	ResumeAction string `json:"resumeAction,omitempty"`

	// Duration is the pause duration in seconds.
	// A value of 0 indicates either:
	//   - The pause lasted less than 1 second (ResumedAt != nil)
	//   - The pause record is not yet completed (ResumedAt == nil)
	// Use ResumedAt to distinguish between these states.
	Duration int `json:"duration"`
}

// NewPauseRecord creates a new PauseRecord from a PauseContext.
func NewPauseRecord(ctx PauseContext) PauseRecord {
	return PauseRecord{
		Reason:   ctx.Reason,
		PausedAt: ctx.PausedAt,
	}
}

// Complete marks the pause record as resumed.
// Duration is calculated as seconds between PausedAt and now.
// Note: Duration may be 0 if the pause lasted less than 1 second;
// this is valid and indicates a very short pause, not an error.
func (pr *PauseRecord) Complete(action string) {
	now := time.Now()
	pr.ResumedAt = &now
	pr.ResumeAction = action
	pr.Duration = int(now.Sub(pr.PausedAt).Seconds())
}

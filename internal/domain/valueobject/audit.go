package valueobject

import "time"

// ActorRole represents the role of an actor in the system.
type ActorRole string

const (
	ActorRoleAdmin       ActorRole = "admin"
	ActorRoleIssueAuthor ActorRole = "issue_author"
)

// IsValid checks if the actor role is valid.
func (ar ActorRole) IsValid() bool {
	return ar == ActorRoleAdmin || ar == ActorRoleIssueAuthor
}

// CanAnswerClarification returns true if the role can answer clarification questions.
func (ar ActorRole) CanAnswerClarification() bool {
	return ar == ActorRoleAdmin || ar == ActorRoleIssueAuthor
}

// CanApprove returns true if the role can approve designs/PRs.
func (ar ActorRole) CanApprove() bool {
	return ar == ActorRoleAdmin
}

// OperationType represents the type of operation in audit logs.
type OperationType string

const (
	OpSessionStart     OperationType = "session_start"
	OpSessionPause     OperationType = "session_pause"
	OpSessionResume    OperationType = "session_resume"
	OpSessionTerminate OperationType = "session_terminate"
	OpStageTransition  OperationType = "stage_transition"
	OpAgentCall        OperationType = "agent_call"
	OpToolUse          OperationType = "tool_use"
	OpFileRead         OperationType = "file_read"
	OpFileWrite        OperationType = "file_write"
	OpBashExecute      OperationType = "bash_execute"
	OpGitOperation     OperationType = "git_operation"
	OpPRCreate         OperationType = "pr_create"
	OpApprovalRequest  OperationType = "approval_request"
	OpApprovalDecision OperationType = "approval_decision"
)

// IsValid checks if the operation type is valid.
func (ot OperationType) IsValid() bool {
	validOps := []OperationType{
		OpSessionStart, OpSessionPause, OpSessionResume, OpSessionTerminate,
		OpStageTransition, OpAgentCall, OpToolUse,
		OpFileRead, OpFileWrite, OpBashExecute, OpGitOperation,
		OpPRCreate, OpApprovalRequest, OpApprovalDecision,
	}
	for _, v := range validOps {
		if ot == v {
			return true
		}
	}
	return false
}

// AuditResult represents the result of an audited operation.
type AuditResult string

const (
	AuditResultSuccess AuditResult = "success"
	AuditResultFailed  AuditResult = "failed"
	AuditResultDenied  AuditResult = "denied"
)

// IsValid checks if the audit result is valid.
func (ar AuditResult) IsValid() bool {
	return ar == AuditResultSuccess || ar == AuditResultFailed || ar == AuditResultDenied
}

// RollbackRecord represents a record of a rollback operation.
type RollbackRecord struct {
	FromStage      Stage     `json:"fromStage"`      // Stage before rollback
	ToStage        Stage     `json:"toStage"`        // Stage after rollback
	Reason         string    `json:"reason"`         // Reason for rollback
	Timestamp      time.Time `json:"timestamp"`      // When rollback occurred
	UserInitiated  bool      `json:"userInitiated"`  // Whether rollback was user-initiated
}

// NewRollbackRecord creates a new rollback record.
func NewRollbackRecord(fromStage, toStage Stage, reason string, userInitiated bool) RollbackRecord {
	return RollbackRecord{
		FromStage:     fromStage,
		ToStage:       toStage,
		Reason:        reason,
		Timestamp:     time.Now(),
		UserInitiated: userInitiated,
	}
}
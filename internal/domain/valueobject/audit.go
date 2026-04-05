package valueobject

import "time"

// ActorRole represents the role of an actor in the system.
type ActorRole string

const (
	ActorRoleAdmin       ActorRole = "admin"
	ActorRoleIssueAuthor ActorRole = "issue_author"
	ActorRoleSystem      ActorRole = "system" // System-triggered operations
)

// IsValid checks if the actor role is valid.
func (ar ActorRole) IsValid() bool {
	return ar == ActorRoleAdmin || ar == ActorRoleIssueAuthor || ar == ActorRoleSystem
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
	// Session operations
	OpSessionStart      OperationType = "session_start"
	OpSessionPause      OperationType = "session_pause"
	OpSessionResume     OperationType = "session_resume"
	OpSessionTerminate  OperationType = "session_terminate"
	OpStageTransition   OperationType = "stage_transition"
	OpUserCommand       OperationType = "user_command" // User-issued commands (continue, restart, terminate, etc.)

	// Agent operations
	OpAgentCall  OperationType = "agent_call"
	OpToolUse    OperationType = "tool_use"
	OpFileRead   OperationType = "file_read"
	OpFileWrite  OperationType = "file_write"
	OpBashExecute OperationType = "bash_execute"
	OpGitOperation OperationType = "git_operation"

	// Approval operations
	OpApprovalRequest  OperationType = "approval_request"
	OpApprovalDecision OperationType = "approval_decision"

	// Clarification operations
	OpClarificationStart          OperationType = "clarification_start"
	OpClarificationAnswer         OperationType = "clarification_answer"
	OpClarificationForceStartDesign OperationType = "clarification_force_start_design"
	OpClarityEvaluate             OperationType = "clarity_evaluate"

	// Design operations
	OpDesignStart        OperationType = "design_start"
	OpDesignConfirm      OperationType = "design_confirm"
	OpDesignReject       OperationType = "design_reject"
	OpDesignUpdate       OperationType = "design_update"
	OpComplexityEvaluate OperationType = "complexity_evaluate"

	// Task operations
	OpTaskBreakdown OperationType = "task_breakdown"
	OpTaskExecute   OperationType = "task_execute"
	OpTaskComplete  OperationType = "task_complete"
	OpTaskFail      OperationType = "task_fail"
	OpTaskRetry     OperationType = "task_retry"
	OpTaskSkip      OperationType = "task_skip"

	// PR operations
	OpPRStart           OperationType = "pr_start"
	OpPRCreate          OperationType = "pr_create"
	OpPRUpdate          OperationType = "pr_update"
	OpPRConflictDetect  OperationType = "pr_conflict_detect"
	OpPRConflictResolve OperationType = "pr_conflict_resolve"
	OpPRMerge           OperationType = "pr_merge"
	OpPRClose           OperationType = "pr_close"

	// Repository operations
	OpRepositoryCreate  OperationType = "repository_create"
	OpRepositoryUpdate  OperationType = "repository_update"
	OpRepositoryDelete  OperationType = "repository_delete"
	OpRepositoryEnable  OperationType = "repository_enable"
	OpRepositoryDisable OperationType = "repository_disable"
)

// validOperationTypes stores all valid operation types for O(1) lookup.
// This map is auto-maintained: adding new constants above requires adding them here.
var validOperationTypes = map[OperationType]bool{
	// Session
	OpSessionStart: true, OpSessionPause: true, OpSessionResume: true,
	OpSessionTerminate: true, OpStageTransition: true, OpUserCommand: true,
	// Agent
	OpAgentCall: true, OpToolUse: true, OpFileRead: true, OpFileWrite: true,
	OpBashExecute: true, OpGitOperation: true,
	// Approval
	OpApprovalRequest: true, OpApprovalDecision: true,
	// Clarification
	OpClarificationStart: true, OpClarificationAnswer: true,
	OpClarificationForceStartDesign: true, OpClarityEvaluate: true,
	// Design
	OpDesignStart: true, OpDesignConfirm: true, OpDesignReject: true,
	OpDesignUpdate: true, OpComplexityEvaluate: true,
	// Task
	OpTaskBreakdown: true, OpTaskExecute: true, OpTaskComplete: true,
	OpTaskFail: true, OpTaskRetry: true, OpTaskSkip: true,
	// PR
	OpPRStart: true, OpPRCreate: true, OpPRUpdate: true,
	OpPRConflictDetect: true, OpPRConflictResolve: true, OpPRMerge: true, OpPRClose: true,
	// Repository
	OpRepositoryCreate: true, OpRepositoryUpdate: true, OpRepositoryDelete: true,
	OpRepositoryEnable: true, OpRepositoryDisable: true,
}

// IsValid checks if the operation type is valid.
func (ot OperationType) IsValid() bool {
	return validOperationTypes[ot]
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
	FromStage     Stage     `json:"fromStage"`     // Stage before rollback
	ToStage       Stage     `json:"toStage"`       // Stage after rollback
	Reason        string    `json:"reason"`        // Reason for rollback
	Timestamp     time.Time `json:"timestamp"`     // When rollback occurred
	UserInitiated bool      `json:"userInitiated"` // Whether rollback was user-initiated
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
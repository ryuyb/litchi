package service

// RollbackDecision represents the decision result for a rollback operation.
type RollbackDecision int

const (
	// RollbackAllowed indicates rollback can proceed without restrictions.
	RollbackAllowed RollbackDecision = iota

	// RollbackNeedConfirmation indicates rollback requires user confirmation.
	// This is used when rollback has significant effects that need user awareness.
	RollbackNeedConfirmation

	// RollbackDenied indicates rollback cannot proceed due to precondition failures.
	RollbackDenied
)

// String returns the string representation of the rollback decision.
func (d RollbackDecision) String() string {
	switch d {
	case RollbackAllowed:
		return "allowed"
	case RollbackNeedConfirmation:
		return "need_confirmation"
	case RollbackDenied:
		return "denied"
	default:
		return "unknown"
	}
}

// RollbackType categorizes the rollback operation depth.
// This determines how much context will be preserved or cleared.
type RollbackType int

const (
	// RollbackTypeShallow is a shallow rollback (one stage back).
	// Examples: PR -> Execution, Design -> TaskBreakdown (hypothetical).
	// Preserves most context, minimal cleanup.
	RollbackTypeShallow RollbackType = iota

	// RollbackTypeDeep is a deep rollback (two stages back).
	// Examples: PR -> Design, Execution -> Design.
	// Clears execution context, deprecates branch.
	RollbackTypeDeep

	// RollbackTypeFull is a full rollback (three or more stages back).
	// Examples: PR -> Clarification, Execution -> Clarification.
	// Clears all context, resets to initial state.
	RollbackTypeFull
)

// String returns the string representation of the rollback type.
func (t RollbackType) String() string {
	switch t {
	case RollbackTypeShallow:
		return "shallow"
	case RollbackTypeDeep:
		return "deep"
	case RollbackTypeFull:
		return "full"
	default:
		return "unknown"
	}
}

// RollbackResult represents the complete evaluation result for a rollback operation.
// It provides detailed information about the rollback decision, effects, and recovery guidance.
type RollbackResult struct {
	// Decision is the rollback decision result.
	Decision RollbackDecision `json:"decision"`

	// RollbackType categorizes the rollback depth.
	RollbackType RollbackType `json:"rollbackType"`

	// RollbackRule is the rule identifier (R1-R6).
	// Empty if the rollback path is not one of the defined rules.
	RollbackRule string `json:"rollbackRule,omitempty"`

	// Reason explains why the decision was made.
	Reason string `json:"reason,omitempty"`

	// RequiredAction describes what the user needs to do (if any).
	RequiredAction string `json:"requiredAction,omitempty"`

	// PreconditionsMet indicates whether all preconditions are satisfied.
	PreconditionsMet bool `json:"preconditionsMet"`

	// --- Rollback Effect Flags ---

	// WillDeprecateBranch indicates whether the current branch will be marked as deprecated.
	// True for R1 (Execution->Design), R3 (Execution->Clarification), R5, R6 (PR rollback to Design or earlier).
	WillDeprecateBranch bool `json:"willDeprecateBranch"`

	// WillClosePR indicates whether the PR will be closed.
	// True for R5 (PR->Design) and R6 (PR->Clarification).
	WillClosePR bool `json:"willClosePR"`

	// WillClearTasks indicates whether the task list will be cleared.
	// True for rollbacks to Design or earlier stages.
	WillClearTasks bool `json:"willClearTasks"`

	// WillClearDesign indicates whether the design will be cleared.
	// True for rollbacks to Clarification stage.
	WillClearDesign bool `json:"willClearDesign"`

	// WillIncrementPRRollbackCount indicates whether the PR rollback count will be incremented.
	// True for any rollback from PR stage.
	WillIncrementPRRollbackCount bool `json:"willIncrementPRRollbackCount"`

	// --- Recovery Guidance ---

	// RecoveryActions lists the actions needed after rollback to continue the workflow.
	RecoveryActions []string `json:"recoveryActions,omitempty"`
}

// IsAllowed returns true if the rollback decision is RollbackAllowed.
func (r RollbackResult) IsAllowed() bool {
	return r.Decision == RollbackAllowed
}

// NeedsConfirmation returns true if the rollback decision is RollbackNeedConfirmation.
func (r RollbackResult) NeedsConfirmation() bool {
	return r.Decision == RollbackNeedConfirmation
}

// IsDenied returns true if the rollback decision is RollbackDenied.
func (r RollbackResult) IsDenied() bool {
	return r.Decision == RollbackDenied
}

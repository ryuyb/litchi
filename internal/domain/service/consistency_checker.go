// Package service provides domain services for consistency checking.
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
)

// ConsistencyIssue represents a detected inconsistency.
type ConsistencyIssue struct {
	Type        IssueType `json:"type"`
	Severity    Severity  `json:"severity"`
	Description string    `json:"description"`
	FieldName   string    `json:"fieldName,omitempty"`
	Expected    any       `json:"expected,omitempty"`
	Actual      any       `json:"actual,omitempty"`
	AutoRepair  bool      `json:"autoRepair"` // Whether this can be auto-repaired
}

// IssueType categorizes the type of inconsistency.
type IssueType string

const (
	IssueTypeCacheMismatch      IssueType = "cache_mismatch"       // Database vs cache inconsistent
	IssueTypeStatusMismatch     IssueType = "status_mismatch"      // Stage vs status inconsistent
	IssueTypeTaskProgress       IssueType = "task_progress"        // Task status vs execution progress inconsistent
	IssueTypePauseContextStale  IssueType = "pause_context_stale"  // PauseContext exists but session is active
	IssueTypeExecutionOrphan    IssueType = "execution_orphan"     // Execution exists but no tasks
	IssueTypeDesignMissing      IssueType = "design_missing"       // Past clarification but no design
)

// Severity indicates the severity of an issue.
type Severity string

const (
	SeverityLow      Severity = "low"      // Minor inconsistency, can continue
	SeverityMedium   Severity = "medium"   // May cause issues, should repair
	SeverityHigh     Severity = "high"     // Critical, must repair before continuing
)

// ConsistencyReport contains the results of a consistency check.
type ConsistencyReport struct {
	SessionID     uuid.UUID         `json:"sessionId"`
	HasIssues     bool              `json:"hasIssues"`
	Issues        []ConsistencyIssue `json:"issues"`
	RepairedCount int               `json:"repairedCount"`
	FailedRepairs []ConsistencyIssue `json:"failedRepairs,omitempty"`
}

// RepairAction represents an action taken to repair an inconsistency.
type RepairAction struct {
	IssueType IssueType `json:"issueType"`
	Action    string    `json:"action"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// ConsistencyChecker defines the interface for checking and repairing state consistency.
type ConsistencyChecker interface {
	// Check performs a consistency check on a WorkSession.
	// It compares database state with cache and validates internal consistency.
	Check(ctx context.Context, session *aggregate.WorkSession, cacheWorktreePath string) (*ConsistencyReport, error)

	// CheckAndRepair performs a consistency check and automatically repairs fixable issues.
	// Returns the report with repair actions taken.
	CheckAndRepair(ctx context.Context, session *aggregate.WorkSession, cacheWorktreePath string) (*ConsistencyReport, error)

	// Repair attempts to repair the given issues.
	// Returns actions taken and any issues that could not be repaired.
	Repair(ctx context.Context, session *aggregate.WorkSession, issues []ConsistencyIssue) ([]RepairAction, []ConsistencyIssue)
}

// ConsistencyCheckOptions provides configuration for consistency checks.
type ConsistencyCheckOptions struct {
	CheckCacheConsistency   bool `json:"checkCacheConsistency"`   // Check DB vs cache
	CheckStageStatus        bool `json:"checkStageStatus"`        // Check stage vs status
	CheckTaskProgress       bool `json:"checkTaskProgress"`       // Check task vs execution
	CheckPauseContext       bool `json:"checkPauseContext"`       // Check pause context validity
	AutoRepair              bool `json:"autoRepair"`              // Auto-repair fixable issues
	RegenerateCacheOnMismatch bool `json:"regenerateCacheOnMismatch"` // Regenerate cache if mismatch
}

// DefaultCheckOptions returns the default check options.
func DefaultCheckOptions() ConsistencyCheckOptions {
	return ConsistencyCheckOptions{
		CheckCacheConsistency:   true,
		CheckStageStatus:        true,
		CheckTaskProgress:       true,
		CheckPauseContext:       true,
		AutoRepair:              true,
		RegenerateCacheOnMismatch: true,
	}
}
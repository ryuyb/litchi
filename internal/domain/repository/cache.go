package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ExecutionContextCache stores the session context for Agent execution.
type ExecutionContextCache struct {
	SessionID      uuid.UUID            `json:"sessionId"`
	CurrentStage   string               `json:"currentStage"`
	Status         string               `json:"status"`
	PauseReason    *string              `json:"pauseReason,omitempty"`
	Clarification  *ClarificationCache  `json:"clarification,omitempty"`
	Design         *DesignCache         `json:"design,omitempty"`
	Execution      *ExecutionCache      `json:"execution,omitempty"`
	Tasks          []TaskCache          `json:"tasks"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}

// Validate checks the cache data integrity.
func (c *ExecutionContextCache) Validate() error {
	if c.SessionID == uuid.Nil {
		return errors.New("sessionId is required")
	}
	if c.CurrentStage == "" {
		return errors.New("currentStage is required")
	}
	if c.Status == "" {
		return errors.New("status is required")
	}
	return nil
}

// ClarificationCache stores clarification state for Agent execution.
type ClarificationCache struct {
	Status           string   `json:"status"`
	ConfirmedPoints  []string `json:"confirmedPoints"`
	PendingQuestions []string `json:"pendingQuestions"`
}

// DesignCache stores design state for Agent execution.
type DesignCache struct {
	Status              string `json:"status"`
	CurrentVersion      int    `json:"currentVersion"`
	ComplexityScore     *int   `json:"complexityScore,omitempty"`
	RequireConfirmation bool   `json:"requireConfirmation"`
	Confirmed           bool   `json:"confirmed"`
}

// ExecutionCache stores execution state for Agent execution.
type ExecutionCache struct {
	CurrentTaskID    *uuid.UUID  `json:"currentTaskId,omitempty"`
	CompletedTaskIDs []uuid.UUID `json:"completedTaskIds"`
	FailedTaskID     *uuid.UUID  `json:"failedTaskId,omitempty"`
	Branch           string      `json:"branch"`
	BranchDeprecated bool        `json:"branchDeprecated"`
	WorktreePath     string      `json:"worktreePath"`
}

// TaskCache stores task state for Agent execution.
type TaskCache struct {
	ID         uuid.UUID `json:"id"`
	Status     string    `json:"status"`
	RetryCount int       `json:"retryCount"`
}

// CacheRepository defines the interface for context cache operations.
type CacheRepository interface {
	// Save writes context.json to {worktreePath}/.litchi/
	Save(ctx context.Context, worktreePath string, cache *ExecutionContextCache) error

	// Load reads context.json from {worktreePath}/.litchi/
	// Returns nil if file does not exist.
	Load(ctx context.Context, worktreePath string) (*ExecutionContextCache, error)

	// Delete removes the .litchi directory from worktree.
	Delete(ctx context.Context, worktreePath string) error
}
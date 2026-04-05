package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// AgentStage represents the execution stage type for Agent operations.
type AgentStage string

const (
	// AgentStageClarification is the clarification phase for understanding requirements.
	AgentStageClarification AgentStage = "clarification"
	// AgentStageDesign is the design phase for creating implementation plans.
	AgentStageDesign AgentStage = "design"
	// AgentStageTaskBreakdown is the task breakdown phase for splitting work into tasks.
	AgentStageTaskBreakdown AgentStage = "task_breakdown"
	// AgentStageTaskExecution is the execution phase for implementing tasks.
	AgentStageTaskExecution AgentStage = "task_execution"
	// AgentStagePRCreation is the PR creation phase for finalizing and submitting.
	AgentStagePRCreation AgentStage = "pr_creation"
)

// AgentRequest is the request structure for Agent execution.
type AgentRequest struct {
	// SessionID is the unique identifier of the work session.
	SessionID uuid.UUID `json:"sessionId"`

	// Stage is the current execution stage.
	Stage AgentStage `json:"stage"`

	// WorktreePath is the Git worktree directory for execution.
	WorktreePath string `json:"worktreePath"`

	// Prompt is the task description or instruction for the Agent.
	Prompt string `json:"prompt"`

	// Context contains execution context information.
	Context *AgentContext `json:"context"`

	// Timeout is the maximum execution duration.
	Timeout time.Duration `json:"timeout"`

	// AllowedTools is the list of permitted tools for this execution.
	AllowedTools []string `json:"allowedTools"`

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int `json:"maxRetries"`
}

// Validate validates the AgentRequest fields.
func (r *AgentRequest) Validate() error {
	if r.SessionID == uuid.Nil {
		return fmt.Errorf("sessionId is required")
	}
	if r.Stage == "" {
		return fmt.Errorf("stage is required")
	}
	if r.WorktreePath == "" {
		return fmt.Errorf("worktreePath is required")
	}
	if r.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if r.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

// AgentContext contains execution context information for the Agent.
type AgentContext struct {
	// IssueTitle is the title of the GitHub issue.
	IssueTitle string `json:"issueTitle"`

	// IssueBody is the body content of the GitHub issue.
	IssueBody string `json:"issueBody"`

	// Repository is the repository name (owner/repo format).
	Repository string `json:"repository"`

	// Branch is the current Git branch name.
	Branch string `json:"branch"`

	// DesignContent is the design document content (for execution stage).
	DesignContent string `json:"designContent"`

	// Tasks is the list of tasks context information.
	Tasks []TaskContext `json:"tasks"`

	// ClarifiedPoints is the list of clarified requirement points.
	ClarifiedPoints []string `json:"clarifiedPoints"`

	// History is the execution history entries.
	History []HistoryEntry `json:"history"`
}

// TaskContext contains context information for a single task.
type TaskContext struct {
	// ID is the unique identifier of the task.
	ID uuid.UUID `json:"id"`

	// Description is the task description.
	Description string `json:"description"`

	// Status is the current status of the task.
	Status string `json:"status"`

	// Dependencies is the list of task IDs that must complete first.
	Dependencies []uuid.UUID `json:"dependencies"`
}

// HistoryEntry represents an execution history entry.
type HistoryEntry struct {
	// Timestamp is when the entry was created.
	Timestamp time.Time `json:"timestamp"`

	// Stage is the execution stage at that moment.
	Stage AgentStage `json:"stage"`

	// Action is the action taken.
	Action string `json:"action"`

	// Result is the outcome of the action.
	Result string `json:"result"`
}

// AgentResponse is the response structure for Agent execution.
type AgentResponse struct {
	// SessionID is the unique identifier of the work session.
	SessionID uuid.UUID `json:"sessionId"`

	// Stage is the execution stage.
	Stage AgentStage `json:"stage"`

	// Success indicates whether execution completed successfully.
	Success bool `json:"success"`

	// Output is the raw output content from the Agent.
	Output string `json:"output"`

	// Result contains structured execution result data.
	Result AgentResult `json:"result"`

	// Duration is the total execution time.
	Duration time.Duration `json:"duration"`

	// TokensUsed is the number of tokens consumed.
	TokensUsed int `json:"tokensUsed"`

	// ToolCalls is the record of all tool invocations.
	ToolCalls []ToolCallRecord `json:"toolCalls"`

	// Error contains error information if execution failed.
	Error *AgentErrorInfo `json:"error"`

	// NeedsApproval indicates whether human approval is required.
	NeedsApproval bool `json:"needsApproval"`
}

// AgentResult contains structured execution result data.
type AgentResult struct {
	// Type is the result type (design, task, clarification, etc.).
	Type string `json:"type"`

	// Content is the main result content.
	Content string `json:"content"`

	// StructuredData contains additional structured data (key-value pairs).
	StructuredData map[string]any `json:"structuredData"`

	// FilesChanged is the list of file modifications.
	FilesChanged []FileChange `json:"filesChanged"`

	// TestsRun is the list of test execution results.
	TestsRun []TestResult `json:"testsRun"`
}

// FileChange represents a file modification record.
type FileChange struct {
	// Path is the file path relative to worktree root.
	Path string `json:"path"`

	// Action is the operation type (create, modify, delete).
	Action string `json:"action"`

	// LinesAdded is the number of lines added.
	LinesAdded int `json:"linesAdded"`

	// LinesDeleted is the number of lines deleted.
	LinesDeleted int `json:"linesDeleted"`
}

// TestResult represents a test execution result.
type TestResult struct {
	// Name is the test name or suite name.
	Name string `json:"name"`

	// Status is the test status (passed, failed, skipped).
	Status string `json:"status"`

	// Message is the error message if test failed.
	Message string `json:"message"`

	// Duration is the test execution time.
	Duration time.Duration `json:"duration"`
}

// ToolCallRecord represents a tool invocation record.
type ToolCallRecord struct {
	// Timestamp is when the tool was invoked.
	Timestamp time.Time `json:"timestamp"`

	// ToolName is the name of the tool used.
	ToolName string `json:"toolName"`

	// Input is the input parameters for the tool.
	Input string `json:"input"`

	// Output is the output result from the tool.
	Output string `json:"output"`

	// Success indicates whether the tool call succeeded.
	Success bool `json:"success"`

	// Blocked indicates whether the call was blocked by permission control.
	Blocked bool `json:"blocked"`

	// BlockReason is the reason for blocking (if blocked).
	BlockReason string `json:"blockReason"`
}

// AgentErrorInfo contains error information from Agent execution.
type AgentErrorInfo struct {
	// Code is the error code.
	Code string `json:"code"`

	// Category is the error category (process, timeout, permission, execution).
	Category string `json:"category"`

	// Message is the error message.
	Message string `json:"message"`

	// Detail is the detailed error information.
	Detail string `json:"detail"`

	// Recoverable indicates whether the error can be recovered.
	Recoverable bool `json:"recoverable"`

	// Retryable indicates whether the execution can be retried.
	Retryable bool `json:"retryable"`

	// RetryCount is the number of retries already attempted.
	RetryCount int `json:"retryCount"`
}

// AgentStatus represents the current execution status.
type AgentStatus struct {
	// SessionID is the unique identifier of the work session.
	SessionID uuid.UUID `json:"sessionId"`

	// Status is the execution status (idle, running, paused, cancelled, completed, failed).
	Status string `json:"status"`

	// CurrentStage is the current execution stage.
	CurrentStage AgentStage `json:"currentStage"`

	// StartTime is when execution started.
	StartTime time.Time `json:"startTime"`

	// Progress is the execution progress percentage (0-100).
	Progress float64 `json:"progress"`

	// Message is the status message.
	Message string `json:"message"`
}

// AgentContextCache represents the cacheable context for Agent execution.
// This is a domain-level value object, independent of infrastructure persistence format.
type AgentContextCache struct {
	// SessionID is the unique identifier of the work session.
	SessionID uuid.UUID `json:"sessionId"`

	// CurrentStage is the current execution stage.
	CurrentStage string `json:"currentStage"`

	// Status is the session status.
	Status string `json:"status"`

	// PauseReason is the reason for pause (if paused).
	PauseReason *string `json:"pauseReason,omitempty"`

	// ClarifiedPoints are the confirmed requirement points.
	ClarifiedPoints []string `json:"clarifiedPoints,omitempty"`

	// DesignVersion is the current design version number.
	DesignVersion int `json:"designVersion,omitempty"`

	// ComplexityScore is the design complexity score.
	ComplexityScore *int `json:"complexityScore,omitempty"`

	// CurrentTaskID is the currently executing task ID.
	CurrentTaskID *uuid.UUID `json:"currentTaskId,omitempty"`

	// CompletedTaskIDs are the IDs of completed tasks.
	CompletedTaskIDs []uuid.UUID `json:"completedTaskIds,omitempty"`

	// Branch is the Git branch name.
	Branch string `json:"branch,omitempty"`

	// WorktreePath is the Git worktree path.
	WorktreePath string `json:"worktreePath,omitempty"`

	// UpdatedAt is the last update timestamp.
	UpdatedAt time.Time `json:"updatedAt"`
}

// AgentRunner defines the interface for Agent execution.
// This interface is implemented by infrastructure layer (ClaudeCodeAgent).
type AgentRunner interface {
	// Execute executes an Agent task and returns the result.
	Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error)

	// ExecuteWithRetry executes a task with automatic retry on failure.
	ExecuteWithRetry(ctx context.Context, req *AgentRequest, policy valueobject.RetryPolicy) (*AgentResponse, error)

	// ValidateRequest validates the request parameters.
	ValidateRequest(req *AgentRequest) error

	// PrepareContext prepares execution context from cache.
	PrepareContext(ctx context.Context, sessionID uuid.UUID, worktreePath string) (*AgentContext, error)

	// SaveContext saves execution context to cache.
	SaveContext(ctx context.Context, worktreePath string, cache *AgentContextCache) error

	// Cancel cancels the running execution for a session.
	Cancel(sessionID uuid.UUID) error

	// GetStatus retrieves the current execution status.
	GetStatus(sessionID uuid.UUID) (*AgentStatus, error)

	// IsRunning checks if Agent is currently executing for a session.
	IsRunning(sessionID uuid.UUID) bool

	// Shutdown gracefully shuts down the executor and cleans up resources.
	Shutdown(ctx context.Context) error
}
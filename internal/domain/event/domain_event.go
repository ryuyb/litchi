package event

import (
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// DomainEvent is the base interface for all domain events.
// All domain events must implement this interface to ensure type safety
// and consistent event handling across the system.
type DomainEvent interface {
	// EventType returns the unique identifier for this event type.
	EventType() string

	// SessionID returns the WorkSession ID that originated this event.
	// Returns uuid.Nil for system-level events (e.g., Repository management events)
	// that are not associated with a specific WorkSession.
	SessionID() uuid.UUID

	// OccurredAt returns the timestamp when the event occurred.
	OccurredAt() time.Time

	// ToMap converts the event to a map for serialization purposes.
	// This is useful for JSON encoding, database storage, or logging.
	ToMap() map[string]any
}

// IsSystemEvent returns true if the event is a system-level event
// not associated with a specific WorkSession.
func IsSystemEvent(e DomainEvent) bool {
	return e.SessionID() == uuid.Nil
}

// BaseEvent provides common fields for all domain events.
// Concrete event types should embed this struct to inherit common behavior.
type BaseEvent struct {
	Type      string      `json:"type"`
	Session   uuid.UUID   `json:"sessionId"`
	Timestamp time.Time   `json:"timestamp"`
}

// EventType returns the event type identifier.
func (e BaseEvent) EventType() string {
	return e.Type
}

// SessionID returns the WorkSession ID.
func (e BaseEvent) SessionID() uuid.UUID {
	return e.Session
}

// OccurredAt returns the event timestamp.
func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

// --- Lifecycle Events (Section 6.1) ---

// WorkSessionStarted is emitted when a new work session is created.
type WorkSessionStarted struct {
	BaseEvent
	IssueNumber int    `json:"issueNumber"`
	Repository  string `json:"repository"`
	Title       string `json:"title"`
}

// NewWorkSessionStarted creates a new WorkSessionStarted event.
func NewWorkSessionStarted(sessionID uuid.UUID, issueNumber int, repository, title string) *WorkSessionStarted {
	return &WorkSessionStarted{
		BaseEvent: BaseEvent{
			Type:      "WorkSessionStarted",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		IssueNumber: issueNumber,
		Repository:  repository,
		Title:       title,
	}
}

// ToMap converts the event to a map.
func (e WorkSessionStarted) ToMap() map[string]any {
	return map[string]any{
		"type":        e.Type,
		"sessionId":   e.Session.String(),
		"timestamp":   e.Timestamp,
		"issueNumber": e.IssueNumber,
		"repository":  e.Repository,
		"title":       e.Title,
	}
}

// WorkSessionPaused is emitted when a work session is paused.
type WorkSessionPaused struct {
	BaseEvent
	Reason string `json:"reason"`
}

// NewWorkSessionPaused creates a new WorkSessionPaused event.
func NewWorkSessionPaused(sessionID uuid.UUID, reason string) *WorkSessionPaused {
	return &WorkSessionPaused{
		BaseEvent: BaseEvent{
			Type:      "WorkSessionPaused",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		Reason: reason,
	}
}

// ToMap converts the event to a map.
func (e WorkSessionPaused) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"reason":    e.Reason,
	}
}

// WorkSessionResumed is emitted when a paused work session is resumed.
type WorkSessionResumed struct {
	BaseEvent
	PreviousStage valueobject.Stage `json:"previousStage"` // The stage the session was in when paused
}

// NewWorkSessionResumed creates a new WorkSessionResumed event.
func NewWorkSessionResumed(sessionID uuid.UUID, previousStage valueobject.Stage) *WorkSessionResumed {
	return &WorkSessionResumed{
		BaseEvent: BaseEvent{
			Type:      "WorkSessionResumed",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		PreviousStage: previousStage,
	}
}

// ToMap converts the event to a map.
func (e WorkSessionResumed) ToMap() map[string]any {
	return map[string]any{
		"type":          e.Type,
		"sessionId":     e.Session.String(),
		"timestamp":     e.Timestamp,
		"previousStage": e.PreviousStage.String(),
	}
}

// WorkSessionTerminated is emitted when a work session is terminated by user.
type WorkSessionTerminated struct {
	BaseEvent
	Reason string `json:"reason"`
}

// NewWorkSessionTerminated creates a new WorkSessionTerminated event.
func NewWorkSessionTerminated(sessionID uuid.UUID, reason string) *WorkSessionTerminated {
	return &WorkSessionTerminated{
		BaseEvent: BaseEvent{
			Type:      "WorkSessionTerminated",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		Reason: reason,
	}
}

// ToMap converts the event to a map.
func (e WorkSessionTerminated) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"reason":    e.Reason,
	}
}

// WorkSessionCompleted is emitted when a work session completes successfully.
type WorkSessionCompleted struct {
	BaseEvent
	PRNumber int `json:"prNumber"`
}

// NewWorkSessionCompleted creates a new WorkSessionCompleted event.
func NewWorkSessionCompleted(sessionID uuid.UUID, prNumber int) *WorkSessionCompleted {
	return &WorkSessionCompleted{
		BaseEvent: BaseEvent{
			Type:      "WorkSessionCompleted",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		PRNumber: prNumber,
	}
}

// ToMap converts the event to a map.
func (e WorkSessionCompleted) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"prNumber":  e.PRNumber,
	}
}

// --- Stage Transition Events (Section 6.2) ---

// StageTransitioned is emitted when a stage successfully transitions forward.
type StageTransitioned struct {
	BaseEvent
	FromStage valueobject.Stage `json:"fromStage"`
	ToStage   valueobject.Stage `json:"toStage"`
}

// NewStageTransitioned creates a new StageTransitioned event.
func NewStageTransitioned(sessionID uuid.UUID, fromStage, toStage valueobject.Stage) *StageTransitioned {
	return &StageTransitioned{
		BaseEvent: BaseEvent{
			Type:      "StageTransitioned",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		FromStage: fromStage,
		ToStage:   toStage,
	}
}

// ToMap converts the event to a map.
func (e StageTransitioned) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"fromStage": e.FromStage.String(),
		"toStage":   e.ToStage.String(),
	}
}

// StageRolledBack is emitted when a stage is rolled back.
type StageRolledBack struct {
	BaseEvent
	FromStage     valueobject.Stage `json:"fromStage"`
	ToStage       valueobject.Stage `json:"toStage"`
	Reason        string            `json:"reason"`
	UserInitiated bool              `json:"userInitiated"`
}

// NewStageRolledBack creates a new StageRolledBack event.
func NewStageRolledBack(sessionID uuid.UUID, fromStage, toStage valueobject.Stage, reason string, userInitiated bool) *StageRolledBack {
	return &StageRolledBack{
		BaseEvent: BaseEvent{
			Type:      "StageRolledBack",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		FromStage:     fromStage,
		ToStage:       toStage,
		Reason:        reason,
		UserInitiated: userInitiated,
	}
}

// ToMap converts the event to a map.
func (e StageRolledBack) ToMap() map[string]any {
	return map[string]any{
		"type":          e.Type,
		"sessionId":     e.Session.String(),
		"timestamp":     e.Timestamp,
		"fromStage":     e.FromStage.String(),
		"toStage":       e.ToStage.String(),
		"reason":        e.Reason,
		"userInitiated": e.UserInitiated,
	}
}

// --- Clarification Stage Events (Section 6.3) ---

// QuestionAsked is emitted when the Agent asks a clarification question.
type QuestionAsked struct {
	BaseEvent
	Question string `json:"question"`
}

// NewQuestionAsked creates a new QuestionAsked event.
func NewQuestionAsked(sessionID uuid.UUID, question string) *QuestionAsked {
	return &QuestionAsked{
		BaseEvent: BaseEvent{
			Type:      "QuestionAsked",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		Question: question,
	}
}

// ToMap converts the event to a map.
func (e QuestionAsked) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"question":  e.Question,
	}
}

// QuestionAnswered is emitted when the Issue author or admin answers a question.
type QuestionAnswered struct {
	BaseEvent
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Actor    string `json:"actor"`
}

// NewQuestionAnswered creates a new QuestionAnswered event.
func NewQuestionAnswered(sessionID uuid.UUID, question, answer, actor string) *QuestionAnswered {
	return &QuestionAnswered{
		BaseEvent: BaseEvent{
			Type:      "QuestionAnswered",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		Question: question,
		Answer:   answer,
		Actor:    actor,
	}
}

// ToMap converts the event to a map.
func (e QuestionAnswered) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"question":  e.Question,
		"answer":    e.Answer,
		"actor":     e.Actor,
	}
}

// ClarificationCompleted is emitted when the clarification stage is completed.
type ClarificationCompleted struct {
	BaseEvent
	ClarityScore int `json:"clarityScore"`
}

// NewClarificationCompleted creates a new ClarificationCompleted event.
func NewClarificationCompleted(sessionID uuid.UUID, clarityScore int) *ClarificationCompleted {
	return &ClarificationCompleted{
		BaseEvent: BaseEvent{
			Type:      "ClarificationCompleted",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		ClarityScore: clarityScore,
	}
}

// ToMap converts the event to a map.
func (e ClarificationCompleted) ToMap() map[string]any {
	return map[string]any{
		"type":         e.Type,
		"sessionId":    e.Session.String(),
		"timestamp":    e.Timestamp,
		"clarityScore": e.ClarityScore,
	}
}

// --- Design Stage Events (Section 6.4) ---

// DesignCreated is emitted when a new design version is created.
type DesignCreated struct {
	BaseEvent
	Version int    `json:"version"`
	Reason  string `json:"reason,omitempty"`
}

// NewDesignCreated creates a new DesignCreated event.
func NewDesignCreated(sessionID uuid.UUID, version int, reason string) *DesignCreated {
	return &DesignCreated{
		BaseEvent: BaseEvent{
			Type:      "DesignCreated",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		Version: version,
		Reason:  reason,
	}
}

// ToMap converts the event to a map.
func (e DesignCreated) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"version":   e.Version,
		"reason":    e.Reason,
	}
}

// DesignApproved is emitted when a design is approved.
type DesignApproved struct {
	BaseEvent
	Version int `json:"version"`
}

// NewDesignApproved creates a new DesignApproved event.
func NewDesignApproved(sessionID uuid.UUID, version int) *DesignApproved {
	return &DesignApproved{
		BaseEvent: BaseEvent{
			Type:      "DesignApproved",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		Version: version,
	}
}

// ToMap converts the event to a map.
func (e DesignApproved) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"version":   e.Version,
	}
}

// DesignRejected is emitted when a design is rejected.
type DesignRejected struct {
	BaseEvent
	Version int    `json:"version"`
	Reason  string `json:"reason,omitempty"`
}

// NewDesignRejected creates a new DesignRejected event.
func NewDesignRejected(sessionID uuid.UUID, version int, reason string) *DesignRejected {
	return &DesignRejected{
		BaseEvent: BaseEvent{
			Type:      "DesignRejected",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		Version: version,
		Reason:  reason,
	}
}

// ToMap converts the event to a map.
func (e DesignRejected) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"version":   e.Version,
		"reason":    e.Reason,
	}
}

// --- Task Stage Events (Section 6.5) ---

// TaskListCreated is emitted when the task list is created.
type TaskListCreated struct {
	BaseEvent
	TaskCount int `json:"taskCount"`
}

// NewTaskListCreated creates a new TaskListCreated event.
func NewTaskListCreated(sessionID uuid.UUID, taskCount int) *TaskListCreated {
	return &TaskListCreated{
		BaseEvent: BaseEvent{
			Type:      "TaskListCreated",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		TaskCount: taskCount,
	}
}

// ToMap converts the event to a map.
func (e TaskListCreated) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"taskCount": e.TaskCount,
	}
}

// TaskStarted is emitted when a task starts execution.
type TaskStarted struct {
	BaseEvent
	TaskID          uuid.UUID `json:"taskId"`
	TaskDescription string    `json:"taskDescription"`
}

// NewTaskStarted creates a new TaskStarted event.
func NewTaskStarted(sessionID uuid.UUID, taskID uuid.UUID, description string) *TaskStarted {
	return &TaskStarted{
		BaseEvent: BaseEvent{
			Type:      "TaskStarted",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		TaskID:          taskID,
		TaskDescription: description,
	}
}

// ToMap converts the event to a map.
func (e TaskStarted) ToMap() map[string]any {
	return map[string]any{
		"type":            e.Type,
		"sessionId":       e.Session.String(),
		"timestamp":       e.Timestamp,
		"taskId":          e.TaskID.String(),
		"taskDescription": e.TaskDescription,
	}
}

// TaskCompleted is emitted when a task completes successfully.
type TaskCompleted struct {
	BaseEvent
	TaskID uuid.UUID `json:"taskId"`
}

// NewTaskCompleted creates a new TaskCompleted event.
func NewTaskCompleted(sessionID uuid.UUID, taskID uuid.UUID) *TaskCompleted {
	return &TaskCompleted{
		BaseEvent: BaseEvent{
			Type:      "TaskCompleted",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		TaskID: taskID,
	}
}

// ToMap converts the event to a map.
func (e TaskCompleted) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"taskId":    e.TaskID.String(),
	}
}

// TaskFailed is emitted when a task fails.
type TaskFailed struct {
	BaseEvent
	TaskID    uuid.UUID `json:"taskId"`
	Reason    string    `json:"reason"`
	Suggestion string   `json:"suggestion,omitempty"`
}

// NewTaskFailed creates a new TaskFailed event.
func NewTaskFailed(sessionID uuid.UUID, taskID uuid.UUID, reason, suggestion string) *TaskFailed {
	return &TaskFailed{
		BaseEvent: BaseEvent{
			Type:      "TaskFailed",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		TaskID:     taskID,
		Reason:     reason,
		Suggestion: suggestion,
	}
}

// ToMap converts the event to a map.
func (e TaskFailed) ToMap() map[string]any {
	return map[string]any{
		"type":       e.Type,
		"sessionId":  e.Session.String(),
		"timestamp":  e.Timestamp,
		"taskId":     e.TaskID.String(),
		"reason":     e.Reason,
		"suggestion": e.Suggestion,
	}
}

// TaskSkipped is emitted when a task is skipped.
type TaskSkipped struct {
	BaseEvent
	TaskID uuid.UUID `json:"taskId"`
	Reason string    `json:"reason"`
}

// NewTaskSkipped creates a new TaskSkipped event.
func NewTaskSkipped(sessionID uuid.UUID, taskID uuid.UUID, reason string) *TaskSkipped {
	return &TaskSkipped{
		BaseEvent: BaseEvent{
			Type:      "TaskSkipped",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		TaskID: taskID,
		Reason: reason,
	}
}

// ToMap converts the event to a map.
func (e TaskSkipped) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"taskId":    e.TaskID.String(),
		"reason":    e.Reason,
	}
}

// TaskRetryStarted is emitted when a failed task is retried.
type TaskRetryStarted struct {
	BaseEvent
	TaskID     uuid.UUID `json:"taskId"`
	RetryCount int       `json:"retryCount"`
}

// NewTaskRetryStarted creates a new TaskRetryStarted event.
func NewTaskRetryStarted(sessionID uuid.UUID, taskID uuid.UUID, retryCount int) *TaskRetryStarted {
	return &TaskRetryStarted{
		BaseEvent: BaseEvent{
			Type:      "TaskRetryStarted",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		TaskID:     taskID,
		RetryCount: retryCount,
	}
}

// ToMap converts the event to a map.
func (e TaskRetryStarted) ToMap() map[string]any {
	return map[string]any{
		"type":       e.Type,
		"sessionId":  e.Session.String(),
		"timestamp":  e.Timestamp,
		"taskId":     e.TaskID.String(),
		"retryCount": e.RetryCount,
	}
}

// --- PR Stage Events (Section 6.6) ---

// PullRequestCreated is emitted when a PR is created successfully.
type PullRequestCreated struct {
	BaseEvent
	PRNumber  int    `json:"prNumber"`
	Branch    string `json:"branch"`
	PRTitle   string `json:"prTitle"`
}

// NewPullRequestCreated creates a new PullRequestCreated event.
func NewPullRequestCreated(sessionID uuid.UUID, prNumber int, branch, title string) *PullRequestCreated {
	return &PullRequestCreated{
		BaseEvent: BaseEvent{
			Type:      "PullRequestCreated",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		PRNumber: prNumber,
		Branch:   branch,
		PRTitle:  title,
	}
}

// ToMap converts the event to a map.
func (e PullRequestCreated) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"prNumber":  e.PRNumber,
		"branch":    e.Branch,
		"prTitle":   e.PRTitle,
	}
}

// PullRequestMerged is emitted when a PR is merged successfully.
// This triggers the transition from PullRequest stage to Completed stage.
type PullRequestMerged struct {
	BaseEvent
	PRNumber  int    `json:"prNumber"`
	MergedBy  string `json:"mergedBy"`
	MergeSHA  string `json:"mergeSha"`
}

// NewPullRequestMerged creates a new PullRequestMerged event.
func NewPullRequestMerged(sessionID uuid.UUID, prNumber int, mergedBy, mergeSHA string) *PullRequestMerged {
	return &PullRequestMerged{
		BaseEvent: BaseEvent{
			Type:      "PullRequestMerged",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		PRNumber: prNumber,
		MergedBy: mergedBy,
		MergeSHA: mergeSHA,
	}
}

// ToMap converts the event to a map.
func (e PullRequestMerged) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"prNumber":  e.PRNumber,
		"mergedBy":  e.MergedBy,
		"mergeSha":  e.MergeSHA,
	}
}

// --- PR Rollback Events (R4, R5, R6 from state-machine.md Section 7.1) ---
// These are specialized events for different PR rollback depths.

// PRRolledBackToExecution is emitted when PR rolls back to Execution stage (R4: shallow rollback).
// Use case: PR review found minor issues, CI failure, user requested code changes.
// PR remains open, branch is preserved.
type PRRolledBackToExecution struct {
	BaseEvent
	PRNumber int    `json:"prNumber"`
	Reason   string `json:"reason"`
}

// NewPRRolledBackToExecution creates a new PRRolledBackToExecution event.
func NewPRRolledBackToExecution(sessionID uuid.UUID, prNumber int, reason string) *PRRolledBackToExecution {
	return &PRRolledBackToExecution{
		BaseEvent: BaseEvent{
			Type:      "PRRolledBackToExecution",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		PRNumber: prNumber,
		Reason:   reason,
	}
}

// ToMap converts the event to a map.
func (e PRRolledBackToExecution) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"prNumber":  e.PRNumber,
		"reason":    e.Reason,
	}
}

// PRRolledBackToDesign is emitted when PR rolls back to Design stage (R5: deep rollback).
// Use case: PR review found design issues, requirement changes.
// PR is closed, branch is deprecated.
type PRRolledBackToDesign struct {
	BaseEvent
	PRNumber       int    `json:"prNumber"`
	Reason         string `json:"reason"`
	DeprecatedBranch string `json:"deprecatedBranch"`
}

// NewPRRolledBackToDesign creates a new PRRolledBackToDesign event.
func NewPRRolledBackToDesign(sessionID uuid.UUID, prNumber int, reason, deprecatedBranch string) *PRRolledBackToDesign {
	return &PRRolledBackToDesign{
		BaseEvent: BaseEvent{
			Type:      "PRRolledBackToDesign",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		PRNumber:         prNumber,
		Reason:           reason,
		DeprecatedBranch: deprecatedBranch,
	}
}

// ToMap converts the event to a map.
func (e PRRolledBackToDesign) ToMap() map[string]any {
	return map[string]any{
		"type":             e.Type,
		"sessionId":        e.Session.String(),
		"timestamp":        e.Timestamp,
		"prNumber":         e.PRNumber,
		"reason":           e.Reason,
		"deprecatedBranch": e.DeprecatedBranch,
	}
}

// PRRolledBackToClarification is emitted when PR rolls back to Clarification stage (R6: deepest rollback).
// Use case: Fundamental misunderstanding of requirements, major requirement changes.
// PR is closed, branch is deprecated, design is cleared.
type PRRolledBackToClarification struct {
	BaseEvent
	PRNumber         int    `json:"prNumber"`
	Reason           string `json:"reason"`
	DeprecatedBranch string `json:"deprecatedBranch"`
}

// NewPRRolledBackToClarification creates a new PRRolledBackToClarification event.
func NewPRRolledBackToClarification(sessionID uuid.UUID, prNumber int, reason, deprecatedBranch string) *PRRolledBackToClarification {
	return &PRRolledBackToClarification{
		BaseEvent: BaseEvent{
			Type:      "PRRolledBackToClarification",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		PRNumber:         prNumber,
		Reason:           reason,
		DeprecatedBranch: deprecatedBranch,
	}
}

// ToMap converts the event to a map.
func (e PRRolledBackToClarification) ToMap() map[string]any {
	return map[string]any{
		"type":             e.Type,
		"sessionId":        e.Session.String(),
		"timestamp":        e.Timestamp,
		"prNumber":         e.PRNumber,
		"reason":           e.Reason,
		"deprecatedBranch": e.DeprecatedBranch,
	}
}

// --- User Command Events (Section 6.7) ---

// UserCommandReceived is emitted when a user command is received.
type UserCommandReceived struct {
	BaseEvent
	Command   string `json:"command"`
	Actor     string `json:"actor"`
	ActorRole string `json:"actorRole"`
}

// NewUserCommandReceived creates a new UserCommandReceived event.
func NewUserCommandReceived(sessionID uuid.UUID, command, actor, actorRole string) *UserCommandReceived {
	return &UserCommandReceived{
		BaseEvent: BaseEvent{
			Type:      "UserCommandReceived",
			Session:   sessionID,
			Timestamp: time.Now(),
		},
		Command:   command,
		Actor:     actor,
		ActorRole: actorRole,
	}
}

// ToMap converts the event to a map.
func (e UserCommandReceived) ToMap() map[string]any {
	return map[string]any{
		"type":      e.Type,
		"sessionId": e.Session.String(),
		"timestamp": e.Timestamp,
		"command":   e.Command,
		"actor":     e.Actor,
		"actorRole": e.ActorRole,
	}
}

// --- Repository Management Events (Section 6.8) ---
// Note: These are system-level events not tied to a specific WorkSession.
// SessionID returns uuid.Nil to indicate no associated session.

// RepositoryAdded is emitted when a new repository is added.
type RepositoryAdded struct {
	BaseEvent
	RepositoryName string `json:"repositoryName"`
}

// NewRepositoryAdded creates a new RepositoryAdded event.
// This is a system-level event with no associated WorkSession.
func NewRepositoryAdded(repositoryName string) *RepositoryAdded {
	return &RepositoryAdded{
		BaseEvent: BaseEvent{
			Type:      "RepositoryAdded",
			Session:   uuid.Nil, // System-level event, no associated session
			Timestamp: time.Now(),
		},
		RepositoryName: repositoryName,
	}
}

// ToMap converts the event to a map.
func (e RepositoryAdded) ToMap() map[string]any {
	return map[string]any{
		"type":           e.Type,
		"timestamp":      e.Timestamp,
		"repositoryName": e.RepositoryName,
	}
}

// RepositoryUpdated is emitted when repository configuration is updated.
type RepositoryUpdated struct {
	BaseEvent
	RepositoryName string   `json:"repositoryName"`
	Changes        []string `json:"changes"`
}

// NewRepositoryUpdated creates a new RepositoryUpdated event.
// This is a system-level event with no associated WorkSession.
func NewRepositoryUpdated(repositoryName string, changes []string) *RepositoryUpdated {
	return &RepositoryUpdated{
		BaseEvent: BaseEvent{
			Type:      "RepositoryUpdated",
			Session:   uuid.Nil, // System-level event, no associated session
			Timestamp: time.Now(),
		},
		RepositoryName: repositoryName,
		Changes:        changes,
	}
}

// ToMap converts the event to a map.
func (e RepositoryUpdated) ToMap() map[string]any {
	return map[string]any{
		"type":           e.Type,
		"timestamp":      e.Timestamp,
		"repositoryName": e.RepositoryName,
		"changes":        e.Changes,
	}
}

// RepositoryEnabled is emitted when a repository is enabled.
type RepositoryEnabled struct {
	BaseEvent
	RepositoryName string `json:"repositoryName"`
}

// NewRepositoryEnabled creates a new RepositoryEnabled event.
// This is a system-level event with no associated WorkSession.
func NewRepositoryEnabled(repositoryName string) *RepositoryEnabled {
	return &RepositoryEnabled{
		BaseEvent: BaseEvent{
			Type:      "RepositoryEnabled",
			Session:   uuid.Nil, // System-level event, no associated session
			Timestamp: time.Now(),
		},
		RepositoryName: repositoryName,
	}
}

// ToMap converts the event to a map.
func (e RepositoryEnabled) ToMap() map[string]any {
	return map[string]any{
		"type":           e.Type,
		"timestamp":      e.Timestamp,
		"repositoryName": e.RepositoryName,
	}
}

// RepositoryDisabled is emitted when a repository is disabled.
type RepositoryDisabled struct {
	BaseEvent
	RepositoryName string `json:"repositoryName"`
}

// NewRepositoryDisabled creates a new RepositoryDisabled event.
// This is a system-level event with no associated WorkSession.
func NewRepositoryDisabled(repositoryName string) *RepositoryDisabled {
	return &RepositoryDisabled{
		BaseEvent: BaseEvent{
			Type:      "RepositoryDisabled",
			Session:   uuid.Nil, // System-level event, no associated session
			Timestamp: time.Now(),
		},
		RepositoryName: repositoryName,
	}
}

// ToMap converts the event to a map.
func (e RepositoryDisabled) ToMap() map[string]any {
	return map[string]any{
		"type":           e.Type,
		"timestamp":      e.Timestamp,
		"repositoryName": e.RepositoryName,
	}
}

// RepositoryDeleted is emitted when a repository is deleted.
type RepositoryDeleted struct {
	BaseEvent
	RepositoryName string `json:"repositoryName"`
}

// NewRepositoryDeleted creates a new RepositoryDeleted event.
// This is a system-level event with no associated WorkSession.
func NewRepositoryDeleted(repositoryName string) *RepositoryDeleted {
	return &RepositoryDeleted{
		BaseEvent: BaseEvent{
			Type:      "RepositoryDeleted",
			Session:   uuid.Nil, // System-level event, no associated session
			Timestamp: time.Now(),
		},
		RepositoryName: repositoryName,
	}
}

// ToMap converts the event to a map.
func (e RepositoryDeleted) ToMap() map[string]any {
	return map[string]any{
		"type":           e.Type,
		"timestamp":      e.Timestamp,
		"repositoryName": e.RepositoryName,
	}
}
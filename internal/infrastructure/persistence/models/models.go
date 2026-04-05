// Package models provides GORM model definitions for database persistence.
// These models map to the PostgreSQL tables defined in migrations.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Repository represents a GitHub repository configuration.
type Repository struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string         `gorm:"type:varchar(255);unique;not null"` // e.g. "org/repo"
	Enabled   bool           `gorm:"default:true"`
	Config    datatypes.JSON `gorm:"type:jsonb;default:'{}'"` // repository-level config override
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
}

// Issue represents a GitHub issue entity.
type Issue struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Number     int64          `gorm:"not null;index"` // GitHub issue number
	Title      string         `gorm:"type:varchar(500);not null"`
	Body       string         `gorm:"type:text"`
	Repository string         `gorm:"type:varchar(255);not null;index"` // references repositories.name
	Author     string         `gorm:"type:varchar(255);not null;index"` // GitHub username
	Labels     datatypes.JSON `gorm:"type:jsonb;default:'[]'"`          // Issue labels
	URL        string         `gorm:"type:varchar(500)"`                // Full GitHub URL
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
}

// WorkSession represents the aggregate root for automation workflow.
type WorkSession struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	IssueID      uuid.UUID      `gorm:"type:uuid;not null;index"`
	CurrentStage string         `gorm:"type:varchar(50);not null;default:'clarification';index"` // clarification, design, task_breakdown, execution, pull_request, completed
	Status       string         `gorm:"type:varchar(50);not null;default:'active';index"`        // active, paused, terminated, completed
	Version      int            `gorm:"type:int;not null;default:1"`                             // optimistic lock version
	PauseContext datatypes.JSON `gorm:"type:jsonb"`                                              // current pause context (nullable)
	PauseHistory datatypes.JSON `gorm:"type:jsonb;default:'[]'"`                                 // history of pause/resume records
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`

	// Relations
	Issue         *Issue         `gorm:"foreignKey:IssueID;constraint:OnDelete:CASCADE"`
	Clarification *Clarification `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	Design        *Design        `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	Tasks         []Task         `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	Execution     *Execution     `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

// Clarification represents the clarification phase entity.
type Clarification struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID           uuid.UUID      `gorm:"type:uuid;not null;unique;index"`
	ConfirmedPoints     datatypes.JSON `gorm:"type:jsonb;default:'[]'"`                               // list of confirmed requirement points
	PendingQuestions    datatypes.JSON `gorm:"type:jsonb;default:'[]'"`                               // list of pending questions
	ConversationHistory datatypes.JSON `gorm:"type:jsonb;default:'[]'"`                               // conversation turns history
	Status              string         `gorm:"type:varchar(50);not null;default:'in_progress';index"` // in_progress, completed
	ClarityScore        *int           `gorm:"type:int"`                                              // overall clarity score (0-100)
	ClarityDimensions   datatypes.JSON `gorm:"type:jsonb"`                                            // detailed clarity dimension scores
	CreatedAt           time.Time      `gorm:"autoCreateTime"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime"`

	// Relation
	Session *WorkSession `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

// Design represents the design phase entity.
type Design struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID           uuid.UUID `gorm:"type:uuid;not null;unique;index"`
	CurrentVersion      int       `gorm:"not null;default:0"`
	ComplexityScore     *int      `gorm:"type:int"` // complexity score (0-100)
	RequireConfirmation bool      `gorm:"default:false"`
	Confirmed           bool      `gorm:"default:false"`
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"`

	// Relations
	Session  *WorkSession    `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	Versions []DesignVersion `gorm:"foreignKey:DesignID;constraint:OnDelete:CASCADE"`
}

// DesignVersion represents a single version of the design document.
type DesignVersion struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DesignID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_design_version"`
	Version   int       `gorm:"not null;uniqueIndex:idx_design_version"`
	Content   string    `gorm:"type:text;not null"`
	Reason    string    `gorm:"type:varchar(500)"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	// Relation
	Design *Design `gorm:"foreignKey:DesignID;constraint:OnDelete:CASCADE"`
}

// Task represents an executable task entity.
type Task struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID     uuid.UUID `gorm:"type:uuid;not null;index"`
	Description   string    `gorm:"type:text;not null"`
	Status        string    `gorm:"type:varchar(50);not null;default:'pending';index"` // pending, running, completed, failed, skipped, retrying
	RetryCount    int       `gorm:"default:0"`
	FailureReason string    `gorm:"type:text"`
	Suggestion    string    `gorm:"type:text"`
	Seq           int       `gorm:"not null;index"` // execution sequence
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`

	// Relations
	Session *WorkSession `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	Result  *TaskResult  `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`

	// Many-to-many relation for dependencies
	Dependencies   []Task `gorm:"many2many:task_dependencies;joinForeignKey:task_id;joinReferences:depends_on_task_id"`
	DependentTasks []Task `gorm:"many2many:task_dependencies;joinForeignKey:depends_on_task_id;joinReferences:task_id"`
}

// TaskDependency represents the junction table for task dependencies.
type TaskDependency struct {
	TaskID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	DependsOnTaskID uuid.UUID `gorm:"type:uuid;primaryKey"`

	Task          *Task `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
	DependsOnTask *Task `gorm:"foreignKey:DependsOnTaskID;constraint:OnDelete:CASCADE"`
}

// TaskResult represents the execution result of a task.
type TaskResult struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TaskID      uuid.UUID      `gorm:"type:uuid;not null;unique;index"`
	Output      string         `gorm:"type:text"`
	TestResults datatypes.JSON `gorm:"type:jsonb;default:'[]'"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`

	// Relation
	Task *Task `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
}

// Execution represents the execution phase entity.
type Execution struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID          uuid.UUID      `gorm:"type:uuid;not null;unique;index"`
	WorktreePath       string         `gorm:"type:varchar(500)"`       // git worktree path
	BranchName         string         `gorm:"type:varchar(255)"`       // current branch name
	BranchDeprecated   bool           `gorm:"default:false"`           // whether branch is deprecated
	DeprecatedBranches datatypes.JSON `gorm:"type:jsonb;default:'[]'"` // history of deprecated branches
	CurrentTaskID      *uuid.UUID     `gorm:"type:uuid"`               // currently executing task
	FailedTask         datatypes.JSON `gorm:"type:jsonb"`              // failed task details: {taskId, reason, suggestion}
	FixTasks           datatypes.JSON `gorm:"type:jsonb;default:'[]'"` // fix tasks added on PR rollback
	RollbackHistory    datatypes.JSON `gorm:"type:jsonb;default:'[]'"` // rollback operation history
	CreatedAt          time.Time      `gorm:"autoCreateTime"`
	UpdatedAt          time.Time      `gorm:"autoUpdateTime"`

	// Relation
	Session *WorkSession `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`

	// Many-to-many relation for completed tasks
	CompletedTasks []Task `gorm:"many2many:execution_completed_tasks;joinForeignKey:execution_id;joinReferences:task_id"`
}

// ExecutionCompletedTask represents the junction table for execution completed tasks.
type ExecutionCompletedTask struct {
	ExecutionID uuid.UUID `gorm:"type:uuid;primaryKey"`
	TaskID      uuid.UUID `gorm:"type:uuid;primaryKey"`

	Execution *Execution `gorm:"foreignKey:ExecutionID;constraint:OnDelete:CASCADE"`
	Task      *Task      `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
}

// ExecutionValidationResult represents the validation result after task execution.
type ExecutionValidationResult struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID uuid.UUID `gorm:"type:uuid;not null;index"`
	TaskID    uuid.UUID `gorm:"type:uuid;not null;index"`

	// Formatting results
	FormatSuccess    *bool  `gorm:"type:boolean"`
	FormatOutput     string `gorm:"type:text"`
	FormatDurationMs *int64 `gorm:"type:bigint"`

	// Lint results
	LintSuccess     *bool  `gorm:"type:boolean"`
	LintOutput      string `gorm:"type:text"`
	LintIssuesFound *int   `gorm:"type:int"`
	LintIssuesFixed *int   `gorm:"type:int"`
	LintDurationMs  *int64 `gorm:"type:bigint"`

	// Test results
	TestSuccess    *bool  `gorm:"type:boolean"`
	TestOutput     string `gorm:"type:text"`
	TestPassed     *int   `gorm:"type:int"`
	TestFailed     *int   `gorm:"type:int"`
	TestDurationMs *int64 `gorm:"type:bigint"`

	// Overall result
	OverallStatus string         `gorm:"type:varchar(50);not null;index"` // passed, failed, warned, skipped
	Warnings      datatypes.JSON `gorm:"type:jsonb;default:'[]'"`

	// Timing
	TotalDurationMs *int64    `gorm:"type:bigint"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`

	// Relations
	Session *WorkSession `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
	Task    *Task        `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
}

// DomainEvent represents a stored domain event.
type DomainEvent struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AggregateID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	AggregateType string         `gorm:"type:varchar(100);not null;index"` // e.g. "WorkSession"
	EventType     string         `gorm:"type:varchar(100);not null;index"` // e.g. "WorkSessionStarted"
	Payload       datatypes.JSON `gorm:"type:jsonb;not null"`
	OccurredAt    time.Time      `gorm:"autoCreateTime;index"`
}

// AuditLog represents an audit trail entry.
type AuditLog struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Timestamp   time.Time  `gorm:"autoCreateTime;index"`
	SessionID   *uuid.UUID `gorm:"type:uuid;index"`
	Repository  string     `gorm:"type:varchar(255);not null;index"`
	IssueNumber *int64     `gorm:"type:bigint"`

	// Actor information
	Actor     string `gorm:"type:varchar(255);not null;index"` // GitHub username
	ActorRole string `gorm:"type:varchar(50)"`                 // admin, issue_author

	// Operation details
	Operation    string `gorm:"type:varchar(100);not null;index"` // operation type
	ResourceType string `gorm:"type:varchar(100)"`                // resource type
	ResourceID   string `gorm:"type:varchar(255)"`                // resource identifier

	// Result
	Result     string `gorm:"type:varchar(50);not null;index"` // success, failed, denied
	DurationMs *int64 `gorm:"type:bigint"`                     // operation duration in milliseconds

	// Details
	Parameters   datatypes.JSON `gorm:"type:jsonb"` // operation parameters
	Output       string         `gorm:"type:text"`  // output summary (truncated)
	ErrorMessage string         `gorm:"type:text"`  // error message

	CreatedAt time.Time `gorm:"autoCreateTime"`

	// Relation
	Session *WorkSession `gorm:"foreignKey:SessionID;constraint:OnDelete:SET NULL"`
}

// WebhookDelivery represents a webhook delivery record for idempotency.
type WebhookDelivery struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DeliveryID     string     `gorm:"type:varchar(255);unique;not null;index"` // GitHub delivery ID (X-GitHub-Delivery)
	EventType      string     `gorm:"type:varchar(100);not null"`              // e.g. issues, issue_comment
	Repository     string     `gorm:"type:varchar(255);not null"`              // repository name
	PayloadHash    string     `gorm:"type:varchar(64)"`                        // payload SHA256 hash
	Processed      bool       `gorm:"default:false;index"`                     // whether processed
	ProcessResult  string     `gorm:"type:varchar(50)"`                        // success, ignored, error
	ProcessMessage string     `gorm:"type:text"`                               // processing message
	CreatedAt      time.Time  `gorm:"autoCreateTime;index"`
	ExpiresAt      *time.Time `gorm:"type:timestamp with time zone;index"` // expiration time (default 24h)
}

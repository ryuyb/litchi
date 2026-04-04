package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// PaginationParams holds pagination parameters for list queries.
type PaginationParams struct {
	Page     int // Page number (1-based)
	PageSize int // Number of items per page
}

// PaginationResult holds pagination metadata for query results.
type PaginationResult struct {
	Page       int // Current page
	PageSize   int // Items per page
	TotalItems int // Total number of items
	TotalPages int // Total number of pages
}

// WorkSessionFilter holds filter parameters for WorkSession queries.
type WorkSessionFilter struct {
	Status     *aggregate.SessionStatus // Filter by session status
	Stage      *valueobject.Stage       // Filter by current stage
	Repository *string                  // Filter by repository name
	Author     *string                  // Filter by issue author
}

// WorkSessionRepository defines the repository interface for WorkSession aggregate.
type WorkSessionRepository interface {
	// Create creates a new WorkSession in the database.
	Create(ctx context.Context, session *aggregate.WorkSession) error

	// Update updates an existing WorkSession in the database.
	Update(ctx context.Context, session *aggregate.WorkSession) error

	// FindByID finds a WorkSession by its ID.
	// Returns nil if not found (no error).
	FindByID(ctx context.Context, id uuid.UUID) (*aggregate.WorkSession, error)

	// FindByIssueID finds a WorkSession by its associated Issue ID.
	// Returns nil if not found (no error).
	FindByIssueID(ctx context.Context, issueID uuid.UUID) (*aggregate.WorkSession, error)

	// FindByGitHubIssue finds a WorkSession by GitHub issue number and repository.
	// Returns nil if not found (no error).
	FindByGitHubIssue(ctx context.Context, repository string, issueNumber int) (*aggregate.WorkSession, error)

	// FindByStatus finds all WorkSessions with the given status.
	FindByStatus(ctx context.Context, status aggregate.SessionStatus) ([]*aggregate.WorkSession, error)

	// FindByStage finds all WorkSessions at the given stage.
	FindByStage(ctx context.Context, stage valueobject.Stage) ([]*aggregate.WorkSession, error)

	// ListWithPagination lists WorkSessions with pagination and optional filtering.
	ListWithPagination(ctx context.Context, params PaginationParams, filter *WorkSessionFilter) ([]*aggregate.WorkSession, *PaginationResult, error)

	// FindActiveByRepository finds all active sessions for a repository.
	FindActiveByRepository(ctx context.Context, repository string) ([]*aggregate.WorkSession, error)

	// Delete deletes a WorkSession by its ID.
	// This is a soft delete if supported, otherwise hard delete.
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByGitHubIssue checks if a WorkSession exists for the given GitHub issue.
	ExistsByGitHubIssue(ctx context.Context, repository string, issueNumber int) (bool, error)
}

// DesignRepository defines the repository interface for Design entity.
// It provides CRUD operations and version management for design documents.
type DesignRepository interface {
	// Save saves a design entity to the database.
	// It creates a new design if ID is not set, or updates existing one.
	// Versions are saved automatically when adding new versions.
	Save(ctx context.Context, design *entity.Design, sessionID uuid.UUID) error

	// FindByID finds a design by its unique identifier.
	// Returns nil if not found (no error).
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Design, error)

	// FindBySessionID finds the design associated with a work session.
	// Returns nil if not found (no error).
	FindBySessionID(ctx context.Context, sessionID uuid.UUID) (*entity.Design, error)

	// GetLatestVersion returns the latest version number for a design.
	// Returns 0 if no design exists.
	GetLatestVersion(ctx context.Context, sessionID uuid.UUID) (int, error)

	// FindVersions retrieves all versions of a design in chronological order.
	// Returns empty slice if no versions found.
	FindVersions(ctx context.Context, sessionID uuid.UUID) ([]valueobject.DesignVersion, error)

	// FindVersionByNumber retrieves a specific version of a design.
	// Returns error if version not found.
	FindVersionByNumber(ctx context.Context, sessionID uuid.UUID, versionNum int) (*valueobject.DesignVersion, error)

	// Delete removes a design and all its versions from the database.
	// This is typically called when the parent WorkSession is deleted (handled by CASCADE).
	Delete(ctx context.Context, id uuid.UUID) error
}

// AuditLogFilter defines filter criteria for querying audit logs.
type AuditLogFilter struct {
	SessionID    *uuid.UUID          // Filter by session ID
	Repository   string              // Filter by repository name
	Actor        string              // Filter by actor username
	ActorRole    valueobject.ActorRole // Filter by actor role
	Operation    valueobject.OperationType // Filter by operation type
	Result       valueobject.AuditResult // Filter by result status
	StartTime    *time.Time          // Filter by start time (inclusive)
	EndTime      *time.Time          // Filter by end time (inclusive)
	ResourceType string              // Filter by resource type
}

// AuditLogListOptions defines options for listing audit logs.
type AuditLogListOptions struct {
	Filter   AuditLogFilter // Filter criteria
	Offset   int            // Offset for pagination
	Limit    int            // Limit for pagination (0 means default)
	OrderBy  string         // Order by field (default: timestamp desc)
}

// AuditLogRepository defines the repository interface for AuditLog entity.
type AuditLogRepository interface {
	// Save persists a new audit log entry.
	// Audit logs are immutable - only creation is allowed.
	Save(ctx context.Context, auditLog *entity.AuditLog) error

	// FindByID retrieves an audit log by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error)

	// List retrieves audit logs based on filter criteria and pagination.
	List(ctx context.Context, opts AuditLogListOptions) ([]*entity.AuditLog, int64, error)

	// ListBySessionID retrieves all audit logs for a specific session.
	ListBySessionID(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*entity.AuditLog, int64, error)

	// ListByRepository retrieves audit logs for a specific repository.
	ListByRepository(ctx context.Context, repository string, offset, limit int) ([]*entity.AuditLog, int64, error)

	// ListByActor retrieves audit logs by a specific actor.
	ListByActor(ctx context.Context, actor string, offset, limit int) ([]*entity.AuditLog, int64, error)

	// ListByTimeRange retrieves audit logs within a time range.
	ListByTimeRange(ctx context.Context, startTime, endTime time.Time, offset, limit int) ([]*entity.AuditLog, int64, error)

	// CountBySession counts audit logs for a specific session.
	CountBySession(ctx context.Context, sessionID uuid.UUID) (int64, error)

	// DeleteBeforeTime deletes audit logs older than the specified time.
	// This is used for data retention/cleanup.
	DeleteBeforeTime(ctx context.Context, before time.Time) (int64, error)
}
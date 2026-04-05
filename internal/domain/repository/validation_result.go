package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// ValidationResultFilter is the filter for querying validation results.
type ValidationResultFilter struct {
	SessionID *uuid.UUID
	TaskID    *uuid.UUID
	Status    *valueobject.ValidationStatus
}

// ValidationResultRepository manages validation result persistence.
type ValidationResultRepository interface {
	// Save saves a validation result.
	Save(ctx context.Context, result *valueobject.ValidationResult, sessionID, taskID uuid.UUID) error

	// FindByTaskID finds a validation result by task ID.
	FindByTaskID(ctx context.Context, taskID uuid.UUID) (*valueobject.ValidationResult, error)

	// FindBySessionID finds all validation results for a session.
	FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*valueobject.ValidationResult, error)

	// FindLatestBySessionID finds the most recent validation result for a session.
	FindLatestBySessionID(ctx context.Context, sessionID uuid.UUID) (*valueobject.ValidationResult, error)

	// List lists validation results with pagination.
	List(ctx context.Context, params PaginationParams, filter *ValidationResultFilter) ([]*valueobject.ValidationResult, *PaginationResult, error)

	// DeleteBySessionID deletes all validation results for a session.
	DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error
}
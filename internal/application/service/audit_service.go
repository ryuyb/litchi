// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// AuditService handles audit log operations.
// It provides query, filtering, and management functions for audit logs.
//
// Core responsibilities:
// 1. Query audit logs - by session, repository, actor, time range
// 2. Filter audit logs - with complex filter criteria
// 3. Statistics - count audit logs for monitoring
// 4. Cleanup - delete expired audit logs for retention
type AuditService struct {
	auditRepo repository.AuditLogRepository
	config    *config.Config
	logger    *zap.Logger
}

// NewAuditService creates a new AuditService.
func NewAuditService(
	auditRepo repository.AuditLogRepository,
	config *config.Config,
	logger *zap.Logger,
) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
		config:    config,
		logger:    logger.Named("audit_service"),
	}
}

// GetAuditLog retrieves a single audit log by ID.
//
// Returns nil if audit log not found.
func (s *AuditService) GetAuditLog(
	ctx context.Context,
	id uuid.UUID,
) (*entity.AuditLog, error) {
	log, err := s.auditRepo.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to find audit log",
			zap.String("id", id.String()),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return log, nil
}

// ListAuditLogs lists audit logs with filtering and pagination.
//
// Supports filtering by:
// - Session ID
// - Repository
// - Actor (username)
// - Actor role
// - Operation type
// - Result status
// - Time range
// - Resource type
//
// Supports pagination with offset and limit.
// Supports ordering by timestamp (default: descending).
//
// Returns audit logs and total count.
func (s *AuditService) ListAuditLogs(
	ctx context.Context,
	filter AuditLogFilterParams,
	offset, limit int,
	orderBy string,
) ([]*entity.AuditLog, int64, error) {
	// Build repository filter
	repoFilter := repository.AuditLogFilter{
		Repository:   filter.Repository,
		Actor:        filter.Actor,
		ActorRole:    filter.ActorRole,
		Operation:    filter.Operation,
		Result:       filter.Result,
		ResourceType: filter.ResourceType,
	}

	if filter.SessionID != nil {
		repoFilter.SessionID = filter.SessionID
	}
	if filter.StartTime != nil {
		repoFilter.StartTime = filter.StartTime
	}
	if filter.EndTime != nil {
		repoFilter.EndTime = filter.EndTime
	}

	// Build list options
	opts := repository.AuditLogListOptions{
		Filter:  repoFilter,
		Offset:  offset,
		Limit:   limit,
		OrderBy: orderBy,
	}

	// Default ordering: timestamp desc
	if opts.OrderBy == "" {
		opts.OrderBy = "timestamp desc"
	}

	// Default limit
	if opts.Limit == 0 {
		opts.Limit = 50
	}

	logs, total, err := s.auditRepo.List(ctx, opts)
	if err != nil {
		s.logger.Error("failed to list audit logs",
			zap.Error(err),
		)
		return nil, 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return logs, total, nil
}

// ListBySession lists all audit logs for a specific session.
//
// Supports pagination with offset and limit.
// Returns audit logs and total count.
func (s *AuditService) ListBySession(
	ctx context.Context,
	sessionID uuid.UUID,
	offset, limit int,
) ([]*entity.AuditLog, int64, error) {
	// Default limit
	if limit == 0 {
		limit = 50
	}

	logs, total, err := s.auditRepo.ListBySessionID(ctx, sessionID, offset, limit)
	if err != nil {
		s.logger.Error("failed to list audit logs by session",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		return nil, 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return logs, total, nil
}

// ListByRepository lists audit logs for a specific repository.
//
// Supports pagination with offset and limit.
// Returns audit logs and total count.
func (s *AuditService) ListByRepository(
	ctx context.Context,
	repository string,
	offset, limit int,
) ([]*entity.AuditLog, int64, error) {
	// Default limit
	if limit == 0 {
		limit = 50
	}

	logs, total, err := s.auditRepo.ListByRepository(ctx, repository, offset, limit)
	if err != nil {
		s.logger.Error("failed to list audit logs by repository",
			zap.String("repository", repository),
			zap.Error(err),
		)
		return nil, 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return logs, total, nil
}

// ListByActor lists audit logs by a specific actor.
//
// Supports pagination with offset and limit.
// Returns audit logs and total count.
func (s *AuditService) ListByActor(
	ctx context.Context,
	actor string,
	offset, limit int,
) ([]*entity.AuditLog, int64, error) {
	// Default limit
	if limit == 0 {
		limit = 50
	}

	logs, total, err := s.auditRepo.ListByActor(ctx, actor, offset, limit)
	if err != nil {
		s.logger.Error("failed to list audit logs by actor",
			zap.String("actor", actor),
			zap.Error(err),
		)
		return nil, 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return logs, total, nil
}

// ListByTimeRange lists audit logs within a time range.
//
// Supports pagination with offset and limit.
// Returns audit logs and total count.
func (s *AuditService) ListByTimeRange(
	ctx context.Context,
	startTime, endTime time.Time,
	offset, limit int,
) ([]*entity.AuditLog, int64, error) {
	// Validate time range
	if startTime.After(endTime) {
		return nil, 0, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"startTime must be before endTime",
		)
	}

	// Default limit
	if limit == 0 {
		limit = 50
	}

	logs, total, err := s.auditRepo.ListByTimeRange(ctx, startTime, endTime, offset, limit)
	if err != nil {
		s.logger.Error("failed to list audit logs by time range",
			zap.Time("start_time", startTime),
			zap.Time("end_time", endTime),
			zap.Error(err),
		)
		return nil, 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return logs, total, nil
}

// CountBySession counts audit logs for a specific session.
//
// Returns the total count.
func (s *AuditService) CountBySession(
	ctx context.Context,
	sessionID uuid.UUID,
) (int64, error) {
	count, err := s.auditRepo.CountBySession(ctx, sessionID)
	if err != nil {
		s.logger.Error("failed to count audit logs by session",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		return 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return count, nil
}

// DeleteExpired deletes audit logs older than the configured retention period.
//
// Uses Audit.RetentionDays from config to determine cutoff time.
// Returns the number of deleted records.
func (s *AuditService) DeleteExpired(
	ctx context.Context,
) (int64, error) {
	// Check if audit is enabled
	if !s.config.Audit.Enabled {
		s.logger.Warn("audit logging is disabled, skipping cleanup")
		return 0, nil
	}

	// Calculate cutoff time based on retention days
	retentionDays := s.config.Audit.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 30 // Default 30 days
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// Delete old audit logs
	count, err := s.auditRepo.DeleteBeforeTime(ctx, cutoffTime)
	if err != nil {
		s.logger.Error("failed to delete expired audit logs",
			zap.Time("cutoff_time", cutoffTime),
			zap.Error(err),
		)
		return 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.logger.Info("deleted expired audit logs",
		zap.Int64("count", count),
		zap.Time("cutoff_time", cutoffTime),
		zap.Int("retention_days", retentionDays),
	)

	return count, nil
}

// GetSessionAuditSummary returns a summary of audit logs for a session.
// This includes counts by operation type and result status.
//
// Returns a summary object with aggregated statistics.
func (s *AuditService) GetSessionAuditSummary(
	ctx context.Context,
	sessionID uuid.UUID,
) (*SessionAuditSummary, error) {
	// Get all audit logs for the session (no pagination)
	logs, _, err := s.auditRepo.ListBySessionID(ctx, sessionID, 0, 1000)
	if err != nil {
		s.logger.Error("failed to get audit logs for summary",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// Aggregate statistics
	summary := &SessionAuditSummary{
		SessionID:  sessionID,
		TotalCount: len(logs),
		ByResult:   make(map[valueobject.AuditResult]int),
		ByOperation: make(map[valueobject.OperationType]int),
	}

	// Calculate total duration
	totalDuration := 0
	for _, log := range logs {
		summary.ByResult[log.Result]++
		summary.ByOperation[log.Operation]++
		totalDuration += log.Duration
	}
	summary.TotalDurationMs = totalDuration

	// Calculate average duration
	if len(logs) > 0 {
		summary.AverageDurationMs = totalDuration / len(logs)
	}

	// Get first and last timestamps
	if len(logs) > 0 {
		summary.FirstTimestamp = logs[0].Timestamp
		summary.LastTimestamp = logs[len(logs)-1].Timestamp
	}

	return summary, nil
}

// AuditLogFilterParams represents filter parameters for audit log queries.
// This is a simplified version of AuditLogFilter for API use.
type AuditLogFilterParams struct {
	SessionID    *uuid.UUID                 `json:"sessionId,omitempty"`
	Repository   string                     `json:"repository,omitempty"`
	Actor        string                     `json:"actor,omitempty"`
	ActorRole    valueobject.ActorRole      `json:"actorRole,omitempty"`
	Operation    valueobject.OperationType  `json:"operation,omitempty"`
	Result       valueobject.AuditResult    `json:"result,omitempty"`
	StartTime    *time.Time                 `json:"startTime,omitempty"`
	EndTime      *time.Time                 `json:"endTime,omitempty"`
	ResourceType string                     `json:"resourceType,omitempty"`
}

// SessionAuditSummary represents a summary of audit logs for a session.
type SessionAuditSummary struct {
	SessionID         uuid.UUID                     `json:"sessionId"`
	TotalCount        int                           `json:"totalCount"`
	TotalDurationMs   int                           `json:"totalDurationMs"`
	AverageDurationMs int                           `json:"averageDurationMs"`
	ByResult          map[valueobject.AuditResult]int `json:"byResult"`
	ByOperation       map[valueobject.OperationType]int `json:"byOperation"`
	FirstTimestamp    time.Time                     `json:"firstTimestamp,omitempty"`
	LastTimestamp     time.Time                     `json:"lastTimestamp,omitempty"`
}

// FormatAuditLog formats an audit log for display.
// This truncates output and error messages to configured max length.
func (s *AuditService) FormatAuditLog(log *entity.AuditLog) *FormattedAuditLog {
	maxOutputLength := s.config.Audit.MaxOutputLength
	if maxOutputLength <= 0 {
		maxOutputLength = 500 // Default
	}

	formatted := &FormattedAuditLog{
		ID:           log.ID,
		Timestamp:    log.Timestamp,
		SessionID:    log.SessionID,
		Repository:   log.Repository,
		IssueNumber:  log.IssueNumber,
		Actor:        log.Actor,
		ActorRole:    log.ActorRole,
		Operation:    log.Operation,
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		Result:       log.Result,
		Duration:     log.Duration,
	}

	// Truncate output
	if len(log.Output) > maxOutputLength {
		formatted.Output = log.Output[:maxOutputLength] + "..."
	} else {
		formatted.Output = log.Output
	}

	// Truncate error
	if len(log.Error) > maxOutputLength {
		formatted.Error = log.Error[:maxOutputLength] + "..."
	} else {
		formatted.Error = log.Error
	}

	return formatted
}

// FormattedAuditLog represents an audit log formatted for display.
// Output and Error fields are truncated for readability.
type FormattedAuditLog struct {
	ID           uuid.UUID                 `json:"id"`
	Timestamp    time.Time                 `json:"timestamp"`
	SessionID    uuid.UUID                 `json:"sessionId"`
	Repository   string                    `json:"repository"`
	IssueNumber  int                       `json:"issueNumber"`
	Actor        string                    `json:"actor"`
	ActorRole    valueobject.ActorRole     `json:"actorRole"`
	Operation    valueobject.OperationType `json:"operation"`
	ResourceType string                    `json:"resourceType"`
	ResourceID   string                    `json:"resourceId"`
	Result       valueobject.AuditResult   `json:"result"`
	Duration     int                       `json:"duration"`
	Output       string                    `json:"output"`
	Error        string                    `json:"error"`
}

// IsSensitiveOperation checks if an operation is marked as sensitive.
// Sensitive operations may require additional handling or logging.
func (s *AuditService) IsSensitiveOperation(operation valueobject.OperationType) bool {
	sensitiveOps := s.config.Audit.SensitiveOperations
	if len(sensitiveOps) == 0 {
		// Default sensitive operations
		return operation == valueobject.OpSessionTerminate ||
			operation == valueobject.OpSessionPause ||
			operation == valueobject.OpBashExecute ||
			operation == valueobject.OpGitOperation ||
			operation == valueobject.OpPRClose ||
			operation == valueobject.OpPRMerge
	}

	// Check against configured list
	for _, sensitiveOp := range sensitiveOps {
		if string(operation) == sensitiveOp {
			return true
		}
	}

	return false
}

// ValidateFilterParams validates filter parameters.
// Returns error if any parameter is invalid.
func (s *AuditService) ValidateFilterParams(filter AuditLogFilterParams) error {
	// Validate actor role if set
	if filter.ActorRole != "" && !filter.ActorRole.IsValid() {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("invalid actor role: %s", filter.ActorRole),
		)
	}

	// Validate operation type if set
	if filter.Operation != "" && !filter.Operation.IsValid() {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("invalid operation type: %s", filter.Operation),
		)
	}

	// Validate result if set
	if filter.Result != "" && !filter.Result.IsValid() {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("invalid audit result: %s", filter.Result),
		)
	}

	// Validate time range if both are set
	if filter.StartTime != nil && filter.EndTime != nil {
		if filter.StartTime.After(*filter.EndTime) {
			return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
				"startTime must be before endTime",
			)
		}
	}

	return nil
}
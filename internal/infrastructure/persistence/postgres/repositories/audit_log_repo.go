// Package repositories provides GORM-based implementations of domain repositories.
package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/models"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AuditLogRepositoryParams holds dependencies for AuditLogRepository.
type AuditLogRepositoryParams struct {
	fx.In

	DB     *gorm.DB
	Logger *zap.Logger
}

// auditLogRepository implements repository.AuditLogRepository using GORM.
type auditLogRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewAuditLogRepository creates a new AuditLogRepository instance.
func NewAuditLogRepository(p AuditLogRepositoryParams) repository.AuditLogRepository {
	return &auditLogRepository{
		db:     p.DB,
		logger: p.Logger,
	}
}

// Save persists a new audit log entry.
func (r *auditLogRepository) Save(ctx context.Context, auditLog *entity.AuditLog) error {
	model := r.toModel(auditLog)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Error("failed to save audit log",
			zap.Error(err),
			zap.String("sessionId", auditLog.SessionID.String()),
			zap.String("operation", string(auditLog.Operation)),
		)
		return err
	}
	// Update entity ID after creation
	auditLog.ID = model.ID
	return nil
}

// FindByID retrieves an audit log by its ID.
func (r *auditLogRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error) {
	var model models.AuditLog
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toEntity(&model), nil
}

// List retrieves audit logs based on filter criteria and pagination.
func (r *auditLogRepository) List(ctx context.Context, opts repository.AuditLogListOptions) ([]*entity.AuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.AuditLog{})

	// Apply filters
	query = r.applyFilter(query, opts.Filter)

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error("failed to count audit logs", zap.Error(err))
		return nil, 0, err
	}

	// Apply ordering
	orderBy := opts.OrderBy
	if orderBy == "" {
		orderBy = "timestamp DESC"
	}
	query = query.Order(orderBy)

	// Apply pagination
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}

	// Execute query
	var modelList []models.AuditLog
	if err := query.Find(&modelList).Error; err != nil {
		r.logger.Error("failed to list audit logs", zap.Error(err))
		return nil, 0, err
	}

	// Convert to entities
	entities := make([]*entity.AuditLog, len(modelList))
	for i, model := range modelList {
		entities[i] = r.toEntity(&model)
	}

	return entities, total, nil
}

// ListBySessionID retrieves all audit logs for a specific session.
func (r *auditLogRepository) ListBySessionID(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return r.List(ctx, repository.AuditLogListOptions{
		Filter: repository.AuditLogFilter{
			SessionID: &sessionID,
		},
		Offset: offset,
		Limit:  limit,
	})
}

// ListByRepository retrieves audit logs for a specific repository.
func (r *auditLogRepository) ListByRepository(ctx context.Context, repositoryName string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return r.List(ctx, repository.AuditLogListOptions{
		Filter: repository.AuditLogFilter{
			Repository: repositoryName,
		},
		Offset: offset,
		Limit:  limit,
	})
}

// ListByActor retrieves audit logs by a specific actor.
func (r *auditLogRepository) ListByActor(ctx context.Context, actor string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return r.List(ctx, repository.AuditLogListOptions{
		Filter: repository.AuditLogFilter{
			Actor: actor,
		},
		Offset: offset,
		Limit:  limit,
	})
}

// ListByTimeRange retrieves audit logs within a time range.
func (r *auditLogRepository) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return r.List(ctx, repository.AuditLogListOptions{
		Filter: repository.AuditLogFilter{
			StartTime: &startTime,
			EndTime:   &endTime,
		},
		Offset: offset,
		Limit:  limit,
	})
}

// CountBySession counts audit logs for a specific session.
func (r *auditLogRepository) CountBySession(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.AuditLog{}).
		Where("session_id = ?", sessionID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteBeforeTime deletes audit logs older than the specified time.
func (r *auditLogRepository) DeleteBeforeTime(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("timestamp < ?", before).
		Delete(&models.AuditLog{})
	if result.Error != nil {
		r.logger.Error("failed to delete old audit logs",
			zap.Error(result.Error),
			zap.Time("before", before),
		)
		return 0, result.Error
	}
	r.logger.Info("deleted old audit logs",
		zap.Int64("count", result.RowsAffected),
		zap.Time("before", before),
	)
	return result.RowsAffected, nil
}

// applyFilter applies filter criteria to the query.
func (r *auditLogRepository) applyFilter(query *gorm.DB, filter repository.AuditLogFilter) *gorm.DB {
	if filter.SessionID != nil {
		query = query.Where("session_id = ?", filter.SessionID)
	}
	if filter.Repository != "" {
		query = query.Where("repository = ?", filter.Repository)
	}
	if filter.Actor != "" {
		query = query.Where("actor = ?", filter.Actor)
	}
	if filter.ActorRole != "" {
		query = query.Where("actor_role = ?", filter.ActorRole)
	}
	if filter.Operation != "" {
		query = query.Where("operation = ?", filter.Operation)
	}
	if filter.Result != "" {
		query = query.Where("result = ?", filter.Result)
	}
	if filter.StartTime != nil {
		query = query.Where("timestamp >= ?", filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("timestamp <= ?", filter.EndTime)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	return query
}

// toModel converts entity.AuditLog to models.AuditLog.
func (r *auditLogRepository) toModel(e *entity.AuditLog) *models.AuditLog {
	model := &models.AuditLog{
		ID:           e.ID,
		Timestamp:    e.Timestamp,
		Repository:   e.Repository,
		Actor:        e.Actor,
		ActorRole:    string(e.ActorRole),
		Operation:    string(e.Operation),
		ResourceType: e.ResourceType,
		ResourceID:   e.ResourceID,
		Result:       string(e.Result),
		DurationMs:   ptrToInt64(e.Duration),
		Output:       e.Output,
		ErrorMessage: e.Error,
		CreatedAt:    time.Now(),
	}

	// Handle nullable SessionID
	if e.SessionID != uuid.Nil {
		model.SessionID = &e.SessionID
	}

	// Handle nullable IssueNumber
	if e.IssueNumber > 0 {
		issueNum := int64(e.IssueNumber)
		model.IssueNumber = &issueNum
	}

	// Convert Parameters map to JSON
	if e.Parameters != nil && len(e.Parameters) > 0 {
		model.Parameters = datatypes.JSON{}
		// Note: In production, you would properly marshal the map to JSON
		// For now, we use json.Marshal which handles the conversion
		if data, err := json.Marshal(e.Parameters); err == nil {
			model.Parameters = datatypes.JSON(data)
		}
	}

	return model
}

// toEntity converts models.AuditLog to entity.AuditLog.
func (r *auditLogRepository) toEntity(m *models.AuditLog) *entity.AuditLog {
	e := &entity.AuditLog{
		ID:           m.ID,
		Timestamp:    m.Timestamp,
		Repository:   m.Repository,
		Actor:        m.Actor,
		ActorRole:    valueobject.ActorRole(m.ActorRole),
		Operation:    valueobject.OperationType(m.Operation),
		ResourceType: m.ResourceType,
		ResourceID:   m.ResourceID,
		Result:       valueobject.AuditResult(m.Result),
		Output:       m.Output,
		Error:        m.ErrorMessage,
		Parameters:   make(map[string]any),
	}

	// Handle nullable SessionID
	if m.SessionID != nil {
		e.SessionID = *m.SessionID
	}

	// Handle nullable IssueNumber
	if m.IssueNumber != nil {
		e.IssueNumber = int(*m.IssueNumber)
	}

	// Handle nullable DurationMs
	if m.DurationMs != nil {
		e.Duration = int(*m.DurationMs)
	}

	// Unmarshal Parameters JSON to map
	if m.Parameters != nil {
		var params map[string]any
		if err := json.Unmarshal(m.Parameters, &params); err == nil {
			e.Parameters = params
		}
	}

	return e
}

// ptrToInt64 returns a pointer to the int64 value of an int.
func ptrToInt64(v int) *int64 {
	if v == 0 {
		return nil
	}
	v64 := int64(v)
	return &v64
}

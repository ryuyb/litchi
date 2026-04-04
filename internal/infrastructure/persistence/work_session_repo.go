// Package persistence provides GORM-based repository implementations.
package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/converter"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/models"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WorkSessionRepositoryModule provides the WorkSession repository via Fx.
var WorkSessionRepositoryModule = fx.Module("work_session_repository",
	fx.Provide(NewWorkSessionRepository),
)

// WorkSessionRepoParams holds dependencies for WorkSessionRepository.
type WorkSessionRepoParams struct {
	fx.In

	DB     *gorm.DB
	Logger *zap.Logger
}

// workSessionRepository implements repository.WorkSessionRepository using GORM.
type workSessionRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewWorkSessionRepository creates a new WorkSessionRepository instance.
func NewWorkSessionRepository(p WorkSessionRepoParams) repository.WorkSessionRepository {
	return &workSessionRepository{
		db:     p.DB,
		logger: p.Logger.Named("work_session_repo"),
	}
}

// preloadAll applies all standard preloads for WorkSession queries.
func (r *workSessionRepository) preloadAll(query *gorm.DB) *gorm.DB {
	return query.
		Preload("Issue").
		Preload("Clarification").
		Preload("Design").
		Preload("Design.Versions").
		Preload("Tasks").
		Preload("Tasks.Dependencies").
		Preload("Execution").
		Preload("Execution.CompletedTasks")
}

// Create creates a new WorkSession in the database.
func (r *workSessionRepository) Create(ctx context.Context, session *aggregate.WorkSession) error {
	if session == nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail("session cannot be nil")
	}

	model, err := converter.WorkSessionToModel(session)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrValidationFailed, err)
	}

	// Use transaction to create session with all related entities
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create Issue first (if exists)
		if model.Issue != nil {
			if err := tx.Create(model.Issue).Error; err != nil {
				return fmt.Errorf("failed to create issue: %w", err)
			}
			model.IssueID = model.Issue.ID
		}

		// Create WorkSession
		if err := tx.Create(model).Error; err != nil {
			return fmt.Errorf("failed to create work session: %w", err)
		}

		// Create Clarification (if exists)
		if model.Clarification != nil {
			model.Clarification.SessionID = model.ID
			if err := tx.Create(model.Clarification).Error; err != nil {
				return fmt.Errorf("failed to create clarification: %w", err)
			}
		}

		// Create Design and its versions (if exists)
		if model.Design != nil {
			model.Design.SessionID = model.ID
			if err := tx.Create(model.Design).Error; err != nil {
				return fmt.Errorf("failed to create design: %w", err)
			}
			// Create design versions
			if len(model.Design.Versions) > 0 {
				for i := range model.Design.Versions {
					model.Design.Versions[i].DesignID = model.Design.ID
				}
				if err := tx.Create(&model.Design.Versions).Error; err != nil {
					return fmt.Errorf("failed to create design versions: %w", err)
				}
			}
		}

		// Create Tasks (if exists)
		if len(model.Tasks) > 0 {
			for i := range model.Tasks {
				model.Tasks[i].SessionID = model.ID
			}
			if err := tx.Create(&model.Tasks).Error; err != nil {
				return fmt.Errorf("failed to create tasks: %w", err)
			}
		}

		// Create Execution (if exists)
		if model.Execution != nil {
			model.Execution.SessionID = model.ID
			if err := tx.Create(model.Execution).Error; err != nil {
				return fmt.Errorf("failed to create execution: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		r.logger.Error("failed to create work session",
			zap.String("session_id", session.ID.String()),
			zap.Error(err),
		)
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	r.logger.Info("work session created",
		zap.String("session_id", session.ID.String()),
	)

	return nil
}

// Update updates an existing WorkSession in the database.
func (r *workSessionRepository) Update(ctx context.Context, session *aggregate.WorkSession) error {
	if session == nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail("session cannot be nil")
	}

	model, err := converter.WorkSessionToModel(session)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrValidationFailed, err)
	}

	// Use transaction to update session with all related entities
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update WorkSession basic fields
		result := tx.Model(&models.WorkSession{}).
			Where("id = ?", session.ID).
			Updates(map[string]any{
				"current_stage": model.CurrentStage,
				"status":        model.Status,
				"updated_at":    model.UpdatedAt,
			})

		if result.Error != nil {
			return fmt.Errorf("failed to update work session: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
				"session not found: " + session.ID.String(),
			)
		}

		// Update Issue (if exists)
		if model.Issue != nil {
			if err := tx.Save(model.Issue).Error; err != nil {
				return fmt.Errorf("failed to update issue: %w", err)
			}
		}

		// Update or create Clarification
		if model.Clarification != nil {
			var existing models.Clarification
			err := tx.Where("session_id = ?", session.ID).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Create new
				model.Clarification.SessionID = session.ID
				if err := tx.Create(model.Clarification).Error; err != nil {
					return fmt.Errorf("failed to create clarification: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("failed to query clarification: %w", err)
			} else {
				// Update existing
				model.Clarification.ID = existing.ID
				if err := tx.Save(model.Clarification).Error; err != nil {
					return fmt.Errorf("failed to update clarification: %w", err)
				}
			}
		}

		// Update or create Design
		if model.Design != nil {
			var existing models.Design
			err := tx.Where("session_id = ?", session.ID).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Create new
				model.Design.SessionID = session.ID
				if err := tx.Create(model.Design).Error; err != nil {
					return fmt.Errorf("failed to create design: %w", err)
				}
				// Create design versions
				if len(model.Design.Versions) > 0 {
					for i := range model.Design.Versions {
						model.Design.Versions[i].DesignID = model.Design.ID
					}
					if err := tx.Create(&model.Design.Versions).Error; err != nil {
						return fmt.Errorf("failed to create design versions: %w", err)
					}
				}
			} else if err != nil {
				return fmt.Errorf("failed to query design: %w", err)
			} else {
				// Update existing
				model.Design.ID = existing.ID
				if err := tx.Save(model.Design).Error; err != nil {
					return fmt.Errorf("failed to update design: %w", err)
				}
				// Add new versions (append, not replace)
				if len(model.Design.Versions) > 0 {
					for i := range model.Design.Versions {
						model.Design.Versions[i].DesignID = model.Design.ID
					}
					if err := tx.Create(&model.Design.Versions).Error; err != nil {
						return fmt.Errorf("failed to create new design versions: %w", err)
					}
				}
			}
		}

		// Update or create Execution
		if model.Execution != nil {
			var existing models.Execution
			err := tx.Where("session_id = ?", session.ID).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Create new
				model.Execution.SessionID = session.ID
				if err := tx.Create(model.Execution).Error; err != nil {
					return fmt.Errorf("failed to create execution: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("failed to query execution: %w", err)
			} else {
				// Update existing
				model.Execution.ID = existing.ID
				if err := tx.Save(model.Execution).Error; err != nil {
					return fmt.Errorf("failed to update execution: %w", err)
				}
			}
		}

		// Update or create Tasks
		if len(model.Tasks) > 0 {
			// Get existing task IDs for this session
			var existingTaskIDs []uuid.UUID
			if err := tx.Model(&models.Task{}).
				Where("session_id = ?", session.ID).
				Pluck("id", &existingTaskIDs).Error; err != nil {
				return fmt.Errorf("failed to query existing tasks: %w", err)
			}

			existingSet := make(map[uuid.UUID]bool)
			for _, id := range existingTaskIDs {
				existingSet[id] = true
			}

			// Create or update each task
			for i := range model.Tasks {
				task := &model.Tasks[i]
				task.SessionID = session.ID

				if existingSet[task.ID] {
					// Update existing task
					if err := tx.Save(task).Error; err != nil {
						return fmt.Errorf("failed to update task %s: %w", task.ID, err)
					}
				} else {
					// Create new task
					if err := tx.Create(task).Error; err != nil {
						return fmt.Errorf("failed to create task %s: %w", task.ID, err)
					}
				}

				// Update task dependencies (many-to-many)
				// GORM's Association mode will handle the junction table
				if len(task.Dependencies) > 0 {
					if err := tx.Model(task).Association("Dependencies").Replace(task.Dependencies); err != nil {
						return fmt.Errorf("failed to update task dependencies for %s: %w", task.ID, err)
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		r.logger.Error("failed to update work session",
			zap.String("session_id", session.ID.String()),
			zap.Error(err),
		)
		return err
	}

	r.logger.Debug("work session updated",
		zap.String("session_id", session.ID.String()),
	)

	return nil
}

// FindByID finds a WorkSession by its ID.
func (r *workSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*aggregate.WorkSession, error) {
	var model models.WorkSession

	err := r.preloadAll(r.db.WithContext(ctx)).
		Where("id = ?", id).
		First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		r.logger.Error("failed to find work session by id",
			zap.String("id", id.String()),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	session, err := converter.WorkSessionFromModel(&model)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrValidationFailed, err)
	}

	return session, nil
}

// FindByIssueID finds a WorkSession by its associated Issue ID.
func (r *workSessionRepository) FindByIssueID(ctx context.Context, issueID uuid.UUID) (*aggregate.WorkSession, error) {
	var model models.WorkSession

	err := r.preloadAll(r.db.WithContext(ctx)).
		Where("issue_id = ?", issueID).
		First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		r.logger.Error("failed to find work session by issue id",
			zap.String("issue_id", issueID.String()),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	session, err := converter.WorkSessionFromModel(&model)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrValidationFailed, err)
	}

	return session, nil
}

// FindByGitHubIssue finds a WorkSession by GitHub issue number and repository.
func (r *workSessionRepository) FindByGitHubIssue(ctx context.Context, repository string, issueNumber int) (*aggregate.WorkSession, error) {
	var model models.WorkSession

	err := r.preloadAll(r.db.WithContext(ctx)).
		Joins("JOIN issues ON issues.id = work_sessions.issue_id").
		Where("issues.repository = ? AND issues.number = ?", repository, issueNumber).
		First(&model).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		r.logger.Error("failed to find work session by github issue",
			zap.String("repository", repository),
			zap.Int("issue_number", issueNumber),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	session, err := converter.WorkSessionFromModel(&model)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrValidationFailed, err)
	}

	return session, nil
}

// FindByStatus finds all WorkSessions with the given status.
func (r *workSessionRepository) FindByStatus(ctx context.Context, status aggregate.SessionStatus) ([]*aggregate.WorkSession, error) {
	var models []models.WorkSession

	err := r.preloadAll(r.db.WithContext(ctx)).
		Where("status = ?", string(status)).
		Order("created_at desc").
		Find(&models).Error

	if err != nil {
		r.logger.Error("failed to find work sessions by status",
			zap.String("status", string(status)),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	sessions := make([]*aggregate.WorkSession, len(models))
	for i, model := range models {
		session, err := converter.WorkSessionFromModel(&model)
		if err != nil {
			return nil, litchierrors.Wrap(litchierrors.ErrValidationFailed, err)
		}
		sessions[i] = session
	}

	return sessions, nil
}

// FindByStage finds all WorkSessions at the given stage.
func (r *workSessionRepository) FindByStage(ctx context.Context, stage valueobject.Stage) ([]*aggregate.WorkSession, error) {
	var models []models.WorkSession

	err := r.preloadAll(r.db.WithContext(ctx)).
		Where("current_stage = ?", stage.String()).
		Order("created_at desc").
		Find(&models).Error

	if err != nil {
		r.logger.Error("failed to find work sessions by stage",
			zap.String("stage", stage.String()),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	sessions := make([]*aggregate.WorkSession, len(models))
	for i, model := range models {
		session, err := converter.WorkSessionFromModel(&model)
		if err != nil {
			return nil, litchierrors.Wrap(litchierrors.ErrValidationFailed, err)
		}
		sessions[i] = session
	}

	return sessions, nil
}

// ListWithPagination lists WorkSessions with pagination and optional filtering.
func (r *workSessionRepository) ListWithPagination(ctx context.Context, params repository.PaginationParams, filter *repository.WorkSessionFilter) ([]*aggregate.WorkSession, *repository.PaginationResult, error) {
	// Validate and set defaults
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Build query with filters
	query := r.preloadAll(r.db.WithContext(ctx)).
		Model(&models.WorkSession{})

	if filter != nil {
		if filter.Status != nil {
			query = query.Where("status = ?", string(*filter.Status))
		}
		if filter.Stage != nil {
			query = query.Where("current_stage = ?", filter.Stage.String())
		}
		// Join issues table only once if either Repository or Author filter is needed
		if filter.Repository != nil || filter.Author != nil {
			query = query.Joins("JOIN issues ON issues.id = work_sessions.issue_id")
		}
		if filter.Repository != nil {
			query = query.Where("issues.repository = ?", *filter.Repository)
		}
		if filter.Author != nil {
			query = query.Where("issues.author = ?", *filter.Author)
		}
	}

	// Count total items
	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		r.logger.Error("failed to count work sessions", zap.Error(err))
		return nil, nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// Calculate pagination metadata
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize > 0 {
		totalPages++
	}

	// Query with pagination
	var modelList []models.WorkSession
	offset := (page - 1) * pageSize

	err := query.
		Order("created_at desc").
		Offset(offset).
		Limit(pageSize).
		Find(&modelList).Error

	if err != nil {
		r.logger.Error("failed to list work sessions", zap.Error(err))
		return nil, nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// Convert models to domain aggregates
	sessions := make([]*aggregate.WorkSession, len(modelList))
	for i, model := range modelList {
		session, err := converter.WorkSessionFromModel(&model)
		if err != nil {
			return nil, nil, litchierrors.Wrap(litchierrors.ErrValidationFailed, err)
		}
		sessions[i] = session
	}

	paginationResult := &repository.PaginationResult{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: int(totalItems),
		TotalPages: totalPages,
	}

	return sessions, paginationResult, nil
}

// FindActiveByRepository finds all active sessions for a repository.
func (r *workSessionRepository) FindActiveByRepository(ctx context.Context, repository string) ([]*aggregate.WorkSession, error) {
	var models []models.WorkSession

	err := r.preloadAll(r.db.WithContext(ctx)).
		Joins("JOIN issues ON issues.id = work_sessions.issue_id").
		Where("issues.repository = ? AND work_sessions.status = ?", repository, aggregate.SessionStatusActive).
		Order("created_at desc").
		Find(&models).Error

	if err != nil {
		r.logger.Error("failed to find active work sessions by repository",
			zap.String("repository", repository),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	sessions := make([]*aggregate.WorkSession, len(models))
	for i, model := range models {
		session, err := converter.WorkSessionFromModel(&model)
		if err != nil {
			return nil, litchierrors.Wrap(litchierrors.ErrValidationFailed, err)
		}
		sessions[i] = session
	}

	return sessions, nil
}

// Delete deletes a WorkSession by its ID.
func (r *workSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.WorkSession{})

	if result.Error != nil {
		r.logger.Error("failed to delete work session",
			zap.String("id", id.String()),
			zap.Error(result.Error),
		)
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, result.Error)
	}

	if result.RowsAffected == 0 {
		return litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			"session not found: " + id.String(),
		)
	}

	r.logger.Info("work session deleted",
		zap.String("id", id.String()),
	)

	return nil
}

// ExistsByGitHubIssue checks if a WorkSession exists for the given GitHub issue.
func (r *workSessionRepository) ExistsByGitHubIssue(ctx context.Context, repository string, issueNumber int) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.WorkSession{}).
		Joins("JOIN issues ON issues.id = work_sessions.issue_id").
		Where("issues.repository = ? AND issues.number = ?", repository, issueNumber).
		Count(&count).Error

	if err != nil {
		r.logger.Error("failed to check work session existence",
			zap.String("repository", repository),
			zap.Int("issue_number", issueNumber),
			zap.Error(err),
		)
		return false, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return count > 0, nil
}

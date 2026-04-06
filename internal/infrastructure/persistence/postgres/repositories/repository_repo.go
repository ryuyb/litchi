// Package repositories provides GORM-based implementations of repository interfaces.
package repositories

import (
	"context"

	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/converter"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/models"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// RepositoryRepoParams holds dependencies for RepositoryRepo.
type RepositoryRepoParams struct {
	fx.In

	DB *gorm.DB `name:"gorm_db"`
}

// RepositoryRepo implements repository.RepositoryRepository using GORM.
type RepositoryRepo struct {
	db *gorm.DB
}

// NewRepositoryRepo creates a new RepositoryRepo.
func NewRepositoryRepo(p RepositoryRepoParams) *RepositoryRepo {
	return &RepositoryRepo{db: p.DB}
}

// FindByName finds a repository by its name.
func (r *RepositoryRepo) FindByName(ctx context.Context, name string) (*entity.Repository, error) {
	var model models.Repository
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&model).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return converter.RepositoryFromModel(&model), nil
}

// Save saves a repository configuration.
func (r *RepositoryRepo) Save(ctx context.Context, repo *entity.Repository) error {
	model, err := converter.RepositoryToModel(repo)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if exists
		var existing models.Repository
		err := tx.Where("id = ?", model.ID).First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			// Create new
			return tx.Create(model).Error
		}
		if err != nil {
			return err
		}

		// Update existing
		return tx.Save(model).Error
	})
}

// Delete deletes a repository by its name.
func (r *RepositoryRepo) Delete(ctx context.Context, name string) error {
	return r.db.WithContext(ctx).
		Where("name = ?", name).
		Delete(&models.Repository{}).
		Error
}

// FindAll finds all repository configurations.
func (r *RepositoryRepo) FindAll(ctx context.Context) ([]*entity.Repository, error) {
	var modelList []*models.Repository
	err := r.db.WithContext(ctx).Find(&modelList).Error
	if err != nil {
		return nil, err
	}

	entities := make([]*entity.Repository, len(modelList))
	for i, m := range modelList {
		entities[i] = converter.RepositoryFromModel(m)
	}

	return entities, nil
}

// FindEnabled finds all enabled repositories.
func (r *RepositoryRepo) FindEnabled(ctx context.Context) ([]*entity.Repository, error) {
	var modelList []*models.Repository
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Find(&modelList).
		Error
	if err != nil {
		return nil, err
	}

	entities := make([]*entity.Repository, len(modelList))
	for i, m := range modelList {
		entities[i] = converter.RepositoryFromModel(m)
	}

	return entities, nil
}

// ExistsByName checks if a repository exists by name.
func (r *RepositoryRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Repository{}).
		Where("name = ?", name).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ListWithPagination lists repositories with pagination and optional filtering.
func (r *RepositoryRepo) ListWithPagination(ctx context.Context, params repository.PaginationParams, filter *repository.RepositoryFilter) ([]*entity.Repository, *repository.PaginationResult, error) {
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
	query := r.db.WithContext(ctx).Model(&models.Repository{})

	if filter != nil && filter.Enabled != nil {
		query = query.Where("enabled = ?", *filter.Enabled)
	}

	// Count total items
	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		return nil, nil, err
	}

	// Calculate pagination metadata
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize > 0 {
		totalPages++
	}

	// Query with pagination
	var modelList []*models.Repository
	offset := (page - 1) * pageSize

	err := query.
		Order("created_at desc").
		Offset(offset).
		Limit(pageSize).
		Find(&modelList).Error

	if err != nil {
		return nil, nil, err
	}

	// Convert models to domain entities
	entities := make([]*entity.Repository, len(modelList))
	for i, m := range modelList {
		entities[i] = converter.RepositoryFromModel(m)
	}

	paginationResult := &repository.PaginationResult{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: int(totalItems),
		TotalPages: totalPages,
	}

	return entities, paginationResult, nil
}

// Ensure RepositoryRepo implements repository.RepositoryRepository interface.
var _ repository.RepositoryRepository = (*RepositoryRepo)(nil)
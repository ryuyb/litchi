// Package repositories provides GORM-based implementations of repository interfaces.
package repositories

import (
	"context"

	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/converter"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// RepositoryRepo implements repository.RepositoryRepository using GORM.
type RepositoryRepo struct {
	db *gorm.DB
}

// NewRepositoryRepo creates a new RepositoryRepo.
func NewRepositoryRepo(db *gorm.DB) *RepositoryRepo {
	return &RepositoryRepo{db: db}
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
	model := converter.RepositoryToModel(repo)

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

// Ensure RepositoryRepo implements repository.RepositoryRepository interface.
var _ repository.RepositoryRepository = (*RepositoryRepo)(nil)
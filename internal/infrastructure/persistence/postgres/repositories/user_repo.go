// Package repositories provides GORM-based implementations of repository interfaces.
package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/converter"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/models"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// UserRepoParams holds dependencies for UserRepo.
type UserRepoParams struct {
	fx.In

	DB *gorm.DB `name:"gorm_db"`
}

// UserRepo implements repository.UserRepository using GORM.
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(p UserRepoParams) *UserRepo {
	return &UserRepo{db: p.DB}
}

// Create creates a new user in the database.
func (r *UserRepo) Create(ctx context.Context, user *entity.User) error {
	model := converter.UserToModel(user)
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing user in the database.
func (r *UserRepo) Update(ctx context.Context, user *entity.User) error {
	model := converter.UserToModel(user)
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete deletes a user by its ID.
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.User{}).
		Error
}

// FindByID finds a user by its ID.
// Returns nil if not found (no error).
func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var model models.User
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&model).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return converter.UserFromModel(&model), nil
}

// FindByUsername finds a user by username.
// Returns nil if not found (no error).
func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var model models.User
	err := r.db.WithContext(ctx).
		Where("username = ?", username).
		First(&model).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return converter.UserFromModel(&model), nil
}

// ExistsByUsername checks if a user exists by username.
func (r *UserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("username = ?", username).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// FindAll finds all users.
func (r *UserRepo) FindAll(ctx context.Context) ([]*entity.User, error) {
	var modelList []*models.User
	err := r.db.WithContext(ctx).
		Order("created_at desc").
		Find(&modelList).
		Error
	if err != nil {
		return nil, err
	}

	entities := make([]*entity.User, len(modelList))
	for i, m := range modelList {
		entities[i] = converter.UserFromModel(m)
	}

	return entities, nil
}

// ListWithPagination lists users with pagination.
func (r *UserRepo) ListWithPagination(ctx context.Context, params repository.PaginationParams) ([]*entity.User, *repository.PaginationResult, error) {
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

	// Count total items
	var totalItems int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&totalItems).Error; err != nil {
		return nil, nil, err
	}

	// Calculate pagination metadata
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize > 0 {
		totalPages++
	}

	// Query with pagination
	var modelList []*models.User
	offset := (page - 1) * pageSize

	err := r.db.WithContext(ctx).
		Order("created_at desc").
		Offset(offset).
		Limit(pageSize).
		Find(&modelList).
		Error

	if err != nil {
		return nil, nil, err
	}

	// Convert models to domain entities
	entities := make([]*entity.User, len(modelList))
	for i, m := range modelList {
		entities[i] = converter.UserFromModel(m)
	}

	paginationResult := &repository.PaginationResult{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: int(totalItems),
		TotalPages: totalPages,
	}

	return entities, paginationResult, nil
}

// Count returns the total number of users.
func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Count(&count).
		Error
	return count, err
}

// Ensure UserRepo implements repository.UserRepository interface.
var _ repository.UserRepository = (*UserRepo)(nil)
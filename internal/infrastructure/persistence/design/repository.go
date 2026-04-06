// Package design provides GORM-based repository implementation for Design entity.
// It handles persistence operations including version management.
package design

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/models"
	"github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "design_repository",
		Provides: []string{"repository.DesignRepository"},
		Invokes:  []string{},
		Depends:  []string{"*gorm.DB", "*zap.Logger"},
	})
}

// Module provides the DesignRepository module for Fx.
var Module = fx.Module("design_repository",
	fx.Provide(NewRepository),
)

// Repository implements DesignRepository using GORM.
type Repository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// Params holds dependencies for creating a Repository.
type Params struct {
	fx.In

	DB     *gorm.DB
	Logger *zap.Logger
}

// NewRepository creates a new GORM-based DesignRepository.
func NewRepository(p Params) repository.DesignRepository {
	return &Repository{
		db:     p.DB,
		logger: p.Logger.Named("design_repo"),
	}
}

// Save saves a design entity to the database.
func (r *Repository) Save(ctx context.Context, design *entity.Design, sessionID uuid.UUID) error {
	// Convert domain entity to persistence model
	dbDesign := toDBModel(design, sessionID)

	// Use transaction to save design and its versions
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if design already exists for this session
		var existing models.Design
		result := tx.Where("session_id = ?", sessionID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new design
			if err := tx.Create(&dbDesign).Error; err != nil {
				return errors.Wrap(errors.ErrDatabaseOperation, err).
					WithDetail("failed to create design").
					WithContext("session_id", sessionID.String())
			}
		} else if result.Error != nil {
			return errors.Wrap(errors.ErrDatabaseOperation, result.Error).
				WithDetail("failed to query existing design").
				WithContext("session_id", sessionID.String())
		} else {
			// Update existing design
			dbDesign.ID = existing.ID
			dbDesign.CreatedAt = existing.CreatedAt

			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"current_version":      dbDesign.CurrentVersion,
				"complexity_score":     dbDesign.ComplexityScore,
				"require_confirmation": dbDesign.RequireConfirmation,
				"confirmed":            dbDesign.Confirmed,
				"updated_at":           dbDesign.UpdatedAt,
			}).Error; err != nil {
				return errors.Wrap(errors.ErrDatabaseOperation, err).
					WithDetail("failed to update design").
					WithContext("session_id", sessionID.String())
			}

			// Save new versions (only save versions that don't exist)
			for _, version := range dbDesign.Versions {
				var existingVersion models.DesignVersion
				result := tx.Where("design_id = ? AND version = ?", dbDesign.ID, version.Version).First(&existingVersion)
				if result.Error == gorm.ErrRecordNotFound {
					version.DesignID = dbDesign.ID
					if err := tx.Create(&version).Error; err != nil {
						return errors.Wrap(errors.ErrDatabaseOperation, err).
							WithDetail("failed to create design version").
							WithContext("version", version.Version)
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		r.logger.Error("failed to save design",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		return err
	}

	r.logger.Debug("design saved successfully",
		zap.String("session_id", sessionID.String()),
		zap.Int("current_version", design.CurrentVersion),
	)

	return nil
}

// FindByID finds a design by its unique identifier.
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Design, error) {
	var dbDesign models.Design
	result := r.db.WithContext(ctx).
		Preload("Versions").
		Where("id = ?", id).
		First(&dbDesign)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		err := errors.Wrap(errors.ErrDatabaseOperation, result.Error).
			WithDetail("failed to find design by ID").
			WithContext("id", id.String())
		r.logger.Error("failed to find design", zap.String("id", id.String()), zap.Error(err))
		return nil, err
	}

	return toDomainEntity(&dbDesign), nil
}

// FindBySessionID finds the design associated with a work session.
func (r *Repository) FindBySessionID(ctx context.Context, sessionID uuid.UUID) (*entity.Design, error) {
	var dbDesign models.Design
	result := r.db.WithContext(ctx).
		Preload("Versions").
		Where("session_id = ?", sessionID).
		First(&dbDesign)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		err := errors.Wrap(errors.ErrDatabaseOperation, result.Error).
			WithDetail("failed to find design by session ID").
			WithContext("session_id", sessionID.String())
		r.logger.Error("failed to find design by session",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	return toDomainEntity(&dbDesign), nil
}

// GetLatestVersion returns the latest version number for a design.
func (r *Repository) GetLatestVersion(ctx context.Context, sessionID uuid.UUID) (int, error) {
	var dbDesign models.Design
	result := r.db.WithContext(ctx).
		Select("current_version").
		Where("session_id = ?", sessionID).
		First(&dbDesign)

	if result.Error == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if result.Error != nil {
		err := errors.Wrap(errors.ErrDatabaseOperation, result.Error).
			WithDetail("failed to get latest version").
			WithContext("session_id", sessionID.String())
		r.logger.Error("failed to get latest version",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		return 0, err
	}

	return dbDesign.CurrentVersion, nil
}

// FindVersions retrieves all versions of a design in chronological order.
func (r *Repository) FindVersions(ctx context.Context, sessionID uuid.UUID) ([]valueobject.DesignVersion, error) {
	var dbVersions []models.DesignVersion

	// First get the design ID
	var dbDesign models.Design
	result := r.db.WithContext(ctx).
		Select("id").
		Where("session_id = ?", sessionID).
		First(&dbDesign)

	if result.Error == gorm.ErrRecordNotFound {
		return []valueobject.DesignVersion{}, nil
	}
	if result.Error != nil {
		err := errors.Wrap(errors.ErrDatabaseOperation, result.Error).
			WithDetail("failed to find design for versions").
			WithContext("session_id", sessionID.String())
		return nil, err
	}

	// Query versions ordered by version number
	result = r.db.WithContext(ctx).
		Where("design_id = ?", dbDesign.ID).
		Order("version ASC").
		Find(&dbVersions)

	if result.Error != nil {
		err := errors.Wrap(errors.ErrDatabaseOperation, result.Error).
			WithDetail("failed to find design versions").
			WithContext("session_id", sessionID.String())
		r.logger.Error("failed to find design versions",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	versions := make([]valueobject.DesignVersion, len(dbVersions))
	for i, v := range dbVersions {
		versions[i] = toDomainDesignVersion(&v)
	}

	return versions, nil
}

// FindVersionByNumber retrieves a specific version of a design.
func (r *Repository) FindVersionByNumber(ctx context.Context, sessionID uuid.UUID, versionNum int) (*valueobject.DesignVersion, error) {
	// First get the design ID
	var dbDesign models.Design
	result := r.db.WithContext(ctx).
		Select("id").
		Where("session_id = ?", sessionID).
		First(&dbDesign)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrSessionNotFound).
			WithDetail("design not found for session").
			WithContext("session_id", sessionID.String())
	}
	if result.Error != nil {
		err := errors.Wrap(errors.ErrDatabaseOperation, result.Error).
			WithDetail("failed to find design").
			WithContext("session_id", sessionID.String())
		return nil, err
	}

	// Query specific version
	var dbVersion models.DesignVersion
	result = r.db.WithContext(ctx).
		Where("design_id = ? AND version = ?", dbDesign.ID, versionNum).
		First(&dbVersion)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, errors.New(errors.ErrValidationFailed).
			WithDetail(fmt.Sprintf("version %d not found", versionNum)).
			WithContext("session_id", sessionID.String())
	}
	if result.Error != nil {
		err := errors.Wrap(errors.ErrDatabaseOperation, result.Error).
			WithDetail("failed to find design version").
			WithContext("version", versionNum)
		return nil, err
	}

	return new(toDomainDesignVersion(&dbVersion)), nil
}

// Delete removes a design and all its versions from the database.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Design{}, "id = ?", id)
	if result.Error != nil {
		err := errors.Wrap(errors.ErrDatabaseOperation, result.Error).
			WithDetail("failed to delete design").
			WithContext("id", id.String())
		r.logger.Error("failed to delete design", zap.String("id", id.String()), zap.Error(err))
		return err
	}

	r.logger.Debug("design deleted successfully", zap.String("id", id.String()))
	return nil
}

// ============================================
// Model Conversion Functions
// ============================================

// toDBModel converts a domain Design entity to a persistence model.
func toDBModel(design *entity.Design, sessionID uuid.UUID) models.Design {
	dbDesign := models.Design{
		SessionID:           sessionID,
		CurrentVersion:      design.CurrentVersion,
		RequireConfirmation: design.RequireConfirmation,
		Confirmed:           design.Confirmed,
	}

	// Set complexity score
	if design.ComplexityScore.Value() > 0 {
		dbDesign.ComplexityScore = new(design.ComplexityScore.Value())
	}

	// Convert versions
	dbDesign.Versions = make([]models.DesignVersion, len(design.Versions))
	for i, v := range design.Versions {
		dbDesign.Versions[i] = toDBDesignVersion(&v)
	}

	return dbDesign
}

// toDBDesignVersion converts a domain DesignVersion to a persistence model.
func toDBDesignVersion(version *valueobject.DesignVersion) models.DesignVersion {
	return models.DesignVersion{
		Version:   version.Version,
		Content:   version.Content,
		Reason:    version.Reason,
		CreatedAt: version.CreatedAt,
	}
}

// toDomainEntity converts a persistence model to a domain Design entity.
func toDomainEntity(dbDesign *models.Design) *entity.Design {
	// Create design with basic fields
	design := &entity.Design{
		CurrentVersion:      dbDesign.CurrentVersion,
		RequireConfirmation: dbDesign.RequireConfirmation,
		Confirmed:           dbDesign.Confirmed,
		Versions:            make([]valueobject.DesignVersion, len(dbDesign.Versions)),
	}

	// Set complexity score
	if dbDesign.ComplexityScore != nil {
		score, err := valueobject.NewComplexityScore(*dbDesign.ComplexityScore)
		if err != nil {
			// Use default score if invalid
			score = valueobject.ComplexityScore{}
		}
		design.ComplexityScore = score
	}

	// Convert versions
	for i, v := range dbDesign.Versions {
		design.Versions[i] = toDomainDesignVersion(&v)
	}

	return design
}

// toDomainDesignVersion converts a persistence model to a domain DesignVersion.
func toDomainDesignVersion(dbVersion *models.DesignVersion) valueobject.DesignVersion {
	return valueobject.DesignVersion{
		Version:   dbVersion.Version,
		Content:   dbVersion.Content,
		Reason:    dbVersion.Reason,
		CreatedAt: dbVersion.CreatedAt,
	}
}

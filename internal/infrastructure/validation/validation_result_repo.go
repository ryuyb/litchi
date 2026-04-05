package validation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GormValidationResultRepository implements ValidationResultRepository using GORM.
type GormValidationResultRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewGormValidationResultRepository creates a new GORM-based validation result repository.
func NewGormValidationResultRepository(db *gorm.DB, logger *zap.Logger) *GormValidationResultRepository {
	return &GormValidationResultRepository{
		db:     db,
		logger: logger.Named("validation-result-repo"),
	}
}

// Save saves a validation result.
func (r *GormValidationResultRepository) Save(ctx context.Context, result *valueobject.ValidationResult, sessionID, taskID uuid.UUID) error {
	model := r.toModel(result, sessionID, taskID)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Error("failed to save validation result",
			zap.Error(err),
			zap.String("sessionID", sessionID.String()),
			zap.String("taskID", taskID.String()),
		)
		return err
	}

	r.logger.Debug("saved validation result",
		zap.String("id", model.ID.String()),
		zap.String("status", string(result.OverallStatus)),
	)
	return nil
}

// FindByTaskID finds a validation result by task ID.
func (r *GormValidationResultRepository) FindByTaskID(ctx context.Context, taskID uuid.UUID) (*valueobject.ValidationResult, error) {
	var model models.ExecutionValidationResult
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.fromModel(&model), nil
}

// FindBySessionID finds all validation results for a session.
func (r *GormValidationResultRepository) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*valueobject.ValidationResult, error) {
	var models []*models.ExecutionValidationResult
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	results := make([]*valueobject.ValidationResult, len(models))
	for i, m := range models {
		results[i] = r.fromModel(m)
	}
	return results, nil
}

// FindLatestBySessionID finds the most recent validation result for a session.
func (r *GormValidationResultRepository) FindLatestBySessionID(ctx context.Context, sessionID uuid.UUID) (*valueobject.ValidationResult, error) {
	var model models.ExecutionValidationResult
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.fromModel(&model), nil
}

// List lists validation results with pagination.
func (r *GormValidationResultRepository) List(ctx context.Context, params repository.PaginationParams, filter *repository.ValidationResultFilter) ([]*valueobject.ValidationResult, *repository.PaginationResult, error) {
	query := r.db.WithContext(ctx).Model(&models.ExecutionValidationResult{})

	// Apply filters
	if filter != nil {
		if filter.SessionID != nil {
			query = query.Where("session_id = ?", filter.SessionID)
		}
		if filter.TaskID != nil {
			query = query.Where("task_id = ?", filter.TaskID)
		}
		if filter.Status != nil {
			query = query.Where("overall_status = ?", string(*filter.Status))
		}
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, nil, err
	}

	// Apply pagination
	offset := (params.Page - 1) * params.PageSize
	var models []*models.ExecutionValidationResult
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(params.PageSize).
		Find(&models).Error; err != nil {
		return nil, nil, err
	}

	results := make([]*valueobject.ValidationResult, len(models))
	for i, m := range models {
		results[i] = r.fromModel(m)
	}

	pagination := &repository.PaginationResult{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: int(total),
		TotalPages: int((total + int64(params.PageSize) - 1) / int64(params.PageSize)),
	}

	return results, pagination, nil
}

// DeleteBySessionID deletes all validation results for a session.
func (r *GormValidationResultRepository) DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&models.ExecutionValidationResult{}).Error; err != nil {
		r.logger.Error("failed to delete validation results",
			zap.Error(err),
			zap.String("sessionID", sessionID.String()),
		)
		return err
	}
	return nil
}

// toModel converts domain model to GORM model.
func (r *GormValidationResultRepository) toModel(result *valueobject.ValidationResult, sessionID, taskID uuid.UUID) *models.ExecutionValidationResult {
	model := &models.ExecutionValidationResult{
		ID:        uuid.New(),
		SessionID: sessionID,
		TaskID:    taskID,
		CreatedAt: time.Now(),
	}

	// Format result
	if result.FormatResult != nil {
		model.FormatSuccess = &result.FormatResult.Success
		model.FormatOutput = result.FormatResult.Output
		model.FormatDurationMs = &result.FormatResult.Duration
		model.FormatToolName = result.FormatResult.ToolName
	}

	// Lint result
	if result.LintResult != nil {
		model.LintSuccess = &result.LintResult.Success
		model.LintOutput = result.LintResult.Output
		model.LintIssuesFound = &result.LintResult.IssuesFound
		model.LintIssuesFixed = &result.LintResult.IssuesFixed
		model.LintDurationMs = &result.LintResult.Duration
		model.LintToolName = result.LintResult.ToolName
	}

	// Test result
	if result.TestResult != nil {
		model.TestSuccess = &result.TestResult.Success
		model.TestOutput = result.TestResult.Output
		model.TestPassed = &result.TestResult.Passed
		model.TestFailed = &result.TestResult.Failed
		model.TestDurationMs = &result.TestResult.Duration
		model.TestToolName = result.TestResult.ToolName
		// Serialize test failures
		if len(result.TestResult.Failures) > 0 {
			failuresJSON, err := json.Marshal(result.TestResult.Failures)
			if err != nil {
				r.logger.Warn("failed to marshal test failures", zap.Error(err))
				failuresJSON = []byte("[]")
			}
			model.TestFailures = failuresJSON
		} else {
			model.TestFailures = []byte("[]")
		}
	}

	// Overall
	model.OverallStatus = string(result.OverallStatus)
	model.TotalDurationMs = &result.Duration

	// Warnings
	if len(result.Warnings) > 0 {
		warningsJSON, err := json.Marshal(result.Warnings)
		if err != nil {
			r.logger.Warn("failed to marshal warnings", zap.Error(err))
			warningsJSON = []byte("[]")
		}
		model.Warnings = warningsJSON
	} else {
		model.Warnings = []byte("[]")
	}

	return model
}

// fromModel converts GORM model to domain model.
func (r *GormValidationResultRepository) fromModel(model *models.ExecutionValidationResult) *valueobject.ValidationResult {
	result := valueobject.NewValidationResult()

	// Format result - check all required pointer fields
	if model.FormatSuccess != nil && model.FormatDurationMs != nil {
		result.FormatResult = &valueobject.FormatResult{
			Success:  *model.FormatSuccess,
			Output:   model.FormatOutput,
			Duration: *model.FormatDurationMs,
			ToolName: model.FormatToolName,
		}
	}

	// Lint result - check all required pointer fields
	if model.LintSuccess != nil && model.LintIssuesFound != nil && model.LintIssuesFixed != nil && model.LintDurationMs != nil {
		result.LintResult = &valueobject.LintResult{
			Success:     *model.LintSuccess,
			Output:      model.LintOutput,
			IssuesFound: *model.LintIssuesFound,
			IssuesFixed: *model.LintIssuesFixed,
			Duration:    *model.LintDurationMs,
			ToolName:    model.LintToolName,
		}
	}

	// Test result - check all required pointer fields
	if model.TestSuccess != nil && model.TestPassed != nil && model.TestFailed != nil && model.TestDurationMs != nil {
		result.TestResult = &valueobject.ValidationTestResult{
			Success:  *model.TestSuccess,
			Output:   model.TestOutput,
			Passed:   *model.TestPassed,
			Failed:   *model.TestFailed,
			Duration: *model.TestDurationMs,
			ToolName: model.TestToolName,
		}
		// Deserialize test failures
		if len(model.TestFailures) > 0 {
			var failures []valueobject.TestFailure
			if err := json.Unmarshal(model.TestFailures, &failures); err == nil {
				result.TestResult.Failures = failures
			}
		}
	}

	// Overall
	result.OverallStatus = valueobject.ValidationStatus(model.OverallStatus)
	if model.TotalDurationMs != nil {
		result.Duration = *model.TotalDurationMs
	}

	// Warnings
	if len(model.Warnings) > 0 {
		var warnings []string
		if err := json.Unmarshal(model.Warnings, &warnings); err == nil {
			result.Warnings = warnings
		}
	}

	return result
}
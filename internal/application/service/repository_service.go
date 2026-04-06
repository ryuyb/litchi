// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// RepositoryService handles repository configuration management.
// It provides CRUD operations for repository entities and enables/disables
// repositories for processing.
//
// Core responsibilities:
// 1. Repository CRUD - create, update, delete, query repositories
// 2. Enable/disable - manage repository processing status
// 3. Config merging - merge repository config with global config
// 4. Config validation - validate repository configuration
type RepositoryService struct {
	repoRepo        repository.RepositoryRepository
	auditRepo       repository.AuditLogRepository
	eventDispatcher *event.Dispatcher
	config          *config.Config
	logger          *zap.Logger
}

// NewRepositoryService creates a new RepositoryService.
func NewRepositoryService(
	repoRepo repository.RepositoryRepository,
	auditRepo repository.AuditLogRepository,
	eventDispatcher *event.Dispatcher,
	config *config.Config,
	logger *zap.Logger,
) *RepositoryService {
	return &RepositoryService{
		repoRepo:        repoRepo,
		auditRepo:       auditRepo,
		eventDispatcher: eventDispatcher,
		config:          config,
		logger:          logger.Named("repository_service"),
	}
}

// CreateRepository creates a new repository configuration.
// This enables the repository for processing by default.
//
// Steps:
// 1. Validate repository name format (owner/repo)
// 2. Check if repository already exists
// 3. Create repository entity with optional config overrides
// 4. Save to database
//
// Returns the created repository entity.
func (s *RepositoryService) CreateRepository(
	ctx context.Context,
	name string,
	configOverrides *entity.RepoConfig,
	actor string,
	actorRole valueobject.ActorRole,
) (*entity.Repository, error) {
	startTime := time.Now()

	// 1. Create repository entity
	repo := entity.NewRepository(name)

	// 2. Apply config overrides if provided
	if configOverrides != nil {
		repo.SetConfig(*configOverrides)
	}

	// 3. Validate repository
	if err := repo.Validate(); err != nil {
		s.recordAuditLog(ctx, repo, actor, actorRole,
			valueobject.OpRepositoryCreate, startTime, false, err.Error())
		return nil, err
	}

	// 4. Check if repository already exists
	exists, err := s.repoRepo.ExistsByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to check repository existence",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if exists {
		s.recordAuditLog(ctx, repo, actor, actorRole,
			valueobject.OpRepositoryCreate, startTime, false, "repository already exists")
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("repository %s already exists", name),
		)
	}

	// 5. Save repository
	if err := s.repoRepo.Save(ctx, repo); err != nil {
		s.logger.Error("failed to save repository",
			zap.String("name", name),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, repo, actor, actorRole,
			valueobject.OpRepositoryCreate, startTime, false, err.Error())
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 6. Record audit log
	s.recordAuditLog(ctx, repo, actor, actorRole,
		valueobject.OpRepositoryCreate, startTime, true, fmt.Sprintf("repository %s created", name))

	s.logger.Info("repository created",
		zap.String("repo_id", repo.ID.String()),
		zap.String("name", name),
		zap.Bool("enabled", repo.Enabled),
	)

	return repo, nil
}

// UpdateRepository updates an existing repository configuration.
//
// Steps:
// 1. Validate repository exists
// 2. Apply new configuration
// 3. Validate configuration
// 4. Save to database
//
// Returns the updated repository entity.
func (s *RepositoryService) UpdateRepository(
	ctx context.Context,
	name string,
	configOverrides *entity.RepoConfig,
	actor string,
	actorRole valueobject.ActorRole,
) (*entity.Repository, error) {
	startTime := time.Now()

	// 1. Find repository
	repo, err := s.repoRepo.FindByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to find repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if repo == nil {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("repository %s not found", name),
		)
	}

	// 2. Apply config overrides if provided
	if configOverrides != nil {
		repo.SetConfig(*configOverrides)
	}

	// 3. Validate repository
	if err := repo.Validate(); err != nil {
		s.recordAuditLog(ctx, repo, actor, actorRole,
			valueobject.OpRepositoryUpdate, startTime, false, err.Error())
		return nil, err
	}

	// 4. Validate config values
	if err := s.ValidateConfig(ctx, repo.Config); err != nil {
		s.recordAuditLog(ctx, repo, actor, actorRole,
			valueobject.OpRepositoryUpdate, startTime, false, err.Error())
		return nil, err
	}

	// 5. Save repository
	if err := s.repoRepo.Save(ctx, repo); err != nil {
		s.logger.Error("failed to update repository",
			zap.String("name", name),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, repo, actor, actorRole,
			valueobject.OpRepositoryUpdate, startTime, false, err.Error())
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 6. Record audit log
	s.recordAuditLog(ctx, repo, actor, actorRole,
		valueobject.OpRepositoryUpdate, startTime, true, fmt.Sprintf("repository %s updated", name))

	s.logger.Info("repository updated",
		zap.String("repo_id", repo.ID.String()),
		zap.String("name", name),
	)

	return repo, nil
}

// DeleteRepository deletes a repository configuration.
// This disables the repository from processing.
//
// Steps:
// 1. Validate repository exists
// 2. Delete from database
//
// Returns error if deletion fails.
func (s *RepositoryService) DeleteRepository(
	ctx context.Context,
	name string,
	actor string,
	actorRole valueobject.ActorRole,
) error {
	startTime := time.Now()

	// 1. Find repository to get ID for audit log
	repo, err := s.repoRepo.FindByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to find repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if repo == nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("repository %s not found", name),
		)
	}

	// 2. Delete repository
	if err := s.repoRepo.Delete(ctx, name); err != nil {
		s.logger.Error("failed to delete repository",
			zap.String("name", name),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, repo, actor, actorRole,
			valueobject.OpRepositoryDelete, startTime, false, err.Error())
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 3. Record audit log
	s.recordAuditLog(ctx, repo, actor, actorRole,
		valueobject.OpRepositoryDelete, startTime, true, fmt.Sprintf("repository %s deleted", name))

	s.logger.Info("repository deleted",
		zap.String("repo_id", repo.ID.String()),
		zap.String("name", name),
	)

	return nil
}

// GetRepository retrieves a repository configuration by name.
//
// Returns nil if repository not found.
func (s *RepositoryService) GetRepository(
	ctx context.Context,
	name string,
) (*entity.Repository, error) {
	repo, err := s.repoRepo.FindByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to find repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return repo, nil
}

// ListRepositories lists all repository configurations.
//
// Returns empty slice if no repositories found.
func (s *RepositoryService) ListRepositories(
	ctx context.Context,
) ([]*entity.Repository, error) {
	repos, err := s.repoRepo.FindAll(ctx)
	if err != nil {
		s.logger.Error("failed to list repositories",
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return repos, nil
}

// ListRepositoriesWithPagination lists repositories with pagination and optional filtering.
//
// Returns empty slice if no repositories found.
func (s *RepositoryService) ListRepositoriesWithPagination(
	ctx context.Context,
	params repository.PaginationParams,
	filter *repository.RepositoryFilter,
) ([]*entity.Repository, *repository.PaginationResult, error) {
	repos, pagination, err := s.repoRepo.ListWithPagination(ctx, params, filter)
	if err != nil {
		s.logger.Error("failed to list repositories with pagination",
			zap.Error(err),
		)
		return nil, nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return repos, pagination, nil
}

// ListEnabledRepositories lists all enabled repositories.
//
// Returns empty slice if no enabled repositories found.
func (s *RepositoryService) ListEnabledRepositories(
	ctx context.Context,
) ([]*entity.Repository, error) {
	repos, err := s.repoRepo.FindEnabled(ctx)
	if err != nil {
		s.logger.Error("failed to list enabled repositories",
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return repos, nil
}

// EnableRepository enables a repository for processing.
//
// Steps:
// 1. Validate repository exists
// 2. Enable repository
// 3. Save to database
//
// Returns error if operation fails.
func (s *RepositoryService) EnableRepository(
	ctx context.Context,
	name string,
	actor string,
	actorRole valueobject.ActorRole,
) error {
	startTime := time.Now()

	// 1. Find repository
	repo, err := s.repoRepo.FindByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to find repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if repo == nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("repository %s not found", name),
		)
	}

	// 2. Enable repository
	repo.Enable()

	// 3. Save repository
	if err := s.repoRepo.Save(ctx, repo); err != nil {
		s.logger.Error("failed to enable repository",
			zap.String("name", name),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, repo, actor, actorRole,
			valueobject.OpRepositoryEnable, startTime, false, err.Error())
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 4. Record audit log
	s.recordAuditLog(ctx, repo, actor, actorRole,
		valueobject.OpRepositoryEnable, startTime, true, fmt.Sprintf("repository %s enabled", name))

	s.logger.Info("repository enabled",
		zap.String("repo_id", repo.ID.String()),
		zap.String("name", name),
	)

	return nil
}

// DisableRepository disables a repository from processing.
//
// Steps:
// 1. Validate repository exists
// 2. Disable repository
// 3. Save to database
//
// Returns error if operation fails.
func (s *RepositoryService) DisableRepository(
	ctx context.Context,
	name string,
	actor string,
	actorRole valueobject.ActorRole,
) error {
	startTime := time.Now()

	// 1. Find repository
	repo, err := s.repoRepo.FindByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to find repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if repo == nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("repository %s not found", name),
		)
	}

	// 2. Disable repository
	repo.Disable()

	// 3. Save repository
	if err := s.repoRepo.Save(ctx, repo); err != nil {
		s.logger.Error("failed to disable repository",
			zap.String("name", name),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, repo, actor, actorRole,
			valueobject.OpRepositoryDisable, startTime, false, err.Error())
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 4. Record audit log
	s.recordAuditLog(ctx, repo, actor, actorRole,
		valueobject.OpRepositoryDisable, startTime, true, fmt.Sprintf("repository %s disabled", name))

	s.logger.Info("repository disabled",
		zap.String("repo_id", repo.ID.String()),
		zap.String("name", name),
	)

	return nil
}

// GetEffectiveConfig returns the effective configuration for a repository.
// Repository config overrides take precedence over global config.
//
// Steps:
// 1. Get repository config
// 2. Build global config from application config
// 3. Merge configs (repository config takes precedence)
//
// Returns the merged effective configuration.
func (s *RepositoryService) GetEffectiveConfig(
	ctx context.Context,
	name string,
) (*EffectiveConfig, error) {
	// 1. Get repository
	repo, err := s.repoRepo.FindByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to find repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 2. Build global config from application config
	globalConfig := s.buildGlobalRepoConfig()

	// 3. If repository not found, return global config only
	if repo == nil {
		return &EffectiveConfig{
			RepositoryName: name,
			GlobalConfig:   globalConfig,
			RepoConfig:     entity.RepoConfig{},
			Effective:      globalConfig,
			HasRepoConfig:  false,
		}, nil
	}

	// 4. Merge configs (repository config takes precedence)
	effective := repo.GetEffectiveConfig(globalConfig)

	return &EffectiveConfig{
		RepositoryName: name,
		GlobalConfig:   globalConfig,
		RepoConfig:     repo.Config,
		Effective:      effective,
		HasRepoConfig:  true,
		RepositoryID:   repo.ID,
		Enabled:        repo.Enabled,
	}, nil
}

// ValidateConfig validates repository configuration values.
//
// Validates:
// - MaxConcurrency: must be >= 1 if set
// - ComplexityThreshold: must be >= 0 and <= 100 if set
// - TaskRetryLimit: must be >= 0 if set
// - DefaultModel: must be non-empty if set
func (s *RepositoryService) ValidateConfig(
	ctx context.Context,
	config entity.RepoConfig,
) error {
	// Validate MaxConcurrency
	if config.MaxConcurrency != nil {
		if *config.MaxConcurrency < 1 {
			return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
				"maxConcurrency must be at least 1",
			)
		}
	}

	// Validate ComplexityThreshold
	if config.ComplexityThreshold != nil {
		if *config.ComplexityThreshold < 0 || *config.ComplexityThreshold > 100 {
			return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
				"complexityThreshold must be between 0 and 100",
			)
		}
	}

	// Validate TaskRetryLimit
	if config.TaskRetryLimit != nil {
		if *config.TaskRetryLimit < 0 {
			return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
				"taskRetryLimit must be at least 0",
			)
		}
	}

	// Validate DefaultModel (must be non-empty if set)
	if config.DefaultModel != nil {
		if *config.DefaultModel == "" {
			return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
				"defaultModel cannot be empty if set",
			)
		}
	}

	return nil
}

// EffectiveConfig represents the effective configuration for a repository.
// It includes the global config, repository-specific config, and the merged result.
type EffectiveConfig struct {
	RepositoryName string            `json:"repositoryName"`
	RepositoryID   uuid.UUID         `json:"repositoryId,omitempty"`
	Enabled        bool              `json:"enabled"`
	GlobalConfig   entity.RepoConfig `json:"globalConfig"`
	RepoConfig     entity.RepoConfig `json:"repoConfig"`
	Effective      entity.RepoConfig `json:"effective"`
	HasRepoConfig  bool              `json:"hasRepoConfig"`
}

// --- Internal Helper Methods ---

// buildGlobalRepoConfig builds RepoConfig from application global config.
func (s *RepositoryService) buildGlobalRepoConfig() entity.RepoConfig {
	global := entity.RepoConfig{}

	// Set values from application config
	global.MaxConcurrency = &s.config.Agent.MaxConcurrency
	global.ComplexityThreshold = &s.config.Complexity.Threshold
	global.ForceDesignConfirm = &s.config.Complexity.ForceDesignConfirm
	global.DefaultModel = &s.config.Agent.Type
	global.TaskRetryLimit = &s.config.Agent.TaskRetryLimit

	return global
}

// recordAuditLog records an audit log entry.
func (s *RepositoryService) recordAuditLog(
	ctx context.Context,
	repo *entity.Repository,
	actor string,
	actorRole valueobject.ActorRole,
	operation valueobject.OperationType,
	startTime time.Time,
	success bool,
	errMsg string,
) {
	if repo == nil {
		return
	}

	auditLog := entity.NewAuditLog(
		repo.ID,
		repo.Name,
		0, // No issue number for repository operations
		actor,
		actorRole,
		operation,
		"repository",
		repo.ID.String(),
	)

	auditLog.SetDuration(int(time.Since(startTime).Milliseconds()))

	if success {
		auditLog.MarkSuccess()
	} else if errMsg != "" {
		auditLog.MarkFailed(errMsg)
	}

	if err := s.auditRepo.Save(ctx, auditLog); err != nil {
		s.logger.Warn("failed to save audit log",
			zap.String("repo_id", repo.ID.String()),
			zap.Error(err),
		)
	}
}
// ============================================
// Validation Configuration Methods
// ============================================

// GetValidationConfig retrieves the validation configuration for a repository.
// Returns an empty config if repository not found or no validation config set.
func (s *RepositoryService) GetValidationConfig(
	ctx context.Context,
	name string,
) (*valueobject.ExecutionValidationConfig, error) {
	repo, err := s.repoRepo.FindByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to find repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	if repo == nil {
		return nil, litchierrors.New(litchierrors.ErrRepositoryNotFound).
			WithDetail("repository not found: " + name)
	}

	if repo.ValidationConfig == nil {
		// Return default config
		return &valueobject.ExecutionValidationConfig{
			Enabled: false,
		}, nil
	}

	return repo.ValidationConfig, nil
}

// UpdateValidationConfig updates the validation configuration for a repository.
func (s *RepositoryService) UpdateValidationConfig(
	ctx context.Context,
	name string,
	config *valueobject.ExecutionValidationConfig,
	actor string,
	actorRole valueobject.ActorRole,
) (*valueobject.ExecutionValidationConfig, error) {
	startTime := time.Now()

	// 1. Find repository
	repo, err := s.repoRepo.FindByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to find repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if repo == nil {
		return nil, litchierrors.New(litchierrors.ErrRepositoryNotFound).
			WithDetail("repository not found: " + name)
	}

	// 2. Update validation config
	repo.SetValidationConfig(config)

	// 3. Save repository
	if err := s.repoRepo.Save(ctx, repo); err != nil {
		s.logger.Error("failed to update validation config",
			zap.String("name", name),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, repo, actor, actorRole,
			valueobject.OpValidationConfigUpdate, startTime, false, err.Error())
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 4. Record audit log
	s.recordAuditLog(ctx, repo, actor, actorRole,
		valueobject.OpValidationConfigUpdate, startTime, true, "validation config updated")

	s.logger.Info("validation config updated",
		zap.String("repo_id", repo.ID.String()),
		zap.String("name", name),
	)

	return config, nil
}

// GetDetectionResult retrieves the project detection result for a repository.
// Returns nil if no detection has been run.
func (s *RepositoryService) GetDetectionResult(
	ctx context.Context,
	name string,
) (*valueobject.DetectedProject, error) {
	repo, err := s.repoRepo.FindByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to find repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	if repo == nil {
		return nil, litchierrors.New(litchierrors.ErrRepositoryNotFound).
			WithDetail("repository not found: " + name)
	}

	return repo.DetectedProject, nil
}

// RunDetection triggers project detection for a repository.
// This performs automatic detection of project type, language, and tools.
// Note: This is a placeholder implementation that returns mock data.
// The actual detection logic requires cloning the repository and analyzing files.
func (s *RepositoryService) RunDetection(
	ctx context.Context,
	name string,
) (*valueobject.DetectedProject, error) {
	startTime := time.Now()

	// 1. Find repository
	repo, err := s.repoRepo.FindByName(ctx, name)
	if err != nil {
		s.logger.Error("failed to find repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if repo == nil {
		return nil, litchierrors.New(litchierrors.ErrRepositoryNotFound).
			WithDetail("repository not found: " + name)
	}

	// 2. Run detection (placeholder - returns mock data for now)
	// TODO: Implement actual project detection using ProjectDetector service
	detectedProject := s.performMockDetection(name)

	// 3. Save detection result to repository
	repo.SetDetectedProject(detectedProject)

	// 4. Save repository
	if err := s.repoRepo.Save(ctx, repo); err != nil {
		s.logger.Error("failed to save detection result",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.logger.Info("project detection completed",
		zap.String("repo_id", repo.ID.String()),
		zap.String("name", name),
		zap.String("type", string(detectedProject.Type)),
		zap.Int("confidence", detectedProject.Confidence),
		zap.Duration("duration", time.Since(startTime)),
	)

	return detectedProject, nil
}

// performMockDetection returns a mock detection result.
// TODO: This is a PLACEHOLDER implementation for development/testing purposes only.
// It uses simple heuristics based on repository name patterns and should NOT be used
// in production. The actual ProjectDetector service should be implemented to:
// 1. Clone the repository to a temporary location
// 2. Analyze configuration files (go.mod, package.json, pyproject.toml, etc.)
// 3. Detect language, framework, and tool configurations
// 4. Return accurate detection results with high confidence
func (s *RepositoryService) performMockDetection(name string) *valueobject.DetectedProject {
	// Detect project type based on repository name pattern
	projectType := valueobject.ProjectTypeUnknown
	primaryLanguage := "Unknown"
	confidence := 50

	// Simple heuristics based on common patterns
	if len(name) > 0 {
		switch {
		case containsAny(name, []string{"go", "golang"}):
			projectType = valueobject.ProjectTypeGo
			primaryLanguage = "Go"
			confidence = 85
		case containsAny(name, []string{"node", "js", "ts", "react", "vue", "next"}):
			projectType = valueobject.ProjectTypeNodeJS
			primaryLanguage = "TypeScript"
			confidence = 80
		case containsAny(name, []string{"py", "python", "django", "flask"}):
			projectType = valueobject.ProjectTypePython
			primaryLanguage = "Python"
			confidence = 80
		case containsAny(name, []string{"rs", "rust", "cargo"}):
			projectType = valueobject.ProjectTypeRust
			primaryLanguage = "Rust"
			confidence = 85
		case containsAny(name, []string{"java", "spring", "kotlin"}):
			projectType = valueobject.ProjectTypeJava
			primaryLanguage = "Java"
			confidence = 80
		}
	}

	project := valueobject.NewDetectedProject(projectType, primaryLanguage, confidence)

	// Add detected tools based on project type
	switch projectType {
	case valueobject.ProjectTypeGo:
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeFormatter,
			"gofmt",
			"Go standard formatter",
			valueobject.NewToolCommand("gofmt", "gofmt", []string{"-s", "-w"}, 30),
		))
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeLinter,
			"golangci-lint",
			"Common Go linter",
			valueobject.NewToolCommand("golangci-lint", "golangci-lint", []string{"run"}, 60),
		).WithConfigFile(".golangci.yml"))
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeTester,
			"go test",
			"Go test",
			valueobject.NewToolCommand("go test", "go", []string{"test", "./..."}, 120),
		))
	case valueobject.ProjectTypeNodeJS:
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeFormatter,
			"prettier",
			"Common JS/TS formatter",
			valueobject.NewToolCommand("prettier", "npx", []string{"prettier", "--write", "."}, 60),
		).WithConfigFile(".prettierrc"))
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeLinter,
			"eslint",
			"Common JS/TS linter",
			valueobject.NewToolCommand("eslint", "npx", []string{"eslint", "."}, 60),
		).WithConfigFile(".eslintrc"))
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeTester,
			"jest",
			"Common JS/TS test runner",
			valueobject.NewToolCommand("jest", "npx", []string{"jest"}, 120),
		))
	case valueobject.ProjectTypePython:
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeFormatter,
			"black",
			"Common Python formatter",
			valueobject.NewToolCommand("black", "black", []string{"."}, 60),
		))
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeLinter,
			"ruff",
			"Modern Python linter",
			valueobject.NewToolCommand("ruff", "ruff", []string{"check", "."}, 60),
		))
		project.AddTool(valueobject.NewDetectedTool(
			valueobject.ToolTypeTester,
			"pytest",
			"Common Python test runner",
			valueobject.NewToolCommand("pytest", "pytest", []string{}, 120),
		))
	}

	return project
}

// containsAny checks if s contains any of the substrings (case-insensitive).
func containsAny(s string, substrings []string) bool {
	sLower := strings.ToLower(s)
	for _, sub := range substrings {
		if strings.Contains(sLower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

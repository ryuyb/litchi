// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"fmt"
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
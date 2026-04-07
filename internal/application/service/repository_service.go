// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/repository"
	domainservice "github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/git"
	"github.com/ryuyb/litchi/internal/infrastructure/github"
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
// 5. Project detection - detect project type and tools via file analysis
type RepositoryService struct {
	repoRepo        repository.RepositoryRepository
	auditRepo       repository.AuditLogRepository
	eventDispatcher *event.Dispatcher
	config          *config.Config
	detector        domainservice.CompositeProjectDetector
	gitClient       git.GitClient
	githubClient    *github.ClientManager
	logger          *zap.Logger
}

// NewRepositoryService creates a new RepositoryService.
func NewRepositoryService(
	repoRepo repository.RepositoryRepository,
	auditRepo repository.AuditLogRepository,
	eventDispatcher *event.Dispatcher,
	config *config.Config,
	detector domainservice.CompositeProjectDetector,
	gitClient git.GitClient,
	githubClient *github.ClientManager,
	logger *zap.Logger,
) *RepositoryService {
	return &RepositoryService{
		repoRepo:        repoRepo,
		auditRepo:       auditRepo,
		eventDispatcher: eventDispatcher,
		config:          config,
		detector:        detector,
		gitClient:       gitClient,
		githubClient:    githubClient,
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
// This performs automatic detection of project type, language, and tools
// by cloning the repository to a temporary location and analyzing configuration files.
func (s *RepositoryService) RunDetection(
	ctx context.Context,
	name string,
) (*valueobject.DetectedProject, error) {
	startTime := time.Now()

	// 1. Validate required dependencies
	if s.gitClient == nil {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("git client not configured for detection")
	}
	if s.detector == nil {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("project detector not configured for detection")
	}

	// 2. Find repository
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

	// 3. Get authentication token for clone
	authToken, err := s.getCloneAuthToken(ctx, name)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrGitCloneFailed, err).
			WithDetail("failed to get authentication token for clone")
	}

	// 4. Create temporary directory for clone
	tempPath, err := os.MkdirTemp("", "litchi-detection-*")
	if err != nil {
		s.logger.Error("failed to create temp directory",
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrGitCloneFailed, err).
			WithDetail("failed to create temporary directory for detection")
	}

	// Track cleanup state - cleanup on all exit paths
	cleanupDone := false
	defer func() {
		if !cleanupDone {
			s.cleanupTempDir(tempPath)
		}
	}()

	// 5. Build clone URL with authentication
	cloneURL := s.buildCloneURL(name, authToken)

	// 6. Clone repository with shallow clone (depth=1)
	cloneOpts := git.CloneOptions{
		Depth:        1,
		SingleBranch: true,
	}
	if err := s.gitClient.CloneRepository(ctx, cloneURL, tempPath, cloneOpts); err != nil {
		s.logger.Error("failed to clone repository",
			zap.String("name", name),
			zap.String("tempPath", tempPath),
			zap.Error(err),
		)
		return nil, err // GitClient already wraps error appropriately
	}

	s.logger.Debug("repository cloned for detection",
		zap.String("name", name),
		zap.String("tempPath", tempPath),
	)

	// 7. Run project detection
	detectedProject, err := s.detector.DetectWithAll(ctx, tempPath)
	if err != nil {
		s.logger.Warn("detection failed",
			zap.String("name", name),
			zap.Error(err),
		)
		// Detection failure is not critical - return unknown project
		detectedProject = valueobject.NewDetectedProject(
			valueobject.ProjectTypeUnknown,
			"Unknown",
			30,
		)
	}

	// 8. Cleanup temp directory (done before save to free resources early)
	s.cleanupTempDir(tempPath)
	cleanupDone = true

	// 9. Save detection result to repository entity
	repo.SetDetectedProject(detectedProject)

	// 10. Save repository
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
		zap.String("language", detectedProject.PrimaryLanguage),
		zap.Int("confidence", detectedProject.Confidence),
		zap.Int("tools", len(detectedProject.DetectedTools)),
		zap.Duration("duration", time.Since(startTime)),
	)

	return detectedProject, nil
}

// getCloneAuthToken returns an authentication token for cloning the repository.
// For PAT mode: returns the PAT token directly.
// For GitHub App mode: returns the installation token for the repository.
func (s *RepositoryService) getCloneAuthToken(ctx context.Context, repoName string) (string, error) {
	// Check if detector/gitClient are available (may be nil in tests)
	if s.githubClient == nil {
		// Fallback: try to use PAT from config directly
		if s.config.GitHub.Token != "" && !config.IsEnvPlaceholder(s.config.GitHub.Token) {
			return s.config.GitHub.Token, nil
		}
		return "", litchierrors.New(litchierrors.ErrGitHubAuthFailed).
			WithDetail("no GitHub client available and no PAT configured")
	}

	// Get client for this repository (handles both PAT and App modes)
	client, err := s.githubClient.GetClient(ctx, repoName)
	if err != nil {
		return "", err
	}

	// For PAT mode, the token is stored in the PATAuthStrategy
	if s.githubClient.GetAuthType() == github.AuthTypePAT {
		// Get token from config (PAT is stored there)
		return s.config.GitHub.Token, nil
	}

	// For GitHub App mode, we need to get the installation token
	// The client is already authenticated with the installation token
	// We need to extract it from the token cache
	if client == nil {
		return "", litchierrors.New(litchierrors.ErrGitHubAuthFailed).
			WithDetail("failed to get GitHub client for repository")
	}

	// Use the installation token from the cache
	// For App mode, ClientManager caches tokens - we can fetch a fresh one
	installationID, err := s.findInstallationID(ctx, repoName)
	if err != nil {
		return "", err
	}

	// Create fresh installation token for clone
	appStrategy, ok := s.githubClient.GetAuthStrategy().(*github.GitHubAppAuthStrategy)
	if !ok {
		return "", litchierrors.New(litchierrors.ErrGitHubAuthFailed).
			WithDetail("expected GitHubAppAuthStrategy for App mode")
	}

	// Check cache first
	cachedToken := appStrategy.GetTokenCache().Get(installationID)
	if cachedToken != nil {
		return cachedToken.Token, nil
	}

	// Fetch new token
	return s.fetchInstallationToken(ctx, installationID, appStrategy)
}

// findInstallationID finds the installation ID for a repository.
func (s *RepositoryService) findInstallationID(ctx context.Context, repoName string) (int64, error) {
	// Check if repository already has installation ID stored
	repo, err := s.repoRepo.FindByName(ctx, repoName)
	if err == nil && repo != nil && repo.HasInstallation() {
		return repo.InstallationID, nil
	}

	// Find via GitHub API (ClientManager handles this)
	return 0, litchierrors.New(litchierrors.ErrGitHubAuthFailed).
		WithDetail("installation ID not found for repository: " + repoName)
}

// fetchInstallationToken fetches a fresh installation token.
func (s *RepositoryService) fetchInstallationToken(
	ctx context.Context,
	installationID int64,
	appStrategy *github.GitHubAppAuthStrategy,
) (string, error) {
	// Use ClientManager's method to fetch token
	if s.githubClient == nil {
		return "", litchierrors.New(litchierrors.ErrGitHubAuthFailed).
			WithDetail("GitHub client manager not available")
	}
	return s.githubClient.FetchInstallationToken(ctx, installationID, appStrategy)
}

// buildCloneURL builds a GitHub clone URL with authentication token embedded.
// Format: https://<token>@github.com/<owner>/<repo>.git
func (s *RepositoryService) buildCloneURL(repoName, token string) string {
	return fmt.Sprintf("https://%s@github.com/%s.git", token, repoName)
}

// cleanupTempDir removes the temporary directory and logs warning on failure.
func (s *RepositoryService) cleanupTempDir(tempPath string) {
	if err := os.RemoveAll(tempPath); err != nil {
		s.logger.Warn("failed to cleanup temp directory",
			zap.String("path", tempPath),
			zap.Error(err),
		)
	} else {
		s.logger.Debug("temp directory cleaned up",
			zap.String("path", tempPath),
		)
	}
}

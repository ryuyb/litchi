package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// --- Helper functions for tests ---

func newTestRepositoryService(
	repoRepo repository.RepositoryRepository,
	auditRepo repository.AuditLogRepository,
) *RepositoryService {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			MaxConcurrency: 5,
			TaskRetryLimit: 3,
			Type:           "claude-opus-4-6",
		},
		Complexity: config.ComplexityConfig{
			Threshold:          70,
			ForceDesignConfirm: false,
		},
	}

	dispatcher := event.NewDispatcher()

	// Pass nil for detector, gitClient, githubClient - these are only needed for RunDetection tests
	return NewRepositoryService(
		repoRepo,
		auditRepo,
		dispatcher,
		cfg,
		nil, // detector - not needed for most tests
		nil, // gitClient - not needed for most tests
		nil, // githubClient - not needed for most tests
		zap.NewNop(),
	)
}

// --- Tests for CreateRepository ---

func TestRepositoryService_CreateRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repoRepo.EXPECT().ExistsByName(ctx, "owner/repo").Return(false, nil)
	repoRepo.EXPECT().Save(ctx, mockRepoSaved("owner/repo")).Return(nil)
	auditRepo.EXPECT().Save(ctx, mockAuditLogSaved()).Return(nil)

	repo, err := svc.CreateRepository(ctx, "owner/repo", nil, "admin", valueobject.ActorRoleAdmin)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	if repo.Name != "owner/repo" {
		t.Errorf("expected name 'owner/repo', got '%s'", repo.Name)
	}
	if !repo.Enabled {
		t.Errorf("expected repo to be enabled by default")
	}
	if repo.ID == uuid.Nil {
		t.Errorf("expected repo to have a valid ID")
	}
}

func TestRepositoryService_CreateRepository_WithConfigOverrides(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	configOverrides := &entity.RepoConfig{
		MaxConcurrency:      new(10),
		ComplexityThreshold: new(80),
		ForceDesignConfirm:  new(true),
		DefaultModel:        new("claude-sonnet-4-6"),
		TaskRetryLimit:      new(5),
	}

	repoRepo.EXPECT().ExistsByName(ctx, "owner/repo").Return(false, nil)
	repoRepo.EXPECT().Save(ctx, mockRepoSaved("owner/repo")).Return(nil)
	auditRepo.EXPECT().Save(ctx, mockAuditLogSaved()).Return(nil)

	repo, err := svc.CreateRepository(ctx, "owner/repo", configOverrides, "admin", valueobject.ActorRoleAdmin)
	if err != nil {
		t.Fatalf("CreateRepository failed: %v", err)
	}

	// Verify config overrides were applied
	if *repo.Config.MaxConcurrency != 10 {
		t.Errorf("expected MaxConcurrency 10, got %d", *repo.Config.MaxConcurrency)
	}
	if *repo.Config.ComplexityThreshold != 80 {
		t.Errorf("expected ComplexityThreshold 80, got %d", *repo.Config.ComplexityThreshold)
	}
	if !*repo.Config.ForceDesignConfirm {
		t.Errorf("expected ForceDesignConfirm true, got false")
	}
	if *repo.Config.DefaultModel != "claude-sonnet-4-6" {
		t.Errorf("expected DefaultModel 'claude-sonnet-4-6', got '%s'", *repo.Config.DefaultModel)
	}
	if *repo.Config.TaskRetryLimit != 5 {
		t.Errorf("expected TaskRetryLimit 5, got %d", *repo.Config.TaskRetryLimit)
	}
}

func TestRepositoryService_CreateRepository_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repoRepo.EXPECT().ExistsByName(ctx, "owner/repo").Return(true, nil)
	auditRepo.EXPECT().Save(ctx, mockAuditLogSaved()).Return(nil)

	_, err := svc.CreateRepository(ctx, "owner/repo", nil, "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for duplicate repository")
	}

	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
	if litchiErr.Code != litchierrors.ErrValidationFailed {
		t.Errorf("expected ErrValidationFailed, got %s", litchiErr.Code.Code)
	}
}

func TestRepositoryService_CreateRepository_InvalidName(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	auditRepo.EXPECT().Save(ctx, mockAuditLogSaved()).Return(nil)

	_, err := svc.CreateRepository(ctx, "invalidname", nil, "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}

	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestRepositoryService_CreateRepository_DatabaseError(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repoRepo.EXPECT().ExistsByName(ctx, "owner/repo").Return(false, nil)
	repoRepo.EXPECT().Save(ctx, mockRepoSaved("owner/repo")).Return(errors.New("database error"))
	auditRepo.EXPECT().Save(ctx, mockAuditLogSaved()).Return(nil)

	_, err := svc.CreateRepository(ctx, "owner/repo", nil, "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for database failure")
	}

	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
	if litchiErr.Code != litchierrors.ErrDatabaseOperation {
		t.Errorf("expected ErrDatabaseOperation, got %s", litchiErr.Code.Code)
	}
}

// --- Tests for UpdateRepository ---

func TestRepositoryService_UpdateRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	existingRepo := entity.NewRepository("owner/repo")
	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(existingRepo, nil)
	repoRepo.EXPECT().Save(ctx, mockRepoSaved("owner/repo")).Return(nil)
	auditRepo.EXPECT().Save(ctx, mockAuditLogSaved()).Return(nil)

	configOverrides := &entity.RepoConfig{
		MaxConcurrency: new(15),
	}

	repo, err := svc.UpdateRepository(ctx, "owner/repo", configOverrides, "admin", valueobject.ActorRoleAdmin)
	if err != nil {
		t.Fatalf("UpdateRepository failed: %v", err)
	}

	if *repo.Config.MaxConcurrency != 15 {
		t.Errorf("expected MaxConcurrency 15, got %d", *repo.Config.MaxConcurrency)
	}
}

func TestRepositoryService_UpdateRepository_NotFound(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(nil, nil)

	configOverrides := &entity.RepoConfig{
		MaxConcurrency: new(15),
	}

	_, err := svc.UpdateRepository(ctx, "owner/repo", configOverrides, "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for repository not found")
	}

	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestRepositoryService_UpdateRepository_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	existingRepo := entity.NewRepository("owner/repo")
	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(existingRepo, nil)
	auditRepo.EXPECT().Save(ctx, mockAuditLogSaved()).Return(nil)

	configOverrides := &entity.RepoConfig{
		MaxConcurrency: new(0),
	}

	_, err := svc.UpdateRepository(ctx, "owner/repo", configOverrides, "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}

	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for DeleteRepository ---

func TestRepositoryService_DeleteRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	existingRepo := entity.NewRepository("owner/repo")
	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(existingRepo, nil)
	repoRepo.EXPECT().Delete(ctx, "owner/repo").Return(nil)
	auditRepo.EXPECT().Save(ctx, mockAuditLogSaved()).Return(nil)

	err := svc.DeleteRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err != nil {
		t.Fatalf("DeleteRepository failed: %v", err)
	}
}

func TestRepositoryService_DeleteRepository_NotFound(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(nil, nil)

	err := svc.DeleteRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for repository not found")
	}

	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for GetRepository ---

func TestRepositoryService_GetRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	existingRepo := entity.NewRepository("owner/repo")
	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(existingRepo, nil)

	repo, err := svc.GetRepository(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("GetRepository failed: %v", err)
	}

	if repo == nil {
		t.Fatal("expected repository to be found")
	}
	if repo.Name != "owner/repo" {
		t.Errorf("expected name 'owner/repo', got '%s'", repo.Name)
	}
}

func TestRepositoryService_GetRepository_NotFound(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(nil, nil)

	repo, err := svc.GetRepository(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("GetRepository failed: %v", err)
	}

	if repo != nil {
		t.Errorf("expected nil for not found repository, got %+v", repo)
	}
}

// --- Tests for ListRepositories ---

func TestRepositoryService_ListRepositories_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repos := []*entity.Repository{
		entity.NewRepository("owner/repo1"),
		entity.NewRepository("owner/repo2"),
	}
	repoRepo.EXPECT().FindAll(ctx).Return(repos, nil)

	result, err := svc.ListRepositories(ctx)
	if err != nil {
		t.Fatalf("ListRepositories failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 repositories, got %d", len(result))
	}
}

func TestRepositoryService_ListRepositories_Empty(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repoRepo.EXPECT().FindAll(ctx).Return([]*entity.Repository{}, nil)

	repos, err := svc.ListRepositories(ctx)
	if err != nil {
		t.Fatalf("ListRepositories failed: %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("expected 0 repositories for empty storage, got %d", len(repos))
	}
}

// --- Tests for ListEnabledRepositories ---

func TestRepositoryService_ListEnabledRepositories_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repo1 := entity.NewRepository("owner/repo1")
	repos := []*entity.Repository{repo1}
	repoRepo.EXPECT().FindEnabled(ctx).Return(repos, nil)

	result, err := svc.ListEnabledRepositories(ctx)
	if err != nil {
		t.Fatalf("ListEnabledRepositories failed: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 enabled repository, got %d", len(result))
	}
	if result[0].Name != "owner/repo1" {
		t.Errorf("expected 'owner/repo1', got '%s'", result[0].Name)
	}
}

// --- Tests for EnableRepository ---

func TestRepositoryService_EnableRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repo := entity.NewRepository("owner/repo")
	repo.Disable()
	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(repo, nil)
	repoRepo.EXPECT().Save(ctx, repo).Return(nil)
	auditRepo.EXPECT().Save(ctx, mockAuditLogSaved()).Return(nil)

	err := svc.EnableRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err != nil {
		t.Fatalf("EnableRepository failed: %v", err)
	}

	if !repo.Enabled {
		t.Errorf("expected repository to be enabled")
	}
}

func TestRepositoryService_EnableRepository_NotFound(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(nil, nil)

	err := svc.EnableRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for repository not found")
	}

	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for DisableRepository ---

func TestRepositoryService_DisableRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repo := entity.NewRepository("owner/repo")
	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(repo, nil)
	repoRepo.EXPECT().Save(ctx, repo).Return(nil)
	auditRepo.EXPECT().Save(ctx, mockAuditLogSaved()).Return(nil)

	err := svc.DisableRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err != nil {
		t.Fatalf("DisableRepository failed: %v", err)
	}

	if repo.Enabled {
		t.Errorf("expected repository to be disabled")
	}
}

func TestRepositoryService_DisableRepository_NotFound(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(nil, nil)

	err := svc.DisableRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for repository not found")
	}

	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for GetEffectiveConfig ---

func TestRepositoryService_GetEffectiveConfig_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repo := entity.NewRepository("owner/repo")
	maxConcurrency := 10
	repo.SetMaxConcurrency(maxConcurrency)
	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(repo, nil)

	effectiveConfig, err := svc.GetEffectiveConfig(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("GetEffectiveConfig failed: %v", err)
	}

	if effectiveConfig.RepositoryName != "owner/repo" {
		t.Errorf("expected RepositoryName 'owner/repo', got '%s'", effectiveConfig.RepositoryName)
	}
	if !effectiveConfig.HasRepoConfig {
		t.Errorf("expected HasRepoConfig to be true")
	}
	if !effectiveConfig.Enabled {
		t.Errorf("expected Enabled to be true")
	}

	if *effectiveConfig.Effective.MaxConcurrency != 10 {
		t.Errorf("expected Effective.MaxConcurrency 10 (from repo), got %d", *effectiveConfig.Effective.MaxConcurrency)
	}
	if *effectiveConfig.Effective.ComplexityThreshold != 70 {
		t.Errorf("expected Effective.ComplexityThreshold 70 (from global), got %d", *effectiveConfig.Effective.ComplexityThreshold)
	}
}

func TestRepositoryService_GetEffectiveConfig_NoRepository(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	repoRepo.EXPECT().FindByName(ctx, "owner/repo").Return(nil, nil)

	effectiveConfig, err := svc.GetEffectiveConfig(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("GetEffectiveConfig failed: %v", err)
	}

	if effectiveConfig.HasRepoConfig {
		t.Errorf("expected HasRepoConfig to be false for non-existent repo")
	}
	if effectiveConfig.Enabled {
		t.Errorf("expected Enabled to be false for non-existent repo")
	}

	if *effectiveConfig.Effective.MaxConcurrency != 5 {
		t.Errorf("expected Effective.MaxConcurrency 5 (from global), got %d", *effectiveConfig.Effective.MaxConcurrency)
	}
}

// --- Tests for ValidateConfig ---

func TestRepositoryService_ValidateConfig_Valid(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	config := entity.RepoConfig{
		MaxConcurrency:      new(10),
		ComplexityThreshold: new(80),
		TaskRetryLimit:      new(5),
		DefaultModel:        new("claude-sonnet-4-6"),
	}

	err := svc.ValidateConfig(ctx, config)
	if err != nil {
		t.Fatalf("ValidateConfig failed for valid config: %v", err)
	}
}

func TestRepositoryService_ValidateConfig_InvalidMaxConcurrency(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	config := entity.RepoConfig{
		MaxConcurrency: new(0),
	}

	err := svc.ValidateConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error for invalid MaxConcurrency")
	}

	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestRepositoryService_ValidateConfig_InvalidComplexityThreshold(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	complexityThreshold := -1
	config := entity.RepoConfig{
		ComplexityThreshold: &complexityThreshold,
	}

	err := svc.ValidateConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error for negative ComplexityThreshold")
	}

	complexityThreshold = 101
	config = entity.RepoConfig{
		ComplexityThreshold: &complexityThreshold,
	}

	err = svc.ValidateConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error for ComplexityThreshold > 100")
	}
}

func TestRepositoryService_ValidateConfig_InvalidTaskRetryLimit(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	config := entity.RepoConfig{
		TaskRetryLimit: new(-1),
	}

	err := svc.ValidateConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error for invalid TaskRetryLimit")
	}

	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestRepositoryService_ValidateConfig_EmptyDefaultModel(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	config := entity.RepoConfig{
		DefaultModel: new(""),
	}

	err := svc.ValidateConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error for empty DefaultModel")
	}

	if _, ok := errors.AsType[*litchierrors.Error](err); !ok {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestRepositoryService_ValidateConfig_EmptyConfig(t *testing.T) {
	ctx := context.Background()
	repoRepo := repository.NewMockRepositoryRepository(t)
	auditRepo := repository.NewMockAuditLogRepository(t)
	svc := newTestRepositoryService(repoRepo, auditRepo)

	config := entity.RepoConfig{}

	err := svc.ValidateConfig(ctx, config)
	if err != nil {
		t.Fatalf("ValidateConfig failed for empty config: %v", err)
	}
}

// --- Mock matchers for testify mock ---

// mockRepoSaved matches any repository with the given name
func mockRepoSaved(name string) any {
	return mock.MatchedBy(func(repo *entity.Repository) bool {
		return repo.Name == name
	})
}

// mockAuditLogSaved matches any audit log
func mockAuditLogSaved() any {
	return mock.MatchedBy(func(log *entity.AuditLog) bool {
		return log != nil
	})
}

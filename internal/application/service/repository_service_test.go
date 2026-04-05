package service

import (
	"context"
	"errors"
	"testing"
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

// --- Mocks for RepositoryService tests (following design_service_test.go pattern) ---

type mockRepoRepository struct {
	repos      map[string]*entity.Repository
	saveErr    error
	findErr    error
	deleteErr  error
	existsErr  error
	findAllErr error
}

func newMockRepoRepository() *mockRepoRepository {
	return &mockRepoRepository{
		repos: make(map[string]*entity.Repository),
	}
}

func (m *mockRepoRepository) FindByName(ctx context.Context, name string) (*entity.Repository, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.repos[name], nil
}

func (m *mockRepoRepository) Save(ctx context.Context, repo *entity.Repository) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.repos[repo.Name] = repo
	return nil
}

func (m *mockRepoRepository) Delete(ctx context.Context, name string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.repos, name)
	return nil
}

func (m *mockRepoRepository) FindAll(ctx context.Context) ([]*entity.Repository, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}
	result := make([]*entity.Repository, 0, len(m.repos))
	for _, repo := range m.repos {
		result = append(result, repo)
	}
	return result, nil
}

func (m *mockRepoRepository) FindEnabled(ctx context.Context) ([]*entity.Repository, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}
	result := make([]*entity.Repository, 0)
	for _, repo := range m.repos {
		if repo.Enabled {
			result = append(result, repo)
		}
	}
	return result, nil
}

func (m *mockRepoRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	return m.repos[name] != nil, nil
}

type mockAuditLogRepositoryForRepo struct {
	logs   []*entity.AuditLog
	saveErr error
}

func newMockAuditLogRepositoryForRepo() *mockAuditLogRepositoryForRepo {
	return &mockAuditLogRepositoryForRepo{
		logs: make([]*entity.AuditLog, 0),
	}
}

func (m *mockAuditLogRepositoryForRepo) Save(ctx context.Context, log *entity.AuditLog) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAuditLogRepositoryForRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error) {
	return nil, nil
}

func (m *mockAuditLogRepositoryForRepo) List(ctx context.Context, opts repository.AuditLogListOptions) ([]*entity.AuditLog, int64, error) {
	return m.logs, int64(len(m.logs)), nil
}

func (m *mockAuditLogRepositoryForRepo) ListBySessionID(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditLogRepositoryForRepo) ListByRepository(ctx context.Context, repo string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditLogRepositoryForRepo) ListByActor(ctx context.Context, actor string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditLogRepositoryForRepo) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditLogRepositoryForRepo) CountBySession(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockAuditLogRepositoryForRepo) DeleteBeforeTime(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

// --- Helper functions for tests ---

func newTestRepositoryService(
	repoRepo repository.RepositoryRepository,
	auditRepo repository.AuditLogRepository,
) *RepositoryService {
	cfg := &config.Config{
		Agent: config.AgentConfig{
			MaxConcurrency:  5,
			TaskRetryLimit:  3,
			Type:            "claude-opus-4-6",
		},
		Complexity: config.ComplexityConfig{
			Threshold:          70,
			ForceDesignConfirm: false,
		},
	}

	dispatcher := event.NewDispatcher()

	return NewRepositoryService(
		repoRepo,
		auditRepo,
		dispatcher,
		cfg,
		zap.NewNop(),
	)
}

// --- Tests for CreateRepository ---

func TestRepositoryService_CreateRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

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

	// Verify audit log was recorded
	if len(auditRepo.logs) != 1 {
		t.Errorf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
}

func TestRepositoryService_CreateRepository_WithConfigOverrides(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	maxConcurrency := 10
	complexityThreshold := 80
	forceDesignConfirm := true
	defaultModel := "claude-sonnet-4-6"
	taskRetryLimit := 5

	configOverrides := &entity.RepoConfig{
		MaxConcurrency:      &maxConcurrency,
		ComplexityThreshold: &complexityThreshold,
		ForceDesignConfirm:  &forceDesignConfirm,
		DefaultModel:        &defaultModel,
		TaskRetryLimit:      &taskRetryLimit,
	}

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
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Create a repository first
	existingRepo := entity.NewRepository("owner/repo")
	repoRepo.repos["owner/repo"] = existingRepo

	// Try to create the same repository again
	_, err := svc.CreateRepository(ctx, "owner/repo", nil, "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for duplicate repository")
	}

	// Verify error type
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
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Invalid name without slash
	_, err := svc.CreateRepository(ctx, "invalidname", nil, "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}

	// Verify error type
	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestRepositoryService_CreateRepository_DatabaseError(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	repoRepo.saveErr = errors.New("database error")
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	_, err := svc.CreateRepository(ctx, "owner/repo", nil, "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for database failure")
	}

	// Verify error type
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
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Create a repository first
	existingRepo := entity.NewRepository("owner/repo")
	repoRepo.repos["owner/repo"] = existingRepo

	// Update with new config
	maxConcurrency := 15
	configOverrides := &entity.RepoConfig{
		MaxConcurrency: &maxConcurrency,
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
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	maxConcurrency := 15
	configOverrides := &entity.RepoConfig{
		MaxConcurrency: &maxConcurrency,
	}

	_, err := svc.UpdateRepository(ctx, "owner/repo", configOverrides, "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for repository not found")
	}

	// Verify error type
	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestRepositoryService_UpdateRepository_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Create a repository first
	existingRepo := entity.NewRepository("owner/repo")
	repoRepo.repos["owner/repo"] = existingRepo

	// Invalid config: MaxConcurrency < 1
	maxConcurrency := 0
	configOverrides := &entity.RepoConfig{
		MaxConcurrency: &maxConcurrency,
	}

	_, err := svc.UpdateRepository(ctx, "owner/repo", configOverrides, "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}

	// Verify error type
	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for DeleteRepository ---

func TestRepositoryService_DeleteRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Create a repository first
	existingRepo := entity.NewRepository("owner/repo")
	repoRepo.repos["owner/repo"] = existingRepo

	err := svc.DeleteRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err != nil {
		t.Fatalf("DeleteRepository failed: %v", err)
	}

	// Verify repository is deleted
	if repoRepo.repos["owner/repo"] != nil {
		t.Errorf("expected repository to be deleted")
	}

	// Verify audit log was recorded
	if len(auditRepo.logs) != 1 {
		t.Errorf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
}

func TestRepositoryService_DeleteRepository_NotFound(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	err := svc.DeleteRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for repository not found")
	}

	// Verify error type
	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for GetRepository ---

func TestRepositoryService_GetRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Create a repository first
	existingRepo := entity.NewRepository("owner/repo")
	repoRepo.repos["owner/repo"] = existingRepo

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
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

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
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Create some repositories
	repoRepo.repos["owner/repo1"] = entity.NewRepository("owner/repo1")
	repoRepo.repos["owner/repo2"] = entity.NewRepository("owner/repo2")

	repos, err := svc.ListRepositories(ctx)
	if err != nil {
		t.Fatalf("ListRepositories failed: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("expected 2 repositories, got %d", len(repos))
	}
}

func TestRepositoryService_ListRepositories_Empty(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

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
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Create some repositories, one disabled
	repo1 := entity.NewRepository("owner/repo1")
	repo2 := entity.NewRepository("owner/repo2")
	repo2.Disable()
	repoRepo.repos["owner/repo1"] = repo1
	repoRepo.repos["owner/repo2"] = repo2

	repos, err := svc.ListEnabledRepositories(ctx)
	if err != nil {
		t.Fatalf("ListEnabledRepositories failed: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("expected 1 enabled repository, got %d", len(repos))
	}
	if repos[0].Name != "owner/repo1" {
		t.Errorf("expected 'owner/repo1', got '%s'", repos[0].Name)
	}
}

// --- Tests for EnableRepository ---

func TestRepositoryService_EnableRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Create a disabled repository
	repo := entity.NewRepository("owner/repo")
	repo.Disable()
	repoRepo.repos["owner/repo"] = repo

	err := svc.EnableRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err != nil {
		t.Fatalf("EnableRepository failed: %v", err)
	}

	// Verify repository is enabled
	if !repoRepo.repos["owner/repo"].Enabled {
		t.Errorf("expected repository to be enabled")
	}
}

func TestRepositoryService_EnableRepository_NotFound(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	err := svc.EnableRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for repository not found")
	}

	// Verify error type
	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for DisableRepository ---

func TestRepositoryService_DisableRepository_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Create an enabled repository
	repo := entity.NewRepository("owner/repo")
	repoRepo.repos["owner/repo"] = repo

	err := svc.DisableRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err != nil {
		t.Fatalf("DisableRepository failed: %v", err)
	}

	// Verify repository is disabled
	if repoRepo.repos["owner/repo"].Enabled {
		t.Errorf("expected repository to be disabled")
	}
}

func TestRepositoryService_DisableRepository_NotFound(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	err := svc.DisableRepository(ctx, "owner/repo", "admin", valueobject.ActorRoleAdmin)
	if err == nil {
		t.Fatal("expected error for repository not found")
	}

	// Verify error type
	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

// --- Tests for GetEffectiveConfig ---

func TestRepositoryService_GetEffectiveConfig_Success(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Create a repository with custom config
	repo := entity.NewRepository("owner/repo")
	maxConcurrency := 10
	repo.SetMaxConcurrency(maxConcurrency)
	repoRepo.repos["owner/repo"] = repo

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

	// Verify effective config: MaxConcurrency from repo (10), others from global
	if *effectiveConfig.Effective.MaxConcurrency != 10 {
		t.Errorf("expected Effective.MaxConcurrency 10 (from repo), got %d", *effectiveConfig.Effective.MaxConcurrency)
	}
	if *effectiveConfig.Effective.ComplexityThreshold != 70 {
		t.Errorf("expected Effective.ComplexityThreshold 70 (from global), got %d", *effectiveConfig.Effective.ComplexityThreshold)
	}
}

func TestRepositoryService_GetEffectiveConfig_NoRepository(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// No repository in storage
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

	// Verify effective config equals global config
	if *effectiveConfig.Effective.MaxConcurrency != 5 {
		t.Errorf("expected Effective.MaxConcurrency 5 (from global), got %d", *effectiveConfig.Effective.MaxConcurrency)
	}
}

// --- Tests for ValidateConfig ---

func TestRepositoryService_ValidateConfig_Valid(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	maxConcurrency := 10
	complexityThreshold := 80
	taskRetryLimit := 5
	defaultModel := "claude-sonnet-4-6"

	config := entity.RepoConfig{
		MaxConcurrency:      &maxConcurrency,
		ComplexityThreshold: &complexityThreshold,
		TaskRetryLimit:      &taskRetryLimit,
		DefaultModel:        &defaultModel,
	}

	err := svc.ValidateConfig(ctx, config)
	if err != nil {
		t.Fatalf("ValidateConfig failed for valid config: %v", err)
	}
}

func TestRepositoryService_ValidateConfig_InvalidMaxConcurrency(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	maxConcurrency := 0 // Invalid: must be >= 1
	config := entity.RepoConfig{
		MaxConcurrency: &maxConcurrency,
	}

	err := svc.ValidateConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error for invalid MaxConcurrency")
	}

	// Verify error type
	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestRepositoryService_ValidateConfig_InvalidComplexityThreshold(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Test negative threshold
	complexityThreshold := -1
	config := entity.RepoConfig{
		ComplexityThreshold: &complexityThreshold,
	}

	err := svc.ValidateConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error for negative ComplexityThreshold")
	}

	// Test threshold > 100
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
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	taskRetryLimit := -1 // Invalid: must be >= 0
	config := entity.RepoConfig{
		TaskRetryLimit: &taskRetryLimit,
	}

	err := svc.ValidateConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error for invalid TaskRetryLimit")
	}

	// Verify error type
	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestRepositoryService_ValidateConfig_EmptyDefaultModel(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	defaultModel := "" // Invalid: must be non-empty if set
	config := entity.RepoConfig{
		DefaultModel: &defaultModel,
	}

	err := svc.ValidateConfig(ctx, config)
	if err == nil {
		t.Fatal("expected error for empty DefaultModel")
	}

	// Verify error type
	var litchiErr *litchierrors.Error
	if !errors.As(err, &litchiErr) {
		t.Errorf("expected litchierrors.Error, got %T", err)
	}
}

func TestRepositoryService_ValidateConfig_EmptyConfig(t *testing.T) {
	ctx := context.Background()
	repoRepo := newMockRepoRepository()
	auditRepo := newMockAuditLogRepositoryForRepo()
	svc := newTestRepositoryService(repoRepo, auditRepo)

	// Empty config with all nil values should be valid
	config := entity.RepoConfig{}

	err := svc.ValidateConfig(ctx, config)
	if err != nil {
		t.Fatalf("ValidateConfig failed for empty config: %v", err)
	}
}
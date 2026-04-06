package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/github"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// mockSessionRepo is a mock implementation of WorkSessionRepository.
type mockSessionRepo struct {
	sessions    map[uuid.UUID]*aggregate.WorkSession
	issueIndex  map[string]uuid.UUID // "owner/repo:number" -> sessionID
	createErr   error
	updateErr   error
	findErr     error
	findNil     bool
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{
		sessions:   make(map[uuid.UUID]*aggregate.WorkSession),
		issueIndex: make(map[string]uuid.UUID),
	}
}

func (m *mockSessionRepo) Create(ctx context.Context, session *aggregate.WorkSession) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.sessions[session.ID] = session
	key := session.Issue.Repository + ":" + string(rune(session.Issue.Number))
	m.issueIndex[key] = session.ID
	return nil
}

func (m *mockSessionRepo) Update(ctx context.Context, session *aggregate.WorkSession) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionRepo) FindByID(ctx context.Context, id uuid.UUID) (*aggregate.WorkSession, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.sessions[id], nil
}

func (m *mockSessionRepo) FindByIssueID(ctx context.Context, issueID uuid.UUID) (*aggregate.WorkSession, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, s := range m.sessions {
		if s.Issue.ID == issueID {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockSessionRepo) FindByGitHubIssue(ctx context.Context, repository string, issueNumber int) (*aggregate.WorkSession, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if m.findNil {
		return nil, nil
	}
	key := repository + ":" + string(rune(issueNumber))
	if id, ok := m.issueIndex[key]; ok {
		return m.sessions[id], nil
	}
	return nil, nil
}

func (m *mockSessionRepo) FindByStatus(ctx context.Context, status aggregate.SessionStatus) ([]*aggregate.WorkSession, error) {
	return nil, nil
}

func (m *mockSessionRepo) FindByStage(ctx context.Context, stage valueobject.Stage) ([]*aggregate.WorkSession, error) {
	return nil, nil
}

func (m *mockSessionRepo) ListWithPagination(ctx context.Context, params repository.PaginationParams, filter *repository.WorkSessionFilter) ([]*aggregate.WorkSession, *repository.PaginationResult, error) {
	return nil, nil, nil
}

func (m *mockSessionRepo) FindActiveByRepository(ctx context.Context, repository string) ([]*aggregate.WorkSession, error) {
	return nil, nil
}

func (m *mockSessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockSessionRepo) ExistsByGitHubIssue(ctx context.Context, repository string, issueNumber int) (bool, error) {
	key := repository + ":" + string(rune(issueNumber))
	_, ok := m.issueIndex[key]
	return ok, nil
}

// mockRepoRepo is a mock implementation of RepositoryRepository.
type mockRepoRepo struct {
	repos     map[string]*entity.Repository
	findErr   error
}

func newMockRepoRepo() *mockRepoRepo {
	return &mockRepoRepo{
		repos: make(map[string]*entity.Repository),
	}
}

func (m *mockRepoRepo) FindByName(ctx context.Context, name string) (*entity.Repository, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.repos[name], nil
}

func (m *mockRepoRepo) Save(ctx context.Context, repo *entity.Repository) error {
	m.repos[repo.Name] = repo
	return nil
}

func (m *mockRepoRepo) Delete(ctx context.Context, name string) error {
	delete(m.repos, name)
	return nil
}

func (m *mockRepoRepo) FindAll(ctx context.Context) ([]*entity.Repository, error) {
	result := make([]*entity.Repository, 0, len(m.repos))
	for _, r := range m.repos {
		result = append(result, r)
	}
	return result, nil
}

func (m *mockRepoRepo) FindEnabled(ctx context.Context) ([]*entity.Repository, error) {
	result := make([]*entity.Repository, 0)
	for _, r := range m.repos {
		if r.IsEnabled() {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRepoRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	_, ok := m.repos[name]
	return ok, nil
}

func (m *mockRepoRepo) ListWithPagination(ctx context.Context, params repository.PaginationParams, filter *repository.RepositoryFilter) ([]*entity.Repository, *repository.PaginationResult, error) {
	// Apply filter
	var filtered []*entity.Repository
	for _, repo := range m.repos {
		if filter != nil && filter.Enabled != nil {
			if repo.Enabled != *filter.Enabled {
				continue
			}
		}
		filtered = append(filtered, repo)
	}

	// Calculate pagination
	total := len(filtered)
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return filtered[start:end], &repository.PaginationResult{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, nil
}

// mockAuditRepo is a mock implementation of AuditLogRepository.
type mockAuditRepo struct {
	logs    []*entity.AuditLog
	saveErr error
}

func newMockAuditRepo() *mockAuditRepo {
	return &mockAuditRepo{
		logs: make([]*entity.AuditLog, 0),
	}
}

func (m *mockAuditRepo) Save(ctx context.Context, auditLog *entity.AuditLog) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.logs = append(m.logs, auditLog)
	return nil
}

func (m *mockAuditRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error) {
	return nil, nil
}

func (m *mockAuditRepo) List(ctx context.Context, opts repository.AuditLogListOptions) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditRepo) ListBySessionID(ctx context.Context, sessionID uuid.UUID, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditRepo) ListByRepository(ctx context.Context, repository string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditRepo) ListByActor(ctx context.Context, actor string, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditRepo) ListByTimeRange(ctx context.Context, startTime, endTime time.Time, offset, limit int) ([]*entity.AuditLog, int64, error) {
	return nil, 0, nil
}

func (m *mockAuditRepo) CountBySession(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockAuditRepo) DeleteBeforeTime(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

// mockGitHubIssueService is a mock implementation of GitHub IssueService.
type mockGitHubIssueService struct {
	issue      *entity.Issue
	isAdmin    bool
	getErr     error
	adminErr   error
}

func (m *mockGitHubIssueService) GetIssue(ctx context.Context, owner, repo string, number int) (*entity.Issue, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.issue, nil
}

func (m *mockGitHubIssueService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (int64, error) {
	return 1, nil
}

func (m *mockGitHubIssueService) UpdateComment(ctx context.Context, owner, repo string, commentID int64, body string) error {
	return nil
}

func (m *mockGitHubIssueService) CloseIssue(ctx context.Context, owner, repo string, number int) error {
	return nil
}

func (m *mockGitHubIssueService) ReopenIssue(ctx context.Context, owner, repo string, number int) error {
	return nil
}

func (m *mockGitHubIssueService) AddLabels(ctx context.Context, owner, repo string, number int, labels []string) error {
	return nil
}

func (m *mockGitHubIssueService) RemoveLabel(ctx context.Context, owner, repo string, number int, label string) error {
	return nil
}

func (m *mockGitHubIssueService) GetLabels(ctx context.Context, owner, repo string, number int) ([]string, error) {
	return nil, nil
}

func (m *mockGitHubIssueService) ListComments(ctx context.Context, owner, repo string, number int) ([]*github.IssueComment, error) {
	return nil, nil
}

func (m *mockGitHubIssueService) CreateIssue(ctx context.Context, owner, repo, title, body string, labels []string) (*entity.Issue, error) {
	return nil, nil
}

func (m *mockGitHubIssueService) UpdateIssue(ctx context.Context, owner, repo string, number int, title, body *string) error {
	return nil
}

func (m *mockGitHubIssueService) AssignIssue(ctx context.Context, owner, repo string, number int, assignees []string) error {
	return nil
}

func (m *mockGitHubIssueService) GetPermissionLevel(ctx context.Context, owner, repo, username string) (string, error) {
	return "", nil
}

func (m *mockGitHubIssueService) IsRepoAdmin(ctx context.Context, owner, repo, username string) (bool, error) {
	if m.adminErr != nil {
		return false, m.adminErr
	}
	return m.isAdmin, nil
}

func TestProcessIssueEvent_NewSession(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepo()
	repoRepo := newMockRepoRepo()
	auditRepo := newMockAuditRepo()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{}

	svc := NewIssueService(
		sessionRepo,
		repoRepo,
		auditRepo,
		&github.IssueService{},
		dispatcher,
		cfg,
		zap.NewNop(),
	)

	session, isNew, err := svc.ProcessIssueEvent(
		ctx,
		"owner/repo",
		123,
		"Test Issue",
		"Test body",
		"testuser",
		[]string{"bug"},
		"https://github.com/owner/repo/issues/123",
		time.Now(),
	)

	if err != nil {
		t.Fatalf("ProcessIssueEvent failed: %v", err)
	}

	if !isNew {
		t.Error("Expected new session to be created")
	}

	if session == nil {
		t.Fatal("Expected session to be returned")
	}

	if session.Issue.Number != 123 {
		t.Errorf("Expected issue number 123, got %d", session.Issue.Number)
	}

	if session.Issue.Repository != "owner/repo" {
		t.Errorf("Expected repository owner/repo, got %s", session.Issue.Repository)
	}

	if session.CurrentStage != valueobject.StageClarification {
		t.Errorf("Expected stage Clarification, got %s", session.CurrentStage)
	}

	// Check audit log was created
	if len(auditRepo.logs) != 1 {
		t.Errorf("Expected 1 audit log, got %d", len(auditRepo.logs))
	}
}

func TestProcessIssueEvent_ExistingSession(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepo()
	repoRepo := newMockRepoRepo()
	auditRepo := newMockAuditRepo()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{}

	// Pre-create a session
	issue := entity.NewIssueFromGitHub(
		123,
		"Test Issue",
		"Test body",
		"owner/repo",
		"testuser",
		[]string{"bug"},
		"https://github.com/owner/repo/issues/123",
		time.Now(),
	)
	existingSession, _ := aggregate.NewWorkSession(issue)
	sessionRepo.Create(ctx, existingSession)

	svc := NewIssueService(
		sessionRepo,
		repoRepo,
		auditRepo,
		&github.IssueService{},
		dispatcher,
		cfg,
		zap.NewNop(),
	)

	session, isNew, err := svc.ProcessIssueEvent(
		ctx,
		"owner/repo",
		123,
		"Test Issue",
		"Test body",
		"testuser",
		[]string{"bug"},
		"https://github.com/owner/repo/issues/123",
		time.Now(),
	)

	if err != nil {
		t.Fatalf("ProcessIssueEvent failed: %v", err)
	}

	if isNew {
		t.Error("Expected existing session to be returned, not new")
	}

	if session.ID != existingSession.ID {
		t.Error("Expected same session ID")
	}
}

func TestProcessIssueEvent_DisabledRepository(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepo()
	repoRepo := newMockRepoRepo()
	auditRepo := newMockAuditRepo()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{}

	// Create disabled repository
	repo := entity.NewRepository("owner/repo")
	repo.Disable()
	repoRepo.repos["owner/repo"] = repo

	svc := NewIssueService(
		sessionRepo,
		repoRepo,
		auditRepo,
		&github.IssueService{},
		dispatcher,
		cfg,
		zap.NewNop(),
	)

	_, _, err := svc.ProcessIssueEvent(
		ctx,
		"owner/repo",
		123,
		"Test Issue",
		"Test body",
		"testuser",
		[]string{"bug"},
		"https://github.com/owner/repo/issues/123",
		time.Now(),
	)

	if err == nil {
		t.Fatal("Expected error for disabled repository")
	}

	if !litchierrors.Is(err, litchierrors.ErrPermissionDenied) {
		t.Errorf("Expected PermissionDenied error, got %v", err)
	}
}

func TestProcessIssueEvent_DatabaseError(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepo()
	repoRepo := newMockRepoRepo()
	auditRepo := newMockAuditRepo()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{}

	// Set database error
	sessionRepo.createErr = errors.New("database error")

	svc := NewIssueService(
		sessionRepo,
		repoRepo,
		auditRepo,
		&github.IssueService{},
		dispatcher,
		cfg,
		zap.NewNop(),
	)

	_, _, err := svc.ProcessIssueEvent(
		ctx,
		"owner/repo",
		123,
		"Test Issue",
		"Test body",
		"testuser",
		[]string{"bug"},
		"https://github.com/owner/repo/issues/123",
		time.Now(),
	)

	if err == nil {
		t.Fatal("Expected database error")
	}

	if !litchierrors.Is(err, litchierrors.ErrDatabaseOperation) {
		t.Errorf("Expected DatabaseOperation error, got %v", err)
	}
}

func TestProcessIssueCommandEvent_Continue(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepo()
	repoRepo := newMockRepoRepo()
	auditRepo := newMockAuditRepo()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{}

	// Create paused session
	issue := entity.NewIssueFromGitHub(
		123,
		"Test Issue",
		"Test body",
		"owner/repo",
		"testuser",
		[]string{"bug"},
		"https://github.com/owner/repo/issues/123",
		time.Now(),
	)
	session, _ := aggregate.NewWorkSession(issue)
	session.Pause("test_pause")
	sessionRepo.Create(ctx, session)

	svc := NewIssueService(
		sessionRepo,
		repoRepo,
		auditRepo,
		&github.IssueService{},
		dispatcher,
		cfg,
		zap.NewNop(),
	)

	updatedSession, err := svc.ProcessIssueCommandEvent(
		ctx,
		"owner/repo",
		123,
		"testuser",
		"continue",
	)

	if err != nil {
		t.Fatalf("ProcessIssueCommandEvent failed: %v", err)
	}

	if updatedSession.SessionStatus != aggregate.SessionStatusActive {
		t.Errorf("Expected session to be active, got %s", updatedSession.SessionStatus)
	}
}

func TestProcessIssueCommandEvent_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepo()
	repoRepo := newMockRepoRepo()
	auditRepo := newMockAuditRepo()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{}

	// Create session with different author
	issue := entity.NewIssueFromGitHub(
		123,
		"Test Issue",
		"Test body",
		"owner/repo",
		"originalauthor",
		[]string{"bug"},
		"https://github.com/owner/repo/issues/123",
		time.Now(),
	)
	session, _ := aggregate.NewWorkSession(issue)
	sessionRepo.Create(ctx, session)

	// Note: This test will fail at the GitHub API call because we don't have a mock.
	// In a real scenario, we would use a mock interface for ghIssueService.
	// For now, we skip this test as it requires integration testing.
	t.Skip("Skipping test that requires GitHub API mock - use integration test instead")

	svc := NewIssueService(
		sessionRepo,
		repoRepo,
		auditRepo,
		&github.IssueService{},
		dispatcher,
		cfg,
		zap.NewNop(),
	)

	_, err := svc.ProcessIssueCommandEvent(
		ctx,
		"owner/repo",
		123,
		"otheruser", // Not the author
		"continue",
	)

	if err == nil {
		t.Fatal("Expected permission denied error")
	}

	if !litchierrors.Is(err, litchierrors.ErrPermissionDenied) {
		t.Errorf("Expected PermissionDenied error, got %v", err)
	}
}

func TestProcessIssueCommandEvent_UnknownCommand(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepo()
	repoRepo := newMockRepoRepo()
	auditRepo := newMockAuditRepo()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{}

	// Create session
	issue := entity.NewIssueFromGitHub(
		123,
		"Test Issue",
		"Test body",
		"owner/repo",
		"testuser",
		[]string{"bug"},
		"https://github.com/owner/repo/issues/123",
		time.Now(),
	)
	session, _ := aggregate.NewWorkSession(issue)
	sessionRepo.Create(ctx, session)

	svc := NewIssueService(
		sessionRepo,
		repoRepo,
		auditRepo,
		&github.IssueService{},
		dispatcher,
		cfg,
		zap.NewNop(),
	)

	_, err := svc.ProcessIssueCommandEvent(
		ctx,
		"owner/repo",
		123,
		"testuser",
		"unknown_command",
	)

	if err == nil {
		t.Fatal("Expected error for unknown command")
	}

	if !litchierrors.Is(err, litchierrors.ErrBadRequest) {
		t.Errorf("Expected BadRequest error, got %v", err)
	}
}

func TestGetSession(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepo()
	repoRepo := newMockRepoRepo()
	auditRepo := newMockAuditRepo()
	dispatcher := event.NewDispatcher()
	cfg := &config.Config{}

	// Create session
	issue := entity.NewIssueFromGitHub(
		123,
		"Test Issue",
		"Test body",
		"owner/repo",
		"testuser",
		[]string{"bug"},
		"https://github.com/owner/repo/issues/123",
		time.Now(),
	)
	session, _ := aggregate.NewWorkSession(issue)
	sessionRepo.Create(ctx, session)

	svc := NewIssueService(
		sessionRepo,
		repoRepo,
		auditRepo,
		&github.IssueService{},
		dispatcher,
		cfg,
		zap.NewNop(),
	)

	// Test found
	found, err := svc.GetSession(ctx, "owner/repo", 123)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if found.ID != session.ID {
		t.Error("Expected same session ID")
	}

	// Test not found
	_, err = svc.GetSession(ctx, "owner/repo", 999)
	if err == nil {
		t.Fatal("Expected session not found error")
	}
	if !litchierrors.Is(err, litchierrors.ErrSessionNotFound) {
		t.Errorf("Expected SessionNotFound error, got %v", err)
	}
}


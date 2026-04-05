// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"fmt"
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
	"github.com/ryuyb/litchi/internal/pkg/utils"
	"go.uber.org/zap"
)

// IssueService handles GitHub Issue-related operations.
// It manages the lifecycle of WorkSessions triggered by GitHub Issues.
type IssueService struct {
	sessionRepo    repository.WorkSessionRepository
	repoRepo       repository.RepositoryRepository
	auditRepo      repository.AuditLogRepository
	ghIssueService *github.IssueService
	eventDispatcher *event.Dispatcher
	config         *config.Config
	logger         *zap.Logger
}

// NewIssueService creates a new IssueService.
func NewIssueService(
	sessionRepo repository.WorkSessionRepository,
	repoRepo repository.RepositoryRepository,
	auditRepo repository.AuditLogRepository,
	ghIssueService *github.IssueService,
	eventDispatcher *event.Dispatcher,
	config *config.Config,
	logger *zap.Logger,
) *IssueService {
	return &IssueService{
		sessionRepo:    sessionRepo,
		repoRepo:       repoRepo,
		auditRepo:      auditRepo,
		ghIssueService: ghIssueService,
		eventDispatcher: eventDispatcher,
		config:         config,
		logger:         logger.Named("issue_service"),
	}
}

// ProcessIssueEvent processes a GitHub Issue webhook event.
// This is the main entry point for handling Issue opened/reopened events.
// It creates a new WorkSession if one doesn't exist, validates permissions,
// and triggers the workflow.
//
// Returns:
// - session: The created or existing WorkSession
// - isNew: True if a new session was created, false if existing
// - err: Error if processing failed
func (s *IssueService) ProcessIssueEvent(
	ctx context.Context,
	repoName string,
	issueNumber int,
	issueTitle string,
	issueBody string,
	author string,
	labels []string,
	issueURL string,
	createdAt time.Time,
) (session *aggregate.WorkSession, isNew bool, err error) {
	startTime := time.Now()

	// 1. Check if repository is enabled
	_, err = s.checkRepositoryEnabled(ctx, repoName)
	if err != nil {
		return nil, false, err
	}

	// 2. Check if session already exists for this issue
	existingSession, err := s.sessionRepo.FindByGitHubIssue(ctx, repoName, issueNumber)
	if err != nil {
		s.logger.Error("failed to check existing session",
			zap.String("repository", repoName),
			zap.Int("issue_number", issueNumber),
			zap.Error(err),
		)
		return nil, false, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	if existingSession != nil {
		s.logger.Info("session already exists for issue",
			zap.String("session_id", existingSession.ID.String()),
			zap.String("repository", repoName),
			zap.Int("issue_number", issueNumber),
		)
		return existingSession, false, nil
	}

	// 3. Create Issue entity
	issue := entity.NewIssueFromGitHub(
		issueNumber,
		issueTitle,
		issueBody,
		repoName,
		author,
		labels,
		issueURL,
		createdAt,
	)

	if err := issue.Validate(); err != nil {
		return nil, false, err
	}

	// 4. Create new WorkSession
	session, err = aggregate.NewWorkSession(issue)
	if err != nil {
		return nil, false, err
	}

	// 5. Persist session to database
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		s.logger.Error("failed to create session",
			zap.String("repository", repoName),
			zap.Int("issue_number", issueNumber),
			zap.Error(err),
		)
		return nil, false, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 6. Record audit log
	s.recordAuditLog(ctx, session, author, valueobject.ActorRoleIssueAuthor, valueobject.OpSessionStart, startTime, true, "")

	// 7. Publish domain events
	s.publishEvents(ctx, session)

	s.logger.Info("new session created for issue",
		zap.String("session_id", session.ID.String()),
		zap.String("repository", repoName),
		zap.Int("issue_number", issueNumber),
		zap.String("author", author),
	)

	return session, true, nil
}

// ProcessIssueCommandEvent processes a user command on an Issue (e.g., @bot continue).
// Commands can be issued by the Issue author or repository admins.
//
// Supported commands:
// - continue: Resume a paused session
// - restart: Terminate current session and start fresh
// - rollback: Rollback to a previous stage
// - terminate: Terminate the session
func (s *IssueService) ProcessIssueCommandEvent(
	ctx context.Context,
	repoName string,
	issueNumber int,
	actor string,
	command string,
) (session *aggregate.WorkSession, err error) {
	startTime := time.Now()

	// 1. Check permission
	actorRole, err := s.checkActorPermission(ctx, repoName, issueNumber, actor)
	if err != nil {
		return nil, err
	}

	// 2. Get existing session
	session, err = s.sessionRepo.FindByGitHubIssue(ctx, repoName, issueNumber)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("no session found for issue #%d", issueNumber),
		)
	}

	// 3. Process command based on type
	var operation valueobject.OperationType
	var success bool
	var errMsg string

	switch command {
	case "continue":
		operation = valueobject.OpSessionResume
		err = s.handleContinueCommand(ctx, session)
		if err != nil {
			errMsg = err.Error()
		} else {
			success = true
		}

	case "restart":
		operation = valueobject.OpSessionTerminate
		err = s.handleRestartCommand(ctx, repoName, issueNumber, actor)
		if err != nil {
			errMsg = err.Error()
		} else {
			success = true
			// Session will be different after restart
			session, _ = s.sessionRepo.FindByGitHubIssue(ctx, repoName, issueNumber)
		}

	case "terminate":
		operation = valueobject.OpSessionTerminate
		err = s.handleTerminateCommand(ctx, session)
		if err != nil {
			errMsg = err.Error()
		} else {
			success = true
		}

	default:
		operation = valueobject.OpUserCommand
		errMsg = fmt.Sprintf("unknown command: %s", command)
		err = litchierrors.New(litchierrors.ErrBadRequest).WithDetail(errMsg)
	}

	// 4. Record audit log
	s.recordAuditLog(ctx, session, actor, actorRole, operation, startTime, success, errMsg)

	// 5. Publish events if successful
	if success && session != nil {
		s.publishEvents(ctx, session)
	}

	return session, err
}

// checkRepositoryEnabled checks if the repository is enabled for processing.
// If no repository config exists, it's considered enabled by default.
func (s *IssueService) checkRepositoryEnabled(ctx context.Context, repoName string) (*entity.Repository, error) {
	repo, err := s.repoRepo.FindByName(ctx, repoName)
	if err != nil {
		s.logger.Error("failed to check repository config",
			zap.String("repository", repoName),
			zap.Error(err),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// If no config exists, repository is enabled by default
	if repo == nil {
		s.logger.Debug("no repository config found, enabled by default",
			zap.String("repository", repoName),
		)
		return nil, nil
	}

	if !repo.IsEnabled() {
		s.logger.Info("repository is disabled",
			zap.String("repository", repoName),
		)
		return nil, litchierrors.New(litchierrors.ErrPermissionDenied).WithDetail(
			fmt.Sprintf("repository %s is disabled", repoName),
		)
	}

	return repo, nil
}

// checkActorPermission checks if the actor has permission to issue commands.
// Returns the actor's role (IssueAuthor or Admin).
func (s *IssueService) checkActorPermission(
	ctx context.Context,
	repoName string,
	issueNumber int,
	actor string,
) (valueobject.ActorRole, error) {
	// 1. Get the session to find the issue author
	session, err := s.sessionRepo.FindByGitHubIssue(ctx, repoName, issueNumber)
	if err != nil {
		return "", litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return "", litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("no session found for issue #%d", issueNumber),
		)
	}

	// 2. Check if actor is the issue author
	if session.Issue.Author == actor {
		return valueobject.ActorRoleIssueAuthor, nil
	}

	// 3. Check if actor is repo admin/maintain
	isAdmin, err := s.ghIssueService.IsRepoAdmin(ctx, utils.ExtractOwner(repoName), utils.ExtractRepo(repoName), actor)
	if err != nil {
		s.logger.Warn("failed to check admin permission",
			zap.String("repository", repoName),
			zap.String("actor", actor),
			zap.Error(err),
		)
		// Permission check failed, deny access
		return "", litchierrors.New(litchierrors.ErrPermissionDenied).WithDetail(
			"failed to verify admin permission",
		)
	}

	if isAdmin {
		return valueobject.ActorRoleAdmin, nil
	}

	// Actor is neither issue author nor admin
	return "", litchierrors.New(litchierrors.ErrPermissionDenied).WithDetail(
		fmt.Sprintf("actor %s is not authorized (must be issue author or admin)", actor),
	)
}

// handleContinueCommand handles the "continue" command to resume a paused session.
func (s *IssueService) handleContinueCommand(ctx context.Context, session *aggregate.WorkSession) error {
	if !session.IsPaused() {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not paused",
		)
	}

	if err := session.ResumeWithAction("user_continue_command"); err != nil {
		return err
	}

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.logger.Info("session resumed by continue command",
		zap.String("session_id", session.ID.String()),
	)

	return nil
}

// handleRestartCommand handles the "restart" command to terminate and create new session.
func (s *IssueService) handleRestartCommand(
	ctx context.Context,
	repoName string,
	issueNumber int,
	actor string,
) error {
	session, err := s.sessionRepo.FindByGitHubIssue(ctx, repoName, issueNumber)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("no session found for issue #%d", issueNumber),
		)
	}

	// Terminate existing session
	if err := session.Terminate("user_restart_command"); err != nil {
		return err
	}

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// Get fresh issue data from GitHub
	issue, err := s.ghIssueService.GetIssue(ctx, utils.ExtractOwner(repoName), utils.ExtractRepo(repoName), issueNumber)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrGitHubAPIError, err)
	}

	// Create new session
	newSession, err := aggregate.NewWorkSession(issue)
	if err != nil {
		return err
	}

	if err := s.sessionRepo.Create(ctx, newSession); err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.logger.Info("session restarted",
		zap.String("old_session_id", session.ID.String()),
		zap.String("new_session_id", newSession.ID.String()),
		zap.String("actor", actor),
	)

	return nil
}

// handleTerminateCommand handles the "terminate" command.
func (s *IssueService) handleTerminateCommand(ctx context.Context, session *aggregate.WorkSession) error {
	if session.SessionStatus.IsTerminal() {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is already terminated",
		)
	}

	if err := session.Terminate("user_terminate_command"); err != nil {
		return err
	}

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.logger.Info("session terminated by user command",
		zap.String("session_id", session.ID.String()),
	)

	return nil
}

// recordAuditLog records an audit log entry for the operation.
func (s *IssueService) recordAuditLog(
	ctx context.Context,
	session *aggregate.WorkSession,
	actor string,
	actorRole valueobject.ActorRole,
	operation valueobject.OperationType,
	startTime time.Time,
	success bool,
	errMsg string,
) {
	if session == nil {
		return
	}

	auditLog := entity.NewAuditLog(
		session.ID,
		session.Issue.Repository,
		session.Issue.Number,
		actor,
		actorRole,
		operation,
		"work_session",
		session.ID.String(),
	)

	auditLog.SetDuration(int(time.Since(startTime).Milliseconds()))

	if success {
		auditLog.MarkSuccess()
	} else if errMsg != "" {
		auditLog.MarkFailed(errMsg)
	}

	if err := s.auditRepo.Save(ctx, auditLog); err != nil {
		s.logger.Warn("failed to save audit log",
			zap.String("session_id", session.ID.String()),
			zap.Error(err),
		)
	}
}

// publishEvents publishes all collected domain events from the session.
func (s *IssueService) publishEvents(ctx context.Context, session *aggregate.WorkSession) {
	events := session.GetEvents()
	if len(events) == 0 {
		return
	}

	if err := s.eventDispatcher.DispatchBatch(ctx, events); err != nil {
		s.logger.Warn("failed to dispatch events",
			zap.String("session_id", session.ID.String()),
			zap.Int("event_count", len(events)),
			zap.Error(err),
		)
	}

	session.ClearEvents()
}

// GetSession retrieves a WorkSession by repository and issue number.
func (s *IssueService) GetSession(ctx context.Context, repoName string, issueNumber int) (*aggregate.WorkSession, error) {
	session, err := s.sessionRepo.FindByGitHubIssue(ctx, repoName, issueNumber)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("no session found for issue #%d in %s", issueNumber, repoName),
		)
	}
	return session, nil
}

// GetSessionByID retrieves a WorkSession by its ID.
func (s *IssueService) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*aggregate.WorkSession, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("no session found with ID %s", sessionID.String()),
		)
	}
	return session, nil
}

// ListSessions lists WorkSessions with pagination and optional filtering.
func (s *IssueService) ListSessions(
	ctx context.Context,
	page, pageSize int,
	filter *repository.WorkSessionFilter,
) ([]*aggregate.WorkSession, *repository.PaginationResult, error) {
	params := repository.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}

	sessions, result, err := s.sessionRepo.ListWithPagination(ctx, params, filter)
	if err != nil {
		return nil, nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	return sessions, result, nil
}
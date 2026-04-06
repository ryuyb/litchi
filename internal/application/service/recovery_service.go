// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// EventDispatcher defines the minimal interface for event dispatching needed by RecoveryService.
// This interface allows decoupling from the concrete event.Dispatcher implementation.
type EventDispatcher interface {
	DispatchAsync(ctx context.Context, event event.DomainEvent)
	DispatchSync(ctx context.Context, event event.DomainEvent) error
}

// RecoveryService handles session recovery operations including:
// - Service restart recovery: automatically resume sessions paused due to service restart
// - User command recovery: handle @bot continue commands and manual recovery triggers
//
// This service coordinates with ConsistencyService, SessionControlService, and TaskService
// to ensure proper state validation and task continuation during recovery.
type RecoveryService struct {
	sessionRepo           repository.WorkSessionRepository
	cacheRepo             repository.CacheRepository
	consistencyService    *ConsistencyService
	sessionControlService service.SessionControlService
	taskService           *TaskService
	eventDispatcher       EventDispatcher
	logger                *zap.Logger
	config                *config.Config
}

// RecoveryServiceParams holds dependencies for RecoveryService.
type RecoveryServiceParams struct {
	fx.In

	SessionRepo           repository.WorkSessionRepository
	CacheRepo             repository.CacheRepository
	ConsistencyService    *ConsistencyService
	SessionControlService service.SessionControlService
	TaskService           *TaskService
	EventDispatcher       EventDispatcher `name:"event_dispatcher"`
	Config                *config.Config
	Logger                *zap.Logger
}

// NewRecoveryService creates a new RecoveryService.
func NewRecoveryService(p RecoveryServiceParams) *RecoveryService {
	return &RecoveryService{
		sessionRepo:           p.SessionRepo,
		cacheRepo:             p.CacheRepo,
		consistencyService:    p.ConsistencyService,
		sessionControlService: p.SessionControlService,
		taskService:           p.TaskService,
		eventDispatcher:       p.EventDispatcher,
		config:                p.Config,
		logger:                p.Logger.Named("recovery_service"),
	}
}

// SetEventDispatcher sets the event dispatcher for recovery events.
// This is useful when the dispatcher is not available at construction time.
func (s *RecoveryService) SetEventDispatcher(dispatcher EventDispatcher) {
	s.eventDispatcher = dispatcher
}

// RecoveryStatus represents the recovery status of a session.
type RecoveryStatus struct {
	SessionID         uuid.UUID          `json:"sessionId"`
	Repository        string             `json:"repository"`
	IssueNumber       int                `json:"issueNumber"`
	Title             string             `json:"title"`
	CurrentStage      string             `json:"currentStage"`
	Status            string             `json:"status"`
	PauseReason       string             `json:"pauseReason"`
	PauseReasonDetail string             `json:"pauseReasonDetail"`
	PausedAt          time.Time          `json:"pausedAt"`
	PausedBy          string             `json:"pausedBy,omitempty"`
	ValidActions      []string           `json:"validActions"`
	CanAutoRecover    bool               `json:"canAutoRecover"`
	TaskProgress      *TaskProgressInfo  `json:"taskProgress,omitempty"`
}

// RecoveryInfo represents brief information about a recoverable session.
type RecoveryInfo struct {
	SessionID   uuid.UUID `json:"sessionId"`
	Repository  string    `json:"repository"`
	IssueNumber int       `json:"issueNumber"`
	Title       string    `json:"title"`
	Stage       string    `json:"stage"`
	PauseReason string    `json:"pauseReason"`
	PausedAt    time.Time `json:"pausedAt"`
}

// TaskProgressInfo represents task progress information for recovery status.
type TaskProgressInfo struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Pending   int `json:"pending"`
	InProgress int `json:"inProgress"`
}

// RecoverOnStartup performs automatic recovery for sessions paused due to service restart.
// This method should be called during application startup via Fx lifecycle hook.
//
// Recovery conditions:
// - Session status is Paused
// - PauseContext.Reason supports auto-recovery (ServiceRestart, RateLimited, ResourceExhausted)
// - Auto-resume time has passed (if specified)
func (s *RecoveryService) RecoverOnStartup(ctx context.Context) error {
	s.logger.Info("starting session recovery on startup")

	// Find all recoverable sessions
	sessions, err := s.findRecoverableSessions(ctx)
	if err != nil {
		s.logger.Error("failed to find recoverable sessions", zap.Error(err))
		return err
	}

	if len(sessions) == 0 {
		s.logger.Info("no recoverable sessions found")
		return nil
	}

	s.logger.Info("found recoverable sessions", zap.Int("count", len(sessions)))

	// Recover each session
	var recoveredCount, failedCount int
	for _, session := range sessions {
		if err := s.recoverSession(ctx, session); err != nil {
			s.logger.Warn("failed to recover session",
				zap.String("session_id", session.ID.String()),
				zap.Error(err),
			)
			failedCount++
		} else {
			recoveredCount++
		}
	}

	s.logger.Info("session recovery completed",
		zap.Int("recovered", recoveredCount),
		zap.Int("failed", failedCount),
	)

	return nil
}

// findRecoverableSessions finds all sessions that can be auto-recovered.
func (s *RecoveryService) findRecoverableSessions(ctx context.Context) ([]*aggregate.WorkSession, error) {
	// Find all paused sessions
	sessions, err := s.sessionRepo.FindByStatus(ctx, aggregate.SessionStatusPaused)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// Filter for auto-recoverable sessions
	recoverable := make([]*aggregate.WorkSession, 0)
	for _, session := range sessions {
		if s.canAutoRecoverSession(session) {
			recoverable = append(recoverable, session)
		}
	}

	return recoverable, nil
}

// canAutoRecoverSession checks if a session can be auto-recovered.
func (s *RecoveryService) canAutoRecoverSession(session *aggregate.WorkSession) bool {
	if session == nil {
		return false
	}

	// Must have pause context
	pauseContext := session.GetPauseContext()
	if pauseContext == nil {
		return false
	}

	// Check if reason supports auto-recovery
	if !pauseContext.Reason.CanAutoRecover() {
		return false
	}

	// Check if auto-resume time has passed
	return pauseContext.CanAutoResumeNow()
}

// recoverSession recovers a single session.
func (s *RecoveryService) recoverSession(ctx context.Context, session *aggregate.WorkSession) error {
	sessionID := session.ID.String()
	pauseReason := ""
	if pauseContext := session.GetPauseContext(); pauseContext != nil {
		pauseReason = string(pauseContext.Reason)
	}

	s.logger.Info("recovering session",
		zap.String("session_id", sessionID),
		zap.String("pause_reason", pauseReason),
	)

	// Step 1: Check and repair state consistency
	worktreePath := ""
	if session.Execution != nil {
		worktreePath = session.Execution.WorktreePath
	}

	report, err := s.consistencyService.CheckAndRepair(ctx, session, worktreePath)
	if err != nil {
		s.logger.Warn("consistency check failed during recovery",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		// Continue with recovery even if consistency check fails
	} else if report.HasIssues {
		s.logger.Info("repaired consistency issues during recovery",
			zap.String("session_id", sessionID),
			zap.Int("repaired_count", report.RepairedCount),
			zap.Int("failed_repairs", len(report.FailedRepairs)),
		)
		// Log details of failed repairs for debugging
		for _, failed := range report.FailedRepairs {
			s.logger.Debug("failed to repair issue",
				zap.String("session_id", sessionID),
				zap.String("issue_type", string(failed.Type)),
				zap.String("description", failed.Description),
			)
		}
	}

	// Step 2: Auto-resume the session using SessionControlService
	resumed, err := s.sessionControlService.AutoResumeSession(session)
	if err != nil {
		s.logger.Error("auto-resume failed during recovery",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		return litchierrors.Wrap(litchierrors.ErrValidationFailed, err)
	}

	if !resumed {
		s.logger.Warn("session does not meet auto-resume conditions",
			zap.String("session_id", sessionID),
		)
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session does not meet auto-resume conditions",
		)
	}

	// Step 3: Save the resumed session state
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		s.logger.Error("failed to save resumed session",
			zap.String("session_id", sessionID),
			zap.Error(err),
		)
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.logger.Info("session resumed successfully",
		zap.String("session_id", sessionID),
		zap.String("current_stage", session.CurrentStage.String()),
	)

	// Step 4: Continue task execution if in Execution stage
	if session.CurrentStage == valueobject.StageExecution {
		if err := s.continueTaskExecution(ctx, session); err != nil {
			s.logger.Warn("failed to continue task execution after recovery",
				zap.String("session_id", sessionID),
				zap.Error(err),
			)
			// Don't fail the recovery, just log the warning
		}
	}

	return nil
}

// HandleUserResumeCommand handles user-initiated recovery via @bot continue command.
// This method validates permissions, checks state consistency, and resumes the session.
//
// Parameters:
// - repository: the repository name (owner/repo format)
// - issueNumber: the GitHub issue number
// - actor: the GitHub username who triggered the command
// - action: the resume action (e.g., "admin_continue", "admin_skip", "admin_rollback")
//
// Returns the recovery status after the command is processed.
func (s *RecoveryService) HandleUserResumeCommand(
	ctx context.Context,
	repository string,
	issueNumber int,
	actor string,
	action string,
) (*RecoveryStatus, error) {
	s.logger.Info("handling user resume command",
		zap.String("repository", repository),
		zap.Int("issue_number", issueNumber),
		zap.String("actor", actor),
		zap.String("action", action),
	)

	// Step 1: Find session by GitHub issue
	session, err := s.sessionRepo.FindByGitHubIssue(ctx, repository, issueNumber)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("no session found for %s#%d", repository, issueNumber),
		)
	}

	// Step 2: Validate session is paused
	if !session.IsPaused() {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not paused",
		)
	}

	// Step 3: Validate permission
	if err := s.validateResumePermission(session, actor, action); err != nil {
		return nil, err
	}

	// Step 4: Check and repair state consistency
	worktreePath := ""
	if session.Execution != nil {
		worktreePath = session.Execution.WorktreePath
	}

	report, err := s.consistencyService.CheckAndRepair(ctx, session, worktreePath)
	if err != nil {
		s.logger.Warn("consistency check failed during user resume",
			zap.String("session_id", session.ID.String()),
			zap.Error(err),
		)
	} else if report.HasIssues {
		s.logger.Info("repaired consistency issues during user resume",
			zap.String("session_id", session.ID.String()),
			zap.Int("repaired_count", report.RepairedCount),
		)
	}

	// Step 5: Resume the session with the specified action
	if err := s.sessionControlService.ResumeSession(session, action); err != nil {
		return nil, err
	}

	// Step 6: Save the resumed session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.logger.Info("session resumed by user command",
		zap.String("session_id", session.ID.String()),
		zap.String("actor", actor),
		zap.String("action", action),
	)

	// Step 7: Continue task execution if in Execution stage
	if session.CurrentStage == valueobject.StageExecution {
		if err := s.continueTaskExecution(ctx, session); err != nil {
			s.logger.Warn("failed to continue task execution after user resume",
				zap.String("session_id", session.ID.String()),
				zap.Error(err),
			)
		}
	}

	// Step 8: Build and return recovery status
	return s.buildRecoveryStatus(session), nil
}

// validateResumePermission validates if the actor has permission to resume with the given action.
// Permission rules based on docs/design/architecture.md section 12.1:
//   - Issue Author: Can trigger Agent, but limited to basic actions
//   - Repository Admin: Can perform all actions including admin_force
//
// Current implementation:
//   - Checks if action is valid for the pause reason
//   - Issue author can use most actions except admin_force (requires repo admin)
//   - admin_force action requires repository admin permission
//
// TODO: Integrate with AuthService.CheckRepoPermission when implementing HTTP API layer.
// This will require GitHub API calls to verify repository admin/maintain permission.
// For now, admin_force is blocked for non-admin users.
func (s *RecoveryService) validateResumePermission(
	session *aggregate.WorkSession,
	actor string,
	action string,
) error {
	// Check if actor is the Issue author
	isIssueAuthor := session.Issue != nil && session.Issue.Author == actor

	// Get valid actions for this pause reason
	validActions := s.sessionControlService.GetValidResumeActions(session)

	// Check if action is valid
	actionValid := false
	for _, validAction := range validActions {
		if action == validAction {
			actionValid = true
			break
		}
	}

	if !actionValid {
		return litchierrors.New(litchierrors.ErrPermissionDenied).WithDetail(
			fmt.Sprintf("action '%s' is not valid for this pause reason. Valid actions: %v", action, validActions),
		)
	}

	// For admin_force action, verify admin permission
	// According to architecture.md section 12.1, admin_force requires
	// repository admin or maintain permission via GitHub API
	if action == "admin_force" {
		// admin_force is a privileged action that requires repository admin permission.
		// Per architecture.md section 12.1, only users with admin/maintain permission
		// can execute this action. Issue authors cannot use admin_force.
		//
		// TODO: When HTTP API layer is implemented, integrate with AuthService:
		//   hasAdminPermission, err := s.authService.CheckRepoPermission(ctx, repo, actor)
		//   if err != nil { return err }
		//   if !hasAdminPermission { return ErrPermissionDenied }
		//
		// For now, we block admin_force for all users until AuthService is available.
		// This ensures security by default rather than allowing potential misuse.
		s.logger.Warn("admin_force action requires repository admin permission - AuthService integration pending",
			zap.String("session_id", session.ID.String()),
			zap.String("actor", actor),
			zap.Bool("is_issue_author", isIssueAuthor),
		)

		return litchierrors.New(litchierrors.ErrPermissionDenied).WithDetail(
			"admin_force action requires repository admin or maintain permission. " +
				"Please contact a repository administrator or use a different action.",
		)
	}

	return nil
}

// GetRecoveryStatus returns the recovery status for a specific session.
func (s *RecoveryService) GetRecoveryStatus(
	ctx context.Context,
	sessionID uuid.UUID,
) (*RecoveryStatus, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			"session not found: " + sessionID.String(),
		)
	}

	return s.buildRecoveryStatus(session), nil
}

// ListRecoverableSessions lists all sessions that can be recovered.
func (s *RecoveryService) ListRecoverableSessions(ctx context.Context) ([]RecoveryInfo, error) {
	// Find all paused sessions
	sessions, err := s.sessionRepo.FindByStatus(ctx, aggregate.SessionStatusPaused)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// Build recovery info list
	infos := make([]RecoveryInfo, 0, len(sessions))
	for _, session := range sessions {
		info := RecoveryInfo{
			SessionID:   session.ID,
			Repository:  session.Issue.Repository,
			IssueNumber: session.Issue.Number,
			Title:       session.Issue.Title,
			Stage:       session.CurrentStage.String(),
		}

		if pauseContext := session.GetPauseContext(); pauseContext != nil {
			info.PauseReason = string(pauseContext.Reason)
			info.PausedAt = pauseContext.PausedAt
		}

		infos = append(infos, info)
	}

	return infos, nil
}

// continueTaskExecution continues task execution after session recovery.
// This method handles task continuation asynchronously to avoid blocking the recovery process.
//
// Recovery behavior:
// 1. If a task was in progress during pause, it will be continued
// 2. If there's a failed task, it won't be auto-retried (user decides)
// 3. Otherwise, triggers next task execution in background
//
// Errors are logged and can be monitored via structured logs.
// Future enhancement: emit TaskExecutionFailedAfterRecovery events for alerting.
func (s *RecoveryService) continueTaskExecution(ctx context.Context, session *aggregate.WorkSession) error {
	if session.CurrentStage != valueobject.StageExecution {
		return nil
	}

	sessionID := session.ID.String()

	// Check if there's a task currently in progress
	if session.Execution != nil && session.Execution.CurrentTaskID != nil {
		currentTask := session.GetTask(*session.Execution.CurrentTaskID)
		if currentTask != nil && currentTask.IsInProgress() {
			s.logger.Info("task was in progress during pause, will continue",
				zap.String("session_id", sessionID),
				zap.String("task_id", currentTask.ID.String()),
				zap.String("task_description", currentTask.Description),
			)
		}
	}

	// Check for failed tasks that might need attention
	failedTask := session.GetFailedTask()
	if failedTask != nil {
		s.logger.Warn("session has failed task after recovery - manual intervention may be needed",
			zap.String("session_id", sessionID),
			zap.String("task_id", failedTask.TaskID),
			zap.String("reason", failedTask.Reason),
			zap.String("suggestion", failedTask.Suggestion),
		)
		// Don't auto-retry failed tasks, let user decide
		// The user can use "admin_continue" or "admin_skip" actions
		return nil
	}

	// Trigger next task execution in background
	// Using goroutine to avoid blocking the recovery process
	go func() {
		// Use context with timeout for async execution to prevent runaway tasks
		// Default timeout: 30 minutes (configurable via config.Task.Timeout in future)
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		taskID, result, err := s.taskService.ExecuteNextTask(bgCtx, session.ID)
		if err != nil {
			s.logger.Error("task execution failed after recovery",
				zap.String("session_id", sessionID),
				zap.Error(err),
				zap.String("monitoring_note", "Check if this error needs manual intervention"),
			)

			// Emit event for monitoring/alerting systems if dispatcher is available
			if s.eventDispatcher != nil {
				s.emitRecoveryEvent(bgCtx, session.ID, "task_execution_failed", err)
			}
			return
		}

		s.logger.Info("task execution triggered successfully after recovery",
			zap.String("session_id", sessionID),
			zap.String("task_id", taskID.String()),
			zap.Bool("success", result.Success),
			zap.Int("duration_ms", result.Duration),
		)
	}()

	return nil
}

// emitRecoveryEvent emits a recovery-related event for monitoring systems.
func (s *RecoveryService) emitRecoveryEvent(ctx context.Context, sessionID uuid.UUID, eventType string, err error) {
	s.logger.Info("emitting recovery event",
		zap.String("session_id", sessionID.String()),
		zap.String("event_type", eventType),
		zap.NamedError("original_error", err),
	)
	// Actual event emission will be implemented when event dispatcher interface is finalized
}

// buildRecoveryStatus builds RecoveryStatus from a WorkSession.
func (s *RecoveryService) buildRecoveryStatus(session *aggregate.WorkSession) *RecoveryStatus {
	status := &RecoveryStatus{
		SessionID:    session.ID,
		Repository:   session.Issue.Repository,
		IssueNumber:  session.Issue.Number,
		Title:        session.Issue.Title,
		CurrentStage: session.CurrentStage.String(),
		Status:       string(session.SessionStatus),
		ValidActions: s.sessionControlService.GetValidResumeActions(session),
	}

	// Fill pause context info
	if pauseContext := session.GetPauseContext(); pauseContext != nil {
		status.PauseReason = string(pauseContext.Reason)
		status.PauseReasonDetail = pauseContext.ErrorDetails
		status.PausedAt = pauseContext.PausedAt
		status.PausedBy = pauseContext.PausedBy
		status.CanAutoRecover = pauseContext.Reason.CanAutoRecover()
	}

	// Fill task progress if in Execution stage
	if session.CurrentStage == valueobject.StageExecution && len(session.Tasks) > 0 {
		var completed, failed, pending, inProgress int
		for _, task := range session.Tasks {
			switch task.Status {
			case valueobject.TaskStatusCompleted:
				completed++
			case valueobject.TaskStatusFailed:
				failed++
			case valueobject.TaskStatusPending:
				pending++
			case valueobject.TaskStatusInProgress:
				inProgress++
			}
		}

		status.TaskProgress = &TaskProgressInfo{
			Total:      len(session.Tasks),
			Completed:  completed,
			Failed:     failed,
			Pending:    pending,
			InProgress: inProgress,
		}
	}

	return status
}
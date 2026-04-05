// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"go.uber.org/zap"
)

// ConsistencyService implements the ConsistencyChecker domain service.
// It checks and repairs inconsistencies between database and cache,
// as well as internal state consistency within WorkSession.
type ConsistencyService struct {
	sessionRepo repository.WorkSessionRepository
	cacheRepo   repository.CacheRepository
	logger      *zap.Logger
	options     service.ConsistencyCheckOptions
}

// NewConsistencyService creates a new ConsistencyService.
func NewConsistencyService(
	sessionRepo repository.WorkSessionRepository,
	cacheRepo repository.CacheRepository,
	logger *zap.Logger,
) *ConsistencyService {
	return &ConsistencyService{
		sessionRepo: sessionRepo,
		cacheRepo:   cacheRepo,
		logger:      logger.Named("consistency_service"),
		options:     service.DefaultCheckOptions(),
	}
}

// WithOptions returns a new service with custom options.
func (s *ConsistencyService) WithOptions(options service.ConsistencyCheckOptions) *ConsistencyService {
	return &ConsistencyService{
		sessionRepo: s.sessionRepo,
		cacheRepo:   s.cacheRepo,
		logger:      s.logger,
		options:     options,
	}
}

// Check performs a consistency check on a WorkSession.
func (s *ConsistencyService) Check(
	ctx context.Context,
	session *aggregate.WorkSession,
	cacheWorktreePath string,
) (*service.ConsistencyReport, error) {
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail("session cannot be nil")
	}

	report := &service.ConsistencyReport{
		SessionID: session.ID,
		Issues:    []service.ConsistencyIssue{},
	}

	// 1. Check cache consistency
	if s.options.CheckCacheConsistency && cacheWorktreePath != "" {
		s.checkCacheConsistency(ctx, session, cacheWorktreePath, report)
	}

	// 2. Check stage vs status consistency
	if s.options.CheckStageStatus {
		s.checkStageStatusConsistency(session, report)
	}

	// 3. Check task progress consistency
	if s.options.CheckTaskProgress && session.CurrentStage == valueobject.StageExecution {
		s.checkTaskProgressConsistency(session, report)
	}

	// 4. Check pause context validity
	if s.options.CheckPauseContext {
		s.checkPauseContextValidity(session, report)
	}

	report.HasIssues = len(report.Issues) > 0
	return report, nil
}

// CheckAndRepair performs a consistency check and repairs fixable issues.
func (s *ConsistencyService) CheckAndRepair(
	ctx context.Context,
	session *aggregate.WorkSession,
	cacheWorktreePath string,
) (*service.ConsistencyReport, error) {
	report, err := s.Check(ctx, session, cacheWorktreePath)
	if err != nil {
		return nil, err
	}

	if !report.HasIssues {
		return report, nil
	}

	// Perform repairs
	actions, failedRepairs := s.Repair(ctx, session, report.Issues)
	report.RepairedCount = len(actions) - len(failedRepairs)
	report.FailedRepairs = failedRepairs

	// Log repair results
	for _, action := range actions {
		if action.Success {
			s.logger.Info("repaired consistency issue",
				zap.String("session_id", session.ID.String()),
				zap.String("issue_type", string(action.IssueType)),
				zap.String("action", action.Action),
			)
		} else {
			s.logger.Warn("failed to repair consistency issue",
				zap.String("session_id", session.ID.String()),
				zap.String("issue_type", string(action.IssueType)),
				zap.String("error", action.Error),
			)
		}
	}

	return report, nil
}

// Repair attempts to repair the given issues.
func (s *ConsistencyService) Repair(
	ctx context.Context,
	session *aggregate.WorkSession,
	issues []service.ConsistencyIssue,
) ([]service.RepairAction, []service.ConsistencyIssue) {
	actions := []service.RepairAction{}
	failedRepairs := []service.ConsistencyIssue{}

	for _, issue := range issues {
		if !issue.AutoRepair {
			actions = append(actions, service.RepairAction{
				IssueType: issue.Type,
				Action:    "skipped",
				Success:   false,
				Error:     "auto-repair not supported for this issue type",
			})
			failedRepairs = append(failedRepairs, issue)
			continue
		}

		action, success, errMsg := s.repairIssue(ctx, session, issue)
		actions = append(actions, service.RepairAction{
			IssueType: issue.Type,
			Action:    action,
			Success:   success,
			Error:     errMsg,
		})

		if !success {
			failedRepairs = append(failedRepairs, issue)
		}
	}

	return actions, failedRepairs
}

// checkCacheConsistency checks if database state matches cache.
func (s *ConsistencyService) checkCacheConsistency(
	ctx context.Context,
	session *aggregate.WorkSession,
	worktreePath string,
	report *service.ConsistencyReport,
) {
	cache, err := s.cacheRepo.Load(ctx, worktreePath)
	if err != nil {
		s.logger.Debug("cache load failed, will regenerate",
			zap.String("session_id", session.ID.String()),
			zap.Error(err),
		)
		// Cache file not existing is not an error, just needs regeneration
		return
	}

	if cache == nil {
		// No cache exists, not an inconsistency
		return
	}

	// Check session ID match
	if cache.SessionID != session.ID {
		report.Issues = append(report.Issues, service.ConsistencyIssue{
			Type:        service.IssueTypeCacheMismatch,
			Severity:    service.SeverityHigh,
			Description: "Cache session ID does not match database",
			FieldName:   "sessionId",
			Expected:    session.ID,
			Actual:      cache.SessionID,
			AutoRepair:  true,
		})
		return // Other checks are meaningless if session ID differs
	}

	// Check stage match
	if cache.CurrentStage != session.CurrentStage.String() {
		report.Issues = append(report.Issues, service.ConsistencyIssue{
			Type:        service.IssueTypeCacheMismatch,
			Severity:    service.SeverityMedium,
			Description: "Cache currentStage does not match database",
			FieldName:   "currentStage",
			Expected:    session.CurrentStage.String(),
			Actual:      cache.CurrentStage,
			AutoRepair:  true,
		})
	}

	// Check status match
	if cache.Status != string(session.SessionStatus) {
		report.Issues = append(report.Issues, service.ConsistencyIssue{
			Type:        service.IssueTypeCacheMismatch,
			Severity:    service.SeverityMedium,
			Description: "Cache status does not match database",
			FieldName:   "status",
			Expected:    string(session.SessionStatus),
			Actual:      cache.Status,
			AutoRepair:  true,
		})
	}
}

// checkStageStatusConsistency checks if stage and status are internally consistent.
func (s *ConsistencyService) checkStageStatusConsistency(
	session *aggregate.WorkSession,
	report *service.ConsistencyReport,
) {
	// Rule: Completed stage should have Completed status
	if session.CurrentStage == valueobject.StageCompleted &&
		session.SessionStatus != aggregate.SessionStatusCompleted {
		report.Issues = append(report.Issues, service.ConsistencyIssue{
			Type:        service.IssueTypeStatusMismatch,
			Severity:    service.SeverityHigh,
			Description: "Session in completed stage but status is not completed",
			FieldName:   "status",
			Expected:    aggregate.SessionStatusCompleted,
			Actual:      session.SessionStatus,
			AutoRepair:  true,
		})
	}

	// Rule: Active status should not have PauseContext
	if session.SessionStatus == aggregate.SessionStatusActive && session.PauseContext != nil {
		report.Issues = append(report.Issues, service.ConsistencyIssue{
			Type:        service.IssueTypePauseContextStale,
			Severity:    service.SeverityMedium,
			Description: "Session is active but has stale PauseContext",
			FieldName:   "pauseContext",
			Expected:    nil,
			Actual:      "non-nil",
			AutoRepair:  true,
		})
	}

	// Rule: Past clarification stage should have design
	if valueobject.StageOrder(session.CurrentStage) > valueobject.StageOrder(valueobject.StageClarification) {
		if session.Design == nil {
			report.Issues = append(report.Issues, service.ConsistencyIssue{
				Type:        service.IssueTypeDesignMissing,
				Severity:    service.SeverityHigh,
				Description: "Session past clarification stage but has no design",
				FieldName:   "design",
				Expected:    "non-nil",
				Actual:      nil,
				AutoRepair:  false, // Cannot auto-create design
			})
		}
	}
}

// checkTaskProgressConsistency checks if task status matches execution progress.
func (s *ConsistencyService) checkTaskProgressConsistency(
	session *aggregate.WorkSession,
	report *service.ConsistencyReport,
) {
	if session.Execution == nil {
		return
	}

	// Check for orphan execution (execution exists but no tasks)
	if len(session.Tasks) == 0 {
		report.Issues = append(report.Issues, service.ConsistencyIssue{
			Type:        service.IssueTypeExecutionOrphan,
			Severity:    service.SeverityHigh,
			Description: "Execution exists but no tasks are defined",
			FieldName:   "tasks",
			Expected:    "non-empty tasks list",
			Actual:      "empty",
			AutoRepair:  false, // Cannot auto-generate tasks
		})
		return // Other checks are meaningless if no tasks exist
	}

	// Check current task is in progress
	if session.Execution.CurrentTaskID != nil {
		currentTask := session.GetTask(*session.Execution.CurrentTaskID)
		if currentTask == nil {
			report.Issues = append(report.Issues, service.ConsistencyIssue{
				Type:        service.IssueTypeTaskProgress,
				Severity:    service.SeverityHigh,
				Description: "Current task ID references non-existent task",
				FieldName:   "execution.currentTaskId",
				Expected:    "valid task ID",
				Actual:      session.Execution.CurrentTaskID,
				AutoRepair:  true, // Can clear the current task ID
			})
		} else if currentTask.Status != valueobject.TaskStatusInProgress {
			report.Issues = append(report.Issues, service.ConsistencyIssue{
				Type:        service.IssueTypeTaskProgress,
				Severity:    service.SeverityMedium,
				Description: "Current task is not in progress status",
				FieldName:   "execution.currentTaskId",
				Expected:    valueobject.TaskStatusInProgress,
				Actual:      currentTask.Status,
				AutoRepair:  true,
			})
		}
	}

	// Check completed tasks match task statuses
	for _, completedID := range session.Execution.CompletedTasks {
		task := session.GetTask(completedID)
		if task == nil {
			report.Issues = append(report.Issues, service.ConsistencyIssue{
				Type:        service.IssueTypeTaskProgress,
				Severity:    service.SeverityMedium,
				Description: "Completed task ID references non-existent task",
				FieldName:   "execution.completedTaskIds",
				Expected:    "valid task ID",
				Actual:      completedID,
				AutoRepair:  true,
			})
		} else if task.Status != valueobject.TaskStatusCompleted && task.Status != valueobject.TaskStatusSkipped {
			report.Issues = append(report.Issues, service.ConsistencyIssue{
				Type:        service.IssueTypeTaskProgress,
				Severity:    service.SeverityMedium,
				Description: "Task in completed list but status is not completed/skipped",
				FieldName:   "execution.completedTaskIds",
				Expected:    valueobject.TaskStatusCompleted,
				Actual:      task.Status,
				AutoRepair:  false, // Would need to re-run task
			})
		}
	}
}

// checkPauseContextValidity checks if pause context is valid for current status.
func (s *ConsistencyService) checkPauseContextValidity(
	session *aggregate.WorkSession,
	report *service.ConsistencyReport,
) {
	// Rule: Paused status should have PauseContext
	if session.SessionStatus == aggregate.SessionStatusPaused && session.PauseContext == nil {
		report.Issues = append(report.Issues, service.ConsistencyIssue{
			Type:        service.IssueTypePauseContextStale,
			Severity:    service.SeverityMedium,
			Description: "Session is paused but has no PauseContext",
			FieldName:   "pauseContext",
			Expected:    "non-nil",
			Actual:      nil,
			AutoRepair:  false, // Cannot auto-generate pause context
		})
	}
}

// repairIssue attempts to repair a single issue.
func (s *ConsistencyService) repairIssue(
	ctx context.Context,
	session *aggregate.WorkSession,
	issue service.ConsistencyIssue,
) (action string, success bool, errMsg string) {
	switch issue.Type {
	case service.IssueTypeCacheMismatch:
		// Regenerate cache from database
		if s.cacheRepo != nil {
			worktreePath := ""
			if session.Execution != nil {
				worktreePath = session.Execution.WorktreePath
			}
			if worktreePath != "" {
				cache := buildCacheFromSession(session)
				if err := s.cacheRepo.Save(ctx, worktreePath, cache); err != nil {
					return "regenerate_cache", false, err.Error()
				}
				return "regenerate_cache", true, ""
			}
		}
		return "regenerate_cache", false, "no worktree path available"

	case service.IssueTypeStatusMismatch:
		// Fix status based on stage - use aggregate method to ensure invariants
		if session.CurrentStage == valueobject.StageCompleted {
			if err := session.Complete(); err != nil {
				return "set_status_completed", false, err.Error()
			}
			return "set_status_completed", true, ""
		}
		return "fix_status", false, "unsupported status fix"

	case service.IssueTypePauseContextStale:
		// Clear stale pause context using aggregate method
		if err := session.ClearStalePauseContext(); err != nil {
			return "clear_pause_context", false, err.Error()
		}
		return "clear_pause_context", true, ""

	case service.IssueTypeTaskProgress:
		// Clear invalid current task ID using aggregate method
		if issue.FieldName == "execution.currentTaskId" {
			if err := session.ClearCurrentTask(); err != nil {
				return "clear_current_task_id", false, err.Error()
			}
			return "clear_current_task_id", true, ""
		}
		return "fix_task_progress", false, "unsupported task progress fix"

	default:
		return "unknown", false, fmt.Sprintf("unsupported issue type: %s", issue.Type)
	}
}

// buildCacheFromSession creates a cache structure from a WorkSession.
func buildCacheFromSession(session *aggregate.WorkSession) *repository.ExecutionContextCache {
	cache := &repository.ExecutionContextCache{
		SessionID:    session.ID,
		CurrentStage: session.CurrentStage.String(),
		Status:       string(session.SessionStatus),
		Tasks:        []repository.TaskCache{},
		UpdatedAt:    time.Now(),
	}

	// Add clarification info
	if session.Clarification != nil {
		cache.Clarification = &repository.ClarificationCache{
			Status:           string(session.Clarification.Status),
			ConfirmedPoints:  session.Clarification.ConfirmedPoints,
			PendingQuestions: session.Clarification.PendingQuestions,
		}
	}

	// Add design info
	if session.Design != nil {
		var complexityScore *int
		if session.Design.ComplexityScore.Value() > 0 {
			score := session.Design.ComplexityScore.Value()
			complexityScore = &score
		}
		cache.Design = &repository.DesignCache{
			Status:              "approved",
			CurrentVersion:      session.Design.CurrentVersion,
			ComplexityScore:     complexityScore,
			RequireConfirmation: session.Design.RequireConfirmation,
			Confirmed:           session.Design.Confirmed,
		}
	}

	// Add execution info
	if session.Execution != nil {
		cache.Execution = &repository.ExecutionCache{
			CurrentTaskID:    session.Execution.CurrentTaskID,
			CompletedTaskIDs: session.Execution.CompletedTasks,
			FailedTaskID:     nil, // Will be set if failed task exists
			Branch:           session.Execution.Branch.Name,
			BranchDeprecated: session.Execution.Branch.IsDeprecated,
			WorktreePath:     session.Execution.WorktreePath,
		}
		if session.Execution.FailedTask != nil {
			if taskID, err := uuid.Parse(session.Execution.FailedTask.TaskID); err == nil {
				cache.Execution.FailedTaskID = &taskID
			}
		}
	}

	// Add tasks
	for _, task := range session.Tasks {
		cache.Tasks = append(cache.Tasks, repository.TaskCache{
			ID:         task.ID,
			Status:     task.Status.String(),
			RetryCount: task.RetryCount,
		})
	}

	return cache
}
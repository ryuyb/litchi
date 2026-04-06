// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/github"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/ryuyb/litchi/internal/pkg/utils"
	"go.uber.org/zap"
)

// TaskService handles the task breakdown and execution phase of WorkSession.
// It manages task generation, dependency resolution, execution scheduling, and monitoring.
//
// Core responsibilities:
// 1. Task scheduling - determine execution order based on dependencies
// 2. Dependency resolution - check if all dependencies are satisfied before execution
// 3. Execution monitoring - track task status, handle failures and retries
type TaskService struct {
	sessionRepo     repository.WorkSessionRepository
	auditRepo       repository.AuditLogRepository
	agentRunner     service.AgentRunner
	ghIssueService  *github.IssueService
	eventDispatcher *event.Dispatcher
	config          *config.Config
	logger          *zap.Logger
}

// NewTaskService creates a new TaskService.
func NewTaskService(
	sessionRepo repository.WorkSessionRepository,
	auditRepo repository.AuditLogRepository,
	agentRunner service.AgentRunner,
	ghIssueService *github.IssueService,
	eventDispatcher *event.Dispatcher,
	config *config.Config,
	logger *zap.Logger,
) *TaskService {
	return &TaskService{
		sessionRepo:     sessionRepo,
		auditRepo:       auditRepo,
		agentRunner:     agentRunner,
		ghIssueService:  ghIssueService,
		eventDispatcher: eventDispatcher,
		config:          config,
		logger:          logger.Named("task_service"),
	}
}

// StartTaskBreakdown starts the task breakdown process for a session.
// This method generates the task list from the design document using Agent.
//
// Steps:
// 1. Validate session is in TaskBreakdown stage
// 2. Prepare Agent context with design and clarification info
// 3. Execute Agent to generate task breakdown
// 4. Parse tasks with dependency relationships
// 5. Initialize execution entity
// 6. Transition to Execution stage
//
// Returns the generated task list.
func (s *TaskService) StartTaskBreakdown(
	ctx context.Context,
	sessionID uuid.UUID,
) ([]*entity.Task, error) {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate session is in TaskBreakdown stage
	if session.GetCurrentStage() != valueobject.StageTaskBreakdown {
		return nil, litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected TaskBreakdown", session.GetCurrentStage()),
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Validate design exists
	if session.Design == nil {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"design not initialized",
		)
	}

	// 5. Check if tasks already exist
	if len(session.Tasks) > 0 {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"tasks already generated",
		)
	}

	// 6. Prepare Agent request for task breakdown
	agentReq := &service.AgentRequest{
		SessionID: session.ID,
		Stage:     service.AgentStageTaskBreakdown,
		Prompt:    s.buildTaskBreakdownPrompt(session),
		Context: &service.AgentContext{
			IssueTitle:      session.Issue.Title,
			IssueBody:       session.Issue.Body,
			Repository:      session.Issue.Repository,
			ClarifiedPoints: session.Clarification.ConfirmedPoints,
			DesignContent:   session.Design.GetCurrentContent(),
		},
		Timeout: s.parseTimeout(s.config.Failure.Timeout.TaskBreakdown),
	}

	// 7. Execute Agent to generate tasks
	response, err := s.agentRunner.Execute(ctx, agentReq)
	if err != nil {
		s.logger.Error("failed to execute agent for task breakdown",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpTaskBreakdown, startTime, false, err.Error())
		return nil, litchierrors.Wrap(litchierrors.ErrAgentExecutionFail, err)
	}

	// 8. Parse tasks from response
	tasks, err := s.parseTaskBreakdown(response.Output)
	if err != nil {
		s.logger.Error("failed to parse task breakdown",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpTaskBreakdown, startTime, false, err.Error())
		return nil, litchierrors.New(litchierrors.ErrAgentExecutionFail).WithDetail(
			fmt.Sprintf("failed to parse tasks: %v", err),
		)
	}

	// 9. Validate parsed tasks
	if len(tasks) == 0 {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"no tasks generated from design",
		)
	}

	// 10. Set tasks to session
	session.SetTasks(tasks)

	// 11. Initialize execution entity (will be set properly when entering Execution stage)
	// Use placeholder values that will be updated by infrastructure layer
	execution := entity.NewExecution("", "")
	session.Execution = execution

	// 12. Transition to Execution stage
	if err := session.TransitionTo(valueobject.StageExecution); err != nil {
		return nil, err
	}

	// 13. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 14. Record audit log
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpTaskBreakdown, startTime, true, fmt.Sprintf("generated %d tasks", len(tasks)))

	// 15. Publish events
	s.publishEvents(ctx, session)

	// 16. Post task breakdown to GitHub issue
	if err := s.postTaskBreakdownToIssue(ctx, session, tasks); err != nil {
		s.logger.Warn("failed to post task breakdown to issue",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
	}

	s.logger.Info("task breakdown started",
		zap.String("session_id", sessionID.String()),
		zap.Int("task_count", len(tasks)),
	)

	return tasks, nil
}

// ExecuteNextTask executes the next executable task in the session.
// This method finds the next task with satisfied dependencies and executes it.
//
// Steps:
// 1. Validate session is in Execution stage
// 2. Find next executable task (dependency resolution)
// 3. Start the task
// 4. Execute Agent with task context
// 5. Handle execution result (complete/fail)
//
// Returns the execution result and task ID.
func (s *TaskService) ExecuteNextTask(
	ctx context.Context,
	sessionID uuid.UUID,
) (taskID uuid.UUID, result valueobject.ExecutionResult, err error) {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return uuid.Nil, valueobject.ExecutionResult{}, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return uuid.Nil, valueobject.ExecutionResult{}, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate session is in Execution stage
	if session.GetCurrentStage() != valueobject.StageExecution {
		return uuid.Nil, valueobject.ExecutionResult{}, litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected Execution", session.GetCurrentStage()),
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return uuid.Nil, valueobject.ExecutionResult{}, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Get max retry limit from config
	maxRetryLimit := s.config.Agent.TaskRetryLimit
	if maxRetryLimit <= 0 {
		maxRetryLimit = 3 // Default
	}

	// 5. Find next executable task
	nextTask := session.GetNextExecutableTask(maxRetryLimit)
	if nextTask == nil {
		// Check if all tasks are completed
		if session.AreAllTasksCompleted() {
			return uuid.Nil, valueobject.ExecutionResult{}, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
				"all tasks are completed",
			)
		}
		// Check if there's a failed task blocking progress
		if session.HasFailedTask() {
			return uuid.Nil, valueobject.ExecutionResult{}, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
				"execution blocked by failed task, please retry or skip",
			)
		}
		// No executable task (dependencies not satisfied)
		return uuid.Nil, valueobject.ExecutionResult{}, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"no executable task available (dependency constraints)",
		)
	}

	taskID = nextTask.ID

	// 6. Start the task (marks as InProgress)
	if nextTask.IsFailed() {
		// Retry failed task
		if err := session.RetryTask(taskID, maxRetryLimit); err != nil {
			return uuid.Nil, valueobject.ExecutionResult{}, err
		}
	} else {
		// Start new task
		if err := session.StartTask(taskID); err != nil {
			return uuid.Nil, valueobject.ExecutionResult{}, err
		}
	}

	// 7. Build task execution context
	taskContexts := s.buildTaskContexts(session)

	// 8. Prepare Agent request for task execution
	agentReq := &service.AgentRequest{
		SessionID: session.ID,
		Stage:     service.AgentStageTaskExecution,
		Prompt:    s.buildTaskExecutionPrompt(session, nextTask),
		Context: &service.AgentContext{
			IssueTitle:      session.Issue.Title,
			IssueBody:       session.Issue.Body,
			Repository:      session.Issue.Repository,
			DesignContent:   session.Design.GetCurrentContent(),
			ClarifiedPoints: session.Clarification.ConfirmedPoints,
			Tasks:           taskContexts,
		},
		Timeout: s.parseTimeout(s.config.Failure.Timeout.TaskExecution),
	}

	// 9. Execute Agent
	response, err := s.agentRunner.Execute(ctx, agentReq)
	if err != nil {
		s.logger.Error("failed to execute agent for task",
			zap.String("session_id", sessionID.String()),
			zap.String("task_id", taskID.String()),
			zap.Error(err),
		)

		// Mark task as failed
		failReason := fmt.Sprintf("Agent execution failed: %v", err)
		if err := session.FailTask(taskID, failReason, "Check Agent logs for details"); err != nil {
			s.logger.Error("failed to mark task as failed",
				zap.String("task_id", taskID.String()),
				zap.Error(err),
			)
		}

		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpTaskExecute, startTime, false, failReason)
		s.publishEvents(ctx, session)

		if saveErr := s.sessionRepo.Update(ctx, session); saveErr != nil {
			s.logger.Error("failed to save session after task failure",
				zap.String("session_id", sessionID.String()),
				zap.Error(saveErr),
			)
		}

		return taskID, valueobject.ExecutionResult{}, litchierrors.Wrap(litchierrors.ErrAgentExecutionFail, err)
	}

	// 10. Process execution result
	result = valueobject.NewExecutionResult(response.Output, response.Success, int(response.Duration.Milliseconds()))

	// Parse test results if available
	if len(response.Result.TestsRun) > 0 {
		for _, tr := range response.Result.TestsRun {
			result.AddTestResult(tr.Name, tr.Status, tr.Message)
		}
	}

	// 11. Handle execution outcome
	if response.Success && !result.HasTestFailures() {
		// Task completed successfully
		if err := session.CompleteTask(taskID, result); err != nil {
			return taskID, result, err
		}

		s.logger.Info("task completed",
			zap.String("session_id", sessionID.String()),
			zap.String("task_id", taskID.String()),
			zap.Int("duration_ms", result.Duration),
		)
	} else {
		// Task failed
		failReason := "Task execution failed"
		if result.HasTestFailures() {
			failReason = "Tests failed"
		}
		if response.Error != nil {
			failReason = response.Error.Message
		}

		suggestion := s.extractSuggestion(response.Output)

		if err := session.FailTask(taskID, failReason, suggestion); err != nil {
			return taskID, result, err
		}

		s.logger.Warn("task failed",
			zap.String("session_id", sessionID.String()),
			zap.String("task_id", taskID.String()),
			zap.String("reason", failReason),
			zap.String("suggestion", suggestion),
		)
	}

	// 12. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return taskID, result, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 13. Record audit log
	auditResult := "success"
	if !response.Success {
		auditResult = "failed"
	}
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpTaskExecute, startTime, response.Success, auditResult)

	// 14. Publish events
	s.publishEvents(ctx, session)

	// 15. Check if all tasks completed (can proceed to PR)
	if session.AreAllTasksCompleted() {
		s.logger.Info("all tasks completed, ready for PR creation",
			zap.String("session_id", sessionID.String()),
		)
	}

	return taskID, result, nil
}

// GetNextExecutableTask returns the next task that can be executed.
// This is a query method that does not modify session state.
func (s *TaskService) GetNextExecutableTask(
	ctx context.Context,
	sessionID uuid.UUID,
) (*TaskInfo, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	maxRetryLimit := s.config.Agent.TaskRetryLimit
	if maxRetryLimit <= 0 {
		maxRetryLimit = 3
	}

	nextTask := session.GetNextExecutableTask(maxRetryLimit)
	if nextTask == nil {
		return nil, nil // No executable task
	}

	// Build dependency info
	dependencyIDs := make([]string, len(nextTask.Dependencies))
	for i, depID := range nextTask.Dependencies {
		dependencyIDs[i] = depID.String()
	}

	// Check if dependencies are satisfied
	dependenciesSatisfied := session.AreDependenciesSatisfiedForTask(nextTask)

	return &TaskInfo{
		ID:                  nextTask.ID,
		Description:         nextTask.Description,
		Status:              nextTask.Status.String(),
		Dependencies:        dependencyIDs,
		DependenciesSatisfied: dependenciesSatisfied,
		RetryCount:          nextTask.RetryCount,
		Order:               nextTask.Order,
	}, nil
}

// CompleteTask manually marks a task as completed.
// This is used for manual intervention or when Agent execution needs manual confirmation.
func (s *TaskService) CompleteTask(
	ctx context.Context,
	sessionID uuid.UUID,
	taskID uuid.UUID,
	result valueobject.ExecutionResult,
) error {
	startTime := time.Now()

	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	if err := session.CompleteTask(taskID, result); err != nil {
		return err
	}

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpTaskComplete, startTime, true, "")
	s.publishEvents(ctx, session)

	s.logger.Info("task manually completed",
		zap.String("session_id", sessionID.String()),
		zap.String("task_id", taskID.String()),
	)

	return nil
}

// FailTask manually marks a task as failed.
// This is used for manual intervention or when external factors cause task failure.
func (s *TaskService) FailTask(
	ctx context.Context,
	sessionID uuid.UUID,
	taskID uuid.UUID,
	reason string,
	suggestion string,
) error {
	startTime := time.Now()

	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	if err := session.FailTask(taskID, reason, suggestion); err != nil {
		return err
	}

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpTaskFail, startTime, false, reason)
	s.publishEvents(ctx, session)

	s.logger.Warn("task manually failed",
		zap.String("session_id", sessionID.String()),
		zap.String("task_id", taskID.String()),
		zap.String("reason", reason),
	)

	return nil
}

// RetryTask retries a failed task.
// This resets the task to InProgress status and allows re-execution.
//
// Returns error if:
// - Task is not in Failed status
// - Task has reached maximum retry limit
func (s *TaskService) RetryTask(
	ctx context.Context,
	sessionID uuid.UUID,
	taskID uuid.UUID,
) error {
	startTime := time.Now()

	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	maxRetryLimit := s.config.Agent.TaskRetryLimit
	if maxRetryLimit <= 0 {
		maxRetryLimit = 3
	}

	if err := session.RetryTask(taskID, maxRetryLimit); err != nil {
		return err
	}

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpTaskRetry, startTime, true, "")
	s.publishEvents(ctx, session)

	s.logger.Info("task retry initiated",
		zap.String("session_id", sessionID.String()),
		zap.String("task_id", taskID.String()),
	)

	return nil
}

// SkipTask skips a task and marks it as Skipped status.
// Skipped tasks are considered "completed" for dependency resolution purposes.
func (s *TaskService) SkipTask(
	ctx context.Context,
	sessionID uuid.UUID,
	taskID uuid.UUID,
	reason string,
) error {
	startTime := time.Now()

	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	if err := session.SkipTask(taskID, reason); err != nil {
		return err
	}

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpTaskSkip, startTime, true, reason)
	s.publishEvents(ctx, session)

	s.logger.Info("task skipped",
		zap.String("session_id", sessionID.String()),
		zap.String("task_id", taskID.String()),
		zap.String("reason", reason),
	)

	return nil
}

// GetTaskStatus returns the status of a specific task.
func (s *TaskService) GetTaskStatus(
	ctx context.Context,
	sessionID uuid.UUID,
	taskID uuid.UUID,
) (*TaskStatus, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	task := session.GetTask(taskID)
	if task == nil {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			fmt.Sprintf("task %s not found", taskID),
		)
	}

	// Build dependency status
	dependencyStatus := make([]DependencyStatus, len(task.Dependencies))
	completedSet := session.CompletedTaskIDSet()
	for i, depID := range task.Dependencies {
		depTask := session.GetTask(depID)
		status := "unknown"
		if depTask != nil {
			status = depTask.Status.String()
		}
		dependencyStatus[i] = DependencyStatus{
			ID:          depID,
			Status:      status,
			IsCompleted: completedSet[depID],
		}
	}

	return &TaskStatus{
		SessionID:         sessionID,
		TaskID:            taskID,
		Description:       task.Description,
		Status:            task.Status.String(),
		StatusDisplayName: task.Status.DisplayName(),
		RetryCount:        task.RetryCount,
		FailureReason:     task.FailureReason,
		Suggestion:        task.Suggestion,
		Order:             task.Order,
		Dependencies:      dependencyStatus,
		ExecutionResult:   task.ExecutionResult,
	}, nil
}

// GetTaskList returns the status of all tasks in a session with optional filtering and pagination.
func (s *TaskService) GetTaskList(
	ctx context.Context,
	sessionID uuid.UUID,
	page, pageSize int,
	statusFilter *valueobject.TaskStatus,
) (*TaskListStatus, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	tasks := session.GetTasks()

	maxRetryLimit := s.config.Agent.TaskRetryLimit
	if maxRetryLimit <= 0 {
		maxRetryLimit = 3
	}

	// Build task summaries
	allTaskSummaries := make([]TaskSummary, 0, len(tasks))
	var completedCount, inProgressCount, pendingCount, failedCount, skippedCount int

	for _, task := range tasks {
		summary := TaskSummary{
			ID:          task.ID,
			Description: task.Description,
			Status:      task.Status.String(),
			Order:       task.Order,
			RetryCount:  task.RetryCount,
		}

		// Check dependency satisfaction
		summary.DependenciesSatisfied = session.AreDependenciesSatisfiedForTask(task)

		// Check if can execute
		if task.IsPending() && summary.DependenciesSatisfied {
			summary.CanExecute = true
		} else if task.IsFailed() && task.CanRetry(maxRetryLimit) && summary.DependenciesSatisfied {
			summary.CanExecute = true
		}

		// Count by status (for overall statistics)
		switch task.Status {
		case valueobject.TaskStatusCompleted:
			completedCount++
		case valueobject.TaskStatusInProgress:
			inProgressCount++
		case valueobject.TaskStatusPending:
			pendingCount++
		case valueobject.TaskStatusFailed:
			failedCount++
		case valueobject.TaskStatusSkipped:
			skippedCount++
		}

		allTaskSummaries = append(allTaskSummaries, summary)
	}

	// Apply status filter if provided
	var filteredSummaries []TaskSummary
	if statusFilter != nil {
		filteredSummaries = make([]TaskSummary, 0)
		for _, summary := range allTaskSummaries {
			if summary.Status == statusFilter.String() {
				filteredSummaries = append(filteredSummaries, summary)
			}
		}
	} else {
		filteredSummaries = allTaskSummaries
	}

	// Calculate pagination
	totalItems := len(filteredSummaries)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	totalPages := totalItems / pageSize
	if totalItems%pageSize > 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	// Apply pagination
	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize
	if startIndex >= totalItems {
		startIndex = totalItems
	}
	if endIndex > totalItems {
		endIndex = totalItems
	}

	paginatedSummaries := filteredSummaries[startIndex:endIndex]

	return &TaskListStatus{
		SessionID:      sessionID,
		TotalTasks:     len(tasks),
		Completed:      completedCount,
		InProgress:     inProgressCount,
		Pending:        pendingCount,
		Failed:         failedCount,
		Skipped:        skippedCount,
		AllCompleted:   session.AreAllTasksCompleted(),
		HasFailedTask:  session.HasFailedTask(),
		CurrentTaskID:  session.GetExecution().CurrentTaskID,
		Tasks:          paginatedSummaries,
		MaxRetryLimit:  maxRetryLimit,
		TotalItems:     totalItems,
		Page:           page,
		PageSize:       pageSize,
		TotalPages:     totalPages,
	}, nil
}

// TaskInfo represents information about a single task.
type TaskInfo struct {
	ID                   uuid.UUID `json:"id"`
	Description          string    `json:"description"`
	Status               string    `json:"status"`
	Dependencies         []string  `json:"dependencies"`
	DependenciesSatisfied bool     `json:"dependenciesSatisfied"`
	RetryCount           int       `json:"retryCount"`
	Order                int       `json:"order"`
}

// TaskStatus represents the detailed status of a task.
type TaskStatus struct {
	SessionID         uuid.UUID               `json:"sessionId"`
	TaskID            uuid.UUID               `json:"taskId"`
	Description       string                  `json:"description"`
	Status            string                  `json:"status"`
	StatusDisplayName string                  `json:"statusDisplayName"`
	RetryCount        int                     `json:"retryCount"`
	FailureReason     string                  `json:"failureReason,omitempty"`
	Suggestion        string                  `json:"suggestion,omitempty"`
	Order             int                     `json:"order"`
	Dependencies      []DependencyStatus      `json:"dependencies"`
	ExecutionResult   valueobject.ExecutionResult `json:"executionResult,omitempty"`
}

// DependencyStatus represents the status of a task dependency.
type DependencyStatus struct {
	ID          uuid.UUID `json:"id"`
	Status      string    `json:"status"`
	IsCompleted bool      `json:"isCompleted"`
} // @name DependencyStatus

// TaskSummary represents a summary of a task in the list.
type TaskSummary struct {
	ID                 uuid.UUID `json:"id"`
	Description        string    `json:"description"`
	Status             string    `json:"status"`
	Order              int       `json:"order"`
	RetryCount         int       `json:"retryCount"`
	DependenciesSatisfied bool   `json:"dependenciesSatisfied"`
	CanExecute         bool      `json:"canExecute"`
}

// TaskListStatus represents the status of all tasks in a session.
type TaskListStatus struct {
	SessionID     uuid.UUID      `json:"sessionId"`
	TotalTasks    int            `json:"totalTasks"`    // Total tasks in session (unfiltered)
	Completed     int            `json:"completed"`     // Completed tasks count
	InProgress    int            `json:"inProgress"`    // In progress tasks count
	Pending       int            `json:"pending"`       // Pending tasks count
	Failed        int            `json:"failed"`        // Failed tasks count
	Skipped       int            `json:"skipped"`       // Skipped tasks count
	AllCompleted  bool           `json:"allCompleted"`
	HasFailedTask bool           `json:"hasFailedTask"`
	CurrentTaskID *uuid.UUID     `json:"currentTaskId,omitempty"`
	Tasks         []TaskSummary  `json:"tasks"`
	MaxRetryLimit int            `json:"maxRetryLimit"`
	// Pagination info (based on filtered results)
	TotalItems int `json:"totalItems"` // Total items after filter
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalPages int `json:"totalPages"`
}

// --- Internal Helper Methods ---

// buildTaskBreakdownPrompt builds the prompt for Agent to generate tasks.
func (s *TaskService) buildTaskBreakdownPrompt(session *aggregate.WorkSession) string {
	return fmt.Sprintf(`Please break down the following design into executable tasks.

Design Document:
%s

Issue Title: %s
Issue Body: %s

Requirements:
1. Break down the design into discrete, executable tasks
2. Each task should be a clear, self-contained unit of work
3. Identify dependencies between tasks (which tasks must complete before others)
4. Order tasks logically based on dependencies
5. Each task description should be specific and actionable

Task Output Format (JSON array):
[
  {
    "description": "Task description (clear, specific, actionable)",
    "dependencies": [0, 1],  // Indices of tasks this depends on (0-based, empty if no dependencies)
    "order": 0  // Execution order (for sorting, starts from 0)
  }
]

Example:
[
  {
    "description": "Create database schema for user entity",
    "dependencies": [],
    "order": 0
  },
  {
    "description": "Implement UserRepository interface",
    "dependencies": [0],
    "order": 1
  },
  {
    "description": "Add UserService with CRUD operations",
    "dependencies": [1],
    "order": 2
  }
]

Output: Provide the complete task breakdown as a JSON array.`,
		session.Design.GetCurrentContent(),
		session.Issue.Title,
		session.Issue.Body,
	)
}

// buildTaskExecutionPrompt builds the prompt for Agent to execute a task.
func (s *TaskService) buildTaskExecutionPrompt(session *aggregate.WorkSession, task *entity.Task) string {
	// Build completed tasks summary
	completedSummary := ""
	if len(session.Execution.CompletedTasks) > 0 {
		completedSummary = "Previously completed tasks:\n"
		for _, taskID := range session.Execution.CompletedTasks {
			completedTask := session.GetTask(taskID)
			if completedTask != nil {
				completedSummary += fmt.Sprintf("- %s\n", completedTask.Description)
			}
		}
	}

	return fmt.Sprintf(`Please execute the following task.

Task Description:
%s

Design Document:
%s

Repository: %s
Issue Title: %s

%s

Requirements:
1. Implement the task according to the design
2. Follow existing code patterns and conventions
3. Write clean, maintainable code
4. Add appropriate comments for complex logic
5. Ensure the implementation is complete and functional

Execution Steps:
1. Read relevant existing code files to understand context
2. Implement the required changes
3. Verify the implementation works correctly
4. Run any existing tests to ensure no regressions

Output: After completing the task, provide a brief summary of what was implemented.`,
		task.Description,
		session.Design.GetCurrentContent(),
		session.Issue.Repository,
		session.Issue.Title,
		completedSummary,
	)
}

// buildTaskContexts builds task context information for Agent.
func (s *TaskService) buildTaskContexts(session *aggregate.WorkSession) []service.TaskContext {
	tasks := session.GetTasks()
	contexts := make([]service.TaskContext, len(tasks))

	for i, task := range tasks {
		contexts[i] = service.TaskContext{
			ID:           task.ID,
			Description:  task.Description,
			Status:       task.Status.String(),
			Dependencies: task.Dependencies,
		}
	}

	return contexts
}

// parseTaskBreakdown parses tasks from Agent response.
func (s *TaskService) parseTaskBreakdown(output string) ([]*entity.Task, error) {
	if output == "" {
		return nil, fmt.Errorf("empty output")
	}

	// Find JSON array in response
	jsonStart := strings.Index(output, "[")
	jsonEnd := strings.LastIndex(output, "]")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	jsonStr := output[jsonStart : jsonEnd+1]

	// Parse into intermediate struct
	var taskDefs []struct {
		Description  string `json:"description"`
		Dependencies []int  `json:"dependencies"`
		Order        int    `json:"order"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &taskDefs); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(taskDefs) == 0 {
		return nil, fmt.Errorf("no tasks defined")
	}

	// Convert to entity.Task with proper dependency IDs
	// First create all tasks to get their IDs
	tasks := make([]*entity.Task, len(taskDefs))
	taskIDs := make([]uuid.UUID, len(taskDefs))

	for i, def := range taskDefs {
		task := entity.NewTask(def.Description, nil, def.Order)
		tasks[i] = task
		taskIDs[i] = task.ID
	}

	// Now resolve dependencies using task IDs
	for i, def := range taskDefs {
		if len(def.Dependencies) > 0 {
			depIDs := make([]uuid.UUID, len(def.Dependencies))
			for j, depIndex := range def.Dependencies {
				if depIndex < 0 || depIndex >= len(taskIDs) {
					return nil, fmt.Errorf("invalid dependency index %d for task %d", depIndex, i)
				}
				depIDs[j] = taskIDs[depIndex]
			}
			tasks[i].Dependencies = depIDs
		}
	}

	return tasks, nil
}

// extractSuggestion extracts a suggestion from Agent output.
func (s *TaskService) extractSuggestion(output string) string {
	// Look for common suggestion patterns
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Suggestion:") ||
			strings.HasPrefix(line, "Fix:") ||
			strings.HasPrefix(line, "Next step:") {
			return line
		}
	}

	// Return first non-empty line as fallback
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 && len(line) < 200 {
			return line
		}
	}

	return "Check Agent logs for details"
}

// postTaskBreakdownToIssue posts the task breakdown to GitHub issue.
func (s *TaskService) postTaskBreakdownToIssue(
	ctx context.Context,
	session *aggregate.WorkSession,
	tasks []*entity.Task,
) error {
	commentBody := "## Task Breakdown\n\n"
	commentBody += fmt.Sprintf("Generated %d tasks:\n\n", len(tasks))

	for _, task := range tasks {
		status := "⏳ Pending"
		if len(task.Dependencies) > 0 {
			status += " (has dependencies)"
		}
		commentBody += fmt.Sprintf("%d. %s - %s\n", task.Order+1, task.Description, status)
	}

	commentBody += "\n---\n"
	commentBody += "*Tasks will be executed in dependency order. The system handles scheduling and retries automatically.*"

	owner := utils.ExtractOwner(session.Issue.Repository)
	repo := utils.ExtractRepo(session.Issue.Repository)

	if _, err := s.ghIssueService.CreateComment(ctx, owner, repo, session.Issue.Number, commentBody); err != nil {
		return err
	}

	return nil
}

// recordAuditLog records an audit log entry.
func (s *TaskService) recordAuditLog(
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
		"task",
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

// publishEvents publishes domain events from the session.
func (s *TaskService) publishEvents(ctx context.Context, session *aggregate.WorkSession) {
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

// parseTimeout parses timeout string to Duration.
func (s *TaskService) parseTimeout(timeoutStr string) time.Duration {
	if timeoutStr == "" {
		return 10 * time.Minute // Default
	}
	d, err := time.ParseDuration(timeoutStr)
	if err != nil {
		s.logger.Warn("failed to parse timeout, using default",
			zap.String("timeout", timeoutStr),
			zap.Error(err))
		return 10 * time.Minute
	}
	return d
}


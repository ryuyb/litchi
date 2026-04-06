// Package task provides HTTP handlers for task management API endpoints.
package task

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/dto"
	"github.com/ryuyb/litchi/internal/application/service"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"go.uber.org/zap"
)

// Handler handles task management HTTP requests.
type Handler struct {
	taskService *service.TaskService
	logger      *zap.Logger
}

// HandlerParams contains dependencies for creating a task handler.
// Fx will automatically inject TaskService and Logger.
type HandlerParams struct {
	fx.In

	TaskService *service.TaskService
	Logger      *zap.Logger
}

// NewHandler creates a new task handler.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		taskService: p.TaskService,
		logger:      p.Logger.Named("task-handler"),
	}
}

// GetTaskList returns the list of tasks for a session with statistics.
// @Summary        Get task list
// @Description    Retrieve all tasks for a work session with status statistics and execution progress
// @Tags           tasks
// @Accept         json
// @Produce        json
// @Param          sessionId  path      string  true  "Session ID (UUID format)"
// @Param          page       query     int     false "Page number (default: 1)"           minimum(1)
// @Param          pageSize   query     int     false "Page size (default: 20, max: 100)"  minimum(1) maximum(100)
// @Param          status     query     string  false "Filter by status (pending, in_progress, completed, failed, skipped)"
// @Success        200        {object}  dto.TaskListResponse  "Task list retrieved successfully"
// @Failure        400        {object}  dto.ErrorResponse     "Invalid session ID format"
// @Failure        404        {object}  dto.ErrorResponse     "Session not found"
// @Failure        500        {object}  dto.ErrorResponse     "Internal server error"
// @Router         /api/v1/sessions/{sessionId}/tasks [get]
func (h *Handler) GetTaskList(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID from path
	sessionIDStr := c.Params("sessionId")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Warn("invalid session ID format",
			zap.String("session_id", sessionIDStr),
			zap.Error(err),
		)
		return litchierrors.New(litchierrors.ErrBadRequest).WithDetail(
			"invalid session ID format: must be a valid UUID",
		)
	}

	// Parse pagination query parameters
	page := dto.ParseQueryInt(c, "page", 1)
	pageSize := dto.ParseQueryInt(c, "pageSize", 20)

	// Normalize pagination parameters
	page, pageSize = dto.NormalizePagination(page, pageSize, dto.DefaultPageSize)

	// Parse status filter (optional)
	var statusFilter *valueobject.TaskStatus
	statusFilterStr := c.Query("status")
	if statusFilterStr != "" {
		status, err := valueobject.ParseTaskStatus(statusFilterStr)
		if err != nil {
			return litchierrors.New(litchierrors.ErrInvalidQueryParam).
				WithDetail("invalid status filter: " + statusFilterStr)
		}
		statusFilter = &status
	}

	// Get task list from service with pagination and filtering
	taskList, err := h.taskService.GetTaskList(ctx, sessionID, page, pageSize, statusFilter)
	if err != nil {
		h.logger.Error("failed to get task list",
			zap.String("session_id", sessionIDStr),
			zap.Error(err),
		)
		return err
	}

	// Convert service.TaskSummary to dto.TaskDTO
	taskDTOs := make([]dto.TaskDTO, 0, len(taskList.Tasks))
	for _, task := range taskList.Tasks {
		taskDTOs = append(taskDTOs, dto.TaskDTO{
			ID:            task.ID,
			Description:   task.Description,
			Status:        task.Status,
			StatusDisplay: valueobject.TaskStatusDisplayName(valueobject.MustParseTaskStatus(task.Status)),
			Order:         task.Order,
			RetryCount:    task.RetryCount,
			CanExecute:    task.CanExecute,
		})
	}

	// Build response with pagination info
	response := dto.TaskListResponse{
		SessionID:     sessionID,
		TotalTasks:    taskList.TotalTasks,
		Completed:     taskList.Completed,
		InProgress:    taskList.InProgress,
		Pending:       taskList.Pending,
		Failed:        taskList.Failed,
		Skipped:       taskList.Skipped,
		AllCompleted:  taskList.AllCompleted,
		HasFailedTask: taskList.HasFailedTask,
		MaxRetryLimit: taskList.MaxRetryLimit,
		Tasks:         taskDTOs,
		Page:          taskList.Page,
		PageSize:      taskList.PageSize,
		TotalItems:    taskList.TotalItems,
		TotalPages:    taskList.TotalPages,
	}

	h.logger.Info("task list retrieved",
		zap.String("session_id", sessionIDStr),
		zap.Int("total_tasks", taskList.TotalTasks),
		zap.Int("page", page),
		zap.Int("page_size", pageSize),
	)

	return c.JSON(response)
}

// GetTaskStatus returns detailed status of a specific task.
// @Summary        Get task status
// @Description    Retrieve detailed status information for a specific task including dependencies and execution result
// @Tags           tasks
// @Accept         json
// @Produce        json
// @Param          sessionId  path      string  true  "Session ID (UUID format)"
// @Param          taskId     path      string  true  "Task ID (UUID format)"
// @Success        200        {object}  TaskStatusResponse  "Task status retrieved successfully"
// @Failure        400        {object}  dto.ErrorResponse  "Invalid session ID or task ID format"
// @Failure        404        {object}  dto.ErrorResponse  "Session or task not found"
// @Failure        500        {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/sessions/{sessionId}/tasks/{taskId} [get]
func (h *Handler) GetTaskStatus(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID from path
	sessionIDStr := c.Params("sessionId")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Warn("invalid session ID format",
			zap.String("session_id", sessionIDStr),
			zap.Error(err),
		)
		return litchierrors.New(litchierrors.ErrBadRequest).WithDetail(
			"invalid session ID format: must be a valid UUID",
		)
	}

	// Parse task ID from path
	taskIDStr := c.Params("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		h.logger.Warn("invalid task ID format",
			zap.String("session_id", sessionIDStr),
			zap.String("task_id", taskIDStr),
			zap.Error(err),
		)
		return litchierrors.New(litchierrors.ErrBadRequest).WithDetail(
			"invalid task ID format: must be a valid UUID",
		)
	}

	// Get task status from service
	taskStatus, err := h.taskService.GetTaskStatus(ctx, sessionID, taskID)
	if err != nil {
		h.logger.Error("failed to get task status",
			zap.String("session_id", sessionIDStr),
			zap.String("task_id", taskIDStr),
			zap.Error(err),
		)
		return err
	}

	// Convert to response DTO
	response := TaskStatusResponse{
		SessionID:         taskStatus.SessionID,
		TaskID:            taskStatus.TaskID,
		Description:       taskStatus.Description,
		Status:            taskStatus.Status,
		StatusDisplayName: taskStatus.StatusDisplayName,
		RetryCount:        taskStatus.RetryCount,
		FailureReason:     taskStatus.FailureReason,
		Suggestion:        taskStatus.Suggestion,
		Order:             taskStatus.Order,
		Dependencies:      taskStatus.Dependencies,
		ExecutionResult:   taskStatus.ExecutionResult,
	}

	h.logger.Info("task status retrieved",
		zap.String("session_id", sessionIDStr),
		zap.String("task_id", taskIDStr),
		zap.String("status", taskStatus.Status),
	)

	return c.JSON(response)
}

// TaskStatusResponse represents detailed task status for API response.
type TaskStatusResponse struct {
	SessionID         uuid.UUID                `json:"sessionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	TaskID            uuid.UUID                `json:"taskId" example:"660e8400-e29b-41d4-a716-446655440001"`
	Description       string                   `json:"description" example:"Implement login logic"`
	Status            string                   `json:"status" example:"completed"`
	StatusDisplayName string                   `json:"statusDisplayName" example:"Completed"`
	RetryCount        int                      `json:"retryCount" example:"0"`
	FailureReason     string                   `json:"failureReason,omitempty" example:"Test failed"`
	Suggestion        string                   `json:"suggestion,omitempty" example:"Fix the test"`
	Order             int                      `json:"order" example:"1"`
	Dependencies      []service.DependencyStatus `json:"dependencies"`
	ExecutionResult   valueobject.ExecutionResult `json:"executionResult,omitempty"`
} // @name TaskStatus

// SkipTask marks a task as skipped with a reason.
// @Summary        Skip task
// @Description    Mark a task as skipped, removing it from execution queue. Skipped tasks are considered completed for dependency resolution.
// @Tags           tasks
// @Accept         json
// @Produce        json
// @Param          sessionId  path      string            true  "Session ID (UUID format)"
// @Param          taskId     path      string            true  "Task ID (UUID format)"
// @Param          body       body      dto.SkipTaskRequest  true  "Skip reason"
// @Success        200        {object}  dto.SuccessResponse  "Task skipped successfully"
// @Failure        400        {object}  dto.ErrorResponse   "Invalid request body, session ID, or task ID"
// @Failure        404        {object}  dto.ErrorResponse   "Session or task not found"
// @Failure        409        {object}  dto.ErrorResponse   "Task cannot be skipped (invalid status)"
// @Failure        500        {object}  dto.ErrorResponse   "Internal server error"
// @Router         /api/v1/sessions/{sessionId}/tasks/{taskId}/skip [post]
func (h *Handler) SkipTask(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID from path
	sessionIDStr := c.Params("sessionId")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Warn("invalid session ID format",
			zap.String("session_id", sessionIDStr),
			zap.Error(err),
		)
		return litchierrors.New(litchierrors.ErrBadRequest).WithDetail(
			"invalid session ID format: must be a valid UUID",
		)
	}

	// Parse task ID from path
	taskIDStr := c.Params("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		h.logger.Warn("invalid task ID format",
			zap.String("session_id", sessionIDStr),
			zap.String("task_id", taskIDStr),
			zap.Error(err),
		)
		return litchierrors.New(litchierrors.ErrBadRequest).WithDetail(
			"invalid task ID format: must be a valid UUID",
		)
	}

	// Parse request body
	var req dto.SkipTaskRequest
	if err := c.Bind().JSON(&req); err != nil {
		h.logger.Warn("failed to parse skip task request",
			zap.String("session_id", sessionIDStr),
			zap.String("task_id", taskIDStr),
			zap.Error(err),
		)
		return litchierrors.New(litchierrors.ErrBadRequest).WithDetail(
			"invalid request body: must be valid JSON with 'reason' field",
		)
	}

	// Validate request
	if err := dto.Validate(&req); err != nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("Validation failed: " + err.Error())
	}

	// Skip task via service
	if err := h.taskService.SkipTask(ctx, sessionID, taskID, req.Reason); err != nil {
		h.logger.Error("failed to skip task",
			zap.String("session_id", sessionIDStr),
			zap.String("task_id", taskIDStr),
			zap.String("reason", req.Reason),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("task skipped",
		zap.String("session_id", sessionIDStr),
		zap.String("task_id", taskIDStr),
		zap.String("reason", req.Reason),
	)

	return c.JSON(dto.SuccessResponse{
		Status:  "success",
		Message: "Task skipped successfully",
	})
}

// RetryTask retries a failed task.
// @Summary        Retry task
// @Description    Reset a failed task to in-progress status for re-execution. Task must be in failed status and not exceed retry limit.
// @Tags           tasks
// @Accept         json
// @Produce        json
// @Param          sessionId  path      string             true  "Session ID (UUID format)"
// @Param          taskId     path      string             true  "Task ID (UUID format)"
// @Param          body       body      dto.RetryTaskRequest  true  "Retry options"
// @Success        200        {object}  dto.SuccessResponse  "Task retry initiated successfully"
// @Failure        400        {object}  dto.ErrorResponse   "Invalid request body, session ID, or task ID"
// @Failure        404        {object}  dto.ErrorResponse   "Session or task not found"
// @Failure        409        {object}  dto.ErrorResponse   "Task cannot be retried (not failed or exceeded retry limit)"
// @Failure        500        {object}  dto.ErrorResponse   "Internal server error"
// @Router         /api/v1/sessions/{sessionId}/tasks/{taskId}/retry [post]
func (h *Handler) RetryTask(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID from path
	sessionIDStr := c.Params("sessionId")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Warn("invalid session ID format",
			zap.String("session_id", sessionIDStr),
			zap.Error(err),
		)
		return litchierrors.New(litchierrors.ErrBadRequest).WithDetail(
			"invalid session ID format: must be a valid UUID",
		)
	}

	// Parse task ID from path
	taskIDStr := c.Params("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		h.logger.Warn("invalid task ID format",
			zap.String("session_id", sessionIDStr),
			zap.String("task_id", taskIDStr),
			zap.Error(err),
		)
		return litchierrors.New(litchierrors.ErrBadRequest).WithDetail(
			"invalid task ID format: must be a valid UUID",
		)
	}

	// Parse request body (optional, can be empty)
	var req dto.RetryTaskRequest
	if c.Body() != nil && len(c.Body()) > 0 {
		if err := c.Bind().JSON(&req); err != nil {
			h.logger.Warn("failed to parse retry task request",
				zap.String("session_id", sessionIDStr),
				zap.String("task_id", taskIDStr),
				zap.Error(err),
			)
			return litchierrors.New(litchierrors.ErrBadRequest).WithDetail(
				"invalid request body: must be valid JSON",
			)
		}
	}

	// TODO(#issue): Implement force retry functionality for RetryTaskRequest.Force field.
	// When force=true, allow retry even if dependencies are not met.
	// Currently the service does not support this feature.

	// Retry task via service
	if err := h.taskService.RetryTask(ctx, sessionID, taskID); err != nil {
		h.logger.Error("failed to retry task",
			zap.String("session_id", sessionIDStr),
			zap.String("task_id", taskIDStr),
			zap.Bool("force", req.Force),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("task retry initiated",
		zap.String("session_id", sessionIDStr),
		zap.String("task_id", taskIDStr),
	)

	return c.JSON(dto.SuccessResponse{
		Status:  "success",
		Message: "Task retry initiated successfully",
	})
}
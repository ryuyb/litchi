// Package session provides HTTP handlers for session management API.
package session

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/ryuyb/litchi/internal/application/dto"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
)

// Handler handles session management HTTP requests.
// It provides CRUD operations and session control actions (pause, resume, rollback, terminate).
type Handler struct {
	sessionRepo           repository.WorkSessionRepository
	sessionControlService service.SessionControlService
	config                *config.Config
	logger                *zap.Logger
}

// HandlerParams contains dependencies for creating a session handler.
type HandlerParams struct {
	fx.In

	SessionRepo           repository.WorkSessionRepository
	SessionControlService service.SessionControlService
	Config                *config.Config
	Logger                *zap.Logger
}

// NewHandler creates a new session handler.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		sessionRepo:           p.SessionRepo,
		sessionControlService: p.SessionControlService,
		config:                p.Config,
		logger:                p.Logger.Named("session_handler"),
	}
}

// ListSessions lists all work sessions with pagination and filtering.
// @Summary        List work sessions
// @Description    Retrieves a paginated list of work sessions with optional filtering by status, stage, and repository
// @Tags           sessions
// @Produce        json
// @Param          page      query     int     false  "Page number (1-based)"           default(1)
// @Param          pageSize  query     int     false  "Number of items per page"        default(20)
// @Param          status    query     string  false  "Filter by session status"        Enums(active, paused, completed, terminated)
// @Param          stage     query     string  false  "Filter by current stage"         Enums(clarification, design, task_breakdown, execution, pull_request, completed)
// @Param          repo      query     string  false  "Filter by repository (owner/repo format)"
// @Success        200       {object}  dto.PaginatedResponse[dto.SessionResponse]  "Sessions retrieved successfully"
// @Failure        400       {object}  dto.ErrorResponse                          "Invalid query parameters"
// @Failure        500       {object}  dto.ErrorResponse                          "Internal server error"
// @Router         /api/v1/sessions [get]
func (h *Handler) ListSessions(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	req := dto.SessionListRequest{}
	if err := c.Bind().Query(&req); err != nil {
		h.logger.Warn("failed to bind query parameters", zap.Error(err))
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid query parameters: " + err.Error())
	}

	// Normalize pagination parameters
	req.Page, req.PageSize = dto.NormalizePagination(req.Page, req.PageSize, dto.DefaultPageSize)

	// Build filter
	filter, err := h.buildListFilter(&req)
	if err != nil {
		return err
	}

	// Query sessions
	sessions, pagination, err := h.sessionRepo.ListWithPagination(ctx,
		repository.PaginationParams{Page: req.Page, PageSize: req.PageSize},
		filter,
	)
	if err != nil {
		h.logger.Error("failed to list sessions", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// Convert to DTOs
	data := make([]dto.SessionResponse, len(sessions))
	for i, session := range sessions {
		data[i] = dto.ToSessionResponse(session)
	}

	return c.JSON(dto.PaginatedResponse[dto.SessionResponse]{
		Data: data,
		Pagination: dto.PaginationDTO{
			Page:       pagination.Page,
			PageSize:   pagination.PageSize,
			TotalItems: int64(pagination.TotalItems),
			TotalPages: pagination.TotalPages,
		},
	})
}

// GetSession retrieves a single work session by ID.
// @Summary        Get work session
// @Description    Retrieves a work session by its unique identifier
// @Tags           sessions
// @Produce        json
// @Param          id   path      string  true  "Session ID (UUID format)"
// @Success        200  {object}  dto.SessionResponse  "Session retrieved successfully"
// @Failure        400  {object}  dto.ErrorResponse    "Invalid session ID format"
// @Failure        404  {object}  dto.ErrorResponse    "Session not found"
// @Failure        500  {object}  dto.ErrorResponse    "Internal server error"
// @Router         /api/v1/sessions/{id} [get]
func (h *Handler) GetSession(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID
	sessionID, err := h.parseSessionID(c)
	if err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid session ID format: " + err.Error())
	}

	// Retrieve session
	session, err := h.getSession(ctx, sessionID, "get")
	if err != nil {
		return err
	}

	return c.JSON(dto.ToSessionResponse(session))
}

// GetSessionDetail retrieves a work session with full details (issue, design, tasks, execution).
// @Summary        Get session details
// @Description    Retrieves a work session with complete details including issue info, design content, task list, and execution context
// @Tags           sessions
// @Produce        json
// @Param          id   path      string  true  "Session ID (UUID format)"
// @Success        200  {object}  dto.SessionResponse  "Session details retrieved successfully"
// @Failure        400  {object}  dto.ErrorResponse    "Invalid session ID format"
// @Failure        404  {object}  dto.ErrorResponse    "Session not found"
// @Failure        500  {object}  dto.ErrorResponse    "Internal server error"
// @Router         /api/v1/sessions/{id}/detail [get]
func (h *Handler) GetSessionDetail(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID
	sessionID, err := h.parseSessionID(c)
	if err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid session ID format: " + err.Error())
	}

	// Retrieve session
	session, err := h.getSession(ctx, sessionID, "detail")
	if err != nil {
		return err
	}

	return c.JSON(dto.ToSessionDetailResponse(session))
}

// PauseSession pauses an active work session.
// @Summary        Pause work session
// @Description    Pauses an active work session with a specified reason. Only active sessions can be paused.
// @Tags           sessions
// @Accept         json
// @Produce        json
// @Param          id      path      string                true  "Session ID (UUID format)"
// @Param          body    body      dto.PauseSessionRequest  true  "Pause request with reason"
// @Success        200     {object}  dto.SessionResponse   "Session paused successfully"
// @Failure        400     {object}  dto.ErrorResponse     "Invalid request or session ID"
// @Failure        404     {object}  dto.ErrorResponse     "Session not found"
// @Failure        409     {object}  dto.ErrorResponse     "Session cannot be paused (not active)"
// @Failure        500     {object}  dto.ErrorResponse     "Internal server error"
// @Router         /api/v1/sessions/{id}/pause [post]
func (h *Handler) PauseSession(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID
	sessionID, err := h.parseSessionID(c)
	if err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid session ID format: " + err.Error())
	}

	// Parse request body
	req := dto.PauseSessionRequest{}
	if err := c.Bind().Body(&req); err != nil {
		return litchierrors.New(litchierrors.ErrInvalidRequestBody).
			WithDetail("Invalid request body: " + err.Error())
	}

	// Validate request
	if err := dto.Validate(&req); err != nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("Validation failed: " + err.Error())
	}

	// Retrieve session
	session, err := h.getSession(ctx, sessionID, "pause")
	if err != nil {
		return err
	}

	// Check if session can be paused
	if err := h.checkCanPause(session); err != nil {
		return err
	}

	// Create pause context
	pauseReason, err := valueobject.ParsePauseReason(req.Reason)
	if err != nil {
		// Use "other" as fallback for unknown reasons, log warning for audit
		h.logger.Warn("invalid pause reason, using 'other'",
			zap.String("reason", req.Reason),
			zap.Error(err),
		)
		pauseReason = valueobject.PauseReasonOther
	}
	pauseCtx := valueobject.NewPauseContext(pauseReason).WithErrorDetails(req.Reason)

	// Pause session
	if err := h.sessionControlService.PauseSession(session, pauseCtx); err != nil {
		h.logger.Error("failed to pause session", zap.Error(err), zap.String("session_id", sessionID.String()))
		return err
	}

	// Save session
	if err := h.updateSession(ctx, session, "paused"); err != nil {
		return err
	}

	h.logger.Info("session paused",
		zap.String("session_id", sessionID.String()),
		zap.String("reason", req.Reason),
	)

	return c.JSON(dto.ToSessionResponse(session))
}

// ResumeSession resumes a paused work session.
// @Summary        Resume work session
// @Description    Resumes a paused work session. Only paused sessions can be resumed.
// @Tags           sessions
// @Accept         json
// @Produce        json
// @Param          id      path      string                  true  "Session ID (UUID format)"
// @Param          body    body      dto.ResumeSessionRequest  true  "Resume request with optional action"
// @Success        200     {object}  dto.SessionResponse     "Session resumed successfully"
// @Failure        400     {object}  dto.ErrorResponse       "Invalid request or session ID"
// @Failure        404     {object}  dto.ErrorResponse       "Session not found"
// @Failure        409     {object}  dto.ErrorResponse       "Session cannot be resumed (not paused)"
// @Failure        500     {object}  dto.ErrorResponse       "Internal server error"
// @Router         /api/v1/sessions/{id}/resume [post]
func (h *Handler) ResumeSession(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID
	sessionID, err := h.parseSessionID(c)
	if err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid session ID format: " + err.Error())
	}

	// Parse request body
	req := dto.ResumeSessionRequest{}
	if err := c.Bind().Body(&req); err != nil {
		// Resume request body is optional, use default action if empty
		req.Action = "manual_resume"
	}

	// Default action
	if req.Action == "" {
		req.Action = "manual_resume"
	}

	// Retrieve session
	session, err := h.getSession(ctx, sessionID, "resume")
	if err != nil {
		return err
	}

	// Check if session can be resumed
	if err := h.checkCanResume(session); err != nil {
		return err
	}

	// Resume session with action
	if err := h.sessionControlService.ResumeSession(session, req.Action); err != nil {
		h.logger.Error("failed to resume session", zap.Error(err), zap.String("session_id", sessionID.String()))
		return err
	}

	// Save session
	if err := h.updateSession(ctx, session, "resumed"); err != nil {
		return err
	}

	h.logger.Info("session resumed",
		zap.String("session_id", sessionID.String()),
		zap.String("action", req.Action),
	)

	return c.JSON(dto.ToSessionResponse(session))
}

// RollbackSession rolls back a work session to a previous stage.
// @Summary        Rollback work session
// @Description    Rolls back a work session to a specified previous stage (clarification, design, or task_breakdown). Only active sessions can be rolled back.
// @Tags           sessions
// @Accept         json
// @Produce        json
// @Param          id      path      string                     true  "Session ID (UUID format)"
// @Param          body    body      dto.RollbackSessionRequest  true  "Rollback request with target stage and reason"
// @Success        200     {object}  dto.SessionResponse        "Session rolled back successfully"
// @Failure        400     {object}  dto.ErrorResponse          "Invalid request or session ID"
// @Failure        404     {object}  dto.ErrorResponse          "Session not found"
// @Failure        409     {object}  dto.ErrorResponse          "Session cannot be rolled back"
// @Failure        500     {object}  dto.ErrorResponse          "Internal server error"
// @Router         /api/v1/sessions/{id}/rollback [post]
func (h *Handler) RollbackSession(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID
	sessionID, err := h.parseSessionID(c)
	if err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid session ID format: " + err.Error())
	}

	// Parse request body
	req := dto.RollbackSessionRequest{}
	if err := c.Bind().Body(&req); err != nil {
		return litchierrors.New(litchierrors.ErrInvalidRequestBody).
			WithDetail("Invalid request body: " + err.Error())
	}

	// Validate request
	if err := dto.Validate(&req); err != nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("Validation failed: " + err.Error())
	}

	// Parse target stage
	targetStage, err := valueobject.Parse(req.TargetStage)
	if err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid target stage. Valid stages: clarification, design, task_breakdown")
	}

	// Retrieve session
	session, err := h.getSession(ctx, sessionID, "rollback")
	if err != nil {
		return err
	}

	// Check if session can rollback to target stage
	if !session.CanRollbackTo(targetStage) {
		return litchierrors.New(litchierrors.ErrInvalidStageTransition).
			WithDetail("Session cannot be rolled back to the specified stage. Current stage: " + session.CurrentStage.String() + ", Target stage: " + targetStage.String())
	}

	// Record from_stage before rollback for accurate logging
	fromStage := session.CurrentStage.String()

	// Perform rollback via service
	if err := h.sessionControlService.RollbackSession(session, targetStage, req.Reason); err != nil {
		h.logger.Error("failed to rollback session", zap.Error(err), zap.String("session_id", sessionID.String()))
		return err
	}

	// Save session
	if err := h.updateSession(ctx, session, "rolled back"); err != nil {
		return err
	}

	h.logger.Info("session rolled back",
		zap.String("session_id", sessionID.String()),
		zap.String("from_stage", fromStage),
		zap.String("to_stage", targetStage.String()),
		zap.String("reason", req.Reason),
	)

	return c.JSON(dto.ToSessionResponse(session))
}

// TerminateSession terminates a work session.
// @Summary        Terminate work session
// @Description    Terminates a work session permanently. Active or paused sessions can be terminated.
// @Tags           sessions
// @Accept         json
// @Produce        json
// @Param          id      path      string                       true  "Session ID (UUID format)"
// @Param          body    body      dto.TerminateSessionRequest  true  "Terminate request with reason"
// @Success        200     {object}  dto.SessionResponse         "Session terminated successfully"
// @Failure        400     {object}  dto.ErrorResponse           "Invalid request or session ID"
// @Failure        404     {object}  dto.ErrorResponse           "Session not found"
// @Failure        409     {object}  dto.ErrorResponse           "Session cannot be terminated"
// @Failure        500     {object}  dto.ErrorResponse           "Internal server error"
// @Router         /api/v1/sessions/{id}/terminate [post]
func (h *Handler) TerminateSession(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID
	sessionID, err := h.parseSessionID(c)
	if err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid session ID format: " + err.Error())
	}

	// Parse request body
	req := dto.TerminateSessionRequest{}
	if err := c.Bind().Body(&req); err != nil {
		return litchierrors.New(litchierrors.ErrInvalidRequestBody).
			WithDetail("Invalid request body: " + err.Error())
	}

	// Validate request
	if err := dto.Validate(&req); err != nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("Validation failed: " + err.Error())
	}

	// Retrieve session
	session, err := h.getSession(ctx, sessionID, "terminate")
	if err != nil {
		return err
	}

	// Check if session can be terminated
	if err := h.checkCanTerminate(session); err != nil {
		return err
	}

	// Terminate session
	if err := h.sessionControlService.TerminateSession(session, req.Reason); err != nil {
		h.logger.Error("failed to terminate session", zap.Error(err), zap.String("session_id", sessionID.String()))
		return err
	}

	// Save session
	if err := h.updateSession(ctx, session, "terminated"); err != nil {
		return err
	}

	h.logger.Info("session terminated",
		zap.String("session_id", sessionID.String()),
		zap.String("reason", req.Reason),
	)

	return c.JSON(dto.ToSessionResponse(session))
}

// RestartSession restarts a terminated session by creating a new session for the same issue.
// @Summary        Restart work session
// @Description    Creates a new work session for the same GitHub issue as a terminated session. The new session starts from the clarification stage.
// @Tags           sessions
// @Accept         json
// @Produce        json
// @Param          id   path      string  true  "Session ID (UUID format) - the terminated session to restart"
// @Success        201  {object}  dto.SessionResponse  "New session created successfully"
// @Failure        400  {object}  dto.ErrorResponse    "Invalid session ID format"
// @Failure        404  {object}  dto.ErrorResponse    "Session not found"
// @Failure        409  {object}  dto.ErrorResponse    "Session is not terminated"
// @Failure        500  {object}  dto.ErrorResponse    "Internal server error"
// @Router         /api/v1/sessions/{id}/restart [post]
func (h *Handler) RestartSession(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID
	sessionID, err := h.parseSessionID(c)
	if err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid session ID format: " + err.Error())
	}

	// Retrieve the terminated session
	session, err := h.getSession(ctx, sessionID, "restart")
	if err != nil {
		return err
	}

	// Check if session is terminated
	if !session.IsTerminated() {
		return litchierrors.New(litchierrors.ErrSessionActive).
			WithDetail("Only terminated sessions can be restarted. Current status: " + string(session.SessionStatus))
	}

	// Check if there's already an active session for this issue
	existingSession, err := h.sessionRepo.FindByGitHubIssue(ctx, session.Issue.Repository, session.Issue.Number)
	if err != nil {
		h.logger.Error("failed to check existing session", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	if existingSession != nil && !existingSession.SessionStatus.IsTerminal() {
		return litchierrors.New(litchierrors.ErrSessionAlreadyExists).
			WithDetail("Active session already exists for this issue. Session ID: " + existingSession.ID.String() + ", Status: " + string(existingSession.SessionStatus))
	}

	// Create new session from the same issue
	newSession, err := aggregate.NewWorkSession(session.Issue)
	if err != nil {
		h.logger.Error("failed to create new session", zap.Error(err))
		return err
	}

	// Save new session
	if err := h.sessionRepo.Create(ctx, newSession); err != nil {
		h.logger.Error("failed to save new session", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	h.logger.Info("session restarted",
		zap.String("old_session_id", sessionID.String()),
		zap.String("new_session_id", newSession.ID.String()),
	)

	return c.Status(201).JSON(dto.ToSessionResponse(newSession))
}

// --- Helper Methods ---

// parseSessionID parses and validates the session ID from the path parameter.
func (h *Handler) parseSessionID(c fiber.Ctx) (uuid.UUID, error) {
	idStr := c.Params("id")
	if idStr == "" {
		return uuid.Nil, litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("session ID is required")
	}

	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("invalid session ID format: " + idStr)
	}

	return sessionID, nil
}

// getSession retrieves a session by ID, returning an error if not found.
// This is a common helper used by multiple handler methods.
func (h *Handler) getSession(ctx context.Context, sessionID uuid.UUID, operation string) (*aggregate.WorkSession, error) {
	session, err := h.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		h.logger.Error("failed to get session for "+operation,
			zap.Error(err),
			zap.String("session_id", sessionID.String()),
		)
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).
			WithDetail("Session not found: " + sessionID.String())
	}

	return session, nil
}

// checkCanPause checks if the session can be paused, returning a specific error if not.
func (h *Handler) checkCanPause(session *aggregate.WorkSession) error {
	if !session.SessionStatus.CanPause() {
		switch session.SessionStatus {
		case aggregate.SessionStatusPaused:
			return litchierrors.New(litchierrors.ErrSessionPaused).
				WithDetail("Session is already paused")
		case aggregate.SessionStatusTerminated:
			return litchierrors.New(litchierrors.ErrSessionTerminated).
				WithDetail("Session is terminated")
		default:
			return litchierrors.New(litchierrors.ErrSessionActive).
				WithDetail("Session cannot be paused. Current status: " + string(session.SessionStatus))
		}
	}
	return nil
}

// checkCanResume checks if the session can be resumed, returning a specific error if not.
func (h *Handler) checkCanResume(session *aggregate.WorkSession) error {
	if !session.SessionStatus.CanResume() {
		return litchierrors.New(litchierrors.ErrSessionNotPaused).
			WithDetail("Only paused sessions can be resumed. Current status: " + string(session.SessionStatus))
	}
	return nil
}

// checkCanTerminate checks if the session can be terminated, returning a specific error if not.
func (h *Handler) checkCanTerminate(session *aggregate.WorkSession) error {
	if !session.SessionStatus.CanTerminate() {
		switch session.SessionStatus {
		case aggregate.SessionStatusTerminated:
			return litchierrors.New(litchierrors.ErrSessionTerminated).
				WithDetail("Session is already terminated")
		default:
			return litchierrors.New(litchierrors.ErrSessionActive).
				WithDetail("Only active or paused sessions can be terminated. Current status: " + string(session.SessionStatus))
		}
	}
	return nil
}

// updateSession saves the session and handles any database errors.
func (h *Handler) updateSession(ctx context.Context, session *aggregate.WorkSession, operation string) error {
	if err := h.sessionRepo.Update(ctx, session); err != nil {
		h.logger.Error("failed to save "+operation+" session",
			zap.Error(err),
			zap.String("session_id", session.ID.String()),
		)
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	return nil
}

// buildListFilter builds a WorkSessionFilter from the request parameters.
// Returns an error if invalid status or stage values are provided.
func (h *Handler) buildListFilter(req *dto.SessionListRequest) (*repository.WorkSessionFilter, error) {
	filter := &repository.WorkSessionFilter{}

	// Parse status filter
	if req.Status != "" {
		status := aggregate.SessionStatus(req.Status)
		if !status.IsValid() {
			return nil, litchierrors.New(litchierrors.ErrInvalidQueryParam).
				WithDetail("Invalid status value: " + req.Status + ". Valid values: active, paused, completed, terminated")
		}
		filter.Status = &status
	}

	// Parse stage filter
	if req.Stage != "" {
		stage, err := valueobject.Parse(req.Stage)
		if err != nil || !stage.IsValid() {
			validStages := "clarification, design, task_breakdown, execution, pull_request, completed"
			return nil, litchierrors.New(litchierrors.ErrInvalidQueryParam).
				WithDetail("Invalid stage value: " + req.Stage + ". Valid values: " + validStages)
		}
		filter.Stage = &stage
	}

	// Repository filter
	if req.Repo != "" {
		filter.Repository = &req.Repo
	}

	return filter, nil
}
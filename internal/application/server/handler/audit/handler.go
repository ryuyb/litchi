// Package audit provides HTTP handlers for audit log API endpoints.
package audit

import (
	"regexp"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/dto"
	"github.com/ryuyb/litchi/internal/application/service"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"go.uber.org/zap"
)

// repoPattern validates owner/repo format (e.g., "owner/repo")
var repoPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)

// Handler handles audit log HTTP requests.
type Handler struct {
	auditService *service.AuditService
	logger       *zap.Logger
}

// HandlerParams contains dependencies for creating an audit handler.
// Fx will automatically inject AuditService and Logger.
type HandlerParams struct {
	fx.In

	AuditService *service.AuditService
	Logger       *zap.Logger
}

// NewHandler creates a new audit handler.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		auditService: p.AuditService,
		logger:       p.Logger.Named("audit.handler"),
	}
}

// ListAuditLogs lists audit logs with filtering and pagination.
//
// @Summary        List audit logs
// @Description    Retrieve audit logs with filtering by session, repository, actor, operation, result, and time range. Supports pagination.
// @Tags           audit
// @Accept         json
// @Produce        json
// @Param          page        query     int     false  "Page number (default: 1)"                example(1)
// @Param          pageSize    query     int     false  "Page size (default: 50, max: 100)"       example(50)
// @Param          sessionId   query     string  false  "Filter by session ID"                     example("550e8400-e29b-41d4-a716-446655440000")
// @Param          repository  query     string  false  "Filter by repository (owner/repo)"        example("owner/repo")
// @Param          actor       query     string  false  "Filter by actor username"                 example("username")
// @Param          operation   query     string  false  "Filter by operation type"                 example("session_start")
// @Param          result      query     string  false  "Filter by result (success/failed/denied)" example("success")
// @Param          startTime   query     string  false  "Filter by start time (RFC3339)"           example("2024-01-01T00:00:00Z")
// @Param          endTime     query     string  false  "Filter by end time (RFC3339)"             example("2024-01-02T00:00:00Z")
// @Param          orderBy     query     string  false  "Order by field (default: timestamp desc)" example("timestamp desc")
// @Success        200         {object}  dto.PaginatedResponse[dto.AuditLogResponse]  "Audit logs retrieved successfully"
// @Failure        400         {object}  dto.ErrorResponse                            "Invalid query parameters"
// @Failure        500         {object}  dto.ErrorResponse                            "Internal server error"
// @Router         /api/v1/audit [get]
func (h *Handler) ListAuditLogs(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	req := dto.AuditLogListRequest{}
	if err := c.Bind().Query(&req); err != nil {
		h.logger.Warn("failed to bind query parameters", zap.Error(err))
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("failed to bind query parameters: " + err.Error())
	}

	// Normalize pagination parameters
	page, pageSize := dto.NormalizePagination(req.Page, req.PageSize, 50)

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build filter parameters
	filter := service.AuditLogFilterParams{}

	// Parse session ID if provided
	if req.SessionID != "" {
		sessionID, err := uuid.Parse(req.SessionID)
		if err != nil {
			h.logger.Warn("invalid session ID format", zap.String("session_id", req.SessionID))
			return litchierrors.New(litchierrors.ErrInvalidQueryParam).
				WithDetail("invalid session ID format: " + req.SessionID)
		}
		filter.SessionID = &sessionID
	}

	// Set repository filter if provided
	if req.Repository != "" {
		filter.Repository = req.Repository
	}

	// Set actor filter if provided
	if req.Actor != "" {
		filter.Actor = req.Actor
	}

	// Parse operation type if provided
	if req.Operation != "" {
		op := valueobject.OperationType(req.Operation)
		if !op.IsValid() {
			return litchierrors.New(litchierrors.ErrInvalidQueryParam).
				WithDetail("Invalid operation type: " + req.Operation)
		}
		filter.Operation = op
	}

	// Parse result if provided
	if req.Result != "" {
		result := valueobject.AuditResult(req.Result)
		if !result.IsValid() {
			return litchierrors.New(litchierrors.ErrInvalidQueryParam).
				WithDetail("Invalid result value: " + req.Result + ". Valid values: success, failed, denied")
		}
		filter.Result = result
	}

	// Parse start time if provided
	if req.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			h.logger.Warn("invalid start time format", zap.String("start_time", req.StartTime))
			return litchierrors.New(litchierrors.ErrInvalidQueryParam).
				WithDetail("invalid start time format (use RFC3339): " + req.StartTime)
		}
		filter.StartTime = &startTime
	}

	// Parse end time if provided
	if req.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			h.logger.Warn("invalid end time format", zap.String("end_time", req.EndTime))
			return litchierrors.New(litchierrors.ErrInvalidQueryParam).
				WithDetail("invalid end time format (use RFC3339): " + req.EndTime)
		}
		filter.EndTime = &endTime
	}

	// Validate filter parameters
	if err := h.auditService.ValidateFilterParams(filter); err != nil {
		h.logger.Warn("invalid filter parameters", zap.Error(err))
		return err
	}

	// Query audit logs
	logs, total, err := h.auditService.ListAuditLogs(ctx, filter, offset, pageSize, req.OrderBy)
	if err != nil {
		h.logger.Error("failed to list audit logs", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err).
			WithDetail("failed to retrieve audit logs")
	}

	// Convert to response DTO
	response := dto.ToAuditLogList(logs, page, pageSize, total)

	return c.JSON(response)
}

// GetAuditLog retrieves a single audit log by ID.
//
// @Summary        Get audit log
// @Description    Retrieve a single audit log entry by its unique identifier.
// @Tags           audit
// @Accept         json
// @Produce        json
// @Param          id   path      string  true  "Audit log ID (UUID format)"  example("550e8400-e29b-41d4-a716-446655440000")
// @Success        200  {object}  dto.AuditLogResponse  "Audit log retrieved successfully"
// @Failure        400  {object}  dto.ErrorResponse     "Invalid audit log ID format"
// @Failure        404  {object}  dto.ErrorResponse     "Audit log not found"
// @Failure        500  {object}  dto.ErrorResponse     "Internal server error"
// @Router         /api/v1/audit/{id} [get]
func (h *Handler) GetAuditLog(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse audit log ID from path
	idStr := c.Params("id")
	if idStr == "" {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("audit log ID is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn("invalid audit log ID format", zap.String("id", idStr))
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("invalid audit log ID format: " + idStr)
	}

	// Retrieve audit log
	log, err := h.auditService.GetAuditLog(ctx, id)
	if err != nil {
		h.logger.Error("failed to get audit log", zap.String("id", idStr), zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err).
			WithDetail("failed to retrieve audit log: " + idStr)
	}

	// Audit log not found
	if log == nil {
		return litchierrors.New(litchierrors.ErrAuditLogNotFound).
			WithDetail("audit log not found: " + idStr)
	}

	// Convert to response DTO
	response := dto.ToAuditLogResponse(log)

	return c.JSON(response)
}

// ListBySession lists audit logs for a specific session.
//
// @Summary        List audit logs by session
// @Description    Retrieve all audit logs associated with a specific work session. Supports pagination.
// @Tags           audit
// @Accept         json
// @Produce        json
// @Param          sessionId  path      string  true  "Session ID (UUID format)"  example("550e8400-e29b-41d4-a716-446655440000")
// @Param          page       query     int     false "Page number (default: 1)"   example(1)
// @Param          pageSize   query     int     false "Page size (default: 50)"    example(50)
// @Success        200        {object}  dto.PaginatedResponse[dto.AuditLogResponse]  "Audit logs retrieved successfully"
// @Failure        400        {object}  dto.ErrorResponse                            "Invalid session ID or query parameters"
// @Failure        500        {object}  dto.ErrorResponse                            "Internal server error"
// @Router         /api/v1/audit/sessions/{sessionId} [get]
func (h *Handler) ListBySession(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID from path
	sessionIDStr := c.Params("sessionId")
	if sessionIDStr == "" {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("session ID is required")
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Warn("invalid session ID format", zap.String("session_id", sessionIDStr))
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("invalid session ID format: " + sessionIDStr)
	}

	// Parse pagination query parameters and normalize
	page, pageSize := dto.NormalizePagination(
		dto.ParseQueryInt(c, "page", 1),
		dto.ParseQueryInt(c, "pageSize", 50),
		50,
	)

	// Calculate offset
	offset := (page - 1) * pageSize

	// Query audit logs by session
	logs, total, err := h.auditService.ListBySession(ctx, sessionID, offset, pageSize)
	if err != nil {
		h.logger.Error("failed to list audit logs by session",
			zap.String("session_id", sessionIDStr),
			zap.Error(err),
		)
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err).
			WithDetail("failed to retrieve audit logs for session: " + sessionIDStr)
	}

	// Convert to response DTO
	response := dto.ToAuditLogList(logs, page, pageSize, total)

	return c.JSON(response)
}

// GetSessionSummary retrieves audit summary for a session.
//
// @Summary        Get session audit summary
// @Description    Retrieve aggregated statistics of audit logs for a specific work session, including counts by operation type and result status.
// @Tags           audit
// @Accept         json
// @Produce        json
// @Param          sessionId  path      string  true  "Session ID (UUID format)"  example("550e8400-e29b-41d4-a716-446655440000")
// @Success        200        {object}  dto.AuditSummaryResponse  "Audit summary retrieved successfully"
// @Failure        400        {object}  dto.ErrorResponse         "Invalid session ID format"
// @Failure        500        {object}  dto.ErrorResponse         "Internal server error"
// @Router         /api/v1/audit/sessions/{sessionId}/summary [get]
func (h *Handler) GetSessionSummary(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse session ID from path
	sessionIDStr := c.Params("sessionId")
	if sessionIDStr == "" {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("session ID is required")
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Warn("invalid session ID format", zap.String("session_id", sessionIDStr))
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("invalid session ID format: " + sessionIDStr)
	}

	// Get session audit summary
	summary, err := h.auditService.GetSessionAuditSummary(ctx, sessionID)
	if err != nil {
		h.logger.Error("failed to get session audit summary",
			zap.String("session_id", sessionIDStr),
			zap.Error(err),
		)
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err).
			WithDetail("failed to retrieve audit summary for session: " + sessionIDStr)
	}

	// Convert to response DTO
	response := dto.AuditSummaryResponse{
		SessionID:         summary.SessionID,
		TotalCount:        summary.TotalCount,
		TotalDurationMs:   summary.TotalDurationMs,
		AverageDurationMs: summary.AverageDurationMs,
		ByResult:          convertResultMap(summary.ByResult),
		ByOperation:       convertOperationMap(summary.ByOperation),
		FirstTimestamp:    summary.FirstTimestamp,
		LastTimestamp:     summary.LastTimestamp,
	}

	return c.JSON(response)
}

// ListByRepository lists audit logs for a specific repository.
//
// @Summary        List audit logs by repository
// @Description    Retrieve all audit logs for a specific repository (owner/repo format). Supports pagination.
// @Tags           audit
// @Accept         json
// @Produce        json
// @Param          repository  path      string  true  "Repository name (owner/repo)"  example("owner/repo")
// @Param          page        query     int     false "Page number (default: 1)"     example(1)
// @Param          pageSize    query     int     false "Page size (default: 50)"      example(50)
// @Success        200         {object}  dto.PaginatedResponse[dto.AuditLogResponse]  "Audit logs retrieved successfully"
// @Failure        400         {object}  dto.ErrorResponse                            "Invalid repository name or query parameters"
// @Failure        500         {object}  dto.ErrorResponse                            "Internal server error"
// @Router         /api/v1/audit/repositories/{repository} [get]
func (h *Handler) ListByRepository(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse repository from path
	repository := c.Params("repository")
	if repository == "" {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("repository name is required")
	}

	// Validate repository format (owner/repo)
	if !repoPattern.MatchString(repository) {
		h.logger.Warn("invalid repository format", zap.String("repository", repository))
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("invalid repository format (use owner/repo): " + repository)
	}

	// Parse pagination query parameters and normalize
	page, pageSize := dto.NormalizePagination(
		dto.ParseQueryInt(c, "page", 1),
		dto.ParseQueryInt(c, "pageSize", 50),
		50,
	)

	// Calculate offset
	offset := (page - 1) * pageSize

	// Query audit logs by repository
	logs, total, err := h.auditService.ListByRepository(ctx, repository, offset, pageSize)
	if err != nil {
		h.logger.Error("failed to list audit logs by repository",
			zap.String("repository", repository),
			zap.Error(err),
		)
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err).
			WithDetail("failed to retrieve audit logs for repository: " + repository)
	}

	// Convert to response DTO
	response := dto.ToAuditLogList(logs, page, pageSize, total)

	return c.JSON(response)
}

// convertResultMap converts valueobject.AuditResult map to string map for DTO.
func convertResultMap(m map[valueobject.AuditResult]int) map[string]int {
	result := make(map[string]int)
	for k, v := range m {
		result[string(k)] = v
	}
	return result
}

// convertOperationMap converts valueobject.OperationType map to string map for DTO.
func convertOperationMap(m map[valueobject.OperationType]int) map[string]int {
	result := make(map[string]int)
	for k, v := range m {
		result[string(k)] = v
	}
	return result
}
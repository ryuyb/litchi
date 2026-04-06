// Package audit provides HTTP handlers for audit log API endpoints.
package audit

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers audit log routes on the Fiber app.
//
// Route structure:
//   - GET /api/v1/audit                              - ListAuditLogs (with filtering and pagination)
//   - GET /api/v1/audit/:id                          - GetAuditLog (single audit log by ID)
//   - GET /api/v1/audit/sessions/:sessionId          - ListBySession (audit logs for a session)
//   - GET /api/v1/audit/sessions/:sessionId/summary  - GetSessionSummary (aggregated statistics)
//   - GET /api/v1/audit/repositories/:repository     - ListByRepository (audit logs for a repo)
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// Audit log routes group
	auditGroup := app.Group("/api/v1/audit")

	// List audit logs with filtering
	auditGroup.Get("/", handler.ListAuditLogs)

	// Get single audit log by ID
	auditGroup.Get("/:id", handler.GetAuditLog)

	// Session-specific routes (must be before :id to avoid conflict)
	sessionGroup := auditGroup.Group("/sessions")
	sessionGroup.Get("/:sessionId", handler.ListBySession)
	sessionGroup.Get("/:sessionId/summary", handler.GetSessionSummary)

	// Repository-specific routes
	auditGroup.Get("/repositories/:repository", handler.ListByRepository)
}
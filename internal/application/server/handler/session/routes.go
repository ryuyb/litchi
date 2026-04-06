// Package session provides HTTP handlers for session management API.
package session

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers session management routes on the Fiber app.
// Routes are organized under /api/v1/sessions prefix.
//
// Route layout:
//   GET    /api/v1/sessions           - ListSessions (paginated list)
//   GET    /api/v1/sessions/:id       - GetSession (basic info)
//   GET    /api/v1/sessions/:id/detail - GetSessionDetail (full details)
//   POST   /api/v1/sessions/:id/pause - PauseSession
//   POST   /api/v1/sessions/:id/resume - ResumeSession
//   POST   /api/v1/sessions/:id/rollback - RollbackSession
//   POST   /api/v1/sessions/:id/terminate - TerminateSession
//   POST   /api/v1/sessions/:id/restart - RestartSession
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// Create route group for session management
	sessionGroup := app.Group("/api/v1/sessions")

	// CRUD operations
	sessionGroup.Get("/", handler.ListSessions)
	sessionGroup.Get("/:id", handler.GetSession)
	sessionGroup.Get("/:id/detail", handler.GetSessionDetail)

	// Session control operations
	sessionGroup.Post("/:id/pause", handler.PauseSession)
	sessionGroup.Post("/:id/resume", handler.ResumeSession)
	sessionGroup.Post("/:id/rollback", handler.RollbackSession)
	sessionGroup.Post("/:id/terminate", handler.TerminateSession)
	sessionGroup.Post("/:id/restart", handler.RestartSession)
}
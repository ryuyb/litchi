// Package session provides HTTP handlers for session management API.
package session

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers session management routes on the router.
//
// Route layout:
//   GET    /sessions           - ListSessions (paginated list)
//   GET    /sessions/:id       - GetSession (basic info)
//   GET    /sessions/:id/detail - GetSessionDetail (full details)
//   POST   /sessions/:id/pause - PauseSession
//   POST   /sessions/:id/resume - ResumeSession
//   POST   /sessions/:id/rollback - RollbackSession
//   POST   /sessions/:id/terminate - TerminateSession
//   POST   /sessions/:id/restart - RestartSession
func RegisterRoutes(router fiber.Router, handler *Handler) {
	// Create route group for session management
	sessionGroup := router.Group("/sessions")

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
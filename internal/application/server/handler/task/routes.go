// Package task provides HTTP handlers for task management API endpoints.
package task

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers task management routes on the router.
// Routes are registered under the /sessions/:sessionId/tasks prefix.
func RegisterRoutes(router fiber.Router, handler *Handler) {
	// Task management routes
	// GET  /sessions/:sessionId/tasks           - Get task list with pagination
	// GET  /sessions/:sessionId/tasks/:taskId   - Get task status
	// POST /sessions/:sessionId/tasks/:taskId/skip  - Skip task
	// POST /sessions/:sessionId/tasks/:taskId/retry - Retry task

	// Create router group for task endpoints
	tasks := router.Group("/sessions/:sessionId/tasks")

	// Task list endpoint (with pagination support)
	tasks.Get("/", handler.GetTaskList)

	// Single task endpoints
	tasks.Get("/:taskId", handler.GetTaskStatus)

	// Task control endpoints
	tasks.Post("/:taskId/skip", handler.SkipTask)
	tasks.Post("/:taskId/retry", handler.RetryTask)
}
// Package task provides HTTP handlers for task management API endpoints.
package task

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers task management routes on the Fiber app.
// Routes are registered under the /api/v1/sessions/:sessionId/tasks prefix.
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// Task management routes
	// GET  /api/v1/sessions/:sessionId/tasks           - Get task list with pagination
	// GET  /api/v1/sessions/:sessionId/tasks/:taskId   - Get task status
	// POST /api/v1/sessions/:sessionId/tasks/:taskId/skip  - Skip task
	// POST /api/v1/sessions/:sessionId/tasks/:taskId/retry - Retry task

	// Create router group for task endpoints
	tasks := app.Group("/api/v1/sessions/:sessionId/tasks")

	// Task list endpoint (with pagination support)
	tasks.Get("/", handler.GetTaskList)

	// Single task endpoints
	tasks.Get("/:taskId", handler.GetTaskStatus)

	// Task control endpoints
	tasks.Post("/:taskId/skip", handler.SkipTask)
	tasks.Post("/:taskId/retry", handler.RetryTask)
}
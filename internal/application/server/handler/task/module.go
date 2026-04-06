// Package task provides HTTP handlers for task management API endpoints.
package task

import (
	"go.uber.org/fx"

	"github.com/gofiber/fiber/v3"
)

// Module provides the task handler as an Fx module.
// It registers the handler as a provider and invokes route registration.
var Module = fx.Module("task-handler",
	// Provide the task handler
	fx.Provide(
		NewHandler,
	),

	// Register routes with the Fiber app
	fx.Invoke(func(app *fiber.App, handler *Handler) {
		RegisterRoutes(app, handler)
	}),
)
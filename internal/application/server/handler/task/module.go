// Package task provides HTTP handlers for task management API endpoints.
package task

import (
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/server/router"
)

// Module provides the task handler as an Fx module.
// It registers the handler as a provider and invokes route registration.
var Module = fx.Module("task-handler",
	// Provide the task handler
	fx.Provide(NewHandler),

	// Register routes with APIRouter
	fx.Invoke(func(apiRouter router.APIRouter, h *Handler) {
		RegisterRoutes(apiRouter, h)
	}),
)
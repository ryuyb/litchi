// Package session provides HTTP handlers for session management API.
package session

import (
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/server/router"
)

// Module provides the session handler as an Fx module.
// It registers the handler as a Provider and the routes as an Invoke.
var Module = fx.Module("session-handler",
	// Provide the session handler
	fx.Provide(NewHandler),

	// Invoke route registration with APIRouter
	fx.Invoke(func(apiRouter router.APIRouter, h *Handler) {
		RegisterRoutes(apiRouter, h)
	}),
)
// Package user provides HTTP handlers for user management API.
package user

import (
	"go.uber.org/fx"

	authmiddleware "github.com/ryuyb/litchi/internal/application/server/middleware/auth"
	"github.com/ryuyb/litchi/internal/application/server/router"
)

// Module provides the user handler as an Fx module.
var Module = fx.Module("user-handler",
	// Provide the user handler
	fx.Provide(NewHandler),

	// Invoke route registration with APIRouter
	fx.Invoke(func(apiRouter router.APIRouter, h *Handler, auth *authmiddleware.Middleware) {
		RegisterRoutes(apiRouter, h, auth)
	}),
)
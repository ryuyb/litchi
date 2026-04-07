// Package auth provides HTTP handlers for authentication API.
package auth

import (
	"go.uber.org/fx"

	authmiddleware "github.com/ryuyb/litchi/internal/application/server/middleware/auth"
	"github.com/ryuyb/litchi/internal/application/server/router"
)

// Module provides the auth handler as an Fx module.
var Module = fx.Module("auth-handler",
	// Provide the auth handler
	fx.Provide(NewHandler),

	// Invoke route registration with APIRouter
	fx.Invoke(func(apiRouter router.APIRouter, h *Handler, auth *authmiddleware.Middleware) {
		RegisterRoutes(apiRouter, h, auth)
	}),
)
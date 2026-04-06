// Package repository provides HTTP handlers for repository management API via Fx module.
package repository

import (
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/server/router"
)

// Module provides repository management handlers via Fx.
var Module = fx.Module("repository-handler",
	// Provider
	fx.Provide(NewHandler),

	// Invoke - register routes with APIRouter
	fx.Invoke(func(apiRouter router.APIRouter, h *Handler) {
		RegisterRoutes(apiRouter, h)
	}),
)
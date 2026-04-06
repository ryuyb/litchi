// Package health provides HTTP health check handlers via Fx module.
package health

import (
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/server/router"
)

// Module provides health check handlers via Fx.
var Module = fx.Module("health-handler",
	// Provider
	fx.Provide(NewHandler),

	// Invoke - register routes (needs both *fiber.App for root /health and APIRouter for /api/v1/health)
	fx.Invoke(func(app *fiber.App, apiRouter router.APIRouter, h *Handler) {
		RegisterRoutes(app, apiRouter, h)
	}),
)
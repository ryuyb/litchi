// Package health provides HTTP health check handlers.
package health

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers health check routes on the Fiber app.
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// Legacy health endpoint (root level)
	app.Get("/health", handler.BasicHealth)

	// API v1 health endpoints
	app.Get("/api/v1/health", handler.HealthCheck)
	app.Get("/api/v1/health/detail", handler.DetailedHealth)
}
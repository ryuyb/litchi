// Package health provides HTTP health check handlers.
package health

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers health check routes on the Fiber app and API router.
//
// Route structure:
//   - GET /health             - BasicHealth (legacy, root level)
//   - GET /api/v1/health      - HealthCheck
//   - GET /api/v1/health/detail - DetailedHealth
//
// Note: BasicHealth is registered at root level for backward compatibility
// and Kubernetes probes, while HealthCheck and DetailedHealth are under /api/v1.
func RegisterRoutes(app *fiber.App, router fiber.Router, handler *Handler) {
	// Legacy health endpoint (root level for backward compatibility)
	app.Get("/health", handler.BasicHealth)

	// API v1 health endpoints
	health := router.Group("/health")
	health.Get("/", handler.HealthCheck)
	health.Get("/detail", handler.DetailedHealth)
}
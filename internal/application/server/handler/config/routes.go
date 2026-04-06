package config

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers configuration routes on the Fiber app.
// Routes:
//   - GET  /api/v1/config   - Get current configuration
//   - PUT  /api/v1/config   - Update configuration
func RegisterRoutes(app *fiber.App, handler *Handler) {
	configGroup := app.Group("/api/v1/config")

	configGroup.Get("", handler.GetConfig)
	configGroup.Put("", handler.UpdateConfig)
}
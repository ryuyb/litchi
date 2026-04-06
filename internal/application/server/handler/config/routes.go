package config

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers configuration routes on the router.
//
// Routes:
//   - GET  /config   - Get current configuration
//   - PUT  /config   - Update configuration
func RegisterRoutes(router fiber.Router, handler *Handler) {
	configGroup := router.Group("/config")

	configGroup.Get("", handler.GetConfig)
	configGroup.Put("", handler.UpdateConfig)
}
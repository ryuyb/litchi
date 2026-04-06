// Package repository provides HTTP handlers for repository management API.
package repository

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers repository management routes on the Fiber app.
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// Repository CRUD routes
	app.Get("/api/v1/repositories", handler.ListRepositories)
	app.Get("/api/v1/repositories/:name", handler.GetRepository)
	app.Post("/api/v1/repositories", handler.CreateRepository)
	app.Put("/api/v1/repositories/:name", handler.UpdateRepository)
	app.Delete("/api/v1/repositories/:name", handler.DeleteRepository)

	// Repository operation routes
	app.Post("/api/v1/repositories/:name/enable", handler.EnableRepository)
	app.Post("/api/v1/repositories/:name/disable", handler.DisableRepository)
	app.Get("/api/v1/repositories/:name/effective-config", handler.GetEffectiveConfig)

	// Validation configuration routes
	app.Get("/api/v1/repositories/:name/validation-config", handler.GetValidationConfig)
	app.Put("/api/v1/repositories/:name/validation-config", handler.UpdateValidationConfig)

	// Project detection routes
	app.Get("/api/v1/repositories/:name/detection", handler.GetDetectionResult)
	app.Post("/api/v1/repositories/:name/detection", handler.RunDetection)
}
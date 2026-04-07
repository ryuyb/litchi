// Package repository provides HTTP handlers for repository management API.
package repository

import (
	"github.com/gofiber/fiber/v3"

	authmiddleware "github.com/ryuyb/litchi/internal/application/server/middleware/auth"
)

// RegisterRoutes registers repository management routes on the router.
//
// Route structure:
//   - GET    /repositories                   - ListRepositories
//   - POST   /repositories                   - CreateRepository
//   - GET    /repositories/:name             - GetRepository
//   - PUT    /repositories/:name             - UpdateRepository
//   - DELETE /repositories/:name             - DeleteRepository
//   - POST   /repositories/:name/enable      - EnableRepository
//   - POST   /repositories/:name/disable     - DisableRepository
//   - GET    /repositories/:name/effective-config - GetEffectiveConfig
//   - GET    /repositories/:name/validation-config  - GetValidationConfig
//   - PUT    /repositories/:name/validation-config  - UpdateValidationConfig
//   - GET    /repositories/:name/detection   - GetDetectionResult
//   - POST   /repositories/:name/detection   - RunDetection
//
// All routes require authentication and admin role.
func RegisterRoutes(router fiber.Router, handler *Handler, authMiddleware *authmiddleware.Middleware) {
	repos := router.Group("/repositories")

	// All repository routes require authentication and admin role
	repos.Use(authMiddleware.RequireAuth(), authMiddleware.RequireAdmin())

	// CRUD operations
	repos.Get("/", handler.ListRepositories)
	repos.Post("/", handler.CreateRepository)
	repos.Get("/:name", handler.GetRepository)
	repos.Put("/:name", handler.UpdateRepository)
	repos.Delete("/:name", handler.DeleteRepository)

	// Repository control operations
	repos.Post("/:name/enable", handler.EnableRepository)
	repos.Post("/:name/disable", handler.DisableRepository)
	repos.Get("/:name/effective-config", handler.GetEffectiveConfig)

	// Validation configuration
	repos.Get("/:name/validation-config", handler.GetValidationConfig)
	repos.Put("/:name/validation-config", handler.UpdateValidationConfig)

	// Project detection
	repos.Get("/:name/detection", handler.GetDetectionResult)
	repos.Post("/:name/detection", handler.RunDetection)
}
// Package user provides HTTP handlers for user management API.
package user

import (
	"github.com/gofiber/fiber/v3"
	authmiddleware "github.com/ryuyb/litchi/internal/application/server/middleware/auth"
)

// RegisterRoutes registers user management routes on the router.
//
// Route layout:
//   GET    /users         - ListUsers (paginated list, admin only)
//   POST   /users         - CreateUser (admin only)
//   PUT    /users/:id     - UpdateUser (admin only)
//   DELETE /users/:id     - DeleteUser (admin only)
func RegisterRoutes(router fiber.Router, handler *Handler, authMiddleware *authmiddleware.Middleware) {
	userGroup := router.Group("/users")

	// All user routes require authentication and admin role
	userGroup.Use(authMiddleware.RequireAuth(), authMiddleware.RequireAdmin())

	userGroup.Get("", handler.ListUsers)
	userGroup.Post("", handler.CreateUser)
	userGroup.Put("/:id", handler.UpdateUser)
	userGroup.Delete("/:id", handler.DeleteUser)
}
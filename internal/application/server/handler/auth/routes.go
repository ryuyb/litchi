// Package auth provides HTTP handlers for authentication API.
package auth

import (
	"github.com/gofiber/fiber/v3"
	authmiddleware "github.com/ryuyb/litchi/internal/application/server/middleware/auth"
)

// RegisterRoutes registers authentication routes on the router.
//
// Route layout:
//   POST   /auth/login   - Login
//   POST   /auth/logout  - Logout
//   GET    /auth/me      - Get current user
func RegisterRoutes(router fiber.Router, handler *Handler, authMiddleware *authmiddleware.Middleware) {
	authGroup := router.Group("/auth")

	// Public routes
	authGroup.Post("/login", handler.Login)

	// Protected routes
	authGroup.Post("/logout", authMiddleware.RequireAuth(), handler.Logout)
	authGroup.Get("/me", authMiddleware.RequireAuth(), handler.GetMe)
}
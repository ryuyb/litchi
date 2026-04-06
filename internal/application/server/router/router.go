// Package router provides the API router group for Fx dependency injection.
package router

import (
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

// APIRouter wraps the /api/v1 router group for dependency injection.
type APIRouter struct {
	fiber.Router
}

// NewAPIRouter creates the /api/v1 router group.
// All API handlers should use this router to register their routes.
func NewAPIRouter(app *fiber.App) APIRouter {
	return APIRouter{app.Group("/api/v1")}
}

// Module provides the /api/v1 router group as an Fx module.
var Module = fx.Module("api-router",
	fx.Provide(NewAPIRouter),
)
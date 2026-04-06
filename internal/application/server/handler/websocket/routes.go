// Package websocket provides WebSocket handlers for real-time progress updates.
package websocket

import (
	websocket "github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

// RouteConfig contains optional configuration for WebSocket routes.
type RouteConfig struct {
	// PreUpgradeMiddleware applied before WebSocket upgrade check.
	// Useful for authentication, authorization, rate limiting, etc.
	PreUpgradeMiddleware []fiber.Handler
}

// RegisterRoutes registers WebSocket routes to the Fiber app.
// Optional routeConfig can be provided to add middleware before the WebSocket upgrade.
func RegisterRoutes(app *fiber.App, handler *Handler, routeConfig ...RouteConfig) {
	// WebSocket route group
	ws := app.Group("/ws")

	// Apply optional pre-upgrade middleware
	if len(routeConfig) > 0 && len(routeConfig[0].PreUpgradeMiddleware) > 0 {
		for _, m := range routeConfig[0].PreUpgradeMiddleware {
			ws.Use(m)
		}
	}

	// Apply WebSocket upgrade middleware
	ws.Use(WebSocketUpgradeMiddleware)

	// Session progress WebSocket endpoint
	// Route: /ws/sessions/:id
	// Clients connect to receive real-time updates for a specific session
	ws.Get("/sessions/:id", websocket.New(handler.HandleSessionWebSocket))
}
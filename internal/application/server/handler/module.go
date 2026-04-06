// Package handler provides HTTP API handlers for the server.
package handler

import (
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/server/handler/audit"
	"github.com/ryuyb/litchi/internal/application/server/handler/config"
	"github.com/ryuyb/litchi/internal/application/server/handler/health"
	"github.com/ryuyb/litchi/internal/application/server/handler/repository"
	"github.com/ryuyb/litchi/internal/application/server/handler/session"
	"github.com/ryuyb/litchi/internal/application/server/handler/task"
	"github.com/ryuyb/litchi/internal/application/server/handler/websocket"
	"github.com/ryuyb/litchi/internal/application/server/middleware"
)

// Module provides all HTTP API handlers for Fx.
var Module = fx.Module("api-handlers",
	// Middleware (ErrorHandler must be created before App)
	fx.Options(middleware.Module),

	// Handler sub-modules (each handler as independent Fx Module)
	fx.Options(
		session.Module,
		task.Module,
		config.Module,
		repository.Module,
		audit.Module,
		health.Module,
		websocket.Module, // WebSocket handler for real-time updates
	),

	// Note: Error handler is now set in fiber.Config during App creation.
	// See internal/application/server/app.go - NewApp takes ErrorHandler as dependency.
)
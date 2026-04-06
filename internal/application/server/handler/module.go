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
	"github.com/ryuyb/litchi/internal/application/server/middleware"
)

// Module provides all REST API handlers for Fx.
var Module = fx.Module("api-handlers",
	// Middleware
	fx.Options(middleware.Module),

	// Handler sub-modules (each handler as independent Fx Module)
	fx.Options(
		session.Module,   // T6.1.1
		task.Module,      // T6.1.2
		config.Module,    // T6.1.3
		repository.Module, // T6.1.4
		audit.Module,     // T6.1.6
		health.Module,    // T6.1.7
	),

	// Register middleware (error handler)
	fx.Invoke(middleware.RegisterErrorHandler),
)
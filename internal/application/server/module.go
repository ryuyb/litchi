package server

import (
	"github.com/gofiber/fiber/v3"
	"github.com/ryuyb/litchi/internal/application/server/handler"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the HTTP server module for Fx.
var Module = fx.Module("server",
	// Fiber App Provider
	fx.Provide(NewApp),

	// Handler Module (all REST API handlers)
	fx.Options(handler.Module),

	// Lifecycle Hooks
	fx.Invoke(StartAppHook),
)

// StartAppHook registers the app start/stop lifecycle hook.
func StartAppHook(lifecycle fx.Lifecycle, app *fiber.App, cfg *config.Config, logger *zap.Logger) {
	lifecycle.Append(StartApp(app, cfg, logger))
}

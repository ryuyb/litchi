package server

import (
	"context"
	_ "embed"
	"fmt"
	"net"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/contrib/v3/swaggerui"
	"github.com/ryuyb/litchi/internal/application/server/middleware"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

//go:embed swagger.json
var swaggerSpec []byte

// Params for NewApp.
type Params struct {
	fx.In

	Logger      *zap.Logger
	Config      *config.Config
	ErrorHandler *middleware.ErrorHandler // Required for global error handling
}

// NewApp creates a new Fiber application.
func NewApp(p Params) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Litchi v" + p.Config.Server.Version,
		ErrorHandler: p.ErrorHandler.Handle, // Set global error handler
	})

	// Health check endpoint is now registered via health handler module
	// See: internal/application/server/handler/health/routes.go

	// Swagger UI - controlled by configuration
	if p.Config.Server.EnableSwaggerUI {
		app.Use(swaggerui.New(swaggerui.Config{
			BasePath:    "/",
			FileContent: swaggerSpec,
			Path:        "swagger",
			Title:       "Litchi API Documentation",
		}))
		p.Logger.Info("Swagger UI enabled at /swagger")
	}

	return app
}

// StartApp starts the Fiber server.
// Uses pre-bound listener to detect port binding errors immediately.
func StartApp(app *fiber.App, cfg *config.Config, logger *zap.Logger) fx.Hook {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	return fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("starting HTTP server", zap.String("addr", addr))

			// Pre-bind the address to detect port binding errors immediately
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return fmt.Errorf("failed to bind address %s: %w", addr, err)
			}

			// Start server with pre-bound listener in background
			go func() {
				if err := app.Listener(ln); err != nil {
					logger.Error("HTTP server error", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("stopping HTTP server")
			return app.ShutdownWithContext(ctx)
		},
	}
}

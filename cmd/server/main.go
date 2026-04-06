package main

import (
	"os"
	"strings"

	"github.com/ryuyb/litchi/internal/application/server"
	"github.com/ryuyb/litchi/internal/application/service"
	"github.com/ryuyb/litchi/internal/infrastructure"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/static"
	"github.com/ryuyb/litchi/internal/pkg/logger"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	_ "github.com/ryuyb/litchi/docs/api" // Import generated docs for Swagger embedding
)

// @title           Litchi API
// @version         0.1.0
// @description     Automated development agent system - from GitHub Issue to Pull Request

// @servers         [{"url": "http://localhost:8080/api/v1", "description": "Local development server"}]

// isDebugMode checks if the application is running in debug mode based on server.mode config.
func isDebugMode() bool {
	mode := strings.ToLower(config.GetServerMode())
	return mode == "" || mode == "debug" || mode == "dev" || mode == "development"
}

// fxLogger returns the appropriate Fx event logger based on debug mode.
func fxLogger() fxevent.Logger {
	if isDebugMode() {
		return &fxevent.ConsoleLogger{W: os.Stderr}
	}
	return fxevent.NopLogger
}

func main() {
	fx.New(
		fx.WithLogger(fxLogger),
		// Config must be loaded first (no dependencies)
		config.Module,
		// Logger depends on config
		logger.Module,
		// Infrastructure modules (database, repositories)
		infrastructure.Module,
		// Application services
		service.Module,
		// Server depends on logger and config
		// API routes are registered here
		server.Module,
		// Static file serving (embedded frontend)
		// Registers SPA fallback route after API routes
		// Fiber matches exact routes before wildcards, so API routes take priority
		static.Module,
	).Run()
}

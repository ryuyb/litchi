package main

import (
	"github.com/ryuyb/litchi/internal/application/server"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/pkg/logger"
	"go.uber.org/fx"

	_ "github.com/ryuyb/litchi/docs/api" // Import generated docs for Swagger embedding
)

// @title           Litchi API
// @version         0.1.0
// @description     Automated development agent system - from GitHub Issue to Pull Request

// @servers         [{"url": "http://localhost:8080/api/v1", "description": "Local development server"}]

func main() {
	fx.New(
		// Config must be loaded first (no dependencies)
		config.Module,
		// Logger depends on config
		logger.Module,
		// Server depends on logger and config
		// StartAppHook ensures fiber.App is instantiated and starts
		server.Module,
	).Run()
}

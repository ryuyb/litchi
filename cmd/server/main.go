package main

import (
	"github.com/ryuyb/litchi/internal/application/server"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/pkg/logger"
	"go.uber.org/fx"
)

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
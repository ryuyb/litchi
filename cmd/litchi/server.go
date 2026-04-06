package main

// @title           Litchi API
// @version         0.1.0
// @description     Automated development agent system - from GitHub Issue to Pull Request

// @servers         [{"url": "http://localhost:8080/api/v1", "description": "Local development server"}]

import (
	"github.com/ryuyb/litchi/internal/application/server"
	"github.com/ryuyb/litchi/internal/application/service"
	"github.com/ryuyb/litchi/internal/infrastructure"
	"github.com/ryuyb/litchi/internal/infrastructure/static"
	"github.com/ryuyb/litchi/internal/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	_ "github.com/ryuyb/litchi/docs/api" // Import generated docs for Swagger embedding
)

// serverMode controls the server runtime mode (debug, release, test) from CLI flag
var serverMode string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the HTTP server",
	Long: `Start the Litchi HTTP server with all application modules.
The server provides:
  - REST API endpoints for session management
  - WebSocket for real-time updates
  - Swagger UI for API documentation (when enabled)
  - Static file serving for the frontend SPA`,
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			fx.WithLogger(func() fxevent.Logger { return getFxLogger(serverMode) }),
			// Supply pre-loaded config (loaded in rootCmd.PersistentPreRunE)
			fx.Supply(loadedCfg),
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
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&serverMode, "mode", "m", "", "server mode (debug, release, test)")
}

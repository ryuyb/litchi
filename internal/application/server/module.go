package server

import (
	"github.com/gofiber/fiber/v3"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "server",
		Provides: []string{"*fiber.App"},
		Invokes:  []string{"StartAppHook"},
		Depends:  []string{"*zap.Logger"},
	})
}

// Module provides the HTTP server module for Fx.
var Module = fx.Module("server",
	// HTTP server providers will be added in T6.1.*
	fx.Provide(NewApp),
	fx.Invoke(StartAppHook),
)

// StartAppHook registers the app start/stop lifecycle hook.
func StartAppHook(lifecycle fx.Lifecycle, app *fiber.App, cfg *config.Config, logger *zap.Logger) {
	lifecycle.Append(StartApp(app, cfg, logger))
}
package config

import (
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/server/router"
)

// Module provides the config handler module for Fx.
var Module = fx.Module("config-handler",
	fx.Provide(NewHandler),
	fx.Invoke(func(apiRouter router.APIRouter, h *Handler) {
		RegisterRoutes(apiRouter, h)
	}),
)
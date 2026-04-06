package config

import (
	"go.uber.org/fx"
)

// Module provides the config handler module for Fx.
var Module = fx.Module("config-handler",
	fx.Provide(NewHandler),
	fx.Invoke(RegisterRoutes),
)
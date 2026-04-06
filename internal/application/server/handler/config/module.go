package config

import (
	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "config-handler",
		Provides: []string{"*config.Handler"},
		Invokes:  []string{"RegisterRoutes"},
		Depends:  []string{"*config.Config", "*zap.Logger"},
	})
}

// Module provides the config handler module for Fx.
var Module = fx.Module("config-handler",
	fx.Provide(NewHandler),
	fx.Invoke(RegisterRoutes),
)
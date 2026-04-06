// Package health provides HTTP health check handlers via Fx module.
package health

import (
	"go.uber.org/fx"
)

// Module provides health check handlers via Fx.
var Module = fx.Module("health-handler",
	// Provider
	fx.Provide(NewHandler),

	// Invoke - register routes
	fx.Invoke(RegisterRoutes),
)
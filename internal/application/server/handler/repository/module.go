// Package repository provides HTTP handlers for repository management API via Fx module.
package repository

import (
	"go.uber.org/fx"
)

// Module provides repository management handlers via Fx.
var Module = fx.Module("repository-handler",
	// Provider
	fx.Provide(NewHandler),

	// Invoke - register routes
	fx.Invoke(RegisterRoutes),
)
// Package static provides Fx module for static file serving.
//
// Registration Order:
// This module MUST be registered AFTER all API/WebSocket/Swagger route modules.
// It registers a wildcard catch-all route (/*) for SPA fallback, which will match
// any path not matched by previously registered exact routes.
// Fiber matches exact routes before wildcards, ensuring proper routing priority.
package static

import (
	"go.uber.org/fx"
)

// Module provides static file serving for embedded frontend assets.
// Register this module after server.Module to ensure API routes take priority.
var Module = fx.Module("static",
	fx.Provide(NewHandler),
	fx.Invoke(RegisterRoutes),
)
// Package middleware provides HTTP middleware for the server.
package middleware

import "go.uber.org/fx"

// Module provides middleware for Fx.
var Module = fx.Module("middleware",
	fx.Provide(NewErrorHandler),
)
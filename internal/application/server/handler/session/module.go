// Package session provides HTTP handlers for session management API.
package session

import (
	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "session-handler",
		Provides: []string{"*session.Handler"},
		Depends:  []string{"*zap.Logger", "*config.Config", "*repository.WorkSessionRepository", "SessionControlService"},
		Invokes:  []string{"RegisterRoutes"},
	})
}

// Module provides the session handler as an Fx module.
// It registers the handler as a Provider and the routes as an Invoke.
var Module = fx.Module("session-handler",
	// Provide the session handler
	fx.Provide(NewHandler),

	// Invoke route registration (registers routes to Fiber App)
	fx.Invoke(RegisterRoutes),
)
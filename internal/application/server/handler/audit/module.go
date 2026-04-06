// Package audit provides HTTP handlers for audit log API endpoints.
package audit

import (
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/server/router"
)

// Module provides audit log API handlers for Fx dependency injection.
//
// This module provides:
//   - Handler: Audit log HTTP handler with query endpoints
//   - Routes: Registration function for Fiber routes
//
// Dependencies:
//   - AuditService: Application service for audit log operations
//   - Logger: Zap logger for structured logging
var Module = fx.Module("audit-handler",
	// Provide the audit handler as a Fx Provider
	fx.Provide(NewHandler),

	// Register routes with APIRouter
	fx.Invoke(func(apiRouter router.APIRouter, h *Handler) {
		RegisterRoutes(apiRouter, h)
	}),
)
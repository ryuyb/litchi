// Package websocket provides WebSocket handlers for real-time progress updates.
package websocket

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the WebSocket handler as an Fx module.
// It registers:
// - Hub (connection manager) as Provider
// - EventBridge (event -> WebSocket bridge) as Provider
// - Handler as Provider
// - Route registration as Invoke
// - Hub lifecycle (start/stop) as Invoke
// - Event handler registration as Invoke
var Module = fx.Module("websocket-handler",
	// Provide the Hub (connection manager)
	fx.Provide(NewHub),

	// Provide the EventBridge (event -> WebSocket bridge)
	fx.Provide(NewEventBridge),

	// Provide the WebSocket handler
	fx.Provide(NewHandler),

	// Invoke Hub lifecycle - start the hub on app start
	fx.Invoke(func(lc fx.Lifecycle, hub *Hub, logger *zap.Logger) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				logger.Info("starting websocket hub")
				go hub.Run(ctx)
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info("stopping websocket hub")
				hub.Stop()
				return nil
			},
		})
	}),

	// Invoke EventBridge registration - subscribe to domain events
	fx.Invoke(func(eventBridge *EventBridge) {
		eventBridge.RegisterHandlers()
	}),

	// Invoke route registration (registers routes to Fiber App)
	fx.Invoke(RegisterRoutes),
)
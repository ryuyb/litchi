// Package event provides domain event infrastructure via Fx.
package event

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides event dispatcher via Fx.
var Module = fx.Module("event",
	fx.Provide(NewDispatcherFromDeps),
)

// DispatcherParams contains dependencies for creating a Dispatcher.
type DispatcherParams struct {
	fx.In

	Logger *zap.Logger
}

// NewDispatcherFromDeps creates a new Dispatcher with dependencies.
func NewDispatcherFromDeps(p DispatcherParams) *Dispatcher {
	return NewDispatcher(
		WithLogger(p.Logger.Named("event_dispatcher")),
		WithAsyncWorkers(10),
	)
}
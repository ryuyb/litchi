package webhook

import (
	"context"

	"go.uber.org/zap"
)

// EventHandler handles a specific webhook event type.
type EventHandler interface {
	Handle(ctx context.Context, event WebhookEvent) error
}

// EventHandlerFunc is an adapter to allow using functions as handlers.
type EventHandlerFunc func(ctx context.Context, event WebhookEvent) error

// Handle implements EventHandler.
func (f EventHandlerFunc) Handle(ctx context.Context, event WebhookEvent) error {
	return f(ctx, event)
}

// EventDispatcher dispatches webhook events to appropriate handlers.
type EventDispatcher struct {
	handlers map[string]EventHandler
	logger   *zap.Logger
}

// NewEventDispatcher creates a new dispatcher.
func NewEventDispatcher(logger *zap.Logger) *EventDispatcher {
	return &EventDispatcher{
		handlers: make(map[string]EventHandler),
		logger:   logger.Named("webhook.dispatcher"),
	}
}

// Register registers a handler for an event type.
func (d *EventDispatcher) Register(eventType string, handler EventHandler) {
	d.handlers[eventType] = handler
	d.logger.Debug("handler registered",
		zap.String("event_type", eventType),
	)
}

// RegisterFunc registers a handler function for an event type.
func (d *EventDispatcher) RegisterFunc(eventType string, handler EventHandlerFunc) {
	d.Register(eventType, handler)
}

// Dispatch dispatches an event to its registered handler.
func (d *EventDispatcher) Dispatch(ctx context.Context, event WebhookEvent) error {
	handler, ok := d.handlers[event.EventType()]
	if !ok {
		d.logger.Debug("no handler for event type",
			zap.String("event_type", event.EventType()),
		)
		return nil // Not an error, just ignored
	}

	d.logger.Debug("dispatching event",
		zap.String("event_type", event.EventType()),
		zap.String("repository", event.Repository()),
		zap.String("actor", event.Actor()),
		zap.String("action", event.Action()),
	)

	return handler.Handle(ctx, event)
}

// HasHandler returns true if a handler is registered for the event type.
func (d *EventDispatcher) HasHandler(eventType string) bool {
	_, ok := d.handlers[eventType]
	return ok
}

// RegisteredEventTypes returns all registered event types.
func (d *EventDispatcher) RegisteredEventTypes() []string {
	types := make([]string, 0, len(d.handlers))
	for t := range d.handlers {
		types = append(types, t)
	}
	return types
}
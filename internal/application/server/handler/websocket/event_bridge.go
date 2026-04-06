// Package websocket provides WebSocket handlers for real-time progress updates.
package websocket

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ryuyb/litchi/internal/domain/event"
)

// EventBridge connects domain events to WebSocket broadcasting.
// It subscribes to all domain events via wildcard handler and broadcasts
// session-related events to connected WebSocket clients.
type EventBridge struct {
	hub        *Hub
	dispatcher *event.Dispatcher
	logger     *zap.Logger
}

// EventBridgeParams contains dependencies for creating an event bridge.
type EventBridgeParams struct {
	Hub        *Hub
	Dispatcher *event.Dispatcher
	Logger     *zap.Logger
}

// NewEventBridge creates a new event bridge.
func NewEventBridge(p EventBridgeParams) *EventBridge {
	return &EventBridge{
		hub:        p.Hub,
		dispatcher: p.Dispatcher,
		logger:     p.Logger.Named("ws_event_bridge"),
	}
}

// RegisterHandlers registers a wildcard event handler for WebSocket broadcasting.
// All events are processed through eventTypeToMessageType which determines if
// the event should be broadcast and converts it to the appropriate message type.
func (b *EventBridge) RegisterHandlers() {
	b.dispatcher.RegisterAll(b.handleEvent)
	b.logger.Info("websocket event handler registered")
}

// handleEvent handles all domain events for WebSocket broadcasting.
// It uses eventTypeToMessageType to determine if an event should be broadcast
// and to convert it to the appropriate WebSocket message type.
func (b *EventBridge) handleEvent(ctx context.Context, e event.DomainEvent) error {
	// Skip system-level events (no session ID)
	if event.IsSystemEvent(e) {
		return nil
	}

	// Check if event type is supported for WebSocket broadcast
	msg := FromDomainEvent(e)
	if msg == nil {
		// Event type not supported for WebSocket, skip silently
		return nil
	}

	return b.broadcastEvent(e.SessionID(), e, msg)
}

// broadcastEvent broadcasts a domain event to WebSocket clients subscribed to the session.
// Note: This method does not implement retry on failure. WebSocket is a real-time transport,
// and retrying could cause message ordering issues. Clients can recover state by:
// 1. Reconnecting and receiving subsequent events
// 2. Fetching current session state via REST API
// The error is logged with full context for troubleshooting.
func (b *EventBridge) broadcastEvent(sessionID uuid.UUID, e event.DomainEvent, msg *Message) error {
	// Serialize message
	msgBytes, err := msg.ToJSON()
	if err != nil {
		b.logger.Error("failed to serialize websocket message",
			zap.Error(err),
			zap.String("event_type", e.EventType()),
			zap.String("session_id", sessionID.String()),
			zap.Any("message_type", msg.Type),
			zap.Any("payload", msg.Payload),
		)
		return err
	}

	// Broadcast to session subscribers
	b.hub.Broadcast(sessionID, msgBytes)

	b.logger.Debug("event broadcasted to websocket clients",
		zap.String("event_type", e.EventType()),
		zap.String("session_id", sessionID.String()),
		zap.Int("client_count", b.hub.ClientCount(sessionID)),
	)

	return nil
}
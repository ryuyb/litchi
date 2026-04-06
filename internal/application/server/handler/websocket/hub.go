// Package websocket provides WebSocket handlers for real-time progress updates.
package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// HubConfig contains configuration for the WebSocket hub.
type HubConfig struct {
	// RegisterBufferSize is the buffer size for the registration channel.
	RegisterBufferSize int

	// UnregisterBufferSize is the buffer size for the unregistration channel.
	UnregisterBufferSize int

	// BroadcastBufferSize is the buffer size for the broadcast channel.
	BroadcastBufferSize int

	// RegisterTimeout is the timeout for client registration.
	// If the registration channel is full, the client will wait up to this duration
	// before being rejected. Set to 0 for no waiting (immediate rejection).
	RegisterTimeout time.Duration
}

// DefaultHubConfig returns the default hub configuration.
func DefaultHubConfig() HubConfig {
	return HubConfig{
		RegisterBufferSize:    100,
		UnregisterBufferSize:  100,
		BroadcastBufferSize:   1000,
		RegisterTimeout:       5 * time.Second,
	}
}

// Hub manages WebSocket client connections.
// It supports session-based subscription - clients subscribe to events for a specific session.
type Hub struct {
	// clients maps session ID to connected clients
	clients map[uuid.UUID]map[*Client]struct{}

	// register channel for new clients
	register chan *Client

	// unregister channel for disconnected clients
	unregister chan *Client

	// broadcast channel for messages (session ID, message)
	broadcast chan BroadcastMessage

	// config holds the hub configuration
	config HubConfig

	// logger for debugging
	logger *zap.Logger

	// mutex for thread-safe access
	mu sync.RWMutex

	// done channel for shutdown
	done chan struct{}
}

// BroadcastMessage represents a message to broadcast to session subscribers.
type BroadcastMessage struct {
	SessionID uuid.UUID
	Message   []byte
}

// NewHub creates a new WebSocket hub with default configuration.
func NewHub(logger *zap.Logger) *Hub {
	return NewHubWithConfig(logger, DefaultHubConfig())
}

// NewHubWithConfig creates a new WebSocket hub with the specified configuration.
func NewHubWithConfig(logger *zap.Logger, config HubConfig) *Hub {
	// Ensure minimum buffer sizes
	if config.RegisterBufferSize < 1 {
		config.RegisterBufferSize = 100
	}
	if config.UnregisterBufferSize < 1 {
		config.UnregisterBufferSize = 100
	}
	if config.BroadcastBufferSize < 1 {
		config.BroadcastBufferSize = 1000
	}

	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]struct{}),
		register:   make(chan *Client, config.RegisterBufferSize),
		unregister: make(chan *Client, config.UnregisterBufferSize),
		broadcast:  make(chan BroadcastMessage, config.BroadcastBufferSize),
		config:     config,
		logger:     logger.Named("ws_hub"),
		done:       make(chan struct{}),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.shutdown()
			return
		case <-h.done:
			h.shutdown()
			return
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case msg := <-h.broadcast:
			h.broadcastMessage(msg)
		}
	}
}

// Register queues a client for registration with optional timeout.
// If the registration channel is full and RegisterTimeout is configured,
// it will wait up to that duration before rejecting the client.
func (h *Hub) Register(client *Client) {
	// If no timeout configured, use non-blocking behavior
	if h.config.RegisterTimeout <= 0 {
		select {
		case h.register <- client:
			h.logger.Debug("client registration queued",
				zap.String("client_id", client.ID.String()),
				zap.String("session_id", client.SessionID.String()),
			)
		default:
			h.logger.Warn("registration channel full, dropping client",
				zap.String("client_id", client.ID.String()),
			)
			client.Close()
		}
		return
	}

	// With timeout
	ctx, cancel := context.WithTimeout(context.Background(), h.config.RegisterTimeout)
	defer cancel()

	select {
	case h.register <- client:
		h.logger.Debug("client registration queued",
			zap.String("client_id", client.ID.String()),
			zap.String("session_id", client.SessionID.String()),
		)
	case <-ctx.Done():
		h.logger.Warn("registration timeout, dropping client",
			zap.String("client_id", client.ID.String()),
			zap.String("session_id", client.SessionID.String()),
			zap.Duration("timeout", h.config.RegisterTimeout),
		)
		client.Close()
	case <-h.done:
		h.logger.Warn("hub shutting down, dropping client",
			zap.String("client_id", client.ID.String()),
		)
		client.Close()
	}
}

// Unregister queues a client for unregistration.
func (h *Hub) Unregister(client *Client) {
	select {
	case h.unregister <- client:
		h.logger.Debug("client unregistration queued",
			zap.String("client_id", client.ID.String()),
			zap.String("session_id", client.SessionID.String()),
		)
	default:
		h.logger.Warn("unregister channel full, closing client directly",
			zap.String("client_id", client.ID.String()),
		)
		// Channel full, close the client directly to prevent resource leak
		client.Close()
	}
}

// Broadcast queues a message for broadcast to session subscribers.
// If the broadcast channel is full, the message is dropped and a warning is logged.
func (h *Hub) Broadcast(sessionID uuid.UUID, msg []byte) {
	select {
	case h.broadcast <- BroadcastMessage{
		SessionID: sessionID,
		Message:   msg,
	}:
		h.logger.Debug("message broadcast queued",
			zap.String("session_id", sessionID.String()),
			zap.Int("msg_len", len(msg)),
		)
	default:
		h.logger.Warn("broadcast channel full, message dropped",
			zap.String("session_id", sessionID.String()),
			zap.Int("channel_size", h.config.BroadcastBufferSize),
		)
	}
}

// registerClient adds a client to the hub.
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sessionClients, exists := h.clients[client.SessionID]
	if !exists {
		sessionClients = make(map[*Client]struct{})
		h.clients[client.SessionID] = sessionClients
	}

	sessionClients[client] = struct{}{}

	h.logger.Info("client registered",
		zap.String("client_id", client.ID.String()),
		zap.String("session_id", client.SessionID.String()),
		zap.Int("session_client_count", len(sessionClients)),
		zap.Int("total_sessions", len(h.clients)),
	)

	// Send connected confirmation
	connectedMsg, err := NewConnectedMessage(client.SessionID).ToJSON()
	if err != nil {
		h.logger.Error("failed to serialize connected message, rolling back registration",
			zap.Error(err),
			zap.String("client_id", client.ID.String()),
		)
		// Rollback registration
		delete(sessionClients, client)
		if len(sessionClients) == 0 {
			delete(h.clients, client.SessionID)
		}
		return
	}
	client.Send(connectedMsg)
}

// unregisterClient removes a client from the hub.
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sessionClients, exists := h.clients[client.SessionID]
	if !exists {
		return
	}

	delete(sessionClients, client)

	// Clean up empty session maps
	if len(sessionClients) == 0 {
		delete(h.clients, client.SessionID)
	}

	h.logger.Info("client unregistered",
		zap.String("client_id", client.ID.String()),
		zap.String("session_id", client.SessionID.String()),
		zap.Int("total_sessions", len(h.clients)),
	)
}

// broadcastMessage sends a message to all clients subscribed to a session.
func (h *Hub) broadcastMessage(msg BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sessionClients, exists := h.clients[msg.SessionID]
	if !exists {
		h.logger.Debug("no clients for session, skipping broadcast",
			zap.String("session_id", msg.SessionID.String()),
		)
		return
	}

	h.logger.Debug("broadcasting message to session clients",
		zap.String("session_id", msg.SessionID.String()),
		zap.Int("client_count", len(sessionClients)),
	)

	for client := range sessionClients {
		if !client.IsClosed() {
			client.Send(msg.Message)
		}
	}
}

// shutdown closes all client connections.
// It waits for all clients to complete their cleanup before returning.
func (h *Hub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.logger.Info("hub shutting down, closing all connections",
		zap.Int("session_count", len(h.clients)),
	)

	var wg sync.WaitGroup
	for sessionID, sessionClients := range h.clients {
		for client := range sessionClients {
			wg.Add(1)
			go func(c *Client) {
				defer wg.Done()
				c.Close()
			}(client)
		}
		delete(h.clients, sessionID)
	}

	// Wait for all clients to complete cleanup
	wg.Wait()
}

// ClientCount returns the number of clients for a session.
func (h *Hub) ClientCount(sessionID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sessionClients, exists := h.clients[sessionID]
	if !exists {
		return 0
	}
	return len(sessionClients)
}

// TotalClients returns the total number of connected clients.
func (h *Hub) TotalClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, sessionClients := range h.clients {
		total += len(sessionClients)
	}
	return total
}

// Stop stops the hub.
func (h *Hub) Stop() {
	select {
	case <-h.done:
		// Already stopped
		return
	default:
		close(h.done)
	}
}
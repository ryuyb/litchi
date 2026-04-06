// Package websocket provides WebSocket handlers for real-time progress updates.
package websocket

import (
	"context"
	"time"

	websocket "github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/ryuyb/litchi/internal/infrastructure/config"
)

// Handler handles WebSocket connections for real-time session updates.
type Handler struct {
	hub    *Hub
	config *config.Config
	logger *zap.Logger
}

// HandlerParams contains dependencies for creating a WebSocket handler.
type HandlerParams struct {
	fx.In

	Hub    *Hub
	Config *config.Config
	Logger *zap.Logger
}

// NewHandler creates a new WebSocket handler.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		hub:    p.Hub,
		config: p.Config,
		logger: p.Logger.Named("ws_handler"),
	}
}

// Default WebSocket configuration values.
const (
	DefaultPingInterval = 30 * time.Second
	DefaultReadTimeout  = 60 * time.Second
	DefaultWriteTimeout = 10 * time.Second
)

// WebSocketConfig contains WebSocket connection configuration.
type WebSocketConfig struct {
	PingInterval time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultWebSocketConfig returns default WebSocket configuration.
func DefaultWebSocketConfig() WebSocketConfig {
	return WebSocketConfig{
		PingInterval: DefaultPingInterval,
		ReadTimeout:  DefaultReadTimeout,
		WriteTimeout: DefaultWriteTimeout,
	}
}

// HandleSessionWebSocket handles WebSocket connections for session progress updates.
// Route: /ws/sessions/:id
func (h *Handler) HandleSessionWebSocket(c *websocket.Conn) {
	// Get session ID from path parameter
	sessionIDStr := c.Params("id")
	if sessionIDStr == "" {
		h.logger.Error("missing session id in websocket path")
		c.Close()
		return
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		h.logger.Error("invalid session id format",
			zap.Error(err),
			zap.String("session_id", sessionIDStr),
		)
		c.Close()
		return
	}

	// Get WebSocket config
	wsConfig := h.getWebSocketConfig()

	// Create client
	client := NewClient(sessionID, c, h.logger)

	// Register client with hub
	h.hub.Register(client)

	// Create context for this connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start read and write pumps in separate goroutines
	go client.WritePump(ctx, wsConfig.PingInterval, wsConfig.WriteTimeout)
	go client.ReadPump(ctx, wsConfig.ReadTimeout)

	h.logger.Info("websocket connection established",
		zap.String("client_id", client.ID.String()),
		zap.String("session_id", sessionID.String()),
	)

	// Wait for connection to close
	<-client.done

	// Ensure client is unregistered from hub (handles edge cases where Close() was called without Unregister)
	h.hub.Unregister(client)

	h.logger.Info("websocket connection closed",
		zap.String("client_id", client.ID.String()),
		zap.String("session_id", sessionID.String()),
	)
}

// getWebSocketConfig returns WebSocket configuration from config or defaults.
func (h *Handler) getWebSocketConfig() WebSocketConfig {
	// Use defaults if no WebSocket config is set
	if h.config.Server.WebSocket == nil {
		return DefaultWebSocketConfig()
	}

	ws := h.config.Server.WebSocket
	return WebSocketConfig{
		PingInterval: firstNonZeroDuration(ws.PingInterval, DefaultPingInterval),
		ReadTimeout:  firstNonZeroDuration(ws.ReadTimeout, DefaultReadTimeout),
		WriteTimeout: firstNonZeroDuration(ws.WriteTimeout, DefaultWriteTimeout),
	}
}

// firstNonZeroDuration returns the first non-zero duration, or the default.
func firstNonZeroDuration(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}

// WebSocketUpgradeMiddleware checks if the request is a WebSocket upgrade request.
// Returns ErrUpgradeRequired if not a WebSocket upgrade.
func WebSocketUpgradeMiddleware(c fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("websocket_upgrade", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}
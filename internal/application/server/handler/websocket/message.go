// Package websocket provides WebSocket handlers for real-time progress updates.
package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ryuyb/litchi/internal/domain/event"
)

// WebSocket message type constants (mirrors gorilla/websocket values).
const (
	// TextMessage denotes a text data message. The text message payload is
	// interpreted as UTF-8 encoded text data.
	TextMessage = 1

	// BinaryMessage denotes a binary data message.
	BinaryMessage = 2

	// CloseMessage denotes a close control message. The optional message
	// payload contains a numeric code and text.
	CloseMessage = 8

	// PingMessage denotes a ping control message. The optional message
	// payload is UTF-8 encoded text.
	PingMessage = 9

	// PongMessage denotes a pong control message. The optional message
	// payload is UTF-8 encoded text.
	PongMessage = 10
)

// MessageType defines the type of WebSocket message.
type MessageType string

// Message types for WebSocket communication.
const (
	// Event messages (from server to client)
	MessageTypeStageTransitioned MessageType = "stage_transitioned"
	MessageTypeStageRolledBack  MessageType = "stage_rolled_back"
	MessageTypeTaskStarted      MessageType = "task_started"
	MessageTypeTaskCompleted    MessageType = "task_completed"
	MessageTypeTaskFailed       MessageType = "task_failed"
	MessageTypeTaskSkipped      MessageType = "task_skipped"
	MessageTypeTaskRetryStarted MessageType = "task_retry_started"
	MessageTypeQuestionAsked    MessageType = "question_asked"
	MessageTypeQuestionAnswered MessageType = "question_answered"
	MessageTypeDesignCreated    MessageType = "design_created"
	MessageTypeDesignApproved   MessageType = "design_approved"
	MessageTypeDesignRejected   MessageType = "design_rejected"
	MessageTypePRCreated        MessageType = "pr_created"
	MessageTypePRMerged         MessageType = "pr_merged"
	MessageTypeSessionStarted   MessageType = "session_started"
	MessageTypeSessionPaused    MessageType = "session_paused"
	MessageTypeSessionResumed   MessageType = "session_resumed"
	MessageTypeSessionCompleted MessageType = "session_completed"
	MessageTypeSessionTerminated MessageType = "session_terminated"

	// Control messages
	MessageTypePing    MessageType = "ping"
	MessageTypePong    MessageType = "pong"
	MessageTypeError   MessageType = "error"
	MessageTypeConnected MessageType = "connected"
)

// Message represents a WebSocket message sent to clients.
type Message struct {
	Type      MessageType `json:"type"`
	Payload   any         `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

// ErrorPayload represents error message payload.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ConnectedPayload represents connection confirmation payload.
type ConnectedPayload struct {
	SessionID string `json:"sessionId"`
}

// NewMessage creates a new WebSocket message.
func NewMessage(msgType MessageType, payload any) *Message {
	return &Message{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

// NewErrorMessage creates an error message.
func NewErrorMessage(code, message string) *Message {
	return NewMessage(MessageTypeError, ErrorPayload{
		Code:    code,
		Message: message,
	})
}

// NewConnectedMessage creates a connection confirmation message.
func NewConnectedMessage(sessionID uuid.UUID) *Message {
	return NewMessage(MessageTypeConnected, ConnectedPayload{
		SessionID: sessionID.String(),
	})
}

// ToJSON serializes the message to JSON.
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromDomainEvent converts a domain event to a WebSocket message.
func FromDomainEvent(e event.DomainEvent) *Message {
	msgType := eventTypeToMessageType(e.EventType())
	if msgType == "" {
		return nil
	}
	return NewMessage(msgType, e.ToMap())
}

// eventTypeToMessageType maps domain event types to WebSocket message types.
func eventTypeToMessageType(eventType string) MessageType {
	switch eventType {
	case "StageTransitioned":
		return MessageTypeStageTransitioned
	case "StageRolledBack":
		return MessageTypeStageRolledBack
	case "TaskStarted":
		return MessageTypeTaskStarted
	case "TaskCompleted":
		return MessageTypeTaskCompleted
	case "TaskFailed":
		return MessageTypeTaskFailed
	case "TaskSkipped":
		return MessageTypeTaskSkipped
	case "TaskRetryStarted":
		return MessageTypeTaskRetryStarted
	case "QuestionAsked":
		return MessageTypeQuestionAsked
	case "QuestionAnswered":
		return MessageTypeQuestionAnswered
	case "DesignCreated":
		return MessageTypeDesignCreated
	case "DesignApproved":
		return MessageTypeDesignApproved
	case "DesignRejected":
		return MessageTypeDesignRejected
	case "PullRequestCreated":
		return MessageTypePRCreated
	case "PullRequestMerged":
		return MessageTypePRMerged
	case "WorkSessionStarted":
		return MessageTypeSessionStarted
	case "WorkSessionPaused", "WorkSessionPausedWithContext":
		return MessageTypeSessionPaused
	case "WorkSessionResumed", "WorkSessionResumedWithAction", "WorkSessionAutoResumed":
		return MessageTypeSessionResumed
	case "WorkSessionCompleted":
		return MessageTypeSessionCompleted
	case "WorkSessionTerminated":
		return MessageTypeSessionTerminated
	default:
		return ""
	}
}

// Client represents a WebSocket client connection.
type Client struct {
	ID        uuid.UUID
	SessionID uuid.UUID
	conn      Connection
	send      chan []byte
	done      chan struct{}
	mu        sync.Mutex
	logger    *zap.Logger
}

// Connection interface for WebSocket connection (abstracts websocket.Conn).
type Connection interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// NewClient creates a new WebSocket client.
func NewClient(sessionID uuid.UUID, conn Connection, logger *zap.Logger) *Client {
	return &Client{
		ID:        uuid.New(),
		SessionID: sessionID,
		conn:      conn,
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		logger:    logger.Named("ws_client"),
	}
}

// Send queues a message to be sent to the client.
// If the client is already closed, the message is silently dropped.
//
// Concurrency Safety:
// This method is safe for concurrent use. The mutex ensures that:
// 1. The done channel check and message send are atomic with respect to Close()
// 2. Close() acquires the same mutex, so once Close() returns, Send() will not queue messages
// 3. The two-phase select (check done, then send) prevents race conditions:
//    - If done is closed before the first select, we return immediately
//    - If Close() is called between the two selects, the mutex prevents concurrent execution
//    - If done is closed after the first select but before the second, the message is queued
//      but will be drained by WritePump or ignored when WritePump detects done is closed
func (c *Client) Send(msg []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if client is closed first (must be explicit check, not in select)
	select {
	case <-c.done:
		// Client is closed, don't queue the message
		return
	default:
		// Client is still open, continue to send
	}

	// Try to queue the message
	select {
	case c.send <- msg:
	default:
		c.logger.Warn("client send channel full, message dropped",
			zap.String("client_id", c.ID.String()),
			zap.String("session_id", c.SessionID.String()),
		)
	}
}

// WritePump sends messages from the send channel to the WebSocket connection.
func (c *Client) WritePump(ctx context.Context, pingInterval, writeTimeout time.Duration) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case msg, ok := <-c.send:
			if !ok {
				// Channel closed
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(TextMessage, msg); err != nil {
				c.logger.Error("write message failed",
					zap.Error(err),
					zap.String("client_id", c.ID.String()),
				)
				return
			}
		case <-ticker.C:
			// Send ping message
			pingMsg, _ := NewMessage(MessageTypePing, nil).ToJSON()
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(TextMessage, pingMsg); err != nil {
				c.logger.Error("write ping failed",
					zap.Error(err),
					zap.String("client_id", c.ID.String()),
				)
				return
			}
		}
	}
}

// ReadPump reads messages from the WebSocket connection.
// It distinguishes between normal closure (client disconnect) and unexpected errors.
func (c *Client) ReadPump(ctx context.Context, readTimeout time.Duration) {
	defer c.Close()

	c.conn.SetReadDeadline(time.Now().Add(readTimeout))

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		default:
			_, msg, err := c.conn.ReadMessage()
			if err != nil {
				// Distinguish between normal closure and unexpected errors
				if websocket.IsUnexpectedCloseError(err,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNoStatusReceived,
				) {
					c.logger.Warn("unexpected read error, client disconnected abnormally",
						zap.Error(err),
						zap.String("client_id", c.ID.String()),
					)
				} else {
					c.logger.Debug("client disconnected normally",
						zap.Error(err),
						zap.String("client_id", c.ID.String()),
					)
				}
				return
			}

			// Handle incoming message (pong response)
			var incoming Message
			if err := json.Unmarshal(msg, &incoming); err == nil {
				if incoming.Type == MessageTypePong {
					// Reset read deadline on pong
					c.conn.SetReadDeadline(time.Now().Add(readTimeout))
				}
			}
		}
	}
}

// Close closes the client connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		// Already closed
		return
	default:
		close(c.done)
		c.conn.Close()
	}
}

// IsClosed checks if the client is closed.
func (c *Client) IsClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}
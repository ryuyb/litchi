// Package websocket provides WebSocket handlers for real-time progress updates.
package websocket

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockConnection is a mock WebSocket connection for testing.
type mockConnection struct {
	mu          sync.Mutex
	writtenMsgs [][]byte
	closed      bool
	readErr     error
}

func (m *mockConnection) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrConnectionClosed
	}
	m.writtenMsgs = append(m.writtenMsgs, data)
	return nil
}

func (m *mockConnection) ReadMessage() (int, []byte, error) {
	return 0, nil, m.readErr
}

func (m *mockConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

func (m *mockConnection) getWrittenMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writtenMsgs
}

func (m *mockConnection) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// ErrConnectionClosed is a mock error for closed connections.
var ErrConnectionClosed = assert.AnError

func TestHubRegister(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	defer hub.Stop()

	// Wait for hub to start
	time.Sleep(10 * time.Millisecond)

	sessionID := uuid.New()
	conn := &mockConnection{}
	client := NewClient(sessionID, conn, logger)

	hub.Register(client)

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount(sessionID))
	assert.Equal(t, 1, hub.TotalClients())

	// Verify connected message was sent
	msgs := conn.getWrittenMessages()
	require.Len(t, msgs, 1)
	assert.Contains(t, string(msgs[0]), `"type":"connected"`)
}

func TestHubUnregister(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	sessionID := uuid.New()
	conn := &mockConnection{}
	client := NewClient(sessionID, conn, logger)

	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount(sessionID))

	hub.Unregister(client)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.ClientCount(sessionID))
	assert.Equal(t, 0, hub.TotalClients())
}

func TestHubBroadcast(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	sessionID := uuid.New()
	conn1 := &mockConnection{}
	conn2 := &mockConnection{}
	client1 := NewClient(sessionID, conn1, logger)
	client2 := NewClient(sessionID, conn2, logger)

	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(50 * time.Millisecond)

	// Clear the connected messages
	conn1.getWrittenMessages()
	conn2.getWrittenMessages()

	// Broadcast a message
	msg := []byte(`{"type":"test","payload":"hello"}`)
	hub.Broadcast(sessionID, msg)

	time.Sleep(50 * time.Millisecond)

	// Both clients should receive the message
	assert.Len(t, conn1.getWrittenMessages(), 1)
	assert.Len(t, conn2.getWrittenMessages(), 1)
}

func TestHubBroadcastToNoClients(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	// Broadcast to a session with no clients
	sessionID := uuid.New()
	msg := []byte(`{"type":"test"}`)

	// Should not panic
	hub.Broadcast(sessionID, msg)
	time.Sleep(50 * time.Millisecond)
}

func TestHubMultipleSessions(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	sessionID1 := uuid.New()
	sessionID2 := uuid.New()
	conn1 := &mockConnection{}
	conn2 := &mockConnection{}
	client1 := NewClient(sessionID1, conn1, logger)
	client2 := NewClient(sessionID2, conn2, logger)

	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount(sessionID1))
	assert.Equal(t, 1, hub.ClientCount(sessionID2))
	assert.Equal(t, 2, hub.TotalClients())

	// Clear connected messages
	conn1.getWrittenMessages()
	conn2.getWrittenMessages()

	// Broadcast to session1
	msg1 := []byte(`{"type":"test1"}`)
	hub.Broadcast(sessionID1, msg1)
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, conn1.getWrittenMessages(), 1)
	assert.Len(t, conn2.getWrittenMessages(), 0) // session2 client should not receive

	// Broadcast to session2
	msg2 := []byte(`{"type":"test2"}`)
	hub.Broadcast(sessionID2, msg2)
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, conn1.getWrittenMessages(), 1) // unchanged
	assert.Len(t, conn2.getWrittenMessages(), 1)
}

func TestHubShutdown(t *testing.T) {
	logger := zap.NewNop()
	hub := NewHub(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(10 * time.Millisecond)

	sessionID := uuid.New()
	conn := &mockConnection{}
	client := NewClient(sessionID, conn, logger)

	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount(sessionID))

	// Stop the hub
	hub.Stop()
	time.Sleep(50 * time.Millisecond)

	// All clients should be removed
	assert.Equal(t, 0, hub.TotalClients())
}

func TestClientSend(t *testing.T) {
	logger := zap.NewNop()
	conn := &mockConnection{}
	sessionID := uuid.New()
	client := NewClient(sessionID, conn, logger)

	msg := []byte(`{"type":"test"}`)
	client.Send(msg)

	// Message should be in send channel
	select {
	case m := <-client.send:
		assert.Equal(t, msg, m)
	default:
		t.Fatal("expected message in send channel")
	}
}

func TestClientClose(t *testing.T) {
	logger := zap.NewNop()
	conn := &mockConnection{}
	sessionID := uuid.New()
	client := NewClient(sessionID, conn, logger)

	assert.False(t, client.IsClosed())

	client.Close()

	assert.True(t, client.IsClosed())
	assert.True(t, conn.isClosed())

	// Close again should be idempotent
	client.Close()
	assert.True(t, client.IsClosed())
}
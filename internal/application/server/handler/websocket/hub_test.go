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
	writeBlock  chan struct{} // optional: block writes until closed
}

func newMockConnection() *mockConnection {
	return &mockConnection{
		writeBlock: make(chan struct{}),
	}
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

func (m *mockConnection) clearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writtenMsgs = nil
}

func (m *mockConnection) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// ErrConnectionClosed is a mock error for closed connections.
var ErrConnectionClosed = assert.AnError

// testHubRunner runs a hub and provides synchronization for tests.
type testHubRunner struct {
	hub    *Hub
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newTestHubRunner(t *testing.T) *testHubRunner {
	logger := zap.NewNop()
	hub := NewHub(logger)
	ctx, cancel := context.WithCancel(context.Background())

	runner := &testHubRunner{
		hub:    hub,
		ctx:    ctx,
		cancel: cancel,
	}

	runner.wg.Add(1)
	go func() {
		defer runner.wg.Done()
		hub.Run(ctx)
	}()

	return runner
}

func (r *testHubRunner) stop() {
	r.cancel()
	r.wg.Wait()
}

// eventually asserts that a condition becomes true within a timeout.
func eventually(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...any) {
	require.Eventually(t, condition, timeout, 10*time.Millisecond, msgAndArgs...)
}

func TestHubRegister(t *testing.T) {
	runner := newTestHubRunner(t)
	defer runner.stop()

	sessionID := uuid.New()
	conn := newMockConnection()
	client := NewClient(sessionID, conn, zap.NewNop())

	// Start write pump to receive messages
	go client.WritePump(runner.ctx, 30*time.Second, 10*time.Second)

	runner.hub.Register(client)

	// Wait for registration to complete
	eventually(t, func() bool {
		return runner.hub.ClientCount(sessionID) == 1
	}, time.Second, "client should be registered")

	assert.Equal(t, 1, runner.hub.TotalClients())

	// Wait for connected message
	eventually(t, func() bool {
		return len(conn.getWrittenMessages()) >= 1
	}, time.Second, "connected message should be sent")

	msgs := conn.getWrittenMessages()
	require.Len(t, msgs, 1)
	assert.Contains(t, string(msgs[0]), `"type":"connected"`)
}

func TestHubUnregister(t *testing.T) {
	runner := newTestHubRunner(t)
	defer runner.stop()

	sessionID := uuid.New()
	conn := newMockConnection()
	client := NewClient(sessionID, conn, zap.NewNop())

	runner.hub.Register(client)

	// Wait for registration
	eventually(t, func() bool {
		return runner.hub.ClientCount(sessionID) == 1
	}, time.Second, "client should be registered")

	runner.hub.Unregister(client)

	// Wait for unregistration
	eventually(t, func() bool {
		return runner.hub.ClientCount(sessionID) == 0
	}, time.Second, "client should be unregistered")

	assert.Equal(t, 0, runner.hub.TotalClients())
}

func TestHubBroadcast(t *testing.T) {
	runner := newTestHubRunner(t)
	defer runner.stop()

	sessionID := uuid.New()
	conn1 := newMockConnection()
	conn2 := newMockConnection()
	client1 := NewClient(sessionID, conn1, zap.NewNop())
	client2 := NewClient(sessionID, conn2, zap.NewNop())

	// Start write pumps
	go client1.WritePump(runner.ctx, 30*time.Second, 10*time.Second)
	go client2.WritePump(runner.ctx, 30*time.Second, 10*time.Second)

	runner.hub.Register(client1)
	runner.hub.Register(client2)

	// Wait for connected messages
	eventually(t, func() bool {
		return len(conn1.getWrittenMessages()) >= 1 && len(conn2.getWrittenMessages()) >= 1
	}, time.Second, "connected messages should be sent")

	// Clear connected messages
	conn1.clearMessages()
	conn2.clearMessages()

	// Broadcast a message
	msg := []byte(`{"type":"test","payload":"hello"}`)
	runner.hub.Broadcast(sessionID, msg)

	// Wait for broadcast
	eventually(t, func() bool {
		return len(conn1.getWrittenMessages()) >= 1 && len(conn2.getWrittenMessages()) >= 1
	}, time.Second, "broadcast messages should be sent")

	assert.Contains(t, string(conn1.getWrittenMessages()[0]), `"type":"test"`)
	assert.Contains(t, string(conn2.getWrittenMessages()[0]), `"type":"test"`)
}

func TestHubBroadcastToNoClients(t *testing.T) {
	runner := newTestHubRunner(t)
	defer runner.stop()

	// Broadcast to a session with no clients
	sessionID := uuid.New()
	msg := []byte(`{"type":"test"}`)

	// Should not panic
	runner.hub.Broadcast(sessionID, msg)
}

func TestHubMultipleSessions(t *testing.T) {
	runner := newTestHubRunner(t)
	defer runner.stop()

	sessionID1 := uuid.New()
	sessionID2 := uuid.New()
	conn1 := newMockConnection()
	conn2 := newMockConnection()
	client1 := NewClient(sessionID1, conn1, zap.NewNop())
	client2 := NewClient(sessionID2, conn2, zap.NewNop())

	// Start write pumps
	go client1.WritePump(runner.ctx, 30*time.Second, 10*time.Second)
	go client2.WritePump(runner.ctx, 30*time.Second, 10*time.Second)

	runner.hub.Register(client1)
	runner.hub.Register(client2)

	// Wait for registration
	eventually(t, func() bool {
		return runner.hub.ClientCount(sessionID1) == 1 && runner.hub.ClientCount(sessionID2) == 1
	}, time.Second, "clients should be registered")

	assert.Equal(t, 2, runner.hub.TotalClients())

	// Clear connected messages
	eventually(t, func() bool {
		return len(conn1.getWrittenMessages()) >= 1 && len(conn2.getWrittenMessages()) >= 1
	}, time.Second, "connected messages should be sent")
	conn1.clearMessages()
	conn2.clearMessages()

	// Broadcast to session1
	msg1 := []byte(`{"type":"test1"}`)
	runner.hub.Broadcast(sessionID1, msg1)

	eventually(t, func() bool {
		return len(conn1.getWrittenMessages()) >= 1
	}, time.Second, "session1 client should receive broadcast")

	assert.Len(t, conn2.getWrittenMessages(), 0) // session2 client should not receive

	// Broadcast to session2
	msg2 := []byte(`{"type":"test2"}`)
	runner.hub.Broadcast(sessionID2, msg2)

	eventually(t, func() bool {
		return len(conn2.getWrittenMessages()) >= 1
	}, time.Second, "session2 client should receive broadcast")

	assert.Len(t, conn1.getWrittenMessages(), 1) // unchanged
}

func TestHubShutdown(t *testing.T) {
	runner := newTestHubRunner(t)

	sessionID := uuid.New()
	conn := newMockConnection()
	client := NewClient(sessionID, conn, zap.NewNop())

	runner.hub.Register(client)

	eventually(t, func() bool {
		return runner.hub.ClientCount(sessionID) == 1
	}, time.Second, "client should be registered")

	// Stop the hub
	runner.stop()

	// All clients should be removed after shutdown
	assert.Equal(t, 0, runner.hub.TotalClients())
}

func TestClientSend(t *testing.T) {
	conn := newMockConnection()
	sessionID := uuid.New()
	client := NewClient(sessionID, conn, zap.NewNop())

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

func TestClientSendWhenClosed(t *testing.T) {
	conn := newMockConnection()
	sessionID := uuid.New()
	client := NewClient(sessionID, conn, zap.NewNop())

	client.Close()

	// Send should return silently when closed
	msg := []byte(`{"type":"test"}`)
	client.Send(msg)

	// Message should NOT be in send channel
	select {
	case <-client.send:
		t.Fatal("should not have message in send channel when closed")
	default:
		// Expected
	}
}

func TestClientClose(t *testing.T) {
	conn := newMockConnection()
	sessionID := uuid.New()
	client := NewClient(sessionID, conn, zap.NewNop())

	assert.False(t, client.IsClosed())

	client.Close()

	assert.True(t, client.IsClosed())
	assert.True(t, conn.isClosed())

	// Close again should be idempotent
	client.Close()
	assert.True(t, client.IsClosed())
}
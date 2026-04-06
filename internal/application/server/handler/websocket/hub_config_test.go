package websocket

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestDefaultHubConfig(t *testing.T) {
	cfg := DefaultHubConfig()

	assert.Equal(t, 100, cfg.RegisterBufferSize)
	assert.Equal(t, 100, cfg.UnregisterBufferSize)
	assert.Equal(t, 1000, cfg.BroadcastBufferSize)
	assert.Equal(t, 5*time.Second, cfg.RegisterTimeout)
}

func TestNewHubWithConfig(t *testing.T) {
	logger := zap.NewNop()

	t.Run("custom config", func(t *testing.T) {
		cfg := HubConfig{
			RegisterBufferSize:    50,
			UnregisterBufferSize:  25,
			BroadcastBufferSize:   500,
			RegisterTimeout:       10 * time.Second,
		}
		hub := NewHubWithConfig(logger, cfg)

		assert.Equal(t, 50, cap(hub.register))
		assert.Equal(t, 25, cap(hub.unregister))
		assert.Equal(t, 500, cap(hub.broadcast))
		assert.Equal(t, 10*time.Second, hub.config.RegisterTimeout)
	})

	t.Run("minimum buffer sizes enforced", func(t *testing.T) {
		cfg := HubConfig{
			RegisterBufferSize:    0,
			UnregisterBufferSize:  -1,
			BroadcastBufferSize:   0,
		}
		hub := NewHubWithConfig(logger, cfg)

		// Should use minimum values
		assert.GreaterOrEqual(t, cap(hub.register), 1)
		assert.GreaterOrEqual(t, cap(hub.unregister), 1)
		assert.GreaterOrEqual(t, cap(hub.broadcast), 1)
	})

	t.Run("default hub uses default config", func(t *testing.T) {
		hub := NewHub(logger)
		defer hub.Stop()

		assert.Equal(t, 100, cap(hub.register))
		assert.Equal(t, 100, cap(hub.unregister))
		assert.Equal(t, 1000, cap(hub.broadcast))
		assert.Equal(t, 5*time.Second, hub.config.RegisterTimeout)
	})
}

func TestHubConfig_RegisterTimeout(t *testing.T) {
	logger := zap.NewNop()

	t.Run("with timeout", func(t *testing.T) {
		cfg := HubConfig{
			RegisterBufferSize:    1,
			RegisterTimeout:       100 * time.Millisecond,
		}
		hub := NewHubWithConfig(logger, cfg)

		// Fill the channel
		hub.register <- &Client{ID: uuidMustParse("00000000-0000-0000-0000-000000000001")}

		// This should timeout
		sessionID := uuidMustParse("00000000-0000-0000-0000-000000000002")
		client := NewClient(sessionID, &mockConnection{}, zap.NewNop())

		// Register should timeout and close client
		start := time.Now()
		hub.Register(client)
		elapsed := time.Since(start)

		// Should have waited at least the timeout duration
		assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond)
		assert.True(t, client.IsClosed())
	})

	t.Run("no timeout (immediate rejection)", func(t *testing.T) {
		cfg := HubConfig{
			RegisterBufferSize: 1,
			RegisterTimeout:    0, // No timeout
		}
		hub := NewHubWithConfig(logger, cfg)

		// Fill the channel
		hub.register <- &Client{ID: uuidMustParse("00000000-0000-0000-0000-000000000001")}

		// This should be rejected immediately
		sessionID := uuidMustParse("00000000-0000-0000-0000-000000000002")
		client := NewClient(sessionID, &mockConnection{}, zap.NewNop())

		start := time.Now()
		hub.Register(client)
		elapsed := time.Since(start)

		// Should be rejected immediately
		assert.Less(t, elapsed, 50*time.Millisecond)
		assert.True(t, client.IsClosed())
	})
}

func uuidMustParse(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}
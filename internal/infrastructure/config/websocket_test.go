package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  WebSocketConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty config is valid (uses defaults)",
			config:  WebSocketConfig{},
			wantErr: false,
		},
		{
			name: "valid config",
			config: WebSocketConfig{
				PingInterval: 30 * time.Second,
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "ping interval too small",
			config: WebSocketConfig{
				PingInterval: 500 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "ping_interval must be at least 1s",
		},
		{
			name: "read timeout less than ping interval",
			config: WebSocketConfig{
				PingInterval: 30 * time.Second,
				ReadTimeout:  20 * time.Second,
			},
			wantErr: true,
			errMsg:  "read_timeout must be greater than ping_interval",
		},
		{
			name: "write timeout too small",
			config: WebSocketConfig{
				WriteTimeout: 500 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "write_timeout must be at least 1s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServerConfig_Validate_WithWebSocket(t *testing.T) {
	tests := []struct {
		name    string
		config  ServerConfig
		wantErr bool
	}{
		{
			name: "valid config without websocket",
			config: ServerConfig{
				Port: 8080,
				Mode: "debug",
			},
			wantErr: false,
		},
		{
			name: "valid config with websocket",
			config: ServerConfig{
				Port: 8080,
				Mode: "debug",
				WebSocket: &WebSocketConfig{
					PingInterval: 30 * time.Second,
					ReadTimeout:  60 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid websocket config",
			config: ServerConfig{
				Port: 8080,
				Mode: "debug",
				WebSocket: &WebSocketConfig{
					PingInterval: 100 * time.Millisecond, // too small
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
// Package websocket provides WebSocket handlers for real-time progress updates.
package websocket

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

func TestNewMessage(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		payload  any
		expected *Message
	}{
		{
			name:    "stage transitioned message",
			msgType: MessageTypeStageTransitioned,
			payload: map[string]string{
				"sessionId": "test-123",
				"from":      "clarification",
				"to":        "design",
			},
			expected: &Message{
				Type:      MessageTypeStageTransitioned,
				Payload:   map[string]string{"sessionId": "test-123", "from": "clarification", "to": "design"},
			},
		},
		{
			name:     "ping message",
			msgType:  MessageTypePing,
			payload:  nil,
			expected: &Message{Type: MessageTypePing, Payload: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessage(tt.msgType, tt.payload)
			assert.Equal(t, tt.msgType, msg.Type)
			assert.Equal(t, tt.payload, msg.Payload)
			assert.False(t, msg.Timestamp.IsZero())
		})
	}
}

func TestNewErrorMessage(t *testing.T) {
	msg := NewErrorMessage("INVALID_SESSION", "Session not found")

	assert.Equal(t, MessageTypeError, msg.Type)
	payload, ok := msg.Payload.(ErrorPayload)
	require.True(t, ok)
	assert.Equal(t, "INVALID_SESSION", payload.Code)
	assert.Equal(t, "Session not found", payload.Message)
}

func TestNewConnectedMessage(t *testing.T) {
	sessionID := uuid.New()
	msg := NewConnectedMessage(sessionID)

	assert.Equal(t, MessageTypeConnected, msg.Type)
	payload, ok := msg.Payload.(ConnectedPayload)
	require.True(t, ok)
	assert.Equal(t, sessionID.String(), payload.SessionID)
}

func TestMessageToJSON(t *testing.T) {
	msg := NewMessage(MessageTypeTaskStarted, map[string]string{
		"taskId":   "task-123",
		"taskName": "Implement feature",
	})

	jsonBytes, err := msg.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"type":"task_started"`)
	assert.Contains(t, string(jsonBytes), `"taskId":"task-123"`)
	assert.Contains(t, string(jsonBytes), `"timestamp"`)
}

func TestFromDomainEvent(t *testing.T) {
	sessionID := uuid.New()
	tests := []struct {
		name         string
		domainEvent  event.DomainEvent
		expectedType MessageType
		shouldReturn bool
	}{
		{
			name: "StageTransitioned event",
			domainEvent: event.NewStageTransitioned(
				sessionID,
				valueobject.StageClarification,
				valueobject.StageDesign,
			),
			expectedType: MessageTypeStageTransitioned,
			shouldReturn: true,
		},
		{
			name: "TaskStarted event",
			domainEvent: event.NewTaskStarted(
				sessionID,
				uuid.New(),
				"Implement feature X",
			),
			expectedType: MessageTypeTaskStarted,
			shouldReturn: true,
		},
		{
			name: "WorkSessionStarted event",
			domainEvent: event.NewWorkSessionStarted(
				sessionID,
				42,
				"owner/repo",
				"Test Issue",
			),
			expectedType: MessageTypeSessionStarted,
			shouldReturn: true,
		},
		{
			name: "PullRequestCreated event",
			domainEvent: event.NewPullRequestCreated(
				sessionID,
				123,
				"feature-branch",
				"Test PR",
			),
			expectedType: MessageTypePRCreated,
			shouldReturn: true,
		},
		{
			name: "RepositoryAdded event (system event)",
			domainEvent: event.NewRepositoryAdded(
				"owner/repo",
			),
			expectedType: "",
			shouldReturn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := FromDomainEvent(tt.domainEvent)
			if !tt.shouldReturn {
				assert.Nil(t, msg)
				return
			}
			require.NotNil(t, msg)
			assert.Equal(t, tt.expectedType, msg.Type)

			// Verify payload contains session ID
			payload := msg.Payload
			payloadMap, ok := payload.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, sessionID.String(), payloadMap["sessionId"])
		})
	}
}

func TestEventTypeToMessageType(t *testing.T) {
	tests := []struct {
		eventType     string
		expectedType  MessageType
	}{
		{"StageTransitioned", MessageTypeStageTransitioned},
		{"StageRolledBack", MessageTypeStageRolledBack},
		{"TaskStarted", MessageTypeTaskStarted},
		{"TaskCompleted", MessageTypeTaskCompleted},
		{"TaskFailed", MessageTypeTaskFailed},
		{"TaskSkipped", MessageTypeTaskSkipped},
		{"TaskRetryStarted", MessageTypeTaskRetryStarted},
		{"QuestionAsked", MessageTypeQuestionAsked},
		{"QuestionAnswered", MessageTypeQuestionAnswered},
		{"DesignCreated", MessageTypeDesignCreated},
		{"DesignApproved", MessageTypeDesignApproved},
		{"DesignRejected", MessageTypeDesignRejected},
		{"PullRequestCreated", MessageTypePRCreated},
		{"PullRequestMerged", MessageTypePRMerged},
		{"WorkSessionStarted", MessageTypeSessionStarted},
		{"WorkSessionPaused", MessageTypeSessionPaused},
		{"WorkSessionPausedWithContext", MessageTypeSessionPaused},
		{"WorkSessionResumed", MessageTypeSessionResumed},
		{"WorkSessionResumedWithAction", MessageTypeSessionResumed},
		{"WorkSessionAutoResumed", MessageTypeSessionResumed},
		{"WorkSessionCompleted", MessageTypeSessionCompleted},
		{"WorkSessionTerminated", MessageTypeSessionTerminated},
		{"UnknownEvent", MessageType("")},
		{"RepositoryAdded", MessageType("")}, // System event, not for WebSocket
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			result := eventTypeToMessageType(tt.eventType)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestDefaultWebSocketConfig(t *testing.T) {
	cfg := DefaultWebSocketConfig()

	assert.Equal(t, 30*time.Second, cfg.PingInterval)
	assert.Equal(t, 60*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout)
}
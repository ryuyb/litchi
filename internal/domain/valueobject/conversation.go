package valueobject

import "time"

// ConversationTurn represents a single turn in the clarification dialogue.
// Used by Clarification entity to track question-answer history.
type ConversationTurn struct {
	Role      string    `json:"role"`      // "agent" or "user"
	Content   string    `json:"content"`   // Question or answer content
	Timestamp time.Time `json:"timestamp"` // When the turn occurred
}

// NewConversationTurn creates a new conversation turn.
func NewConversationTurn(role, content string) ConversationTurn {
	return ConversationTurn{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// IsAgent returns true if this turn is from the agent.
func (ct ConversationTurn) IsAgent() bool {
	return ct.Role == "agent"
}

// IsUser returns true if this turn is from the user.
func (ct ConversationTurn) IsUser() bool {
	return ct.Role == "user"
}

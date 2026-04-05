package service

import "fmt"

// TransitionDecision represents the decision result for a stage transition.
type TransitionDecision int

const (
	// DecisionAllow allows automatic transition without confirmation.
	DecisionAllow TransitionDecision = iota
	// DecisionNeedConfirmation requires user confirmation before transition.
	DecisionNeedConfirmation
	// DecisionDenied denies transition, must continue clarification.
	DecisionDenied
)

// String returns the string representation of the decision.
func (d TransitionDecision) String() string {
	switch d {
	case DecisionAllow:
		return "allow"
	case DecisionNeedConfirmation:
		return "need_confirmation"
	case DecisionDenied:
		return "denied"
	default:
		return "unknown"
	}
}

// IsAllowed returns true if the decision allows transition.
func (d TransitionDecision) IsAllowed() bool {
	return d == DecisionAllow
}

// NeedsConfirmation returns true if the decision requires confirmation.
func (d TransitionDecision) NeedsConfirmation() bool {
	return d == DecisionNeedConfirmation
}

// IsDenied returns true if the decision denies transition.
func (d TransitionDecision) IsDenied() bool {
	return d == DecisionDenied
}

// TransitionResult represents the complete evaluation result for a stage transition.
type TransitionResult struct {
	Decision       TransitionDecision `json:"decision"`
	Reason         string             `json:"reason,omitempty"`
	RequiredAction string             `json:"requiredAction,omitempty"` // Action user needs to take
	ClarityScore   int                `json:"clarityScore,omitempty"`   // Clarity score if applicable
	CanForce       bool               `json:"canForce"`                 // Whether user can force proceed with command (e.g., "start_design"). Only meaningful when decision is NeedConfirmation or Denied.
	SkipClarity    bool               `json:"skipClarity,omitempty"`    // Whether user used force command to bypass clarity check. Only set when SkipClarityCheck context is true.
}

// IsSkippedClarity returns true if user used force command to bypass clarity check.
func (r TransitionResult) IsSkippedClarity() bool {
	return r.SkipClarity
}

// IsAllowed returns true if transition is allowed.
func (r TransitionResult) IsAllowed() bool {
	return r.Decision.IsAllowed()
}

// NeedsConfirmation returns true if transition needs confirmation.
func (r TransitionResult) NeedsConfirmation() bool {
	return r.Decision.NeedsConfirmation()
}

// IsDenied returns true if transition is denied.
func (r TransitionResult) IsDenied() bool {
	return r.Decision.IsDenied()
}

// HasRequiredAction returns true if there is a required action for the user.
func (r TransitionResult) HasRequiredAction() bool {
	return r.RequiredAction != ""
}

// Error returns an error representation of a denied result.
// Includes both the reason and required action (if any) for user guidance.
func (r TransitionResult) Error() error {
	if r.IsDenied() {
		if r.HasRequiredAction() {
			return fmt.Errorf("transition denied: %s (required action: %s)", r.Reason, r.RequiredAction)
		}
		return fmt.Errorf("transition denied: %s", r.Reason)
	}
	return nil
}

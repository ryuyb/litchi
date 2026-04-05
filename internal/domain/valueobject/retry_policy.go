package valueobject

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// BackoffStrategy represents the retry backoff strategy type.
type BackoffStrategy string

const (
	// BackoffStrategyExponential uses exponential backoff (delay = base * 2^attempt).
	BackoffStrategyExponential BackoffStrategy = "exponential"
	// BackoffStrategyLinear uses linear backoff (delay = base * attempt).
	BackoffStrategyLinear BackoffStrategy = "linear"
	// BackoffStrategyConstant uses constant backoff (delay = base).
	BackoffStrategyConstant BackoffStrategy = "constant"
)

// DefaultBackoffConfig provides default backoff configuration.
// Note: This is a value type. Modifications to copies do not affect the global default.
// Use NewBackoffConfig() to get a copy for customization.
var DefaultBackoffConfig = BackoffConfig{
	Strategy:     BackoffStrategyExponential,
	BaseDelay:    1 * time.Second,
	MaxDelay:     60 * time.Second,
	Multiplier:   2.0,
	JitterFactor: 0.1, // Add up to 10% jitter to prevent thundering herd
}

// BackoffConfig defines the configuration for retry backoff.
type BackoffConfig struct {
	Strategy     BackoffStrategy `json:"strategy"`     // Backoff strategy type
	BaseDelay    time.Duration   `json:"baseDelay"`    // Initial delay duration
	MaxDelay     time.Duration   `json:"maxDelay"`     // Maximum delay cap
	Multiplier   float64         `json:"multiplier"`   // Multiplier for exponential backoff
	JitterFactor float64         `json:"jitterFactor"` // Jitter factor (0.0 to 1.0)
}

// NewBackoffConfig creates a new BackoffConfig with default values.
func NewBackoffConfig() BackoffConfig {
	return DefaultBackoffConfig
}

// BackoffConfigWithStrategy creates a BackoffConfig with specified strategy.
func BackoffConfigWithStrategy(strategy BackoffStrategy) BackoffConfig {
	config := DefaultBackoffConfig
	config.Strategy = strategy
	return config
}

// BackoffConfigWithBaseDelay creates a BackoffConfig with specified base delay.
func BackoffConfigWithBaseDelay(baseDelay time.Duration) BackoffConfig {
	config := DefaultBackoffConfig
	config.BaseDelay = baseDelay
	return config
}

// CalculateDelay calculates the delay for a given retry attempt.
// The attempt number starts from 1 (first retry).
// For exponential: delay = baseDelay * multiplier^(attempt-1)
// For linear: delay = baseDelay * attempt
// For constant: delay = baseDelay
func (c BackoffConfig) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return c.BaseDelay
	}

	var delay time.Duration

	switch c.Strategy {
	case BackoffStrategyExponential:
		// delay = baseDelay * multiplier^(attempt-1)
		// attempt=1 -> baseDelay
		// attempt=2 -> baseDelay * multiplier
		// attempt=3 -> baseDelay * multiplier^2
		delay = time.Duration(float64(c.BaseDelay) * math.Pow(c.Multiplier, float64(attempt-1)))
	case BackoffStrategyLinear:
		// delay = baseDelay * attempt
		delay = c.BaseDelay * time.Duration(attempt)
	case BackoffStrategyConstant:
		// delay = baseDelay (constant)
		delay = c.BaseDelay
	default:
		delay = c.BaseDelay
	}

	// Apply max delay cap
	if delay > c.MaxDelay {
		delay = c.MaxDelay
	}

	// Apply jitter to prevent synchronized retries (thundering herd problem).
	// This uses "additive jitter" mode: delay += random(0, delay * jitterFactor)
	// Result range: [delay, delay * (1 + jitterFactor)]
	// This ensures minimum delay is never reduced below the calculated backoff,
	// which is appropriate for rate-limited APIs and resource protection.
	// For "full jitter" mode that randomizes within [delay*(1-jitterFactor), delay*(1+jitterFactor)],
	// use JitterFactor=0 and apply custom jitter at the call site.
	if c.JitterFactor > 0 {
		jitterMax := float64(delay) * c.JitterFactor
		// rand.Float64() returns a random value in [0.0, 1.0)
		// Go 1.20+ rand is globally seeded and thread-safe
		jitterAmount := rand.Float64() * jitterMax
		delay += time.Duration(jitterAmount)
	}

	return delay
}

// Validate validates the BackoffConfig.
func (c BackoffConfig) Validate() error {
	if c.BaseDelay <= 0 {
		return fmt.Errorf("base delay must be positive: %v", c.BaseDelay)
	}
	if c.MaxDelay <= 0 {
		return fmt.Errorf("max delay must be positive: %v", c.MaxDelay)
	}
	if c.BaseDelay > c.MaxDelay {
		return fmt.Errorf("base delay (%v) cannot exceed max delay (%v)", c.BaseDelay, c.MaxDelay)
	}
	if c.Multiplier <= 0 {
		return fmt.Errorf("multiplier must be positive: %v", c.Multiplier)
	}
	if c.JitterFactor < 0 || c.JitterFactor > 1 {
		return fmt.Errorf("jitter factor must be between 0 and 1: %v", c.JitterFactor)
	}
	return nil
}

// RetryPolicy defines the complete retry policy for tasks.
type RetryPolicy struct {
	MaxRetries    int          `json:"maxRetries"`    // Maximum number of retries (default: 3)
	BackoffConfig BackoffConfig `json:"backoffConfig"` // Backoff configuration
}

// DefaultRetryPolicy provides the default retry policy (3 retries with exponential backoff).
// Note: This is a value type. Modifications to copies do not affect the global default.
// Use NewRetryPolicy() to get a copy for customization.
var DefaultRetryPolicy = RetryPolicy{
	MaxRetries:    3,
	BackoffConfig: DefaultBackoffConfig,
}

// NewRetryPolicy creates a new RetryPolicy with default values.
func NewRetryPolicy() RetryPolicy {
	return DefaultRetryPolicy
}

// RetryPolicyWithMaxRetries creates a RetryPolicy with specified max retries.
func RetryPolicyWithMaxRetries(maxRetries int) RetryPolicy {
	policy := DefaultRetryPolicy
	policy.MaxRetries = maxRetries
	return policy
}

// CanRetry checks if a task can be retried based on current retry count.
func (p RetryPolicy) CanRetry(currentRetryCount int) bool {
	return currentRetryCount < p.MaxRetries
}

// GetNextRetryDelay returns the delay before the next retry attempt.
// The attempt number is currentRetryCount + 1.
func (p RetryPolicy) GetNextRetryDelay(currentRetryCount int) time.Duration {
	return p.BackoffConfig.CalculateDelay(currentRetryCount + 1)
}

// Validate validates the RetryPolicy.
func (p RetryPolicy) Validate() error {
	if p.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative: %d", p.MaxRetries)
	}
	return p.BackoffConfig.Validate()
}

// FinalFailureAction defines what action to take when a task finally fails (all retries exhausted).
type FinalFailureAction string

const (
	// FinalFailureActionPauseSession pauses the entire work session for manual intervention.
	FinalFailureActionPauseSession FinalFailureAction = "pause_session"
	// FinalFailureActionSkipTask skips the failed task and continues with remaining tasks.
	FinalFailureActionSkipTask FinalFailureAction = "skip_task"
	// FinalFailureActionRollback rolls back to a previous stage (design or clarification).
	FinalFailureActionRollback FinalFailureAction = "rollback"
	// FinalFailureActionTerminate terminates the entire work session.
	FinalFailureActionTerminate FinalFailureAction = "terminate"
)

// FinalFailureHandling defines how to handle a task that has exhausted all retries.
type FinalFailureHandling struct {
	Action        FinalFailureAction `json:"action"`        // Action to take
	RollbackStage Stage              `json:"rollbackStage"` // Target stage for rollback (if action is rollback)
	NotifyUser    bool               `json:"notifyUser"`    // Whether to notify user
	Reason        string             `json:"reason"`        // Reason for the action
}

// DefaultFinalFailureHandling provides the default final failure handling (pause session).
var DefaultFinalFailureHandling = FinalFailureHandling{
	Action:     FinalFailureActionPauseSession,
	NotifyUser: true,
	Reason:     "task exhausted all retry attempts",
}

// NewFinalFailureHandling creates a new FinalFailureHandling with specified action.
func NewFinalFailureHandling(action FinalFailureAction) FinalFailureHandling {
	return FinalFailureHandling{
		Action:     action,
		NotifyUser: true,
		Reason:     "task exhausted all retry attempts",
	}
}

// FinalFailureHandlingWithRollback creates a FinalFailureHandling with rollback action.
func FinalFailureHandlingWithRollback(targetStage Stage) FinalFailureHandling {
	return FinalFailureHandling{
		Action:        FinalFailureActionRollback,
		RollbackStage: targetStage,
		NotifyUser:    true,
		Reason:        "task exhausted all retry attempts, rolling back to " + targetStage.String(),
	}
}

// RetryRecord tracks the history of retry attempts for a task.
type RetryRecord struct {
	Attempt     int       `json:"attempt"`     // Retry attempt number (1-based)
	Timestamp   time.Time `json:"timestamp"`   // When this retry was attempted
	Delay       time.Duration `json:"delay"`   // Delay before this retry
	Result      string    `json:"result"`      // Result of this retry attempt
	ErrorReason string    `json:"errorReason"` // Error reason if retry failed
}

// NewRetryRecord creates a new RetryRecord.
func NewRetryRecord(attempt int, delay time.Duration, result, errorReason string) RetryRecord {
	return RetryRecord{
		Attempt:     attempt,
		Timestamp:   time.Now(),
		Delay:       delay,
		Result:      result,
		ErrorReason: errorReason,
	}
}

// RetryContext holds the complete retry context for a task.
type RetryContext struct {
	CurrentAttempt int            `json:"currentAttempt"` // Current retry attempt number
	MaxAttempts    int            `json:"maxAttempts"`    // Maximum allowed attempts
	NextDelay      time.Duration  `json:"nextDelay"`      // Delay for next retry
	History        []RetryRecord  `json:"history"`        // History of retry attempts
	Policy         RetryPolicy    `json:"policy"`         // Retry policy being used
	StartTime      time.Time      `json:"startTime"`      // When retry process started
}

// NewRetryContext creates a new RetryContext with the given policy.
func NewRetryContext(policy RetryPolicy) *RetryContext {
	return &RetryContext{
		CurrentAttempt: 0,
		MaxAttempts:    policy.MaxRetries,
		NextDelay:      policy.BackoffConfig.BaseDelay,
		History:        []RetryRecord{},
		Policy:         policy,
		StartTime:      time.Now(),
	}
}

// RecordAttempt records a retry attempt result.
func (c *RetryContext) RecordAttempt(success bool, errorReason string) {
	result := "success"
	if !success {
		result = "failed"
	}

	record := NewRetryRecord(c.CurrentAttempt+1, c.NextDelay, result, errorReason)
	c.History = append(c.History, record)

	if !success {
		c.CurrentAttempt++
		c.NextDelay = c.Policy.GetNextRetryDelay(c.CurrentAttempt)
	}
}

// CanRetry returns true if more retries are available.
func (c *RetryContext) CanRetry() bool {
	return c.CurrentAttempt < c.MaxAttempts
}

// IsExhausted returns true if all retry attempts have been exhausted.
func (c *RetryContext) IsExhausted() bool {
	return c.CurrentAttempt >= c.MaxAttempts
}

// GetTotalWaitDuration returns the total wait time (sum of all delays) spent between retries.
// Note: This does NOT include the actual task execution time, only the backoff delays.
func (c *RetryContext) GetTotalWaitDuration() time.Duration {
	total := time.Duration(0)
	for _, record := range c.History {
		total += record.Delay
	}
	return total
}

// GetNextRetryTime returns the time when the next retry should occur.
func (c *RetryContext) GetNextRetryTime() time.Time {
	if c.IsExhausted() {
		return time.Time{} // Zero time indicates no more retries
	}
	return time.Now().Add(c.NextDelay)
}
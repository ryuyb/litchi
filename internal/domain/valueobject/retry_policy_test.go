package valueobject

import (
	"testing"
	"time"
)

func TestBackoffConfigCalculateDelay(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		config    BackoffConfig
		attempt   int
		minDelay  time.Duration
		maxDelay  time.Duration
	}{
		{
			name:     "exponential_backoff_attempt_1",
			config:   BackoffConfig{Strategy: BackoffStrategyExponential, BaseDelay: 1 * time.Second, MaxDelay: 60 * time.Second, Multiplier: 2.0, JitterFactor: 0},
			attempt:  1,
			minDelay: 1 * time.Second,
			maxDelay: 1 * time.Second, // base * 2^0 = 1s
		},
		{
			name:     "exponential_backoff_attempt_2",
			config:   BackoffConfig{Strategy: BackoffStrategyExponential, BaseDelay: 1 * time.Second, MaxDelay: 60 * time.Second, Multiplier: 2.0, JitterFactor: 0},
			attempt:  2,
			minDelay: 2 * time.Second, // base * 2^1 = 2s
			maxDelay: 2 * time.Second,
		},
		{
			name:     "exponential_backoff_attempt_3",
			config:   BackoffConfig{Strategy: BackoffStrategyExponential, BaseDelay: 1 * time.Second, MaxDelay: 60 * time.Second, Multiplier: 2.0, JitterFactor: 0},
			attempt:  3,
			minDelay: 4 * time.Second, // base * 2^2 = 4s
			maxDelay: 4 * time.Second,
		},
		{
			name:     "exponential_backoff_max_cap",
			config:   BackoffConfig{Strategy: BackoffStrategyExponential, BaseDelay: 1 * time.Second, MaxDelay: 10 * time.Second, Multiplier: 2.0, JitterFactor: 0},
			attempt:  10,
			minDelay: 10 * time.Second, // capped at max (actual would be 1s * 2^9 = 512s)
			maxDelay: 10 * time.Second,
		},
		{
			name:     "linear_backoff",
			config:   BackoffConfig{Strategy: BackoffStrategyLinear, BaseDelay: 2 * time.Second, MaxDelay: 60 * time.Second, Multiplier: 1.0, JitterFactor: 0},
			attempt:  3,
			minDelay: 6 * time.Second, // base * 3
			maxDelay: 6 * time.Second,
		},
		{
			name:     "constant_backoff",
			config:   BackoffConfig{Strategy: BackoffStrategyConstant, BaseDelay: 5 * time.Second, MaxDelay: 60 * time.Second, Multiplier: 1.0, JitterFactor: 0},
			attempt:  5,
			minDelay: 5 * time.Second,
			maxDelay: 5 * time.Second,
		},
		{
			name:     "zero_attempt_returns_base",
			config:   BackoffConfig{Strategy: BackoffStrategyExponential, BaseDelay: 1 * time.Second, MaxDelay: 60 * time.Second, Multiplier: 2.0, JitterFactor: 0},
			attempt:  0,
			minDelay: 1 * time.Second,
			maxDelay: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			delay := tt.config.CalculateDelay(tt.attempt)
			if delay < tt.minDelay || delay > tt.maxDelay {
				t.Errorf("CalculateDelay(%d) = %v, expected between %v and %v",
					tt.attempt, delay, tt.minDelay, tt.maxDelay)
			}
		})
	}
}

func TestBackoffConfigWithJitter(t *testing.T) {
	t.Parallel()
	config := BackoffConfig{
		Strategy:     BackoffStrategyExponential,
		BaseDelay:    1 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.1,
	}

	// Run multiple times to verify jitter is applied with randomness
	for i := 1; i <= 5; i++ {
		delay := config.CalculateDelay(i)
		// Base delay for attempt i with exponential backoff: 1s * 2^(i-1)
		baseExpected := 1 * time.Second * time.Duration(1<<(i-1))
		maxExpected := baseExpected + time.Duration(float64(baseExpected)*config.JitterFactor)

		// With random jitter, delay should be between baseExpected and baseExpected + jitterMax
		if delay < baseExpected {
			t.Errorf("CalculateDelay(%d) = %v, expected >= %v (base)",
				i, delay, baseExpected)
		}
		if delay > maxExpected {
			t.Errorf("CalculateDelay(%d) = %v, expected <= %v (base + jitter)",
				i, delay, maxExpected)
		}
	}

	// Verify randomness by checking that multiple calls produce different values
	delays := make([]time.Duration, 10)
	for j := 0; j < 10; j++ {
		delays[j] = config.CalculateDelay(1)
	}

	// At least some delays should be different (very likely with random jitter)
	uniqueCount := 0
	for j := 0; j < len(delays); j++ {
		for k := j + 1; k < len(delays); k++ {
			if delays[j] != delays[k] {
				uniqueCount++
			}
		}
	}

	// With random jitter, we expect at least some unique values
	// (probability of all 10 being identical is extremely low with rand.Float64)
	if uniqueCount == 0 {
		t.Errorf("Jitter should produce random delays, but all 10 calls returned the same value: %v", delays[0])
	}
}

func TestBackoffConfigValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  BackoffConfig
		hasError bool
	}{
		{
			name:     "valid_config",
			config:   DefaultBackoffConfig,
			hasError: false,
		},
		{
			name:     "invalid_base_delay",
			config:   BackoffConfig{BaseDelay: 0, MaxDelay: 60 * time.Second, Multiplier: 2.0},
			hasError: true,
		},
		{
			name:     "invalid_max_delay",
			config:   BackoffConfig{BaseDelay: 1 * time.Second, MaxDelay: 0, Multiplier: 2.0},
			hasError: true,
		},
		{
			name:     "base_exceeds_max",
			config:   BackoffConfig{BaseDelay: 100 * time.Second, MaxDelay: 60 * time.Second, Multiplier: 2.0},
			hasError: true,
		},
		{
			name:     "invalid_multiplier",
			config:   BackoffConfig{BaseDelay: 1 * time.Second, MaxDelay: 60 * time.Second, Multiplier: 0},
			hasError: true,
		},
		{
			name:     "invalid_jitter_negative",
			config:   BackoffConfig{BaseDelay: 1 * time.Second, MaxDelay: 60 * time.Second, Multiplier: 2.0, JitterFactor: -0.1},
			hasError: true,
		},
		{
			name:     "invalid_jitter_over_1",
			config:   BackoffConfig{BaseDelay: 1 * time.Second, MaxDelay: 60 * time.Second, Multiplier: 2.0, JitterFactor: 1.5},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.hasError && err == nil {
				t.Errorf("Validate() expected error, got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestRetryPolicyCanRetry(t *testing.T) {
	t.Parallel()
	policy := DefaultRetryPolicy

	tests := []struct {
		name           string
		retryCount     int
		expectedCanRetry bool
	}{
		{"zero_retries", 0, true},
		{"one_retry", 1, true},
		{"two_retries", 2, true},
		{"max_retries_reached", 3, false},
		{"over_max", 4, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := policy.CanRetry(tt.retryCount)
			if result != tt.expectedCanRetry {
				t.Errorf("CanRetry(%d) = %v, expected %v", tt.retryCount, result, tt.expectedCanRetry)
			}
		})
	}
}

func TestRetryPolicyGetNextRetryDelay(t *testing.T) {
	t.Parallel()
	policy := DefaultRetryPolicy

	tests := []struct {
		name         string
		retryCount   int
		expectedMin  time.Duration
		expectedMax  time.Duration
	}{
		{"first_retry", 0, 1 * time.Second, 2 * time.Second},    // base * 2^1
		{"second_retry", 1, 2 * time.Second, 4 * time.Second},   // base * 2^2
		{"third_retry", 2, 4 * time.Second, 8 * time.Second},    // base * 2^3
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			delay := policy.GetNextRetryDelay(tt.retryCount)
			// Without jitter, delay should be predictable
			if delay < tt.expectedMin {
				t.Errorf("GetNextRetryDelay(%d) = %v, expected >= %v",
					tt.retryCount, delay, tt.expectedMin)
			}
		})
	}
}

func TestRetryPolicyValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		policy   RetryPolicy
		hasError bool
	}{
		{"valid_policy", DefaultRetryPolicy, false},
		{"negative_max_retries", RetryPolicy{MaxRetries: -1}, true},
		{"invalid_backoff", RetryPolicy{MaxRetries: 3, BackoffConfig: BackoffConfig{BaseDelay: 0}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.policy.Validate()
			if tt.hasError && err == nil {
				t.Errorf("Validate() expected error, got none")
			}
			if !tt.hasError && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestRetryContext(t *testing.T) {
	t.Parallel()
	policy := DefaultRetryPolicy
	ctx := NewRetryContext(policy)

	// Initial state
	if ctx.CurrentAttempt != 0 {
		t.Errorf("Initial CurrentAttempt = %d, expected 0", ctx.CurrentAttempt)
	}
	if !ctx.CanRetry() {
		t.Errorf("Initial CanRetry should be true")
	}
	if ctx.IsExhausted() {
		t.Errorf("Initial IsExhausted should be false")
	}

	// Record failed attempts
	ctx.RecordAttempt(false, "error 1")
	if ctx.CurrentAttempt != 1 {
		t.Errorf("After first failure, CurrentAttempt = %d, expected 1", ctx.CurrentAttempt)
	}
	if !ctx.CanRetry() {
		t.Errorf("After first failure, CanRetry should still be true")
	}

	ctx.RecordAttempt(false, "error 2")
	ctx.RecordAttempt(false, "error 3")

	// After 3 failures (max retries), should be exhausted
	if !ctx.IsExhausted() {
		t.Errorf("After 3 failures, IsExhausted should be true")
	}
	if ctx.CanRetry() {
		t.Errorf("After exhausting retries, CanRetry should be false")
	}

	// History should have 3 records
	if len(ctx.History) != 3 {
		t.Errorf("History length = %d, expected 3", len(ctx.History))
	}
}

func TestRetryContextGetTotalWaitDuration(t *testing.T) {
	t.Parallel()
	policy := RetryPolicy{
		MaxRetries: 2,
		BackoffConfig: BackoffConfig{
			Strategy:   BackoffStrategyConstant,
			BaseDelay:  5 * time.Second,
			MaxDelay:   60 * time.Second,
			Multiplier: 1.0,
			JitterFactor: 0,
		},
	}
	ctx := NewRetryContext(policy)

	ctx.RecordAttempt(false, "error 1")
	ctx.RecordAttempt(false, "error 2")

	// Total wait duration should be 2 * 5s = 10s
	total := ctx.GetTotalWaitDuration()
	if total != 10*time.Second {
		t.Errorf("GetTotalWaitDuration() = %v, expected 10s", total)
	}
}

func TestFinalFailureHandling(t *testing.T) {
	t.Parallel()
	// Test default handling
	defaultHandling := DefaultFinalFailureHandling
	if defaultHandling.Action != FinalFailureActionPauseSession {
		t.Errorf("Default action = %s, expected pause_session", defaultHandling.Action)
	}
	if !defaultHandling.NotifyUser {
		t.Errorf("Default NotifyUser should be true")
	}

	// Test rollback handling
	rollbackHandling := FinalFailureHandlingWithRollback(StageDesign)
	if rollbackHandling.Action != FinalFailureActionRollback {
		t.Errorf("Rollback action = %s, expected rollback", rollbackHandling.Action)
	}
	if rollbackHandling.RollbackStage != StageDesign {
		t.Errorf("RollbackStage = %s, expected design", rollbackHandling.RollbackStage)
	}
}

func TestNewRetryRecord(t *testing.T) {
	t.Parallel()
	record := NewRetryRecord(1, 5*time.Second, "failed", "connection timeout")

	if record.Attempt != 1 {
		t.Errorf("Attempt = %d, expected 1", record.Attempt)
	}
	if record.Delay != 5*time.Second {
		t.Errorf("Delay = %v, expected 5s", record.Delay)
	}
	if record.Result != "failed" {
		t.Errorf("Result = %s, expected failed", record.Result)
	}
	if record.ErrorReason != "connection timeout" {
		t.Errorf("ErrorReason = %s, expected connection timeout", record.ErrorReason)
	}
	if record.Timestamp.IsZero() {
		t.Errorf("Timestamp should not be zero")
	}
}
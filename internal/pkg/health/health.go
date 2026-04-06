// Package health provides health check interfaces and types.
package health

import (
	"context"
	"time"
)

// CheckResult represents the result of a health check.
type CheckResult struct {
	Name      string         // Component name (e.g., "database", "github", "git")
	Status    string         // "pass", "fail", "warn"
	Message   string         // Human-readable status message
	LatencyMs int            // Check latency in milliseconds
	Error     string         // Error message if failed
	Details   map[string]any // Additional component-specific details
}

// Checker defines the interface for health check providers.
// Components that can be health-checked (database, external APIs, etc.)
// should implement this interface.
type Checker interface {
	// Name returns the component name for logging and response.
	Name() string

	// Check performs the health check and returns the result.
	Check(ctx context.Context) CheckResult
}

// TimedCheck wraps a health check function with timing.
// This is a helper for implementing Checker.
func TimedCheck(name string, checkFunc func(ctx context.Context) (status string, message string, details map[string]any, err error)) CheckResult {
	start := time.Now()
	status, message, details, err := checkFunc(context.Background())
	latency := time.Since(start)

	result := CheckResult{
		Name:      name,
		Status:    status,
		Message:   message,
		LatencyMs: int(latency.Milliseconds()),
		Details:   details,
	}

	if err != nil {
		result.Error = err.Error()
		if status == "" {
			result.Status = "fail"
		}
	}

	return result
}
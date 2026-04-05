// Package retry provides error handling and retry strategies for Agent execution.
package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// RetryHandler defines the interface for retry handling.
type RetryHandler interface {
	// ExecuteWithRetry executes a function with retry logic.
	ExecuteWithRetry(
		ctx context.Context,
		req *service.AgentRequest,
		policy valueobject.RetryPolicy,
		execFunc func(ctx context.Context, req *service.AgentRequest) (*service.AgentResponse, error),
	) (*service.AgentResponse, error)

	// ShouldRetry determines if a retry should be attempted.
	ShouldRetry(err error, retryCount int, policy valueobject.RetryPolicy) bool

	// CalculateBackoff calculates the backoff duration.
	CalculateBackoff(retryCount int, policy valueobject.RetryPolicy) time.Duration
}

// ErrorClassifier defines the interface for error classification.
type ErrorClassifier interface {
	// Classify classifies an error.
	Classify(err error) *ClassifiedError
}

// ClassifiedError represents a classified error.
type ClassifiedError struct {
	OriginalError error
	Type          ErrorType
	Severity      SeverityLevel
	Recoverable   bool
	Retryable     bool
	Category      string
}

// ErrorType represents the type of error.
type ErrorType string

const (
	ErrorTypeProcess    ErrorType = "process"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypePermission ErrorType = "permission"
	ErrorTypeExecution  ErrorType = "execution"
	ErrorTypeContext    ErrorType = "context"
	ErrorTypeNetwork    ErrorType = "network"
)

// SeverityLevel represents error severity.
type SeverityLevel int

const (
	SeverityCritical SeverityLevel = 1
	SeverityHigh     SeverityLevel = 2
	SeverityMedium   SeverityLevel = 3
	SeverityLow      SeverityLevel = 4
)

// DefaultRetryHandler is the default implementation.
type DefaultRetryHandler struct {
	classifier ErrorClassifier
	logger     *zap.Logger
}

// NewDefaultRetryHandler creates a new default retry handler.
func NewDefaultRetryHandler() *DefaultRetryHandler {
	return &DefaultRetryHandler{
		classifier: NewDefaultErrorClassifier(),
	}
}

// NewRetryHandler creates a retry handler with custom classifier.
func NewRetryHandler(classifier ErrorClassifier, logger *zap.Logger) *DefaultRetryHandler {
	return &DefaultRetryHandler{
		classifier: classifier,
		logger:     logger,
	}
}

// ExecuteWithRetry executes a function with retry logic.
// The function is executed once initially, then retried up to MaxRetries times on failure.
// Total maximum executions = 1 (initial) + MaxRetries (retries).
func (h *DefaultRetryHandler) ExecuteWithRetry(
	ctx context.Context,
	req *service.AgentRequest,
	policy valueobject.RetryPolicy,
	execFunc func(ctx context.Context, req *service.AgentRequest) (*service.AgentResponse, error),
) (*service.AgentResponse, error) {
	var lastErr error
	var lastResponse *service.AgentResponse

	retryCtx := valueobject.NewRetryContext(policy)

	// Loop: initial execution (executionNum=0) + up to MaxRetries retries
	// Total iterations: MaxRetries + 1
	for executionNum := 0; executionNum <= policy.MaxRetries; executionNum++ {
		// Execute
		response, err := execFunc(ctx, req)

		if err == nil {
			// Success
			retryCtx.RecordAttempt(true, "")
			return response, nil
		}

		// Classify error
		classified := h.classifier.Classify(err)
		lastErr = err
		if response != nil {
			lastResponse = response
		}

		// Record failed attempt
		retryCtx.RecordAttempt(false, err.Error())

		// Check if retryable (executionNum equals current retry count)
		if !h.ShouldRetry(err, executionNum, policy) {
			if h.logger != nil {
				h.logger.Warn("error not retryable, stopping",
					zap.String("sessionId", req.SessionID.String()),
					zap.Int("executionNum", executionNum),
					zap.Int("retriesDone", executionNum),
					zap.String("errorType", string(classified.Type)),
					zap.Error(err),
				)
			}
			break
		}

		// Check if context cancelled
		if ctx.Err() != nil {
			return lastResponse, ctx.Err()
		}

		// Calculate backoff
		backoff := h.CalculateBackoff(executionNum, policy)

		if h.logger != nil {
			h.logger.Warn("execution failed, retrying",
				zap.String("sessionId", req.SessionID.String()),
				zap.Int("executionNum", executionNum),
				zap.Int("maxRetries", policy.MaxRetries),
				zap.Duration("backoff", backoff),
				zap.Error(err),
			)
		}

		// Wait for backoff
		select {
		case <-ctx.Done():
			return lastResponse, ctx.Err()
		case <-time.After(backoff):
			// Continue to next execution
		}
	}

	// All retries exhausted
	if lastResponse != nil && lastResponse.Error == nil {
		lastResponse.Error = &service.AgentErrorInfo{
			Code:       errors.GetCode(lastErr),
			Category:   "execution",
			Message:    lastErr.Error(),
			Retryable:  false,
			RetryCount: retryCtx.CurrentAttempt,
		}
	}

	return lastResponse, lastErr
}

// ShouldRetry determines if a retry should be attempted.
func (h *DefaultRetryHandler) ShouldRetry(err error, retryCount int, policy valueobject.RetryPolicy) bool {
	if retryCount >= policy.MaxRetries {
		return false
	}

	classified := h.classifier.Classify(err)
	return classified.Retryable
}

// CalculateBackoff calculates the backoff duration.
func (h *DefaultRetryHandler) CalculateBackoff(retryCount int, policy valueobject.RetryPolicy) time.Duration {
	return policy.GetNextRetryDelay(retryCount)
}

// DefaultErrorClassifier is the default error classifier.
type DefaultErrorClassifier struct{}

// NewDefaultErrorClassifier creates a new default error classifier.
func NewDefaultErrorClassifier() *DefaultErrorClassifier {
	return &DefaultErrorClassifier{}
}

// Classify classifies an error.
func (c *DefaultErrorClassifier) Classify(err error) *ClassifiedError {
	classified := &ClassifiedError{
		OriginalError: err,
		Type:          ErrorTypeExecution,
		Severity:      SeverityMedium,
		Recoverable:   false,
		Retryable:     false,
	}

	// Check specific error types
	if errors.Is(err, errors.ErrAgentTimeout) {
		classified.Type = ErrorTypeTimeout
		classified.Severity = SeverityMedium
		classified.Retryable = true
		classified.Category = "timeout"
		return classified
	}

	if errors.Is(err, errors.ErrAgentProcessCrash) {
		classified.Type = ErrorTypeProcess
		classified.Severity = SeverityHigh
		classified.Retryable = true
		classified.Category = "process"
		return classified
	}

	if errors.Is(err, errors.ErrAgentPermissionDenied) {
		classified.Type = ErrorTypePermission
		classified.Severity = SeverityHigh
		classified.Retryable = false // Permission errors need user intervention
		classified.Category = "permission"
		return classified
	}

	if errors.Is(err, errors.ErrAgentContextLost) {
		classified.Type = ErrorTypeContext
		classified.Severity = SeverityHigh
		classified.Retryable = true
		classified.Category = "context"
		return classified
	}

	if errors.Is(err, errors.ErrAgentExecutionFail) {
		classified.Type = ErrorTypeExecution
		classified.Severity = SeverityMedium
		classified.Retryable = true // May be retryable
		classified.Category = "execution"
		return classified
	}

	// Default: execution error, not retryable
	classified.Category = "unknown"
	return classified
}

// RecoveryStrategy determines the recovery strategy for an error.
func RecoveryStrategy(classified *ClassifiedError) string {
	switch classified.Type {
	case ErrorTypeTimeout:
		return "retry_with_extended_timeout"
	case ErrorTypeProcess:
		return "restart_process"
	case ErrorTypeContext:
		return "reload_context_from_db"
	case ErrorTypePermission:
		return "notify_user_for_approval"
	case ErrorTypeNetwork:
		return "retry_with_backoff"
	default:
		return "no_recovery"
	}
}

// IsRecoverable checks if an error is recoverable.
func IsRecoverable(err error) bool {
	classifier := NewDefaultErrorClassifier()
	classified := classifier.Classify(err)
	return classified.Recoverable
}

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	classifier := NewDefaultErrorClassifier()
	classified := classifier.Classify(err)
	return classified.Retryable
}

// GetRetryCount extracts retry count from response.
func GetRetryCount(response *service.AgentResponse) int {
	if response != nil && response.Error != nil {
		return response.Error.RetryCount
	}
	return 0
}

// BuildRetryRequest builds a retry request from the original.
// Creates a deep copy of the Context to avoid modifying the original request.
func BuildRetryRequest(original *service.AgentRequest, attemptNum int) *service.AgentRequest {
	req := *original // Shallow copy of struct

	// Deep copy Context to avoid modifying original
	if original.Context != nil {
		req.Context = &service.AgentContext{
			IssueTitle:      original.Context.IssueTitle,
			IssueBody:       original.Context.IssueBody,
			Repository:      original.Context.Repository,
			Branch:          original.Context.Branch,
			DesignContent:   original.Context.DesignContent,
			ClarifiedPoints: append([]string(nil), original.Context.ClarifiedPoints...),
		}
		// Deep copy Tasks slice
		if len(original.Context.Tasks) > 0 {
			req.Context.Tasks = make([]service.TaskContext, len(original.Context.Tasks))
			copy(req.Context.Tasks, original.Context.Tasks)
		}
		// Deep copy History slice
		if len(original.Context.History) > 0 {
			req.Context.History = make([]service.HistoryEntry, len(original.Context.History))
			copy(req.Context.History, original.Context.History)
		}
	} else {
		req.Context = &service.AgentContext{}
	}

	// Add retry metadata to context
	req.Context.History = append(req.Context.History, service.HistoryEntry{
		Timestamp: time.Now(),
		Stage:     original.Stage,
		Action:    "retry",
		Result:    fmt.Sprintf("retry attempt %d", attemptNum),
	})
	return &req
}
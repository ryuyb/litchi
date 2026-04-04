package event

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Handler processes a domain event.
// Handlers are responsible for reacting to domain events, such as:
// - Updating projections/read models
// - Sending notifications (WebSocket, email, etc.)
// - Triggering side effects
// - Recording audit logs
type Handler func(ctx context.Context, event DomainEvent) error

// Dispatcher manages event distribution to registered handlers.
// It supports both synchronous (blocking) and asynchronous (non-blocking)
// dispatch modes to accommodate different use cases:
// - Synchronous: for critical handlers that must complete before proceeding
// - Asynchronous: for handlers that can be processed in background
type Dispatcher struct {
	logger *zap.Logger

	// handlers maps event type to list of handlers
	// Using mutex for thread-safe access since handlers may be registered at runtime
	handlers map[string][]Handler
	mu       sync.RWMutex

	// asyncWorkers controls the number of concurrent async handler executions
	asyncWorkers int
	asyncSem     chan struct{} // semaphore for controlling concurrency
}

// DispatcherOption configures the Dispatcher.
type DispatcherOption func(*Dispatcher)

// WithLogger sets the logger for the dispatcher.
// If nil is passed, the default no-op logger is retained.
func WithLogger(logger *zap.Logger) DispatcherOption {
	return func(d *Dispatcher) {
		if logger != nil {
			d.logger = logger
		}
	}
}

// WithAsyncWorkers sets the number of concurrent async handler workers.
// Default is 10 if not specified.
func WithAsyncWorkers(n int) DispatcherOption {
	return func(d *Dispatcher) {
		if n > 0 {
			d.asyncWorkers = n
		}
	}
}

// NewDispatcher creates a new event dispatcher.
func NewDispatcher(opts ...DispatcherOption) *Dispatcher {
	d := &Dispatcher{
		handlers:     make(map[string][]Handler),
		asyncWorkers: 10, // default concurrency
		logger:       zap.NewNop(), // default no-op logger
	}

	for _, opt := range opts {
		opt(d)
	}

	d.asyncSem = make(chan struct{}, d.asyncWorkers)
	return d
}

// Register adds a handler for a specific event type.
// Multiple handlers can be registered for the same event type.
// Handlers are called in the order they were registered.
func (d *Dispatcher) Register(eventType string, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.handlers[eventType] = append(d.handlers[eventType], handler)

	d.logger.Debug("event handler registered",
		zap.String("eventType", eventType),
		zap.Int("handlerCount", len(d.handlers[eventType])),
	)
}

// RegisterAll adds a handler that receives all events (wildcard handler).
// The handler will be called for every event type.
// Use sparingly - prefer specific handlers for most cases.
func (d *Dispatcher) RegisterAll(handler Handler) {
	d.Register("*", handler)
}

// DispatchSync dispatches an event synchronously to all registered handlers.
// This blocks until all handlers have completed.
// Use for critical handlers where you need guaranteed execution before proceeding.
//
// IMPORTANT: Handlers are executed in order, but ALL handlers run even if one fails.
// This is by design for event notification scenarios where we want to maximize delivery.
// Only the first error is returned; subsequent errors are logged but not returned.
//
// For transactional requirements (stop on first error), use DispatchSyncStopOnError.
func (d *Dispatcher) DispatchSync(ctx context.Context, event DomainEvent) error {
	return d.dispatch(ctx, event, true, false)
}

// DispatchSyncStopOnError dispatches an event synchronously and stops on first handler error.
// Use when handler order matters and you need transactional-like behavior.
// Note: This does NOT provide true transaction rollback - handlers executed before
// the failure have already completed their side effects.
func (d *Dispatcher) DispatchSyncStopOnError(ctx context.Context, event DomainEvent) error {
	return d.dispatch(ctx, event, true, true)
}

// DispatchAsync dispatches an event asynchronously to all registered handlers.
// This returns immediately; handlers execute in background goroutines.
// Use for non-critical handlers where timing is not essential.
//
// Concurrency is controlled by asyncWorkers semaphore to prevent resource exhaustion.
func (d *Dispatcher) DispatchAsync(ctx context.Context, event DomainEvent) {
	go d.dispatch(ctx, event, false, false)
}

// Dispatch dispatches a batch of events synchronously.
// This is useful after a command produces multiple events.
func (d *Dispatcher) DispatchBatch(ctx context.Context, events []DomainEvent) error {
	for _, event := range events {
		if err := d.DispatchSync(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// DispatchBatchAsync dispatches a batch of events asynchronously.
func (d *Dispatcher) DispatchBatchAsync(ctx context.Context, events []DomainEvent) {
	for _, event := range events {
		d.DispatchAsync(ctx, event)
	}
}

// dispatch is the internal dispatch implementation.
// stopOnError: when true, stops execution on first handler error (for transactional-like behavior)
func (d *Dispatcher) dispatch(ctx context.Context, event DomainEvent, synchronous bool, stopOnError bool) error {
	eventType := event.EventType()

	// Copy handlers under lock protection to prevent data race
	// if new handlers are registered during dispatch
	d.mu.RLock()
	handlers := copyHandlers(d.handlers[eventType])
	wildcardHandlers := copyHandlers(d.handlers["*"])
	d.mu.RUnlock()

	// Combine specific and wildcard handlers
	allHandlers := append(handlers, wildcardHandlers...)

	if len(allHandlers) == 0 {
		d.logger.Debug("no handlers registered for event",
			zap.String("eventType", eventType),
		)
		return nil
	}

	var firstErr error

	for _, handler := range allHandlers {
		if synchronous {
			// Synchronous execution - block until complete
			if err := d.executeHandler(ctx, event, handler); err != nil {
				if stopOnError {
					return err // Stop immediately on error
				}
				if firstErr == nil {
					firstErr = err
				}
			}
		} else {
			// Asynchronous execution - acquire semaphore to limit concurrency
			d.asyncSem <- struct{}{} // acquire

			go func(h Handler) {
				defer func() { <-d.asyncSem }() // release

				d.executeHandler(ctx, event, h)
			}(handler)
		}
	}

	// Note: In async mode, we don't wait for handlers to complete.
	// - Logging is done in executeHandler, so no need to wait for that
	// - firstErr is always nil in async mode (errors happen in goroutines)
	// - This is true fire-and-forget behavior

	return firstErr
}

// copyHandlers creates a copy of a handler slice to prevent data race.
func copyHandlers(handlers []Handler) []Handler {
	if len(handlers) == 0 {
		return nil
	}
	copied := make([]Handler, len(handlers))
	copy(copied, handlers)
	return copied
}

// executeHandler runs a single handler with error handling and logging.
func (d *Dispatcher) executeHandler(ctx context.Context, event DomainEvent, handler Handler) error {
	d.logger.Debug("executing event handler",
		zap.String("eventType", event.EventType()),
		zap.String("sessionId", event.SessionID().String()),
	)

	err := handler(ctx, event)

	if err != nil {
		d.logger.Error("event handler failed",
			zap.String("eventType", event.EventType()),
			zap.String("sessionId", event.SessionID().String()),
			zap.Error(err),
		)
		return err
	}

	d.logger.Debug("event handler completed",
		zap.String("eventType", event.EventType()),
		zap.String("sessionId", event.SessionID().String()),
	)

	return nil
}

// HasHandlers checks if any handlers are registered for an event type.
func (d *Dispatcher) HasHandlers(eventType string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.handlers[eventType]) > 0 || len(d.handlers["*"]) > 0
}

// HandlerCount returns the number of handlers registered for an event type.
func (d *Dispatcher) HandlerCount(eventType string) int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.handlers[eventType]) + len(d.handlers["*"])
}

// Clear removes all registered handlers.
// Use with caution - typically only needed for testing.
func (d *Dispatcher) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.handlers = make(map[string][]Handler)
}

// --- Typed Handler Registration (using generics) ---

// registerTyped registers a type-safe handler for a specific event type.
// The type assertion here is defensive - it should never fail if the dispatcher is used correctly.
func registerTyped[T DomainEvent](d *Dispatcher, eventType string, handler func(ctx context.Context, e T) error) {
	d.Register(eventType, func(ctx context.Context, event DomainEvent) error {
		// Type assertion is defensive programming.
		// This should never fail if the dispatcher is used correctly,
		// because we only dispatch events with matching EventType() values.
		typed, ok := event.(T)
		if !ok {
			// This error indicates a bug in the dispatcher usage.
			// It should never happen in production if registration/dispatch is correct.
			return fmt.Errorf("type assertion failed for %s handler: got %T", eventType, event)
		}
		return handler(ctx, typed)
	})
}

// RegisterWorkSessionStarted registers a handler for WorkSessionStarted events.
func (d *Dispatcher) RegisterWorkSessionStarted(handler func(ctx context.Context, e *WorkSessionStarted) error) {
	registerTyped(d, "WorkSessionStarted", handler)
}

// RegisterStageTransitioned registers a handler for StageTransitioned events.
func (d *Dispatcher) RegisterStageTransitioned(handler func(ctx context.Context, e *StageTransitioned) error) {
	registerTyped(d, "StageTransitioned", handler)
}

// RegisterStageRolledBack registers a handler for StageRolledBack events.
func (d *Dispatcher) RegisterStageRolledBack(handler func(ctx context.Context, e *StageRolledBack) error) {
	registerTyped(d, "StageRolledBack", handler)
}

// RegisterTaskStarted registers a handler for TaskStarted events.
func (d *Dispatcher) RegisterTaskStarted(handler func(ctx context.Context, e *TaskStarted) error) {
	registerTyped(d, "TaskStarted", handler)
}

// RegisterTaskCompleted registers a handler for TaskCompleted events.
func (d *Dispatcher) RegisterTaskCompleted(handler func(ctx context.Context, e *TaskCompleted) error) {
	registerTyped(d, "TaskCompleted", handler)
}

// RegisterTaskFailed registers a handler for TaskFailed events.
func (d *Dispatcher) RegisterTaskFailed(handler func(ctx context.Context, e *TaskFailed) error) {
	registerTyped(d, "TaskFailed", handler)
}

// RegisterPullRequestCreated registers a handler for PullRequestCreated events.
func (d *Dispatcher) RegisterPullRequestCreated(handler func(ctx context.Context, e *PullRequestCreated) error) {
	registerTyped(d, "PullRequestCreated", handler)
}

// RegisterWorkSessionCompleted registers a handler for WorkSessionCompleted events.
func (d *Dispatcher) RegisterWorkSessionCompleted(handler func(ctx context.Context, e *WorkSessionCompleted) error) {
	registerTyped(d, "WorkSessionCompleted", handler)
}
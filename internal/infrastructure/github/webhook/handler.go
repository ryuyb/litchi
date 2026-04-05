package webhook

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"

	"github.com/gofiber/fiber/v3"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Headers used by GitHub webhooks.
const (
	HeaderSignature    = "X-Hub-Signature-256"
	HeaderDelivery     = "X-GitHub-Delivery"
	HeaderEventType    = "X-GitHub-Event"
)

// Handler handles GitHub webhook HTTP requests.
type Handler struct {
	verifier         *SignatureVerifier
	parser           *EventParser
	dispatcher       *EventDispatcher
	dedupRepo        repository.WebhookDeliveryRepository
	logger           *zap.Logger
	idempotencyCfg   config.IdempotencyConfig
}

// HandlerParams contains dependencies for creating a webhook handler.
type HandlerParams struct {
	Verifier   *SignatureVerifier
	Parser     *EventParser
	Dispatcher *EventDispatcher
	DedupRepo  repository.WebhookDeliveryRepository
	Logger     *zap.Logger
	Config     *config.WebhookConfig
}

// NewHandler creates a new webhook handler.
func NewHandler(p HandlerParams) *Handler {
	idempotencyCfg := config.IdempotencyConfig{}
	if p.Config != nil {
		idempotencyCfg = p.Config.Idempotency
	}

	return &Handler{
		verifier:       p.Verifier,
		parser:         p.Parser,
		dispatcher:     p.Dispatcher,
		dedupRepo:      p.DedupRepo,
		logger:         p.Logger.Named("webhook.handler"),
		idempotencyCfg: idempotencyCfg,
	}
}

// Handle handles incoming GitHub webhook requests.
// Route: POST /api/v1/webhooks/github
func (h *Handler) Handle(c fiber.Ctx) error {
	ctx := c.Context()

	// 1. Read raw body (before any parsing)
	payload := c.Body()

	// 2. Extract headers
	signature := c.Get(HeaderSignature)
	deliveryID := c.Get(HeaderDelivery)
	eventType := c.Get(HeaderEventType)

	// 3. Verify signature (MUST be first security check)
	if !h.verifier.Verify(payload, signature) {
		h.logger.Warn("webhook signature verification failed",
			zap.String("delivery_id", deliveryID),
			zap.String("event_type", eventType),
		)
		return c.Status(401).JSON(fiber.Map{
			"error": "invalid signature",
		})
	}

	// 4. Validate required headers
	if deliveryID == "" {
		h.logger.Warn("missing delivery ID header")
		return c.Status(400).JSON(fiber.Map{
			"error": "missing X-GitHub-Delivery header",
		})
	}

	// 5. Parse event first (before idempotency check to get repository info)
	event, err := h.parser.Parse(eventType, payload)
	if err != nil {
		h.logger.Error("failed to parse webhook event",
			zap.Error(err),
			zap.String("event_type", eventType),
			zap.String("delivery_id", deliveryID),
		)
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid payload",
		})
	}

	// 6. Atomically acquire processing rights (database-level idempotency)
	if h.idempotencyCfg.Enabled && h.dedupRepo != nil {
		payloadHash := computePayloadHash(payload)
		acquired, err := h.dedupRepo.TryAcquire(ctx, deliveryID, eventType, event.Repository(), payloadHash)
		if err != nil {
			h.logger.Error("failed to acquire webhook delivery",
				zap.Error(err),
				zap.String("delivery_id", deliveryID),
			)

			// For transient errors (database connectivity, timeout), return 503 to let GitHub retry
			if isTransientError(err) {
				return c.Status(503).JSON(fiber.Map{
					"error":       "service temporarily unavailable",
					"error_code":  "TRANSIENT_ERROR",
					"retry_safe":  true,
				})
			}

			// For persistent errors, return 200 to avoid infinite retry
			return c.Status(200).JSON(fiber.Map{
				"status":      "error_acquiring_lock",
				"error_code":  "ACQUIRE_FAILED",
				"message":     err.Error(),
				"retry_safe":  false,
			})
		}

		if !acquired {
			h.logger.Info("webhook already processed (idempotent)",
				zap.String("delivery_id", deliveryID),
			)
			return c.Status(200).JSON(fiber.Map{
				"status":      "already_processed",
				"error_code":  "DUPLICATE_DELIVERY",
				"retry_safe":  false,
			})
		}
	}

	// 7. Dispatch event to handlers
	err = h.dispatcher.Dispatch(ctx, event)
	if err != nil {
		h.logger.Error("failed to dispatch webhook event",
			zap.Error(err),
			zap.String("event_type", eventType),
			zap.String("delivery_id", deliveryID),
			zap.String("repository", event.Repository()),
		)
		h.updateDeliveryResult(ctx, deliveryID, entity.ProcessResultError, err.Error())
		// Return 200 to avoid GitHub retry - we handle errors internally
		return c.Status(200).JSON(fiber.Map{
			"status":      "processing_error",
			"error_code":  "DISPATCH_FAILED",
			"message":     err.Error(),
			"retry_safe":  false,
		})
	}

	// 8. Update delivery result to success
	if h.idempotencyCfg.Enabled && h.dedupRepo != nil {
		h.updateDeliveryResult(ctx, deliveryID, entity.ProcessResultSuccess, "")
	}

	h.logger.Info("webhook processed successfully",
		zap.String("delivery_id", deliveryID),
		zap.String("event_type", eventType),
		zap.String("repository", event.Repository()),
		zap.String("actor", event.Actor()),
		zap.String("action", event.Action()),
	)

	return c.Status(200).JSON(fiber.Map{
		"status": "processed",
	})
}

// updateDeliveryResult updates the processing result of a delivery.
func (h *Handler) updateDeliveryResult(ctx context.Context, deliveryID, result, message string) {
	if h.dedupRepo == nil {
		return
	}
	if err := h.dedupRepo.UpdateResult(ctx, deliveryID, result, message); err != nil {
		h.logger.Error("failed to update delivery result",
			zap.Error(err),
			zap.String("delivery_id", deliveryID),
		)
	}
}

// computePayloadHash computes SHA256 hash of payload.
func computePayloadHash(payload []byte) string {
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

// HealthCheck returns the health status of the webhook handler.
func (h *Handler) HealthCheck(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":             "healthy",
		"idempotency_enabled": h.idempotencyCfg.Enabled,
		"registered_handlers": h.dispatcher.RegisteredEventTypes(),
	})
}

// RegisterRoutes registers webhook routes on the Fiber app.
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// Webhook endpoint
	app.Post("/api/v1/webhooks/github", handler.Handle)

	// Health check for webhook integration
	app.Get("/api/v1/webhooks/health", handler.HealthCheck)
}

// NoOpHandler returns a handler that does nothing (for testing).
func NoOpHandler() EventHandlerFunc {
	return func(ctx context.Context, event WebhookEvent) error {
		return nil
	}
}

// isTransientError checks if the error is transient and worth retrying.
// Transient errors include: database connectivity issues, timeouts, network errors.
// Persistent errors include: constraint violations, data format errors.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context errors (timeout/canceled)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	// Check for network errors (DNS, connection refused, etc.)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check for GORM database connection errors
	if errors.Is(err, gorm.ErrInvalidDB) || errors.Is(err, gorm.ErrInvalidTransaction) {
		return true
	}

	// Check for connection-related error messages
	errMsg := err.Error()
	transientPatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"timeout",
		"deadline exceeded",
		"too many connections",
		"connection pool exhausted",
	}
	for _, pattern := range transientPatterns {
		if containsIgnoreCase(errMsg, pattern) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		sLower[i] = c
	}
	for i := 0; i < len(substr); i++ {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		substrLower[i] = c
	}
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		match := true
		for j := 0; j < len(substrLower); j++ {
			if sLower[i+j] != substrLower[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
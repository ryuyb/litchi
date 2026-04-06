// Package middleware provides HTTP middleware for the server.
package middleware

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/ryuyb/litchi/internal/application/dto"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
)

// ErrorHandler provides unified error handling for HTTP responses.
type ErrorHandler struct {
	logger *zap.Logger
	debug  bool // Whether to include detailed error info in responses
}

// ErrorHandlerParams contains dependencies for creating an error handler.
type ErrorHandlerParams struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
}

// NewErrorHandler creates a new error handler middleware.
func NewErrorHandler(p ErrorHandlerParams) *ErrorHandler {
	return &ErrorHandler{
		logger: p.Logger.Named("error-handler"),
		debug:  p.Config.Server.Mode == "debug",
	}
}

// Handle processes errors and returns appropriate HTTP responses.
// This middleware should be registered as the global error handler for Fiber.
func (h *ErrorHandler) Handle(c fiber.Ctx, err error) error {
	// Get request context for logging
	path := c.Path()
	method := c.Method()

	// Convert domain error to API error code
	apiErr := litchierrors.ToAPIError(err)

	// Build error response
	response := dto.ErrorResponse{
		Code:    litchierrors.GetCode(err),
		Message: apiErr.Message,
	}

	// Add details in debug mode
	var litchiErr *litchierrors.Error
	if errors.As(err, &litchiErr) {
		if litchiErr.Detail != "" && h.debug {
			response.Details = litchiErr.Detail
		}

		// Log based on severity
		severity := litchierrors.GetSeverity(err)
		switch severity {
		case 1, 2: // Critical, High
			h.logger.Error("request error",
				zap.String("path", path),
				zap.String("method", method),
				zap.String("code", response.Code),
				zap.String("detail", litchiErr.Detail),
				zap.Error(err),
			)
		case 3: // Medium
			h.logger.Warn("request error",
				zap.String("path", path),
				zap.String("method", method),
				zap.String("code", response.Code),
				zap.Error(err),
			)
		case 4: // Low
			h.logger.Info("request error",
				zap.String("path", path),
				zap.String("method", method),
				zap.String("code", response.Code),
			)
		}
	} else {
		// Unknown error type
		h.logger.Error("unknown error",
			zap.String("path", path),
			zap.String("method", method),
			zap.Error(err),
		)
	}

	return c.Status(apiErr.Code).JSON(response)
}
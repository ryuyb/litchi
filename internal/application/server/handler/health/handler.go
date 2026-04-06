// Package health provides HTTP health check handlers.
package health

import (
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/dto"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/pkg/health"
	"go.uber.org/zap"
)

// Handler handles health check requests.
type Handler struct {
	checkers []health.Checker
	config   *config.Config
	logger   *zap.Logger
}

// HandlerParams contains dependencies for creating a health handler.
type HandlerParams struct {
	fx.In

	Checkers []health.Checker `group:"health_checkers"`
	Config   *config.Config
	Logger   *zap.Logger
}

// NewHandler creates a new health handler.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		checkers: p.Checkers,
		config:   p.Config,
		logger:   p.Logger.Named("health-handler"),
	}
}

// BasicHealth returns basic server health status (legacy endpoint).
// @Summary        Basic health check
// @Description    Returns the server health status and version (legacy endpoint)
// @Tags           system
// @Accept         json
// @Produce        json
// @Success        200  {object}  dto.HealthCheckResponse  "Server is healthy"
// @Router         /health [get]
func (h *Handler) BasicHealth(c fiber.Ctx) error {
	return c.JSON(dto.HealthCheckResponse{
		Status:  "healthy",
		Version: h.config.Server.Version,
	})
}

// HealthCheck returns basic API health status.
// @Summary        API health check
// @Description    Returns the basic API health status and version
// @Tags           system
// @Accept         json
// @Produce        json
// @Success        200  {object}  dto.HealthCheckResponse  "Server is healthy"
// @Router         /api/v1/health [get]
func (h *Handler) HealthCheck(c fiber.Ctx) error {
	return c.JSON(dto.HealthCheckResponse{
		Status:  "healthy",
		Version: h.config.Server.Version,
	})
}

// DetailedHealth returns detailed health check with component status.
// @Summary        Detailed health check
// @Description    Returns detailed health status including database, GitHub, and Git checks
// @Tags           system
// @Accept         json
// @Produce        json
// @Success        200  {object}  dto.HealthDetailResponse  "Detailed health status"
// @Router         /api/v1/health/detail [get]
func (h *Handler) DetailedHealth(c fiber.Ctx) error {
	ctx := c.Context()
	checks := make([]dto.HealthCheckItem, 0, len(h.checkers))

	// Run all health checkers
	for _, checker := range h.checkers {
		result := checker.Check(ctx)
		checks = append(checks, dto.HealthCheckItem{
			Name:      result.Name,
			Status:    result.Status,
			Message:   result.Message,
			LatencyMs: result.LatencyMs,
			Error:     result.Error,
			Details:   result.Details,
		})
	}

	// Determine overall status
	status := "healthy"
	for _, check := range checks {
		if check.Status == "fail" {
			status = "unhealthy"
			break
		}
		if check.Status == "warn" && status != "unhealthy" {
			status = "degraded"
		}
	}

	return c.JSON(dto.ToHealthDetailResponse(status, h.config.Server.Version, checks))
}
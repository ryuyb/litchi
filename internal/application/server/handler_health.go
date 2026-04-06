package server

import (
	"github.com/gofiber/fiber/v3"
	"github.com/ryuyb/litchi/internal/application/dto"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	version string
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{version: version}
}

// Handle returns the server health status.
// @Summary        Health check
// @Description    Returns the server health status and version
// @Tags           system
// @Accept         json
// @Produce        json
// @Success        200  {object}  dto.HealthCheckResponse  "Server is healthy"
// @Router         /health [get]
func (h *HealthHandler) Handle(c fiber.Ctx) error {
	return c.JSON(dto.HealthCheckResponse{
		Status:  "healthy",
		Version: h.version,
	})
}
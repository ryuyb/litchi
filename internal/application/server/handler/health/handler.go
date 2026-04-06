// Package health provides HTTP health check handlers.
package health

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/dto"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/github"
	"github.com/ryuyb/litchi/internal/infrastructure/git"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/postgres"
	"go.uber.org/zap"
)

// Handler handles health check requests.
type Handler struct {
	db      *postgres.DB
	ghClient *github.Client
	gitExecutor *git.CommandExecutor
	config  *config.Config
	logger  *zap.Logger
}

// HandlerParams contains dependencies for creating a health handler.
type HandlerParams struct {
	fx.In

	DB          *postgres.DB
	GHClient    *github.Client
	GitExecutor *git.CommandExecutor
	Config      *config.Config
	Logger      *zap.Logger
}

// NewHandler creates a new health handler.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		db:      p.DB,
		ghClient: p.GHClient,
		gitExecutor: p.GitExecutor,
		config:  p.Config,
		logger:  p.Logger.Named("health-handler"),
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
	checks := make([]dto.HealthCheckItem, 0)

	// Database check
	dbCheck := h.checkDatabase(ctx)
	checks = append(checks, dbCheck)

	// GitHub check
	ghCheck := h.checkGitHub(ctx)
	checks = append(checks, ghCheck)

	// Git check
	gitCheck := h.checkGit(ctx)
	checks = append(checks, gitCheck)

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

// checkDatabase checks database connectivity.
func (h *Handler) checkDatabase(ctx context.Context) dto.HealthCheckItem {
	start := time.Now()

	err := h.db.Ping(ctx)
	latency := time.Since(start)

	check := dto.HealthCheckItem{
		Name:      "database",
		LatencyMs: int(latency.Milliseconds()),
	}

	if err != nil {
		check.Status = "fail"
		check.Error = err.Error()
		check.Message = "Database connection failed"
		h.logger.Error("database health check failed", zap.Error(err))
	} else {
		check.Status = "pass"
		check.Message = "Connection OK"

		// Get connection pool stats
		stats, err := h.db.Stats()
		if err == nil {
			check.Details = map[string]any{
				"open_connections": stats["open_connections"],
				"idle_connections": stats["idle"],
				"in_use":           stats["in_use"],
			}
		}
	}

	return check
}

// checkGitHub checks GitHub API connectivity.
func (h *Handler) checkGitHub(ctx context.Context) dto.HealthCheckItem {
	start := time.Now()

	// Use Ping method to test GitHub connectivity
	err := h.ghClient.Ping(ctx)
	latency := time.Since(start)

	check := dto.HealthCheckItem{
		Name:      "github",
		LatencyMs: int(latency.Milliseconds()),
	}

	if err != nil {
		check.Status = "fail"
		check.Error = err.Error()
		check.Message = "GitHub API connection failed"
		h.logger.Error("github health check failed", zap.Error(err))
	} else {
		check.Status = "pass"
		check.Message = "API connection OK"

		// Add rate limit info if available
		remaining := h.ghClient.RateLimiter().GetRemaining()
		if remaining > 0 {
			check.Details = map[string]any{
				"rate_limit_remaining": remaining,
			}
		}
	}

	return check
}

// checkGit checks Git binary availability.
func (h *Handler) checkGit(ctx context.Context) dto.HealthCheckItem {
	start := time.Now()

	// Execute git --version to check if Git is available
	result, err := h.gitExecutor.Exec(ctx, "", "--version")
	latency := time.Since(start)

	check := dto.HealthCheckItem{
		Name:      "git",
		LatencyMs: int(latency.Milliseconds()),
	}

	if err != nil {
		check.Status = "fail"
		check.Error = err.Error()
		check.Message = "Git binary not available"
		h.logger.Error("git health check failed", zap.Error(err))
	} else {
		check.Status = "pass"
		check.Message = "Git binary available"

		// Extract version from output
		version := result.Stdout
		if version != "" {
			check.Details = map[string]any{
				"version": version,
			}
		}
	}

	return check
}
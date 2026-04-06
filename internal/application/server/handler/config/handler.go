// Package config provides HTTP handlers for configuration management API.
package config

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/ryuyb/litchi/internal/application/dto"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// Handler handles configuration-related HTTP requests.
type Handler struct {
	cfg    *config.Config
	logger *zap.Logger
}

// HandlerParams contains dependencies for creating a config handler.
type HandlerParams struct {
	Cfg    *config.Config
	Logger *zap.Logger
}

// NewHandler creates a new config handler.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		cfg:    p.Cfg,
		logger: p.Logger.Named("config-handler"),
	}
}

// GetConfig returns the current configuration.
// @Summary        Get configuration
// @Description   Returns the current application configuration (sensitive fields excluded)
// @Tags          config
// @Accept        json
// @Produce       json
// @Success       200  {object}  dto.ConfigResponse  "Configuration retrieved successfully"
// @Failure       500  {object}  dto.ErrorResponse   "Internal server error"
// @Router        /api/v1/config [get]
func (h *Handler) GetConfig(c fiber.Ctx) error {
	response := dto.ToConfigResponse(h.cfg)
	return c.JSON(response)
}

// UpdateConfig updates the configuration (partial update).
// @Summary        Update configuration
// @Description   Updates application configuration (only certain fields are updatable at runtime)
// @Tags          config
// @Accept        json
// @Produce       json
// @Param         request  body      dto.UpdateConfigRequest  true  "Configuration update request"
// @Success       200      {object}  dto.ConfigResponse       "Configuration updated successfully"
// @Failure       400      {object}  dto.ErrorResponse        "Invalid request or validation failed"
// @Failure       500      {object}  dto.ErrorResponse        "Internal server error"
// @Router        /api/v1/config [put]
//
// Atomicity Guarantee:
// This method uses a snapshot pattern to ensure atomic updates. The configuration is:
// 1. Cloned to a snapshot
// 2. Updates are applied to the snapshot only
// 3. The entire snapshot is validated
// 4. Only if all validations pass, the original config is replaced
// If any step fails, the original configuration remains unchanged.
func (h *Handler) UpdateConfig(c fiber.Ctx) error {
	var req dto.UpdateConfigRequest
	if err := c.Bind().JSON(&req); err != nil {
		h.logger.Warn("failed to parse config update request", zap.Error(err))
		return litchierrors.New(litchierrors.ErrBadRequest).
			WithDetail("invalid JSON body: " + err.Error())
	}

	// Create a snapshot for atomic update
	snapshot := h.cfg.Clone()

	// Apply updates to snapshot
	if err := h.applyUpdates(snapshot, &req); err != nil {
		// Validation failed, snapshot is discarded
		return err
	}

	// Validate the entire snapshot after updates
	if err := snapshot.Validate(); err != nil {
		h.logger.Warn("config validation failed after update", zap.Error(err))
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail(err.Error())
	}

	// All validations passed, replace config with snapshot
	*h.cfg = *snapshot

	h.logger.Info("configuration updated",
		zap.Bool("agent_updated", req.Agent != nil),
		zap.Bool("git_updated", req.Git != nil),
		zap.Bool("clarity_updated", req.Clarity != nil),
		zap.Bool("complexity_updated", req.Complexity != nil),
		zap.Bool("audit_updated", req.Audit != nil),
	)

	// Return updated config
	response := dto.ToConfigResponse(h.cfg)
	return c.JSON(response)
}

// applyUpdates applies the update request to the configuration snapshot.
func (h *Handler) applyUpdates(cfg *config.Config, req *dto.UpdateConfigRequest) error {
	// Apply Agent updates
	if req.Agent != nil {
		if err := h.applyAgentUpdates(cfg, req.Agent); err != nil {
			return err
		}
	}

	// Apply Git updates
	if req.Git != nil {
		if err := h.applyGitUpdates(cfg, req.Git); err != nil {
			return err
		}
	}

	// Apply Clarity updates
	if req.Clarity != nil {
		if err := h.applyClarityUpdates(cfg, req.Clarity); err != nil {
			return err
		}
	}

	// Apply Complexity updates
	if req.Complexity != nil {
		if err := h.applyComplexityUpdates(cfg, req.Complexity); err != nil {
			return err
		}
	}

	// Apply Audit updates
	if req.Audit != nil {
		if err := h.applyAuditUpdates(cfg, req.Audit); err != nil {
			return err
		}
	}

	return nil
}

// applyAgentUpdates applies agent configuration updates.
func (h *Handler) applyAgentUpdates(cfg *config.Config, update *dto.AgentConfigUpdate) error {
	if update.MaxConcurrency != nil {
		if *update.MaxConcurrency < 1 || *update.MaxConcurrency > 10 {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("agent.maxConcurrency must be between 1 and 10")
		}
		cfg.Agent.MaxConcurrency = *update.MaxConcurrency
	}

	if update.TaskRetryLimit != nil {
		if *update.TaskRetryLimit < 0 || *update.TaskRetryLimit > 10 {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("agent.taskRetryLimit must be between 0 and 10")
		}
		cfg.Agent.TaskRetryLimit = *update.TaskRetryLimit
	}

	if update.ApprovalTimeout != nil {
		duration, err := time.ParseDuration(*update.ApprovalTimeout)
		if err != nil {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("agent.approvalTimeout must be a valid duration (e.g., '24h', '1h30m')")
		}
		if duration < 0 {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("agent.approvalTimeout must be positive")
		}
		cfg.Agent.ApprovalTimeout = *update.ApprovalTimeout
	}

	return nil
}

// applyGitUpdates applies git configuration updates.
func (h *Handler) applyGitUpdates(cfg *config.Config, update *dto.GitConfigUpdate) error {
	if update.WorktreeBasePath != nil {
		if *update.WorktreeBasePath == "" {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("git.worktreeBasePath cannot be empty")
		}
		cfg.Git.WorktreeBasePath = *update.WorktreeBasePath
	}

	if update.WorktreeAutoClean != nil {
		cfg.Git.WorktreeAutoClean = *update.WorktreeAutoClean
	}

	if update.BranchNamingPattern != nil {
		if *update.BranchNamingPattern == "" {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("git.branchNamingPattern cannot be empty")
		}
		cfg.Git.BranchNamingPattern = *update.BranchNamingPattern
	}

	if update.DefaultBaseBranch != nil {
		if *update.DefaultBaseBranch == "" {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("git.defaultBaseBranch cannot be empty")
		}
		cfg.Git.DefaultBaseBranch = *update.DefaultBaseBranch
	}

	if update.CommitSignOff != nil {
		cfg.Git.CommitSignOff = *update.CommitSignOff
	}

	if update.CommandTimeout != nil {
		duration, err := time.ParseDuration(*update.CommandTimeout)
		if err != nil {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("git.commandTimeout must be a valid duration (e.g., '5m', '1h')")
		}
		if duration < 0 {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("git.commandTimeout must be positive")
		}
		cfg.Git.CommandTimeout = *update.CommandTimeout
	}

	return nil
}

// applyClarityUpdates applies clarity configuration updates.
func (h *Handler) applyClarityUpdates(cfg *config.Config, update *dto.ClarityConfigUpdate) error {
	if update.Threshold != nil {
		if *update.Threshold < 0 || *update.Threshold > 100 {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("clarity.threshold must be between 0 and 100")
		}
		cfg.Clarity.Threshold = *update.Threshold
	}

	if update.AutoProceedThreshold != nil {
		if *update.AutoProceedThreshold < 0 || *update.AutoProceedThreshold > 100 {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("clarity.autoProceedThreshold must be between 0 and 100")
		}
		cfg.Clarity.AutoProceedThreshold = *update.AutoProceedThreshold
	}

	if update.ForceClarifyThreshold != nil {
		if *update.ForceClarifyThreshold < 0 || *update.ForceClarifyThreshold > 100 {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("clarity.forceClarifyThreshold must be between 0 and 100")
		}
		cfg.Clarity.ForceClarifyThreshold = *update.ForceClarifyThreshold
	}

	// Validate threshold relationships
	if cfg.Clarity.ForceClarifyThreshold >= cfg.Clarity.Threshold {
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("clarity.forceClarifyThreshold must be less than clarity.threshold")
	}
	if cfg.Clarity.Threshold >= cfg.Clarity.AutoProceedThreshold {
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("clarity.threshold must be less than clarity.autoProceedThreshold")
	}

	return nil
}

// applyComplexityUpdates applies complexity configuration updates.
func (h *Handler) applyComplexityUpdates(cfg *config.Config, update *dto.ComplexityConfigUpdate) error {
	if update.Threshold != nil {
		if *update.Threshold < 0 || *update.Threshold > 100 {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("complexity.threshold must be between 0 and 100")
		}
		cfg.Complexity.Threshold = *update.Threshold
	}

	if update.ForceDesignConfirm != nil {
		cfg.Complexity.ForceDesignConfirm = *update.ForceDesignConfirm
	}

	return nil
}

// applyAuditUpdates applies audit configuration updates.
func (h *Handler) applyAuditUpdates(cfg *config.Config, update *dto.AuditConfigUpdate) error {
	if update.Enabled != nil {
		cfg.Audit.Enabled = *update.Enabled
	}

	if update.RetentionDays != nil {
		if *update.RetentionDays < 1 || *update.RetentionDays > 365 {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("audit.retentionDays must be between 1 and 365")
		}
		cfg.Audit.RetentionDays = *update.RetentionDays
	}

	if update.MaxOutputLength != nil {
		if *update.MaxOutputLength < 100 || *update.MaxOutputLength > 100000 {
			return litchierrors.New(litchierrors.ErrValidationFailed).
				WithDetail("audit.maxOutputLength must be between 100 and 100000")
		}
		cfg.Audit.MaxOutputLength = *update.MaxOutputLength
	}

	return nil
}
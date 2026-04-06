// Package repository provides HTTP handlers for repository management API.
package repository

import (
	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"

	"github.com/ryuyb/litchi/internal/application/dto"
	"github.com/ryuyb/litchi/internal/application/service"
	domainrepo "github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// Handler handles repository management HTTP requests.
type Handler struct {
	repoService *service.RepositoryService
	logger      *zap.Logger
}

// HandlerParams contains dependencies for creating a repository handler.
type HandlerParams struct {
	fx.In

	RepoService *service.RepositoryService
	Logger      *zap.Logger
}

// NewHandler creates a new repository handler.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		repoService: p.RepoService,
		logger:      p.Logger.Named("repository_handler"),
	}
}

// ListRepositories lists all repository configurations.
// @Summary        List repositories
// @Description    Get all repository configurations with optional pagination and filtering
// @Tags           repositories
// @Accept         json
// @Produce        json
// @Param          page     query    int     false  "Page number (1-based)"    default(1)
// @Param          pageSize query    int     false  "Items per page"           default(20)
// @Param          enabled  query    string  false  "Filter by enabled status (true, false, all)"  default(all)
// @Success        200  {object}  dto.PaginatedResponse[dto.RepositoryResponse]  "List of repositories"
// @Failure        500  {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/repositories [get]
func (h *Handler) ListRepositories(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	page := dto.ParseQueryInt(c, "page", 1)
	pageSize := dto.ParseQueryInt(c, "pageSize", 20)
	enabledFilter := c.Query("enabled", "all") // true, false, all

	// Normalize pagination params
	page, pageSize = dto.NormalizePagination(page, pageSize, dto.DefaultPageSize)

	// Build filter from query parameter
	var filter *domainrepo.RepositoryFilter
	if enabledFilter != "all" {
		filter = &domainrepo.RepositoryFilter{}
		if enabledFilter == "true" {
			filter.Enabled = boolPtr(true)
		} else if enabledFilter == "false" {
			filter.Enabled = boolPtr(false)
		}
	}

	// Get repositories with pagination from service
	repos, pagination, err := h.repoService.ListRepositoriesWithPagination(ctx,
		domainrepo.PaginationParams{Page: page, PageSize: pageSize},
		filter,
	)
	if err != nil {
		h.logger.Error("failed to list repositories", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// Convert to response DTO
	response := dto.ToRepositoryList(repos, pagination.Page, pagination.PageSize, int64(pagination.TotalItems))

	return c.JSON(response)
}

// boolPtr returns a pointer to a bool value.
func boolPtr(v bool) *bool {
	return &v
}

// GetRepository retrieves a repository configuration by name.
// @Summary        Get repository
// @Description    Get repository configuration by name (owner/repo format)
// @Tags           repositories
// @Accept         json
// @Produce        json
// @Param          name  path    string  true  "Repository name (owner/repo)"
// @Success        200  {object}  dto.RepositoryResponse  "Repository configuration"
// @Failure        404  {object}  dto.ErrorResponse  "Repository not found"
// @Failure        500  {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/repositories/{name} [get]
func (h *Handler) GetRepository(c fiber.Ctx) error {
	ctx := c.Context()
	name := c.Params("name")

	if name == "" {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("repository name is required")
	}

	repo, err := h.repoService.GetRepository(ctx, name)
	if err != nil {
		h.logger.Error("failed to get repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return err
	}

	if repo == nil {
		return litchierrors.New(litchierrors.ErrRepositoryNotFound).
			WithDetail("repository not found: " + name)
	}

	response := dto.ToRepositoryResponse(repo)
	return c.JSON(response)
}

// CreateRepository creates a new repository configuration.
// @Summary        Create repository
// @Description    Create a new repository configuration for processing
// @Tags           repositories
// @Accept         json
// @Produce        json
// @Param          body  body    dto.CreateRepositoryRequest  true  "Repository creation request"
// @Success        201  {object}  dto.RepositoryResponse  "Created repository"
// @Failure        400  {object}  dto.ErrorResponse  "Invalid request body"
// @Failure        409  {object}  dto.ErrorResponse  "Repository already exists"
// @Failure        500  {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/repositories [post]
func (h *Handler) CreateRepository(c fiber.Ctx) error {
	ctx := c.Context()

	var req dto.CreateRepositoryRequest
	if err := c.Bind().JSON(&req); err != nil {
		h.logger.Warn("failed to parse create repository request", zap.Error(err))
		return litchierrors.New(litchierrors.ErrInvalidRequestBody).
			WithDetail("invalid request body")
	}

	// Validate request
	if err := dto.Validate(&req); err != nil {
		h.logger.Warn("validation failed for create repository request",
			zap.String("name", req.Name),
			zap.Error(err),
		)
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("Validation failed: " + err.Error())
	}

	// Convert DTO config to entity config
	var configOverrides *entity.RepoConfig
	if req.Config != nil {
		configOverrides = &entity.RepoConfig{
			MaxConcurrency:      req.Config.MaxConcurrency,
			ComplexityThreshold: req.Config.ComplexityThreshold,
			ForceDesignConfirm:  req.Config.ForceDesignConfirm,
			DefaultModel:        req.Config.DefaultModel,
			TaskRetryLimit:      req.Config.TaskRetryLimit,
		}
	}

	// Create repository via service
	// TODO: Replace "api_user" with actual user from auth context after implementing auth middleware
	repo, err := h.repoService.CreateRepository(
		ctx,
		req.Name,
		configOverrides,
		"api_user", // Default actor for API operations
		valueobject.ActorRoleAdmin,
	)
	if err != nil {
		h.logger.Error("failed to create repository",
			zap.String("name", req.Name),
			zap.Error(err),
		)
		return err
	}

	response := dto.ToRepositoryResponse(repo)
	return c.Status(201).JSON(response)
}

// UpdateRepository updates an existing repository configuration.
// @Summary        Update repository
// @Description    Update repository configuration by name
// @Tags           repositories
// @Accept         json
// @Produce        json
// @Param          name  path    string  true  "Repository name (owner/repo)"
// @Param          body  body    dto.UpdateRepositoryRequest  true  "Repository update request"
// @Success        200  {object}  dto.RepositoryResponse  "Updated repository"
// @Failure        400  {object}  dto.ErrorResponse  "Invalid request body"
// @Failure        404  {object}  dto.ErrorResponse  "Repository not found"
// @Failure        500  {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/repositories/{name} [put]
func (h *Handler) UpdateRepository(c fiber.Ctx) error {
	ctx := c.Context()
	name := c.Params("name")

	if name == "" {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("repository name is required")
	}

	var req dto.UpdateRepositoryRequest
	if err := c.Bind().JSON(&req); err != nil {
		h.logger.Warn("failed to parse update repository request", zap.Error(err))
		return litchierrors.New(litchierrors.ErrInvalidRequestBody).
			WithDetail("invalid request body")
	}

	// Convert DTO config to entity config
	configOverrides := &entity.RepoConfig{
		MaxConcurrency:      req.Config.MaxConcurrency,
		ComplexityThreshold: req.Config.ComplexityThreshold,
		ForceDesignConfirm:  req.Config.ForceDesignConfirm,
		DefaultModel:        req.Config.DefaultModel,
		TaskRetryLimit:      req.Config.TaskRetryLimit,
	}

	// Update repository via service
	repo, err := h.repoService.UpdateRepository(
		ctx,
		name,
		configOverrides,
		"api_user",
		valueobject.ActorRoleAdmin,
	)
	if err != nil {
		h.logger.Error("failed to update repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return err
	}

	response := dto.ToRepositoryResponse(repo)
	return c.JSON(response)
}

// DeleteRepository deletes a repository configuration.
// @Summary        Delete repository
// @Description    Delete repository configuration by name
// @Tags           repositories
// @Accept         json
// @Produce        json
// @Param          name  path    string  true  "Repository name (owner/repo)"
// @Success        200  {object}  dto.SuccessResponse  "Repository deleted"
// @Failure        404  {object}  dto.ErrorResponse  "Repository not found"
// @Failure        500  {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/repositories/{name} [delete]
func (h *Handler) DeleteRepository(c fiber.Ctx) error {
	ctx := c.Context()
	name := c.Params("name")

	if name == "" {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("repository name is required")
	}

	err := h.repoService.DeleteRepository(
		ctx,
		name,
		"api_user",
		valueobject.ActorRoleAdmin,
	)
	if err != nil {
		h.logger.Error("failed to delete repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return err
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "repository deleted",
	})
}

// EnableRepository enables a repository for processing.
// @Summary        Enable repository
// @Description    Enable a repository to process incoming GitHub events
// @Tags           repositories
// @Accept         json
// @Produce        json
// @Param          name  path    string  true  "Repository name (owner/repo)"
// @Success        200  {object}  dto.SuccessResponse  "Repository enabled"
// @Failure        404  {object}  dto.ErrorResponse  "Repository not found"
// @Failure        500  {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/repositories/{name}/enable [post]
func (h *Handler) EnableRepository(c fiber.Ctx) error {
	ctx := c.Context()
	name := c.Params("name")

	if name == "" {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("repository name is required")
	}

	err := h.repoService.EnableRepository(
		ctx,
		name,
		"api_user",
		valueobject.ActorRoleAdmin,
	)
	if err != nil {
		h.logger.Error("failed to enable repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return err
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "repository enabled",
	})
}

// DisableRepository disables a repository from processing.
// @Summary        Disable repository
// @Description    Disable a repository from processing incoming GitHub events
// @Tags           repositories
// @Accept         json
// @Produce        json
// @Param          name  path    string  true  "Repository name (owner/repo)"
// @Success        200  {object}  dto.SuccessResponse  "Repository disabled"
// @Failure        404  {object}  dto.ErrorResponse  "Repository not found"
// @Failure        500  {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/repositories/{name}/disable [post]
func (h *Handler) DisableRepository(c fiber.Ctx) error {
	ctx := c.Context()
	name := c.Params("name")

	if name == "" {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("repository name is required")
	}

	err := h.repoService.DisableRepository(
		ctx,
		name,
		"api_user",
		valueobject.ActorRoleAdmin,
	)
	if err != nil {
		h.logger.Error("failed to disable repository",
			zap.String("name", name),
			zap.Error(err),
		)
		return err
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "repository disabled",
	})
}

// GetEffectiveConfig returns the effective configuration for a repository.
// @Summary        Get effective config
// @Description    Get the merged effective configuration (global + repository overrides)
// @Tags           repositories
// @Accept         json
// @Produce        json
// @Param          name  path    string  true  "Repository name (owner/repo)"
// @Success        200  {object}  dto.EffectiveConfigResponse  "Effective configuration"
// @Failure        500  {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/repositories/{name}/effective-config [get]
func (h *Handler) GetEffectiveConfig(c fiber.Ctx) error {
	ctx := c.Context()
	name := c.Params("name")

	if name == "" {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("repository name is required")
	}

	effectiveConfig, err := h.repoService.GetEffectiveConfig(ctx, name)
	if err != nil {
		h.logger.Error("failed to get effective config",
			zap.String("name", name),
			zap.Error(err),
		)
		return err
	}

	// Convert service EffectiveConfig to DTO response
	response := dto.EffectiveConfigResponse{
		RepositoryName: effectiveConfig.RepositoryName,
		RepositoryID:   effectiveConfig.RepositoryID,
		Enabled:        effectiveConfig.Enabled,
		GlobalConfig:   dto.ToRepoConfigDTO(effectiveConfig.GlobalConfig),
		RepoConfig:     dto.ToRepoConfigDTO(effectiveConfig.RepoConfig),
		Effective:      dto.ToRepoConfigDTO(effectiveConfig.Effective),
		HasRepoConfig:  effectiveConfig.HasRepoConfig,
	}

	return c.JSON(response)
}
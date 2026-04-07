// Package user provides HTTP handlers for user management API.
package user

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/application/dto"
	"github.com/ryuyb/litchi/internal/application/server/handler/auth"
	authmiddleware "github.com/ryuyb/litchi/internal/application/server/middleware/auth"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Handler handles user management HTTP requests.
type Handler struct {
	userRepo repository.UserRepository
	logger   *zap.Logger
}

// HandlerParams contains dependencies for creating a user handler.
type HandlerParams struct {
	fx.In

	UserRepo repository.UserRepository
	Logger   *zap.Logger
}

// NewHandler creates a new user handler.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		userRepo: p.UserRepo,
		logger:   p.Logger.Named("user_handler"),
	}
}

// CreateUserRequest represents the create user request body.
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"required,oneof=admin viewer"`
}

// UpdateUserRequest represents the update user request body.
type UpdateUserRequest struct {
	Username *string `json:"username" validate:"omitempty,min=3,max=50"`
	Password *string `json:"password" validate:"omitempty,min=6"`
	Role     *string `json:"role" validate:"omitempty,oneof=admin viewer"`
}

// UserResponse represents user information in responses.
type UserResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// UserListResponse represents a paginated list of users.
type UserListResponse struct {
	Data       []UserResponse    `json:"data"`
	Pagination dto.PaginationDTO `json:"pagination"`
}

// ListUsers lists all users with pagination.
// @Summary        List users
// @Description    Retrieves a paginated list of users (admin only)
// @Tags           users
// @Produce        json
// @Param          page      query     int  false  "Page number (1-based)"  default(1)
// @Param          pageSize  query     int  false  "Items per page"         default(20)
// @Success        200       {object}  UserListResponse  "Users retrieved successfully"
// @Failure        401       {object}  dto.ErrorResponse  "Unauthorized"
// @Failure        403       {object}  dto.ErrorResponse  "Forbidden"
// @Failure        500       {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/users [get]
func (h *Handler) ListUsers(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse pagination params
	req := dto.PaginationRequest{}
	if err := c.Bind().Query(&req); err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid query parameters: " + err.Error())
	}

	// Normalize pagination
	req.Page, req.PageSize = dto.NormalizePagination(req.Page, req.PageSize, 20)

	// Query users
	users, pagination, err := h.userRepo.ListWithPagination(ctx,
		repository.PaginationParams{Page: req.Page, PageSize: req.PageSize},
	)
	if err != nil {
		h.logger.Error("failed to list users", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// Convert to response
	data := make([]UserResponse, len(users))
	for i, u := range users {
		data[i] = toUserResponse(u)
	}

	return c.JSON(UserListResponse{
		Data: data,
		Pagination: dto.PaginationDTO{
			Page:       pagination.Page,
			PageSize:   pagination.PageSize,
			TotalItems: int64(pagination.TotalItems),
			TotalPages: pagination.TotalPages,
		},
	})
}

// CreateUser creates a new user.
// @Summary        Create user
// @Description    Creates a new user (admin only)
// @Tags           users
// @Accept         json
// @Produce        json
// @Param          body  body      CreateUserRequest  true  "User creation request"
// @Success        201   {object}  UserResponse       "User created successfully"
// @Failure        400   {object}  dto.ErrorResponse  "Invalid request"
// @Failure        401   {object}  dto.ErrorResponse  "Unauthorized"
// @Failure        403   {object}  dto.ErrorResponse  "Forbidden"
// @Failure        409   {object}  dto.ErrorResponse  "User already exists"
// @Failure        500   {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/users [post]
func (h *Handler) CreateUser(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse request
	req := CreateUserRequest{}
	if err := c.Bind().Body(&req); err != nil {
		return litchierrors.New(litchierrors.ErrInvalidRequestBody).
			WithDetail("Invalid request body: " + err.Error())
	}

	// Validate request
	if err := dto.Validate(&req); err != nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("Validation failed: " + err.Error())
	}

	// Check if username already exists
	exists, err := h.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		h.logger.Error("failed to check username existence", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if exists {
		return litchierrors.New(litchierrors.ErrUserAlreadyExists).
			WithDetail("Username already exists: " + req.Username)
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("failed to hash password", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrPasswordHashFailed, err)
	}

	// Create user entity
	user := entity.NewUser(req.Username, passwordHash, entity.UserRole(req.Role))
	if err := user.Validate(); err != nil {
		return err
	}

	// Save user
	if err := h.userRepo.Create(ctx, user); err != nil {
		h.logger.Error("failed to create user", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	h.logger.Info("user created",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username),
		zap.String("role", string(user.Role)),
	)

	return c.Status(201).JSON(toUserResponse(user))
}

// UpdateUser updates an existing user.
// @Summary        Update user
// @Description    Updates an existing user (admin only)
// @Tags           users
// @Accept         json
// @Produce        json
// @Param          id    path      string             true  "User ID (UUID)"
// @Param          body  body      UpdateUserRequest  true  "User update request"
// @Success        200   {object}  UserResponse       "User updated successfully"
// @Failure        400   {object}  dto.ErrorResponse  "Invalid request"
// @Failure        401   {object}  dto.ErrorResponse  "Unauthorized"
// @Failure        403   {object}  dto.ErrorResponse  "Forbidden"
// @Failure        404   {object}  dto.ErrorResponse  "User not found"
// @Failure        409   {object}  dto.ErrorResponse  "Username already exists"
// @Failure        500   {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/users/{id} [put]
func (h *Handler) UpdateUser(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse user ID
	idStr := c.Params("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid user ID format: " + idStr)
	}

	// Parse request
	req := UpdateUserRequest{}
	if err := c.Bind().Body(&req); err != nil {
		return litchierrors.New(litchierrors.ErrInvalidRequestBody).
			WithDetail("Invalid request body: " + err.Error())
	}

	// Validate request
	if err := dto.Validate(&req); err != nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("Validation failed: " + err.Error())
	}

	// Find user
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		h.logger.Error("failed to find user", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if user == nil {
		return litchierrors.New(litchierrors.ErrUserNotFound).
			WithDetail("User not found: " + idStr)
	}

	// Update fields
	if req.Username != nil && *req.Username != "" {
		// Check if new username is taken by another user
		existing, err := h.userRepo.FindByUsername(ctx, *req.Username)
		if err != nil {
			h.logger.Error("failed to check username", zap.Error(err))
			return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
		}
		if existing != nil && existing.ID != userID {
			return litchierrors.New(litchierrors.ErrUserAlreadyExists).
				WithDetail("Username already exists: " + *req.Username)
		}
		if err := user.SetUsername(*req.Username); err != nil {
			return err
		}
	}

	if req.Password != nil && *req.Password != "" {
		passwordHash, err := auth.HashPassword(*req.Password)
		if err != nil {
			h.logger.Error("failed to hash password", zap.Error(err))
			return litchierrors.Wrap(litchierrors.ErrPasswordHashFailed, err)
		}
		user.SetPasswordHash(passwordHash)
	}

	if req.Role != nil && *req.Role != "" {
		if err := user.SetRole(entity.UserRole(*req.Role)); err != nil {
			return err
		}
	}

	// Validate updated user
	if err := user.Validate(); err != nil {
		return err
	}

	// Save user
	if err := h.userRepo.Update(ctx, user); err != nil {
		h.logger.Error("failed to update user", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	h.logger.Info("user updated",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username),
	)

	return c.JSON(toUserResponse(user))
}

// DeleteUser deletes a user.
// @Summary        Delete user
// @Description    Deletes a user (admin only)
// @Tags           users
// @Param          id  path      string  true  "User ID (UUID)"
// @Success        204  "User deleted successfully"
// @Failure        400  {object}  dto.ErrorResponse  "Invalid user ID"
// @Failure        401  {object}  dto.ErrorResponse  "Unauthorized"
// @Failure        403  {object}  dto.ErrorResponse  "Forbidden"
// @Failure        404  {object}  dto.ErrorResponse  "User not found"
// @Failure        500  {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/users/{id} [delete]
func (h *Handler) DeleteUser(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse user ID
	idStr := c.Params("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		return litchierrors.New(litchierrors.ErrInvalidQueryParam).
			WithDetail("Invalid user ID format: " + idStr)
	}

	// Get current user
	claims := authmiddleware.GetUserFromContext(c)
	if claims != nil && claims.ID == userID {
		return litchierrors.New(litchierrors.ErrBadRequest).
			WithDetail("Cannot delete your own account")
	}

	// Check user exists
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		h.logger.Error("failed to find user", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if user == nil {
		return litchierrors.New(litchierrors.ErrUserNotFound).
			WithDetail("User not found: " + idStr)
	}

	// Delete user
	if err := h.userRepo.Delete(ctx, userID); err != nil {
		h.logger.Error("failed to delete user", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	h.logger.Info("user deleted",
		zap.String("user_id", userID.String()),
		zap.String("username", user.Username),
	)

	return c.SendStatus(204)
}

// toUserResponse converts a User entity to UserResponse.
func toUserResponse(user *entity.User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
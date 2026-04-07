// Package auth provides HTTP handlers for authentication API.
package auth

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/ryuyb/litchi/internal/application/dto"
	authmiddleware "github.com/ryuyb/litchi/internal/application/server/middleware/auth"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Handler handles authentication HTTP requests.
type Handler struct {
	userRepo repository.UserRepository
	auth     *authmiddleware.Middleware
	logger   *zap.Logger
}

// HandlerParams contains dependencies for creating an auth handler.
type HandlerParams struct {
	fx.In

	UserRepo repository.UserRepository
	Auth     *authmiddleware.Middleware
	Logger   *zap.Logger
}

// NewHandler creates a new auth handler.
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		userRepo: p.UserRepo,
		auth:     p.Auth,
		logger:   p.Logger.Named("auth_handler"),
	}
}

// LoginRequest represents the login request body.
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

// LoginResponse represents the login response.
type LoginResponse struct {
	User *UserResponse `json:"user"`
}

// UserResponse represents user information in responses.
type UserResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

// Login handles user login.
// @Summary        User login
// @Description    Authenticates a user and creates a session
// @Tags           auth
// @Accept         json
// @Produce        json
// @Param          body  body      LoginRequest  true  "Login credentials"
// @Success        200   {object}  LoginResponse  "Login successful"
// @Failure        400   {object}  dto.ErrorResponse  "Invalid request"
// @Failure        401   {object}  dto.ErrorResponse  "Invalid credentials"
// @Failure        500   {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/auth/login [post]
func (h *Handler) Login(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse request body
	req := LoginRequest{}
	if err := c.Bind().Body(&req); err != nil {
		return litchierrors.New(litchierrors.ErrInvalidRequestBody).
			WithDetail("Invalid request body: " + err.Error())
	}

	// Validate request
	if err := dto.Validate(&req); err != nil {
		return litchierrors.New(litchierrors.ErrValidationFailed).
			WithDetail("Validation failed: " + err.Error())
	}

	// Find user by username
	user, err := h.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		h.logger.Error("failed to find user", zap.Error(err), zap.String("username", req.Username))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	if user == nil {
		return litchierrors.New(litchierrors.ErrInvalidCredentials).
			WithDetail("Invalid username or password")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return litchierrors.New(litchierrors.ErrInvalidCredentials).
			WithDetail("Invalid username or password")
	}

	// Create session
	if err := h.auth.SetUserSession(c, user); err != nil {
		h.logger.Error("failed to create session", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrSessionOperation, err)
	}

	h.logger.Info("user logged in",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username),
	)

	return c.JSON(LoginResponse{
		User: toUserResponse(user),
	})
}

// Logout handles user logout.
// @Summary        User logout
// @Description    Clears the user session
// @Tags           auth
// @Success        200  {object}  map[string]string  "Logout successful"
// @Failure        500  {object}  dto.ErrorResponse  "Internal server error"
// @Router         /api/v1/auth/logout [post]
func (h *Handler) Logout(c fiber.Ctx) error {
	// Clear session
	if err := h.auth.ClearUserSession(c); err != nil {
		h.logger.Error("failed to clear session", zap.Error(err))
		return litchierrors.Wrap(litchierrors.ErrSessionOperation, err)
	}

	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

// GetMe returns the current authenticated user.
// @Summary        Get current user
// @Description    Returns the currently authenticated user's information
// @Tags           auth
// @Produce        json
// @Success        200  {object}  UserResponse  "Current user info"
// @Failure        401  {object}  dto.ErrorResponse  "Unauthorized"
// @Router         /api/v1/auth/me [get]
func (h *Handler) GetMe(c fiber.Ctx) error {
	claims := authmiddleware.GetUserFromContext(c)
	if claims == nil {
		return litchierrors.New(litchierrors.ErrUnauthorized).
			WithDetail("User not found in context")
	}

	// Fetch fresh user data from database
	ctx := c.Context()
	user, err := h.userRepo.FindByID(ctx, claims.ID)
	if err != nil {
		h.logger.Error("failed to find user", zap.Error(err), zap.String("user_id", claims.ID.String()))
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	if user == nil {
		return litchierrors.New(litchierrors.ErrUserNotFound).
			WithDetail("User not found")
	}

	return c.JSON(toUserResponse(user))
}

// toUserResponse converts a User entity to UserResponse.
func toUserResponse(user *entity.User) *UserResponse {
	if user == nil {
		return nil
	}
	return &UserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// GetUserFromContext is a helper for other handlers to get the current user.
func GetUserFromContext(c fiber.Ctx) *authmiddleware.UserClaims {
	return authmiddleware.GetUserFromContext(c)
}

// GetUserIDFromContext returns the current user ID from context.
func GetUserIDFromContext(c fiber.Ctx) (string, error) {
	claims := authmiddleware.GetUserFromContext(c)
	if claims == nil {
		return "", litchierrors.New(litchierrors.ErrUnauthorized).
			WithDetail("User not authenticated")
	}
	return claims.ID.String(), nil
}

// IsAdmin checks if the current user is an admin.
func IsAdmin(c fiber.Ctx) bool {
	claims := authmiddleware.GetUserFromContext(c)
	return claims != nil && claims.Role == entity.UserRoleAdmin
}

// HashPassword creates a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword verifies a password against a hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Ensure UserContext is available for context operations.
type UserContextKey string

const UserContextKeyUser UserContextKey = "user"

// InjectUserIntoContext injects user info into context for service layer.
func InjectUserIntoContext(ctx context.Context, claims *authmiddleware.UserClaims) context.Context {
	return context.WithValue(ctx, UserContextKeyUser, claims)
}
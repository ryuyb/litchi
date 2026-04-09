// Package auth provides authentication middleware for the HTTP server.
package auth

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides authentication components via Fx.
var Module = fx.Module("auth",
	fx.Provide(NewMiddleware),
)

// Middleware provides authentication middleware for Fiber.
type Middleware struct {
	store  *session.Store
	logger *zap.Logger
}

// MiddlewareParams holds dependencies for creating auth middleware.
type MiddlewareParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

// NewMiddleware creates a new auth middleware instance with session store.
func NewMiddleware(p MiddlewareParams) *Middleware {
	// Get session timeout from config (default: 24 hours)
	idleTimeout := p.Config.Session.GetIdleTimeout()

	// Create session store
	store := session.NewStore(session.Config{
		IdleTimeout:    idleTimeout,
		CookieSecure:   p.Config.Server.Mode == "release",
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	})

	p.Logger.Info("session store created",
		zap.Duration("idle_timeout", idleTimeout),
	)

	return &Middleware{
		store:  store,
		logger: p.Logger.Named("auth-middleware"),
	}
}

// ContextKey represents the key for storing user in context.
type ContextKey string

const (
	UserContextKey ContextKey = "user"
)

// UserClaims represents the user information stored in session.
type UserClaims struct {
	ID       uuid.UUID       `json:"id"`
	Username string          `json:"username"`
	Role     entity.UserRole `json:"role"`
}

// IsAdmin returns true if the user has admin role.
func (u *UserClaims) IsAdmin() bool {
	return u.Role == entity.UserRoleAdmin
}

// RequireAuth returns a middleware that requires authentication.
// It validates the session and injects user info into context.
func (m *Middleware) RequireAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		sess, err := m.store.Get(c)
		if err != nil {
			m.logger.Warn("failed to get session", zap.Error(err))
			return litchierrors.New(litchierrors.ErrSessionExpired).
				WithDetail("Failed to retrieve session")
		}

		// Check if session is valid
		userID := sess.Get("user_id")
		if userID == nil {
			return litchierrors.New(litchierrors.ErrUnauthorized).
				WithDetail("No active session")
		}

		// Get user claims from session with safe type assertions
		userIDStr, ok := userID.(string)
		if !ok {
			return litchierrors.New(litchierrors.ErrUnauthorized).
				WithDetail("Invalid session: corrupted user ID")
		}
		userIDTyped, err := uuid.Parse(userIDStr)
		if err != nil {
			return litchierrors.New(litchierrors.ErrUnauthorized).
				WithDetail("Invalid session: corrupted user ID")
		}

		usernameVal := sess.Get("username")
		if usernameVal == nil {
			return litchierrors.New(litchierrors.ErrUnauthorized).
				WithDetail("Invalid session: missing username")
		}
		username, ok := usernameVal.(string)
		if !ok {
			return litchierrors.New(litchierrors.ErrUnauthorized).
				WithDetail("Invalid session: corrupted username")
		}

		roleVal := sess.Get("role")
		if roleVal == nil {
			return litchierrors.New(litchierrors.ErrUnauthorized).
				WithDetail("Invalid session: missing role")
		}
		role, ok := roleVal.(string)
		if !ok {
			return litchierrors.New(litchierrors.ErrUnauthorized).
				WithDetail("Invalid session: corrupted role")
		}

		claims := &UserClaims{
			ID:       userIDTyped,
			Username: username,
			Role:     entity.UserRole(role),
		}

		// Store user in context for handlers
		c.Locals(UserContextKey, claims)

		return c.Next()
	}
}

// RequireAdmin returns a middleware that requires admin role.
// Must be used after RequireAuth middleware.
func (m *Middleware) RequireAdmin() fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetUserFromContext(c)
		if user == nil {
			return litchierrors.New(litchierrors.ErrUnauthorized).
				WithDetail("User not found in context")
		}

		if !user.IsAdmin() {
			return litchierrors.New(litchierrors.ErrPermissionDenied).
				WithDetail("Admin role required")
		}

		return c.Next()
	}
}

// GetUserFromContext retrieves user claims from Fiber context.
// Returns nil if user not found or type assertion fails.
func GetUserFromContext(c fiber.Ctx) *UserClaims {
	user := c.Locals(UserContextKey)
	if user == nil {
		return nil
	}
	claims, ok := user.(*UserClaims)
	if !ok {
		return nil
	}
	return claims
}

// GetSessionStore returns the session store for use in handlers.
func (m *Middleware) GetSessionStore() *session.Store {
	return m.store
}

// SetUserSession stores user information in the session.
// It regenerates the session ID first to prevent session fixation attacks.
func (m *Middleware) SetUserSession(c fiber.Ctx, user *entity.User) error {
	sess, err := m.store.Get(c)
	if err != nil {
		return err
	}

	// Regenerate session ID to prevent session fixation attacks
	if err := sess.Regenerate(); err != nil {
		return err
	}

	sess.Set("user_id", user.ID.String())
	sess.Set("username", user.Username)
	sess.Set("role", string(user.Role))

	return sess.Save()
}

// ClearUserSession clears the user session (logout).
func (m *Middleware) ClearUserSession(c fiber.Ctx) error {
	sess, err := m.store.Get(c)
	if err != nil {
		return err
	}

	return sess.Destroy()
}
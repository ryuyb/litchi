package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// UserRole represents the role of a user.
type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"  // Can manage users and system configuration
	UserRoleViewer UserRole = "viewer" // Can view sessions and logs
)

// IsValid checks if the role is valid.
func (r UserRole) IsValid() bool {
	return r == UserRoleAdmin || r == UserRoleViewer
}

// User represents a user entity for web UI authentication.
type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose password hash in JSON
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// NewUser creates a new User entity with the given attributes.
func NewUser(username, passwordHash string, role UserRole) *User {
	return &User{
		ID:           uuid.New(),
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// Validate validates the User entity.
func (u *User) Validate() error {
	if u.Username == "" {
		return errors.New(errors.ErrValidationFailed).WithDetail("username cannot be empty")
	}
	if len(u.Username) < 3 {
		return errors.New(errors.ErrValidationFailed).WithDetail("username must be at least 3 characters")
	}
	if len(u.Username) > 50 {
		return errors.New(errors.ErrValidationFailed).WithDetail("username must be at most 50 characters")
	}
	if u.PasswordHash == "" {
		return errors.New(errors.ErrValidationFailed).WithDetail("password hash cannot be empty")
	}
	if !u.Role.IsValid() {
		return errors.New(errors.ErrValidationFailed).WithDetail("invalid role, must be 'admin' or 'viewer'")
	}
	return nil
}

// IsAdmin returns true if the user has admin role.
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// SetPasswordHash updates the user's password hash.
func (u *User) SetPasswordHash(hash string) {
	u.PasswordHash = hash
	u.UpdatedAt = time.Now()
}

// SetRole updates the user's role.
func (u *User) SetRole(role UserRole) error {
	if !role.IsValid() {
		return errors.New(errors.ErrValidationFailed).WithDetail("invalid role")
	}
	u.Role = role
	u.UpdatedAt = time.Now()
	return nil
}

// SetUsername updates the user's username.
func (u *User) SetUsername(username string) error {
	if username == "" {
		return errors.New(errors.ErrValidationFailed).WithDetail("username cannot be empty")
	}
	if len(username) < 3 || len(username) > 50 {
		return errors.New(errors.ErrValidationFailed).WithDetail("username must be between 3 and 50 characters")
	}
	u.Username = username
	u.UpdatedAt = time.Now()
	return nil
}
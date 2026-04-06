// Package dto provides Data Transfer Objects for API request/response structures.
package dto

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

// Shared validator instance (lazy initialized with thread-safe sync.Once).
var (
	validate *validator.Validate
	once     sync.Once
)

// GetValidator returns the shared validator instance.
// Uses sync.Once for thread-safe lazy initialization.
func GetValidator() *validator.Validate {
	once.Do(func() {
		validate = validator.New()
	})
	return validate
}

// Validate validates the given struct using the validator tags.
// Returns the validation error if validation fails.
func Validate(s any) error {
	if err := GetValidator().Struct(s); err != nil {
		return err
	}
	return nil
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Code    string `json:"code" example:"BAD_REQUEST"`
	Message string `json:"message" example:"Invalid request parameters"`
	// Details contains additional error context (field names, validation rules, etc.)
	// Type varies by error code: map[string]string for validation, string for general errors
	Details any `json:"details,omitempty" example:"{\"field\":\"email\",\"rule\":\"required\"}"`
} // @name ApiError

// SuccessResponse represents a generic success response.
type SuccessResponse struct {
	Status  string `json:"status" example:"success"`
	Message string `json:"message,omitempty" example:"Operation completed"`
} // @name Success

// HealthCheckResponse represents the health check endpoint response.
type HealthCheckResponse struct {
	Status  string `json:"status" example:"healthy"`
	Version string `json:"version" example:"0.1.0"`
} // @name HealthCheck
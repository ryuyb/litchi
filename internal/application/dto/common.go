// Package dto provides Data Transfer Objects for API request/response structures.
package dto

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Code    string `json:"code" example:"BAD_REQUEST"`
	Message string `json:"message" example:"Invalid request parameters"`
	// Details contains additional error context (field names, validation rules, etc.)
	// Type varies by error code: map[string]string for validation, string for general errors
	Details any `json:"details,omitempty" example:"{\"field\":\"email\",\"rule\":\"required\"}"`
}

// SuccessResponse represents a generic success response.
type SuccessResponse struct {
	Status  string `json:"status" example:"success"`
	Message string `json:"message,omitempty" example:"Operation completed"`
}

// HealthCheckResponse represents the health check endpoint response.
type HealthCheckResponse struct {
	Status  string `json:"status" example:"healthy"`
	Version string `json:"version" example:"0.1.0"`
}
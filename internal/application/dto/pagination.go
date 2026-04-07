// Package dto provides Data Transfer Objects for API request/response structures.
package dto

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
)

const (
	// MaxPageSize is the maximum allowed page size for list queries.
	MaxPageSize = 100
	// DefaultPageSize is the default page size when not specified.
	DefaultPageSize = 20
)

// PaginationDTO represents pagination metadata for list responses.
type PaginationDTO struct {
	Page       int   `json:"page" example:"1"`
	PageSize   int   `json:"pageSize" example:"20"`
	TotalItems int64 `json:"totalItems" example:"100"`
	TotalPages int   `json:"totalPages" example:"5"`
} // @name Pagination

// PaginationRequest represents common pagination query parameters.
type PaginationRequest struct {
	Page     int `query:"page" example:"1"`
	PageSize int `query:"pageSize" example:"20"`
} // @name PaginationRequest

// PaginatedResponse represents a paginated list response with generic data type.
type PaginatedResponse[T any] struct {
	Data       []T           `json:"data"`
	Pagination PaginationDTO `json:"pagination"`
} // @name PaginatedResponse

// ValidationError represents a single field validation error.
type ValidationError struct {
	Field   string `json:"field" example:"session_id"`
	Rule    string `json:"rule" example:"required"`
	Message string `json:"message" example:"Session ID is required"`
} // @name ValidationError

// ValidationErrorResponse represents validation error response with field details.
type ValidationErrorResponse struct {
	Code    string            `json:"code" example:"L4API0002"`
	Message string            `json:"message" example:"Validation failed"`
	Errors  []ValidationError `json:"errors"`
} // @name ValidationErrorResponse

// NormalizePagination validates and normalizes pagination parameters.
// Returns normalized page and pageSize values.
// If page < 1, it defaults to 1.
// If pageSize < 1, it defaults to defaultPageSize.
// If pageSize > MaxPageSize, it is capped to MaxPageSize.
func NormalizePagination(page, pageSize, defaultPageSize int) (normalizedPage, normalizedPageSize int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

// ParseQueryInt parses a query parameter as an integer with a default value.
// Returns the default value if the parameter is missing or invalid.
func ParseQueryInt(c fiber.Ctx, key string, defaultValue int) int {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intVal
}
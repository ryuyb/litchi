// Package dto provides Data Transfer Objects for API request/response structures.
package dto

import "time"

// HealthDetailResponse represents detailed health check response.
type HealthDetailResponse struct {
	Status    string            `json:"status" example:"healthy"` // healthy, degraded, unhealthy
	Version   string            `json:"version" example:"0.1.0"`
	Timestamp time.Time         `json:"timestamp" example:"2024-01-01T00:00:00Z"`
	Checks    []HealthCheckItem `json:"checks"`
} // @name HealthDetail

// HealthCheckItem represents a single health check result.
type HealthCheckItem struct {
	Name      string        `json:"name" example:"database"` // database, github, git
	Status    string        `json:"status" example:"pass"`   // pass, fail, warn
	Message   string        `json:"message,omitempty" example:"Connection OK"`
	LatencyMs int           `json:"latencyMs,omitempty" example:"5"`
	Error     string        `json:"error,omitempty" example:"Connection timeout"`
	Details   map[string]any `json:"details,omitempty"`
} // @name HealthCheckItem

// ToHealthDetailResponse creates detailed health response.
func ToHealthDetailResponse(status, version string, checks []HealthCheckItem) HealthDetailResponse {
	return HealthDetailResponse{
		Status:    status,
		Version:   version,
		Timestamp: time.Now(),
		Checks:    checks,
	}
}
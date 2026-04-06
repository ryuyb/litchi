// Package dto provides Data Transfer Objects for API request/response structures.
package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
)

// AuditLogResponse represents an audit log entry.
type AuditLogResponse struct {
	ID           uuid.UUID `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	SessionID    uuid.UUID `json:"sessionId"`
	Repository   string    `json:"repository"`
	IssueNumber  int       `json:"issueNumber"`
	Actor        string    `json:"actor"`
	ActorRole    string    `json:"actorRole"`
	Operation    string    `json:"operation"`
	ResourceType string    `json:"resourceType"`
	ResourceID   string    `json:"resourceId"`
	Result       string    `json:"result"`
	Duration     int       `json:"duration"` // milliseconds
	Output       string    `json:"output,omitempty"`
	Error        string    `json:"error,omitempty"`
} // @name AuditLog

// AuditLogListRequest represents audit log query parameters.
type AuditLogListRequest struct {
	Page       int    `query:"page" default:"1"`
	PageSize   int    `query:"pageSize" default:"50"`
	SessionID  string `query:"sessionId" example:"550e8400-e29b-41d4-a716-446655440000"`
	Repository string `query:"repository" example:"owner/repo"`
	Actor      string `query:"actor" example:"username"`
	Operation  string `query:"operation" example:"session_start"`
	Result     string `query:"result" example:"success"`
	StartTime  string `query:"startTime" example:"2024-01-01T00:00:00Z"`
	EndTime    string `query:"endTime" example:"2024-01-02T00:00:00Z"`
	OrderBy    string `query:"orderBy" default:"timestamp desc"`
} // @name AuditLogList

// AuditSummaryResponse represents audit summary for a session.
type AuditSummaryResponse struct {
	SessionID         uuid.UUID      `json:"sessionId"`
	TotalCount        int            `json:"totalCount"`
	TotalDurationMs   int            `json:"totalDurationMs"`
	AverageDurationMs int            `json:"averageDurationMs"`
	ByResult          map[string]int `json:"byResult"`
	ByOperation       map[string]int `json:"byOperation"`
	FirstTimestamp    time.Time      `json:"firstTimestamp,omitempty"`
	LastTimestamp     time.Time      `json:"lastTimestamp,omitempty"`
} // @name AuditSummary

// ToAuditLogResponse converts entity.AuditLog to DTO.
func ToAuditLogResponse(log *entity.AuditLog) AuditLogResponse {
	return AuditLogResponse{
		ID:           log.ID,
		Timestamp:    log.Timestamp,
		SessionID:    log.SessionID,
		Repository:   log.Repository,
		IssueNumber:  log.IssueNumber,
		Actor:        log.Actor,
		ActorRole:    string(log.ActorRole),
		Operation:    string(log.Operation),
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		Result:       string(log.Result),
		Duration:     log.Duration,
		Output:       log.Output,
		Error:        log.Error,
	}
}

// ToAuditLogList converts audit logs to paginated response.
func ToAuditLogList(logs []*entity.AuditLog, page, pageSize int, total int64) PaginatedResponse[AuditLogResponse] {
	data := make([]AuditLogResponse, len(logs))
	for i, log := range logs {
		data[i] = ToAuditLogResponse(log)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return PaginatedResponse[AuditLogResponse]{
		Data: data,
		Pagination: PaginationDTO{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}
}
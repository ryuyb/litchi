// Package dto provides Data Transfer Objects for API request/response structures.
package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// SessionResponse represents a work session in API response.
type SessionResponse struct {
	ID           uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Repository   string    `json:"repository" example:"owner/repo"`
	IssueNumber  int       `json:"issueNumber" example:"123"`
	IssueTitle   string    `json:"issueTitle" example:"Fix bug in login"`
	CurrentStage string    `json:"currentStage" example:"execution" enums:"clarification,design,task_breakdown,execution,pull_request,completed"`
	Status       string    `json:"status" example:"active" enums:"active,paused,completed,terminated"`
	CreatedAt    time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt    time.Time `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
	PRNumber     *int      `json:"prNumber,omitempty" example:"456"`

	// Optional embedded details
	Issue     *IssueDTO      `json:"issue,omitempty"`
	Design    *DesignDTO     `json:"design,omitempty"`
	Tasks     []TaskDTO      `json:"tasks,omitempty"`
	Execution *ExecutionDTO  `json:"execution,omitempty"`
} // @name Session

// IssueDTO represents issue details in session response.
type IssueDTO struct {
	Number int    `json:"number" example:"123"`
	Title  string `json:"title" example:"Fix bug in login"`
	Body   string `json:"body,omitempty" example:"Issue description"`
	Author string `json:"author" example:"username"`
	URL    string `json:"url" example:"https://github.com/owner/repo/issues/123"`
} // @name Issue

// DesignDTO represents design document details.
type DesignDTO struct {
	CurrentVersion int    `json:"currentVersion" example:"1"`
	Content        string `json:"content,omitempty" example:"Design document content"`
	Confirmed      bool   `json:"confirmed" example:"false"`
} // @name Design

// ExecutionDTO represents execution phase details.
type ExecutionDTO struct {
	CurrentTaskID   uuid.UUID `json:"currentTaskId,omitempty"`
	CurrentTaskDesc string    `json:"currentTaskDesc,omitempty"`
	StartedAt       time.Time `json:"startedAt,omitempty"`
} // @name Execution

// SessionListRequest represents query parameters for listing sessions.
type SessionListRequest struct {
	Page     int    `query:"page" default:"1"`
	PageSize int    `query:"pageSize" default:"20"`
	Status   string `query:"status" example:"active"`   // active, paused, completed, terminated
	Stage    string `query:"stage" example:"execution"` // clarification, design, task_breakdown, execution, pull_request, completed
	Repo     string `query:"repo" example:"owner/repo"`
} // @name SessionList

// PauseSessionRequest represents pause request body.
type PauseSessionRequest struct {
	Reason string `json:"reason" example:"user_request" validate:"required"`
} // @name PauseSession

// ResumeSessionRequest represents resume request body.
type ResumeSessionRequest struct {
	Action string `json:"action,omitempty" example:"manual_resume"`
} // @name ResumeSession

// RollbackSessionRequest represents rollback request body.
type RollbackSessionRequest struct {
	TargetStage string `json:"targetStage" example:"design" validate:"required,oneof=clarification design task_breakdown"`
	Reason      string `json:"reason" example:"design_needs_update" validate:"required"`
} // @name RollbackSession

// TerminateSessionRequest represents terminate request body.
type TerminateSessionRequest struct {
	Reason string `json:"reason" example:"user_request" validate:"required"`
} // @name TerminateSession

// ToSessionResponse converts WorkSession aggregate to DTO.
func ToSessionResponse(session *aggregate.WorkSession) SessionResponse {
	var prNumber *int
	if session.PRNumber != nil {
		prNumber = session.PRNumber
	}

	return SessionResponse{
		ID:           session.ID,
		Repository:   session.Issue.Repository,
		IssueNumber:  session.Issue.Number,
		IssueTitle:   session.Issue.Title,
		CurrentStage: session.CurrentStage.String(),
		Status:       getSessionStatus(session),
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
		PRNumber:     prNumber,
	}
}

// ToSessionDetailResponse converts WorkSession with full details.
func ToSessionDetailResponse(session *aggregate.WorkSession) SessionResponse {
	resp := ToSessionResponse(session)

	// Issue details
	resp.Issue = &IssueDTO{
		Number: session.Issue.Number,
		Title:  session.Issue.Title,
		Body:   session.Issue.Body,
		Author: session.Issue.Author,
		URL:    session.Issue.URL,
	}

	// Design details (if exists)
	if session.Design != nil {
		resp.Design = &DesignDTO{
			CurrentVersion: session.Design.CurrentVersion,
			Content:        session.Design.GetCurrentContent(),
			Confirmed:      session.Design.Confirmed,
		}
	}

	// Tasks details (if exists)
	if len(session.Tasks) > 0 {
		resp.Tasks = make([]TaskDTO, len(session.Tasks))
		for i, task := range session.Tasks {
			resp.Tasks[i] = ToTaskDTO(task)
		}
	}

	return resp
}

// getSessionStatus determines session status string.
func getSessionStatus(session *aggregate.WorkSession) string {
	switch {
	case session.IsTerminated():
		return "terminated"
	case session.IsPaused():
		return "paused"
	case session.IsCompletedSession():
		return "completed"
	default:
		return "active"
	}
}

// StageDTO represents a workflow stage with metadata.
type StageDTO struct {
	Name        string `json:"name" example:"execution"`
	DisplayName string `json:"displayName" example:"Execution"`
	Order       int    `json:"order" example:"4"`
	CanRollback bool   `json:"canRollback" example:"true"`
} // @name Stage

// ToStageDTOs converts all stages to DTOs.
func ToStageDTOs() []StageDTO {
	stages := valueobject.AllStages()
	result := make([]StageDTO, len(stages))
	for i, stage := range stages {
		// Determine rollback capability based on stage position
		// Stages that are not Clarification or Completed can rollback
		canRollback := stage != valueobject.StageClarification && stage != valueobject.StageCompleted
		result[i] = StageDTO{
			Name:        stage.String(),
			DisplayName: stage.DisplayName(),
			Order:       stage.Order(),
			CanRollback: canRollback,
		}
	}
	return result
}
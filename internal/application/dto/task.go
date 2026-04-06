// Package dto provides Data Transfer Objects for API request/response structures.
package dto

import (
	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// TaskDTO represents a task in API response.
type TaskDTO struct {
	ID            uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Description   string    `json:"description" example:"Implement login logic"`
	Status        string    `json:"status" example:"completed"`
	StatusDisplay string    `json:"statusDisplay" example:"Completed"`
	Order         int       `json:"order" example:"1"`
	RetryCount    int       `json:"retryCount" example:"0"`
	Dependencies  []string  `json:"dependencies" example:"[\"task-id-1\"]"`
	CanExecute    bool      `json:"canExecute" example:"true"`
	FailureReason string    `json:"failureReason,omitempty" example:"Test failed"`
	Suggestion    string    `json:"suggestion,omitempty" example:"Fix the test"`
} // @name Task

// TaskListResponse represents the task list for a session.
type TaskListResponse struct {
	SessionID      uuid.UUID `json:"sessionId"`
	TotalTasks     int       `json:"totalTasks"`
	Completed      int       `json:"completed"`
	InProgress     int       `json:"inProgress"`
	Pending        int       `json:"pending"`
	Failed         int       `json:"failed"`
	Skipped        int       `json:"skipped"`
	AllCompleted   bool      `json:"allCompleted"`
	HasFailedTask  bool      `json:"hasFailedTask"`
	MaxRetryLimit  int       `json:"maxRetryLimit"`
	Tasks          []TaskDTO `json:"tasks"`
} // @name TaskList

// SkipTaskRequest represents skip task request body.
type SkipTaskRequest struct {
	Reason string `json:"reason" example:"not_applicable" validate:"required"`
} // @name SkipTask

// RetryTaskRequest represents retry task request body.
type RetryTaskRequest struct {
	Force bool `json:"force,omitempty" example:"false"` // Force retry even if dependencies not met
} // @name RetryTask

// ToTaskDTO converts entity.Task to DTO.
func ToTaskDTO(task *entity.Task) TaskDTO {
	deps := make([]string, len(task.Dependencies))
	for i, depID := range task.Dependencies {
		deps[i] = depID.String()
	}

	// A task can be executed if it's pending and has no dependencies
	// or all dependencies are satisfied (checked by caller)
	canExecute := task.IsPending() || task.IsFailed()

	return TaskDTO{
		ID:            task.ID,
		Description:   task.Description,
		Status:        task.Status.String(),
		StatusDisplay: task.Status.DisplayName(),
		Order:         task.Order,
		RetryCount:    task.RetryCount,
		Dependencies:  deps,
		CanExecute:    canExecute,
		FailureReason: task.FailureReason,
		Suggestion:    task.Suggestion,
	}
}

// ToTaskListResponse converts tasks to list response with statistics.
func ToTaskListResponse(sessionID uuid.UUID, tasks []*entity.Task, maxRetryLimit int) TaskListResponse {
	taskDTOs := make([]TaskDTO, len(tasks))
	stats := struct {
		completed, inProgress, pending, failed, skipped int
	}{}

	for i, task := range tasks {
		taskDTOs[i] = ToTaskDTO(task)
		switch task.Status {
		case valueobject.TaskStatusCompleted:
			stats.completed++
		case valueobject.TaskStatusInProgress:
			stats.inProgress++
		case valueobject.TaskStatusPending:
			stats.pending++
		case valueobject.TaskStatusFailed:
			stats.failed++
		case valueobject.TaskStatusSkipped:
			stats.skipped++
		}
	}

	return TaskListResponse{
		SessionID:     sessionID,
		TotalTasks:    len(tasks),
		Completed:     stats.completed,
		InProgress:    stats.inProgress,
		Pending:      stats.pending,
		Failed:        stats.failed,
		Skipped:       stats.skipped,
		AllCompleted:  stats.completed+stats.skipped == len(tasks) && len(tasks) > 0,
		HasFailedTask: stats.failed > 0,
		MaxRetryLimit: maxRetryLimit,
		Tasks:         taskDTOs,
	}
}
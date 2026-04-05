package valueobject

import (
	"database/sql/driver"
	"fmt"

	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// TaskStatus represents the task status enum.
// The task state machine: Pending → InProgress → Completed / Failed / Skipped
// Failed tasks can retry (transition back to InProgress).
type TaskStatus string

// Task status constants (5 states as per T2.1.2 requirement)
const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusSkipped    TaskStatus = "skipped"
)

// AllTaskStatuses returns all valid task statuses.
func AllTaskStatuses() []TaskStatus {
	return []TaskStatus{
		TaskStatusPending,
		TaskStatusInProgress,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusSkipped,
	}
}

// ParseTaskStatus parses a string into a TaskStatus.
// Returns an error if the string is not a valid task status.
func ParseTaskStatus(statusStr string) (TaskStatus, error) {
	for _, status := range AllTaskStatuses() {
		if string(status) == statusStr {
			return status, nil
		}
	}
	return "", errors.New(errors.ErrInvalidTaskStatus).WithDetail(
		fmt.Sprintf("invalid task status: %s, valid statuses are: %v", statusStr, AllTaskStatuses()),
	)
}

// MustParseTaskStatus parses a string into a TaskStatus, panics on invalid input.
func MustParseTaskStatus(statusStr string) TaskStatus {
	status, err := ParseTaskStatus(statusStr)
	if err != nil {
		panic(err)
	}
	return status
}

// IsValidTaskStatus checks if the given status is a valid task status.
func IsValidTaskStatus(s TaskStatus) bool {
	return (&s).IsValid()
}

// TaskStatusDisplayName returns the user-friendly display name for the task status.
func TaskStatusDisplayName(s TaskStatus) string {
	return (&s).DisplayName()
}

// Methods with pointer receivers

// String returns the string representation of the task status.
func (s *TaskStatus) String() string {
	if s == nil {
		return ""
	}
	return string(*s)
}

// IsValid checks if the task status is a valid status.
func (s *TaskStatus) IsValid() bool {
	if s == nil {
		return false
	}
	for _, status := range AllTaskStatuses() {
		if *s == status {
			return true
		}
	}
	return false
}

// DisplayName returns the user-friendly display name for the task status.
func (s *TaskStatus) DisplayName() string {
	if s == nil {
		return "未知状态"
	}
	switch *s {
	case TaskStatusPending:
		return "待执行"
	case TaskStatusInProgress:
		return "执行中"
	case TaskStatusCompleted:
		return "已完成"
	case TaskStatusFailed:
		return "失败"
	case TaskStatusSkipped:
		return "已跳过"
	default:
		return "未知状态"
	}
}

// IsTerminal checks if the task status is a terminal status (Completed, Failed, or Skipped).
// Terminal statuses cannot transition to other statuses (except Failed can retry).
func (s *TaskStatus) IsTerminal() bool {
	if s == nil {
		return false
	}
	return *s == TaskStatusCompleted || *s == TaskStatusSkipped
}

// CanStart checks if the task can be started (transition from Pending to InProgress).
func (s *TaskStatus) CanStart() bool {
	if s == nil {
		return false
	}
	return *s == TaskStatusPending
}

// CanComplete checks if the task can be marked as completed (transition from InProgress to Completed).
func (s *TaskStatus) CanComplete() bool {
	if s == nil {
		return false
	}
	return *s == TaskStatusInProgress
}

// CanFail checks if the task can be marked as failed (transition from InProgress to Failed).
func (s *TaskStatus) CanFail() bool {
	if s == nil {
		return false
	}
	return *s == TaskStatusInProgress
}

// CanSkip checks if the task can be skipped (transition from Pending or InProgress to Skipped).
func (s *TaskStatus) CanSkip() bool {
	if s == nil {
		return false
	}
	return *s == TaskStatusPending || *s == TaskStatusInProgress
}

// CanRetry checks if the task can be retried (transition from Failed to InProgress).
func (s *TaskStatus) CanRetry() bool {
	if s == nil {
		return false
	}
	return *s == TaskStatusFailed
}

// CanTransitionTo checks if transition to target status is allowed.
// Valid transitions:
// - Pending → InProgress (start)
// - InProgress → Completed (complete)
// - InProgress → Failed (fail)
// - InProgress → Skipped (skip)
// - Pending → Skipped (skip before starting)
// - Failed → InProgress (retry)
func (s *TaskStatus) CanTransitionTo(target TaskStatus) bool {
	if s == nil {
		return false
	}
	if !s.IsValid() || !target.IsValid() {
		return false
	}

	// Define valid transitions
	validTransitions := map[TaskStatus][]TaskStatus{
		TaskStatusPending:    {TaskStatusInProgress, TaskStatusSkipped},
		TaskStatusInProgress: {TaskStatusCompleted, TaskStatusFailed, TaskStatusSkipped},
		TaskStatusFailed:     {TaskStatusInProgress}, // Retry
		TaskStatusCompleted:  {},                     // Terminal, no transitions
		TaskStatusSkipped:    {},                     // Terminal, no transitions
	}

	allowedTargets, exists := validTransitions[*s]
	if !exists {
		return false
	}

	for _, allowed := range allowedTargets {
		if target == allowed {
			return true
		}
	}
	return false
}

// GORM database serialization implementation

// Value implements driver.Valuer for database serialization.
func (s *TaskStatus) Value() (driver.Value, error) {
	if s == nil {
		return nil, errors.New(errors.ErrInvalidTaskStatus).WithDetail("task status cannot be nil")
	}
	if !s.IsValid() {
		return nil, errors.New(errors.ErrInvalidTaskStatus).WithDetail(
			fmt.Sprintf("cannot serialize invalid task status: %s", *s),
		)
	}
	return s.String(), nil
}

// Scan implements sql.Scanner for database deserialization.
func (s *TaskStatus) Scan(value any) error {
	if value == nil {
		return errors.New(errors.ErrInvalidTaskStatus).WithDetail("task status cannot be null")
	}

	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return errors.New(errors.ErrInvalidTaskStatus).WithDetail(
			fmt.Sprintf("cannot scan task status from type: %T", value),
		)
	}

	status, err := ParseTaskStatus(str)
	if err != nil {
		return err
	}

	*s = status
	return nil
}

// MarshalJSON implements json.Marshaler for JSON serialization.
func (s *TaskStatus) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	if !s.IsValid() {
		return nil, errors.New(errors.ErrInvalidTaskStatus).WithDetail(
			fmt.Sprintf("cannot marshal invalid task status: %s", *s),
		)
	}
	return []byte(fmt.Sprintf(`"%s"`, s.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler for JSON deserialization.
func (s *TaskStatus) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return errors.New(errors.ErrInvalidTaskStatus).WithDetail("task status cannot be null")
	}

	// Remove quotes
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	status, err := ParseTaskStatus(str)
	if err != nil {
		return err
	}
	*s = status
	return nil
}

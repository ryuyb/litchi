package valueobject

// TaskStatus represents the task status enum.
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusSkipped    TaskStatus = "skipped"
	TaskStatusRetrying   TaskStatus = "retrying"
)

// TODO: Implement full value object in T2.1.2
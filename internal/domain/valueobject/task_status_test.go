package valueobject

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ryuyb/litchi/internal/pkg/errors"
)

func TestAllTaskStatuses(t *testing.T) {
	statuses := AllTaskStatuses()
	expected := []TaskStatus{
		TaskStatusPending,
		TaskStatusInProgress,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusSkipped,
	}

	if len(statuses) != len(expected) {
		t.Errorf("AllTaskStatuses() returned %d statuses, expected %d", len(statuses), len(expected))
	}

	for i, status := range statuses {
		if status != expected[i] {
			t.Errorf("AllTaskStatuses()[%d] = %s, expected %s", i, status, expected[i])
		}
	}
}

func TestTaskStatusString(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusInProgress, "in_progress"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusFailed, "failed"},
		{TaskStatusSkipped, "skipped"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			status := tt.status
			if got := (&status).String(); got != tt.expected {
				t.Errorf("(&TaskStatus).String() = %s, expected %s", got, tt.expected)
			}
		})
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		if got := nilStatus.String(); got != "" {
			t.Errorf("nilStatus.String() = %s, expected empty string", got)
		}
	})
}

func TestParseTaskStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected TaskStatus
		hasError bool
	}{
		{"pending", TaskStatusPending, false},
		{"in_progress", TaskStatusInProgress, false},
		{"completed", TaskStatusCompleted, false},
		{"failed", TaskStatusFailed, false},
		{"skipped", TaskStatusSkipped, false},
		// Invalid inputs
		{"invalid", "", true},
		{"", "", true},
		{"running", "", true}, // Old status name, not valid
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseTaskStatus(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("ParseTaskStatus(%s) expected error, got none", tt.input)
				}
				if !errors.Is(err, errors.ErrInvalidTaskStatus) {
					t.Errorf("ParseTaskStatus(%s) error should be ErrInvalidTaskStatus, got %v", tt.input, err)
				}
			} else {
				if err != nil {
					t.Errorf("ParseTaskStatus(%s) unexpected error: %v", tt.input, err)
				}
				if got != tt.expected {
					t.Errorf("ParseTaskStatus(%s) = %s, expected %s", tt.input, got, tt.expected)
				}
			}
		})
	}
}

func TestMustParseTaskStatus(t *testing.T) {
	// Valid input
	status := MustParseTaskStatus("pending")
	if status != TaskStatusPending {
		t.Errorf("MustParseTaskStatus(pending) = %s, expected %s", status, TaskStatusPending)
	}

	// Invalid input should panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustParseTaskStatus(invalid) should have panicked")
		}
	}()
	MustParseTaskStatus("invalid")
}

func TestTaskStatusIsValid(t *testing.T) {
	validStatuses := AllTaskStatuses()
	for _, status := range validStatuses {
		if !(&status).IsValid() {
			t.Errorf("(&TaskStatus(%s)).IsValid() should be true", status)
		}
		if !IsValidTaskStatus(status) {
			t.Errorf("IsValidTaskStatus(%s) should be true", status)
		}
	}

	invalidStatuses := []TaskStatus{"invalid", "", "running", "retrying"}
	for _, status := range invalidStatuses {
		if (&status).IsValid() {
			t.Errorf("(&TaskStatus(%s)).IsValid() should be false", status)
		}
		if IsValidTaskStatus(status) {
			t.Errorf("IsValidTaskStatus(%s) should be false", status)
		}
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		if nilStatus.IsValid() {
			t.Errorf("nilStatus.IsValid() should be false")
		}
	})
}

func TestTaskStatusDisplayName(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPending, "待执行"},
		{TaskStatusInProgress, "执行中"},
		{TaskStatusCompleted, "已完成"},
		{TaskStatusFailed, "失败"},
		{TaskStatusSkipped, "已跳过"},
		{TaskStatus("invalid"), "未知状态"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			status := tt.status
			if got := (&status).DisplayName(); got != tt.expected {
				t.Errorf("(&TaskStatus).DisplayName() = %s, expected %s", got, tt.expected)
			}
			if got := TaskStatusDisplayName(tt.status); got != tt.expected {
				t.Errorf("TaskStatusDisplayName(%s) = %s, expected %s", tt.status, got, tt.expected)
			}
		})
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		if nilStatus.DisplayName() != "未知状态" {
			t.Errorf("nilStatus.DisplayName() should be '未知状态'")
		}
	})
}

func TestTaskStatusIsTerminal(t *testing.T) {
	// Terminal statuses
	terminalStatuses := []TaskStatus{TaskStatusCompleted, TaskStatusSkipped}
	for _, status := range terminalStatuses {
		if !(&status).IsTerminal() {
			t.Errorf("(&TaskStatus(%s)).IsTerminal() should be true", status)
		}
	}

	// Non-terminal statuses
	nonTerminalStatuses := []TaskStatus{TaskStatusPending, TaskStatusInProgress, TaskStatusFailed}
	for _, status := range nonTerminalStatuses {
		if (&status).IsTerminal() {
			t.Errorf("(&TaskStatus(%s)).IsTerminal() should be false", status)
		}
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		if nilStatus.IsTerminal() {
			t.Errorf("nilStatus.IsTerminal() should be false")
		}
	})
}

func TestTaskStatusCanStart(t *testing.T) {
	pending := TaskStatusPending
	if !(&pending).CanStart() {
		t.Errorf("Pending.CanStart() should be true")
	}

	inProgress := TaskStatusInProgress
	if (&inProgress).CanStart() {
		t.Errorf("InProgress.CanStart() should be false")
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		if nilStatus.CanStart() {
			t.Errorf("nilStatus.CanStart() should be false")
		}
	})
}

func TestTaskStatusCanComplete(t *testing.T) {
	inProgress := TaskStatusInProgress
	if !(&inProgress).CanComplete() {
		t.Errorf("InProgress.CanComplete() should be true")
	}

	pending := TaskStatusPending
	if (&pending).CanComplete() {
		t.Errorf("Pending.CanComplete() should be false")
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		if nilStatus.CanComplete() {
			t.Errorf("nilStatus.CanComplete() should be false")
		}
	})
}

func TestTaskStatusCanFail(t *testing.T) {
	inProgress := TaskStatusInProgress
	if !(&inProgress).CanFail() {
		t.Errorf("InProgress.CanFail() should be true")
	}

	pending := TaskStatusPending
	if (&pending).CanFail() {
		t.Errorf("Pending.CanFail() should be false")
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		if nilStatus.CanFail() {
			t.Errorf("nilStatus.CanFail() should be false")
		}
	})
}

func TestTaskStatusCanSkip(t *testing.T) {
	// Can skip from Pending or InProgress
	pending := TaskStatusPending
	if !(&pending).CanSkip() {
		t.Errorf("Pending.CanSkip() should be true")
	}

	inProgress := TaskStatusInProgress
	if !(&inProgress).CanSkip() {
		t.Errorf("InProgress.CanSkip() should be true")
	}

	// Cannot skip from other statuses
	failed := TaskStatusFailed
	if (&failed).CanSkip() {
		t.Errorf("Failed.CanSkip() should be false")
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		if nilStatus.CanSkip() {
			t.Errorf("nilStatus.CanSkip() should be false")
		}
	})
}

func TestTaskStatusCanRetry(t *testing.T) {
	failed := TaskStatusFailed
	if !(&failed).CanRetry() {
		t.Errorf("Failed.CanRetry() should be true")
	}

	completed := TaskStatusCompleted
	if (&completed).CanRetry() {
		t.Errorf("Completed.CanRetry() should be false")
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		if nilStatus.CanRetry() {
			t.Errorf("nilStatus.CanRetry() should be false")
		}
	})
}

func TestTaskStatusCanTransitionTo(t *testing.T) {
	// Valid transitions
	validTransitions := []struct {
		from, to TaskStatus
	}{
		{TaskStatusPending, TaskStatusInProgress},    // Start
		{TaskStatusPending, TaskStatusSkipped},       // Skip before starting
		{TaskStatusInProgress, TaskStatusCompleted},  // Complete
		{TaskStatusInProgress, TaskStatusFailed},     // Fail
		{TaskStatusInProgress, TaskStatusSkipped},    // Skip during execution
		{TaskStatusFailed, TaskStatusInProgress},     // Retry
	}

	for _, tt := range validTransitions {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if !(&tt.from).CanTransitionTo(tt.to) {
				t.Errorf("CanTransitionTo(%s -> %s) should be true", tt.from, tt.to)
			}
		})
	}

	// Invalid transitions
	invalidTransitions := []struct {
		from, to TaskStatus
	}{
		{TaskStatusPending, TaskStatusCompleted},     // Cannot skip InProgress
		{TaskStatusCompleted, TaskStatusInProgress},  // Terminal status
		{TaskStatusSkipped, TaskStatusInProgress},    // Terminal status
		{TaskStatusFailed, TaskStatusCompleted},      // Cannot complete from Failed
		{TaskStatusInProgress, TaskStatusPending},    // Cannot go back
	}

	for _, tt := range invalidTransitions {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			if (&tt.from).CanTransitionTo(tt.to) {
				t.Errorf("CanTransitionTo(%s -> %s) should be false", tt.from, tt.to)
			}
		})
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		if nilStatus.CanTransitionTo(TaskStatusInProgress) {
			t.Errorf("nilStatus.CanTransitionTo() should be false")
		}
	})
}

func TestTaskStatusJSONSerialization(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPending, `"pending"`},
		{TaskStatusInProgress, `"in_progress"`},
		{TaskStatusCompleted, `"completed"`},
		{TaskStatusFailed, `"failed"`},
		{TaskStatusSkipped, `"skipped"`},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			status := tt.status
			data, err := json.Marshal(&status)
			if err != nil {
				t.Errorf("Marshal(&%s) unexpected error: %v", tt.status, err)
			}
			if string(data) != tt.expected {
				t.Errorf("Marshal(&%s) = %s, expected %s", tt.status, data, tt.expected)
			}
		})
	}

	// Invalid status should not marshal
	invalidStatus := TaskStatus("invalid")
	_, err := json.Marshal(&invalidStatus)
	if err == nil {
		t.Errorf("Marshal(&invalid) should return error")
	}

	// Test nil behavior: nil TaskStatus serializes to "null"
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		data, err := json.Marshal(nilStatus)
		if err != nil {
			t.Errorf("Marshal(nil) unexpected error: %v", err)
		}
		if string(data) != "null" {
			t.Errorf("Marshal(nil) = %s, expected 'null'", data)
		}
	})
}

func TestTaskStatusJSONDeserialization(t *testing.T) {
	tests := []struct {
		input    string
		expected TaskStatus
		hasError bool
	}{
		{`"pending"`, TaskStatusPending, false},
		{`"in_progress"`, TaskStatusInProgress, false},
		{`"completed"`, TaskStatusCompleted, false},
		{`"failed"`, TaskStatusFailed, false},
		{`"skipped"`, TaskStatusSkipped, false},
		{`"invalid"`, "", true},
		{`""`, "", true},
		{`null`, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var status TaskStatus
			err := json.Unmarshal([]byte(tt.input), &status)

			if tt.hasError {
				if err == nil {
					t.Errorf("Unmarshal(%s) expected error, got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unmarshal(%s) unexpected error: %v", tt.input, err)
				}
				if status != tt.expected {
					t.Errorf("Unmarshal(%s) = %s, expected %s", tt.input, status, tt.expected)
				}
			}
		})
	}
}

func TestTaskStatusValue(t *testing.T) {
	// Valid statuses
	for _, status := range AllTaskStatuses() {
		value, err := (&status).Value()
		if err != nil {
			t.Errorf("Value(&%s) unexpected error: %v", status, err)
		}
		if value != string(status) {
			t.Errorf("Value(&%s) = %v, expected %s", status, value, status)
		}
	}

	// Invalid status
	invalidStatus := TaskStatus("invalid")
	_, err := (&invalidStatus).Value()
	if err == nil {
		t.Errorf("Value(&invalid) should return error")
	}

	// Test nil safety
	t.Run("nil", func(t *testing.T) {
		var nilStatus *TaskStatus
		_, err := nilStatus.Value()
		if err == nil {
			t.Errorf("nilStatus.Value() should return error")
		}
	})
}

func TestTaskStatusScan(t *testing.T) {
	tests := []struct {
		input    any
		expected TaskStatus
		hasError bool
	}{
		{"pending", TaskStatusPending, false},
		{"in_progress", TaskStatusInProgress, false},
		{[]byte("completed"), TaskStatusCompleted, false},
		{"invalid", "", true},
		{[]byte("invalid"), "", true},
		{nil, "", true},
		{123, "", true}, // Invalid type
	}

	for _, tt := range tests {
		t.Run(testFmtInput(tt.input), func(t *testing.T) {
			var status TaskStatus
			err := (&status).Scan(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Scan(%v) expected error, got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Scan(%v) unexpected error: %v", tt.input, err)
				}
				if status != tt.expected {
					t.Errorf("Scan(%v) = %s, expected %s", tt.input, status, tt.expected)
				}
			}
		})
	}
}

func testFmtInput(input any) string {
	if input == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", input)
}
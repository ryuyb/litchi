package valueobject

// FailedTask represents detailed information about a failed task.
type FailedTask struct {
	TaskID     string `json:"taskId"`     // ID of the failed task
	Reason     string `json:"reason"`     // Failure reason
	Suggestion string `json:"suggestion"` // Suggested solution
}

// NewFailedTask creates a new FailedTask value object.
func NewFailedTask(taskID, reason, suggestion string) FailedTask {
	return FailedTask{
		TaskID:     taskID,
		Reason:     reason,
		Suggestion: suggestion,
	}
}

// ExecutionResult represents the result of a task execution.
type ExecutionResult struct {
	Output      string       `json:"output"`      // Execution output/logs
	TestResults []TestResult `json:"testResults"` // Test results (if applicable)
	Duration    int          `json:"duration"`    // Execution duration in milliseconds
	Success     bool         `json:"success"`     // Whether execution succeeded
}

// TestResult represents a single test result.
type TestResult struct {
	Name    string `json:"name"`    // Test name
	Status  string `json:"status"`  // passed, failed, skipped
	Message string `json:"message"` // Error message (if failed)
}

// NewExecutionResult creates a new execution result.
func NewExecutionResult(output string, success bool, duration int) ExecutionResult {
	return ExecutionResult{
		Output:      output,
		Success:     success,
		Duration:    duration,
		TestResults: []TestResult{},
	}
}

// AddTestResult adds a test result to the execution result.
func (er *ExecutionResult) AddTestResult(name, status, message string) {
	er.TestResults = append(er.TestResults, TestResult{
		Name:    name,
		Status:  status,
		Message: message,
	})
}

// HasTestFailures returns true if any tests failed.
func (er ExecutionResult) HasTestFailures() bool {
	for _, tr := range er.TestResults {
		if tr.Status == "failed" {
			return true
		}
	}
	return false
}
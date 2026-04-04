package aggregate

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

func TestNewWorkSession(t *testing.T) {
	issue := entity.NewIssue(123, "Test Issue", "Test body", "owner/repo", "testuser")

	session, err := NewWorkSession(issue)
	if err != nil {
		t.Errorf("NewWorkSession unexpected error: %v", err)
	}

	if session.ID == uuid.Nil {
		t.Errorf("WorkSession ID should not be nil")
	}
	if session.Issue.Number != 123 {
		t.Errorf("WorkSession Issue.Number mismatch")
	}
	if session.CurrentStage != valueobject.StageClarification {
		t.Errorf("NewWorkSession CurrentStage should be Clarification")
	}
	if session.SessionStatus != SessionStatusActive {
		t.Errorf("NewWorkSession SessionStatus should be Active")
	}
	if session.Clarification == nil {
		t.Errorf("NewWorkSession should initialize Clarification")
	}
	if len(session.Tasks) != 0 {
		t.Errorf("NewWorkSession Tasks should be empty")
	}
	if session.Design != nil {
		t.Errorf("NewWorkSession Design should be nil")
	}
	if session.Execution != nil {
		t.Errorf("NewWorkSession Execution should be nil")
	}
}

func TestNewWorkSessionInvalidIssue(t *testing.T) {
	// Invalid issue (no title)
	issue := entity.NewIssue(123, "", "Test body", "owner/repo", "testuser")

	session, err := NewWorkSession(issue)
	if err == nil {
		t.Errorf("NewWorkSession with invalid issue should return error")
	}
	if session != nil {
		t.Errorf("NewWorkSession with invalid issue should return nil session")
	}
}

func TestWorkSessionValidate(t *testing.T) {
	tests := []struct {
		name    string
		session *WorkSession
		hasError bool
	}{
		{
			name: "valid_new_session",
			session: func() *WorkSession {
				s, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))
				return s
			}(),
			hasError: false,
		},
		{
			name: "nil_issue",
			session: &WorkSession{
				ID: uuid.New(),
				CurrentStage: valueobject.StageClarification,
				SessionStatus: SessionStatusActive,
			},
			hasError: true,
		},
		{
			name: "design_stage_without_design",
			session: func() *WorkSession {
				s, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))
				s.CurrentStage = valueobject.StageDesign
				return s
			}(),
			hasError: true,
		},
		{
			name: "execution_stage_without_tasks",
			session: func() *WorkSession {
				s, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))
				s.CurrentStage = valueobject.StageExecution
				s.Design = entity.NewDesign("Design content")
				return s
			}(),
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if tt.hasError {
				if err == nil {
					t.Errorf("Validate(%s) expected error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Validate(%s) unexpected error: %v", tt.name, err)
				}
			}
		})
	}
}

func TestSessionStatus(t *testing.T) {
	tests := []struct {
		status   SessionStatus
		isValid  bool
		isTerminal bool
		canPause bool
		canResume bool
		canTerminate bool
	}{
		{SessionStatusActive, true, false, true, false, true},
		{SessionStatusPaused, true, false, false, true, true},
		{SessionStatusCompleted, true, true, false, false, false},
		{SessionStatusTerminated, true, true, false, false, false},
		{SessionStatus("invalid"), false, false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if tt.status.IsValid() != tt.isValid {
				t.Errorf("IsValid() = %v, expected %v", tt.status.IsValid(), tt.isValid)
			}
			if tt.status.IsTerminal() != tt.isTerminal {
				t.Errorf("IsTerminal() = %v, expected %v", tt.status.IsTerminal(), tt.isTerminal)
			}
			if tt.status.CanPause() != tt.canPause {
				t.Errorf("CanPause() = %v, expected %v", tt.status.CanPause(), tt.canPause)
			}
			if tt.status.CanResume() != tt.canResume {
				t.Errorf("CanResume() = %v, expected %v", tt.status.CanResume(), tt.canResume)
			}
			if tt.status.CanTerminate() != tt.canTerminate {
				t.Errorf("CanTerminate() = %v, expected %v", tt.status.CanTerminate(), tt.canTerminate)
			}
		})
	}
}

func TestWorkSessionPauseResumeTerminate(t *testing.T) {
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))

	// Pause from active
	err := session.Pause("user_request")
	if err != nil {
		t.Errorf("Pause unexpected error: %v", err)
	}
	if !session.IsPaused() {
		t.Errorf("Session should be paused")
	}

	// Pause again should fail
	err = session.Pause("user_request")
	if err == nil {
		t.Errorf("Pause when already paused should fail")
	}

	// Resume
	err = session.Resume()
	if err != nil {
		t.Errorf("Resume unexpected error: %v", err)
	}
	if !session.IsActive() {
		t.Errorf("Session should be active after resume")
	}

	// Resume when active should fail
	err = session.Resume()
	if err == nil {
		t.Errorf("Resume when active should fail")
	}

	// Terminate
	err = session.Terminate("User requested")
	if err != nil {
		t.Errorf("Terminate unexpected error: %v", err)
	}
	if !session.IsTerminated() {
		t.Errorf("Session should be terminated")
	}

	// Operations on terminated session should fail
	err = session.Pause("user_request")
	if err == nil {
		t.Errorf("Pause on terminated session should fail")
	}
}

func TestWorkSessionTransition(t *testing.T) {
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))

	// Cannot transition to Design without completing clarification
	err := session.TransitionTo(valueobject.StageDesign)
	if err == nil {
		t.Errorf("Transition to Design without completed clarification should fail")
	}

	// Complete clarification
	dims, _ := valueobject.NewClarityDimensions(24, 20, 18, 12, 8) // Total 82
	session.SetClarityDimensions(dims)
	session.ConfirmClarificationPoint("Requirement point")
	session.CompleteClarification()

	// Now can transition to Design
	err = session.TransitionTo(valueobject.StageDesign)
	if err != nil {
		t.Errorf("Transition to Design should succeed: %v", err)
	}
	if session.CurrentStage != valueobject.StageDesign {
		t.Errorf("CurrentStage should be Design")
	}

	// Cannot skip stages (jump to Execution)
	err = session.TransitionTo(valueobject.StageExecution)
	if err == nil {
		t.Errorf("Skipping TaskBreakdown should fail")
	}

	// Set design and confirm
	session.SetDesign(entity.NewDesign("Design content"))
	session.ConfirmDesign()

	// Transition to TaskBreakdown
	err = session.TransitionTo(valueobject.StageTaskBreakdown)
	if err != nil {
		t.Errorf("Transition to TaskBreakdown should succeed: %v", err)
	}

	// Cannot transition to Execution without tasks
	err = session.TransitionTo(valueobject.StageExecution)
	if err == nil {
		t.Errorf("Transition to Execution without tasks should fail")
	}

	// Add tasks
	session.SetTasks([]*entity.Task{
		entity.NewTask("Task 1", nil, 1),
		entity.NewTask("Task 2", nil, 2),
	})

	// Start execution phase
	err = session.TransitionTo(valueobject.StageExecution)
	if err != nil {
		t.Errorf("Transition to Execution should succeed: %v", err)
	}

	// Cannot transition to PullRequest without completed tasks
	err = session.TransitionTo(valueobject.StagePullRequest)
	if err == nil {
		t.Errorf("Transition to PullRequest without completed tasks should fail")
	}
}

func TestWorkSessionRollback(t *testing.T) {
	// Create session in Execution stage
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))
	dims, _ := valueobject.NewClarityDimensions(24, 20, 18, 12, 8)
	session.SetClarityDimensions(dims)
	session.ConfirmClarificationPoint("Requirement point")
	session.CompleteClarification()
	session.TransitionTo(valueobject.StageDesign)
	session.SetDesign(entity.NewDesign("Design content"))
	session.ConfirmDesign()
	session.TransitionTo(valueobject.StageTaskBreakdown)
	session.SetTasks([]*entity.Task{entity.NewTask("Task 1", nil, 1)})
	session.TransitionTo(valueobject.StageExecution)
	session.StartExecution("/path/to/worktree", "feature-branch")

	// Rollback to Design
	err := session.RollbackTo(valueobject.StageDesign, "User requested rollback", true)
	if err != nil {
		t.Errorf("Rollback to Design should succeed: %v", err)
	}
	if session.CurrentStage != valueobject.StageDesign {
		t.Errorf("CurrentStage should be Design")
	}
	if len(session.Tasks) != 0 {
		t.Errorf("Tasks should be cleared after rollback to Design")
	}
	if session.Execution != nil && len(session.Execution.RollbackHistory) != 1 {
		t.Errorf("Rollback should be recorded")
	}

	// Cannot rollback from Clarification
	session.CurrentStage = valueobject.StageClarification
	err = session.RollbackTo(valueobject.StageDesign, "Invalid rollback", true)
	if err == nil {
		t.Errorf("Rollback from Clarification should fail")
	}
}

func TestWorkSessionPRRollback(t *testing.T) {
	// Create session in PullRequest stage with completed tasks
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))
	dims, _ := valueobject.NewClarityDimensions(24, 20, 18, 12, 8)
	session.SetClarityDimensions(dims)
	session.ConfirmClarificationPoint("Requirement point")
	session.CompleteClarification()
	session.TransitionTo(valueobject.StageDesign)
	session.SetDesign(entity.NewDesign("Design content"))
	session.ConfirmDesign()
	session.TransitionTo(valueobject.StageTaskBreakdown)

	task := entity.NewTask("Task 1", nil, 1)
	session.SetTasks([]*entity.Task{task})
	session.TransitionTo(valueobject.StageExecution)
	session.StartExecution("/path/to/worktree", "feature-branch")
	task.Start()
	task.Complete(valueobject.NewExecutionResult("Success", true, 100))
	session.CompleteTask(task.ID, valueobject.NewExecutionResult("Success", true, 100))

	session.TransitionTo(valueobject.StagePullRequest)
	session.SetPRNumber(456)

	// Save the execution reference before rollback for checking deprecated branches
	execBeforeRollback := session.Execution

	// Shallow rollback to Execution (R4)
	err := session.RollbackTo(valueobject.StageExecution, "Fix code issues", true)
	if err != nil {
		t.Errorf("R4 rollback should succeed: %v", err)
	}
	if session.CurrentStage != valueobject.StageExecution {
		t.Errorf("CurrentStage should be Execution after R4")
	}
	if session.PRNumber == nil || *session.PRNumber != 456 {
		t.Errorf("PRNumber should still be set after R4")
	}
	if session.Execution == nil {
		t.Errorf("Execution should exist after R4")
	}

	// Deep rollback to Design (R5)
	session.TransitionTo(valueobject.StageExecution)
	session.TransitionTo(valueobject.StagePullRequest)
	err = session.RollbackTo(valueobject.StageDesign, "Redesign needed", true)
	if err != nil {
		t.Errorf("R5 rollback should succeed: %v", err)
	}
	// After R5, Execution is cleared, check DeprecatedBranches in the saved reference
	if execBeforeRollback == nil || !execBeforeRollback.Branch.IsDeprecated {
		t.Errorf("Branch should be deprecated after R5")
	}
	if len(session.Tasks) != 0 {
		t.Errorf("Tasks should be cleared after R5")
	}
	// Check DeprecatedBranches list
	if session.Execution == nil && len(execBeforeRollback.DeprecatedBranches) == 0 {
		t.Errorf("DeprecatedBranches should have record after R5")
	}
}

func TestWorkSessionTaskManagement(t *testing.T) {
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))
	dims, _ := valueobject.NewClarityDimensions(24, 20, 18, 12, 8)
	session.SetClarityDimensions(dims)
	session.ConfirmClarificationPoint("Requirement point")
	session.CompleteClarification()
	session.TransitionTo(valueobject.StageDesign)
	session.SetDesign(entity.NewDesign("Design content"))
	session.ConfirmDesign()
	session.TransitionTo(valueobject.StageTaskBreakdown)

	task1 := entity.NewTask("Task 1", nil, 1)
	task2 := entity.NewTask("Task 2", []uuid.UUID{task1.ID}, 2) // Task2 depends on Task1
	session.SetTasks([]*entity.Task{task1, task2})

	session.TransitionTo(valueobject.StageExecution)
	session.StartExecution("/path/to/worktree", "feature-branch")

	// Get next executable task (should be Task1 - no dependencies)
	nextTask := session.GetNextExecutableTask(3)
	if nextTask == nil || nextTask.ID != task1.ID {
		t.Errorf("Next task should be Task1 (no dependencies)")
	}

	// Task2 should not be executable yet (depends on Task1)
	session.StartTask(task2.ID)
	if session.GetTask(task2.ID).Status == valueobject.TaskStatusInProgress {
		t.Errorf("Task2 should not start before Task1 completes")
	}

	// Start and complete Task1
	err := session.StartTask(task1.ID)
	if err != nil {
		t.Errorf("StartTask failed: %v", err)
	}
	err = session.CompleteTask(task1.ID, valueobject.NewExecutionResult("Success", true, 100))
	if err != nil {
		t.Errorf("CompleteTask failed: %v", err)
	}

	// Now Task2 should be executable
	nextTask = session.GetNextExecutableTask(3)
	if nextTask == nil || nextTask.ID != task2.ID {
		t.Errorf("Next task should be Task2 after Task1 completed")
	}

	// Check all tasks completed
	if session.AreAllTasksCompleted() {
		t.Errorf("Not all tasks completed yet")
	}

	// Complete Task2
	session.StartTask(task2.ID)
	session.CompleteTask(task2.ID, valueobject.NewExecutionResult("Success", true, 100))

	if !session.AreAllTasksCompleted() {
		t.Errorf("All tasks should be completed")
	}
}

func TestWorkSessionTaskFailAndRetry(t *testing.T) {
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))
	setupSessionForExecution(session)

	task := entity.NewTask("Test task", nil, 1)
	session.SetTasks([]*entity.Task{task})
	session.StartExecution("/path/to/worktree", "feature-branch")

	// Start and fail task
	session.StartTask(task.ID)
	err := session.FailTask(task.ID, "Test failure", "Try again")
	if err != nil {
		t.Errorf("FailTask failed: %v", err)
	}

	if !session.HasFailedTask() {
		t.Errorf("Session should have failed task")
	}

	failedTask := session.GetFailedTask()
	if failedTask == nil {
		t.Errorf("GetFailedTask should return failed task info")
	}
	if failedTask.Reason != "Test failure" {
		t.Errorf("FailedTask reason mismatch")
	}

	// Retry task
	err = session.RetryTask(task.ID, 3)
	if err != nil {
		t.Errorf("RetryTask failed: %v", err)
	}
	if session.GetTask(task.ID).Status != valueobject.TaskStatusInProgress {
		t.Errorf("Task should be InProgress after retry")
	}

	// Fail again and exceed retry limit
	session.FailTask(task.ID, "Second failure", "Try again") // RetryCount = 2
	session.RetryTask(task.ID, 3)
	session.FailTask(task.ID, "Third failure", "Try again")  // RetryCount = 3
	session.RetryTask(task.ID, 3)
	session.FailTask(task.ID, "Fourth failure", "Try again") // RetryCount = 4

	err = session.RetryTask(task.ID, 3)
	if err == nil {
		t.Errorf("RetryTask should fail when exceeding limit")
	}
}

func TestWorkSessionTaskSkip(t *testing.T) {
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))
	setupSessionForExecution(session)

	task := entity.NewTask("Test task", nil, 1)
	session.SetTasks([]*entity.Task{task})

	// Skip task
	err := session.SkipTask(task.ID, "User requested skip")
	if err != nil {
		t.Errorf("SkipTask failed: %v", err)
	}

	if !session.GetTask(task.ID).IsSkipped() {
		t.Errorf("Task should be skipped")
	}

	// Skipped tasks count as "completed" for stage progression
	if !session.AreAllTasksCompleted() {
		t.Errorf("All tasks should be completed (including skipped)")
	}
}

func TestWorkSessionClarification(t *testing.T) {
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))

	// Add question
	session.AddClarificationQuestion("What is the expected behavior?")
	if len(session.Clarification.PendingQuestions) != 1 {
		t.Errorf("Question should be added")
	}

	// Answer question
	err := session.AnswerClarificationQuestion("What is the expected behavior?", "It should work correctly", "test-user")
	if err != nil {
		t.Errorf("AnswerClarificationQuestion failed: %v", err)
	}
	if session.Clarification.HasPendingQuestions() {
		t.Errorf("No pending questions should remain")
	}

	// Confirm point
	session.ConfirmClarificationPoint("User authentication is required")

	// Set clarity dimensions
	dims, _ := valueobject.NewClarityDimensions(24, 20, 18, 12, 8)
	session.SetClarityDimensions(dims)

	// Check can complete
	if !session.CanCompleteClarification(60) {
		t.Errorf("Should be able to complete clarification")
	}

	// Complete clarification
	err = session.CompleteClarification()
	if err != nil {
		t.Errorf("CompleteClarification failed: %v", err)
	}
	if !session.Clarification.IsCompleted() {
		t.Errorf("Clarification should be completed")
	}
}

func TestWorkSessionDesignManagement(t *testing.T) {
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))
	completeClarification(session)

	// Set design
	design := entity.NewDesign("Initial design content")
	session.SetDesign(design)

	if session.Design == nil {
		t.Errorf("Design should be set")
	}

	// Confirm design
	err := session.ConfirmDesign()
	if err != nil {
		t.Errorf("ConfirmDesign failed: %v", err)
	}
	if !session.Design.IsConfirmed() {
		t.Errorf("Design should be confirmed")
	}

	// Reject design
	err = session.RejectDesign("Design needs more work")
	if err != nil {
		t.Errorf("RejectDesign failed: %v", err)
	}
	if session.Design.IsConfirmed() {
		t.Errorf("Design should be rejected")
	}

	// Add new version
	session.AddDesignVersion("Updated design content", "User feedback")
	if session.Design.CurrentVersion != 2 {
		t.Errorf("CurrentVersion should be 2 after adding version")
	}
}

func TestWorkSessionEvents(t *testing.T) {
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))

	// Perform some operations that generate events
	session.AddClarificationQuestion("Question?")
	session.ConfirmClarificationPoint("Point")

	events := session.GetEvents()
	if len(events) == 0 {
		t.Errorf("Events should be recorded")
	}

	// Clear events
	session.ClearEvents()
	if len(session.GetEvents()) != 0 {
		t.Errorf("Events should be cleared")
	}
}

func TestWorkSessionQueryMethods(t *testing.T) {
	session, _ := NewWorkSession(entity.NewIssue(123, "Test", "Body", "owner/repo", "user"))

	// IsActive
	if !session.IsActive() {
		t.Errorf("New session should be active")
	}

	// GetCurrentStage
	if session.GetCurrentStage() != valueobject.StageClarification {
		t.Errorf("GetCurrentStage mismatch")
	}

	// GetIssue
	if session.GetIssue() == nil {
		t.Errorf("GetIssue should return issue")
	}

	// Complete clarification and set design
	completeClarification(session)
	session.TransitionTo(valueobject.StageDesign)
	session.SetDesign(entity.NewDesign("Design"))

	// GetDesign
	if session.GetDesign() == nil {
		t.Errorf("GetDesign should return design")
	}

	// Set tasks and get
	task := entity.NewTask("Task", nil, 1)
	session.SetTasks([]*entity.Task{task})
	session.ConfirmDesign()
	session.TransitionTo(valueobject.StageTaskBreakdown)
	session.TransitionTo(valueobject.StageExecution)

	// GetTasks
	if len(session.GetTasks()) != 1 {
		t.Errorf("GetTasks length mismatch")
	}

	// GetTask
	if session.GetTask(task.ID) == nil {
		t.Errorf("GetTask should return task")
	}

	// GetExecution
	session.StartExecution("/path", "branch")
	if session.GetExecution() == nil {
		t.Errorf("GetExecution should return execution")
	}

	// Set PR number and get
	session.SetPRNumber(123)
	if session.GetPRNumber() == nil || *session.GetPRNumber() != 123 {
		t.Errorf("GetPRNumber mismatch")
	}
}

// Helper functions for test setup

func setupSessionForExecution(session *WorkSession) {
	dims, _ := valueobject.NewClarityDimensions(24, 20, 18, 12, 8)
	session.SetClarityDimensions(dims)
	session.ConfirmClarificationPoint("Requirement point")
	session.CompleteClarification()
	session.TransitionTo(valueobject.StageDesign)
	session.SetDesign(entity.NewDesign("Design content"))
	session.ConfirmDesign()
	session.TransitionTo(valueobject.StageTaskBreakdown)
	session.TransitionTo(valueobject.StageExecution)
}

func completeClarification(session *WorkSession) {
	dims, _ := valueobject.NewClarityDimensions(24, 20, 18, 12, 8)
	session.SetClarityDimensions(dims)
	session.ConfirmClarificationPoint("Requirement point")
	session.CompleteClarification()
}
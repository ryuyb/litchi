package entity

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

func TestNewIssue(t *testing.T) {
	issue := NewIssue(123, "Test Issue", "Test body", "owner/repo", "testuser")

	if issue.Number != 123 {
		t.Errorf("NewIssue Number = %d, expected 123", issue.Number)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("NewIssue Title = %s, expected 'Test Issue'", issue.Title)
	}
	if issue.Repository != "owner/repo" {
		t.Errorf("NewIssue Repository = %s, expected 'owner/repo'", issue.Repository)
	}
	if issue.Author != "testuser" {
		t.Errorf("NewIssue Author = %s, expected 'testuser'", issue.Author)
	}
	if issue.ID == uuid.Nil {
		t.Errorf("NewIssue ID should not be nil")
	}
}

func TestIssueValidate(t *testing.T) {
	tests := []struct {
		name     string
		issue    *Issue
		hasError bool
	}{
		{"valid", NewIssue(123, "Test", "Body", "owner/repo", "user"), false},
		{"invalid_number", NewIssue(0, "Test", "Body", "owner/repo", "user"), true},
		{"empty_title", NewIssue(123, "", "Body", "owner/repo", "user"), true},
		{"empty_repository", NewIssue(123, "Test", "Body", "", "user"), true},
		{"empty_author", NewIssue(123, "Test", "Body", "owner/repo", ""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.issue.Validate()
			if tt.hasError {
				if err == nil {
					t.Errorf("Issue(%s).Validate() expected error", tt.name)
				}
				if !errors.Is(err, errors.ErrValidationFailed) {
					t.Errorf("Issue(%s).Validate() error should be ErrValidationFailed", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Issue(%s).Validate() unexpected error: %v", tt.name, err)
				}
			}
		})
	}
}

func TestIssueLabels(t *testing.T) {
	issue := NewIssue(123, "Test", "Body", "owner/repo", "user")

	// Add labels
	issue.AddLabel("bug")
	issue.AddLabel("enhancement")

	if len(issue.Labels) != 2 {
		t.Errorf("Issue.Labels length = %d, expected 2", len(issue.Labels))
	}

	// Add duplicate label
	issue.AddLabel("bug")
	if len(issue.Labels) != 2 {
		t.Errorf("Adding duplicate label should not increase count")
	}

	// Check HasLabel
	if !issue.HasLabel("bug") {
		t.Errorf("Issue.HasLabel('bug') should be true")
	}
	if issue.HasLabel("feature") {
		t.Errorf("Issue.HasLabel('feature') should be false")
	}

	// Remove label
	issue.RemoveLabel("bug")
	if len(issue.Labels) != 1 {
		t.Errorf("Issue.Labels length after removal = %d, expected 1", len(issue.Labels))
	}
	if issue.HasLabel("bug") {
		t.Errorf("Issue.HasLabel('bug') after removal should be false")
	}
}

func TestNewClarification(t *testing.T) {
	clarification := NewClarification()

	if clarification.Status != ClarificationStatusInProgress {
		t.Errorf("NewClarification Status = %s, expected 'in_progress'", clarification.Status)
	}
	if len(clarification.ConfirmedPoints) != 0 {
		t.Errorf("NewClarification ConfirmedPoints should be empty")
	}
	if len(clarification.PendingQuestions) != 0 {
		t.Errorf("NewClarification PendingQuestions should be empty")
	}
}

func TestClarificationQuestions(t *testing.T) {
	c := NewClarification()

	// Add question
	c.AddQuestion("What is the expected behavior?")
	if len(c.PendingQuestions) != 1 {
		t.Errorf("AddQuestion should add one pending question")
	}
	if !c.HasPendingQuestions() {
		t.Errorf("HasPendingQuestions should be true")
	}
	if len(c.History) != 1 {
		t.Errorf("AddQuestion should add one history turn")
	}
	if !c.History[0].IsAgent() {
		t.Errorf("Question turn should be from agent")
	}

	// Answer question
	err := c.AnswerQuestion("What is the expected behavior?", "It should work correctly")
	if err != nil {
		t.Errorf("AnswerQuestion unexpected error: %v", err)
	}
	if len(c.PendingQuestions) != 0 {
		t.Errorf("AnswerQuestion should remove from pending list")
	}
	if len(c.History) != 2 {
		t.Errorf("AnswerQuestion should add history turn")
	}
	if !c.History[1].IsUser() {
		t.Errorf("Answer turn should be from user")
	}

	// Answer non-existent question
	err = c.AnswerQuestion("Non-existent question", "Answer")
	if err == nil {
		t.Errorf("AnswerQuestion for non-existent question should return error")
	}
}

func TestClarificationConfirmPoint(t *testing.T) {
	c := NewClarification()

	c.ConfirmPoint("User authentication is required")
	c.ConfirmPoint("Data should be persisted in PostgreSQL")

	if len(c.ConfirmedPoints) != 2 {
		t.Errorf("ConfirmPoint should add points")
	}

	// Duplicate confirm
	c.ConfirmPoint("User authentication is required")
	if len(c.ConfirmedPoints) != 2 {
		t.Errorf("Duplicate ConfirmPoint should not add")
	}
}

func TestClarificationComplete(t *testing.T) {
	c := NewClarification()

	// Set up valid clarity dimensions (score >= 60)
	dims, _ := valueobject.NewClarityDimensions(24, 20, 18, 12, 8) // Total 82
	c.SetClarityDimensions(dims)
	c.ConfirmPoint("Requirement point")

	// Cannot complete with pending questions
	c.AddQuestion("Unanswered question")
	if c.CanComplete(60) {
		t.Errorf("CanComplete should be false with pending questions")
	}

	// Answer the question
	c.AnswerQuestion("Unanswered question", "Answer")
	if !c.CanComplete(60) {
		t.Errorf("CanComplete should be true after answering all questions")
	}

	// Complete
	c.Complete()
	if !c.IsCompleted() {
		t.Errorf("IsCompleted should be true after Complete()")
	}
}

func TestNewDesign(t *testing.T) {
	content := "# Design Document\n\n## Overview\nThis is the design."
	design := NewDesign(content)

	if design.CurrentVersion != 1 {
		t.Errorf("NewDesign CurrentVersion = %d, expected 1", design.CurrentVersion)
	}
	if len(design.Versions) != 1 {
		t.Errorf("NewDesign should have 1 version")
	}
	if design.Versions[0].Reason != "initial" {
		t.Errorf("First version reason should be 'initial'")
	}
	if design.Confirmed {
		t.Errorf("NewDesign should not be confirmed by default")
	}
}

func TestDesignVersionManagement(t *testing.T) {
	design := NewDesign("Initial content")

	// Add new version
	design.AddVersion("Updated content", "rollback to design")
	if design.CurrentVersion != 2 {
		t.Errorf("AddVersion should increment CurrentVersion")
	}
	if len(design.Versions) != 2 {
		t.Errorf("AddVersion should add new version")
	}
	if design.Confirmed {
		t.Errorf("AddVersion should reset confirmation")
	}

	// Get current content
	if design.GetCurrentContent() != "Updated content" {
		t.Errorf("GetCurrentContent should return current version content")
	}

	// Get specific version
	v1, err := design.GetVersion(1)
	if err != nil {
		t.Errorf("GetVersion(1) unexpected error: %v", err)
	}
	if v1.Content != "Initial content" {
		t.Errorf("GetVersion(1) content mismatch")
	}

	// Get non-existent version
	_, err = design.GetVersion(999)
	if err == nil {
		t.Errorf("GetVersion(999) should return error")
	}
}

func TestDesignComplexity(t *testing.T) {
	design := NewDesign("Initial content")

	// High complexity score
	highScore, _ := valueobject.NewComplexityScore(80)
	design.SetComplexityScore(highScore, 70)

	if !design.RequireConfirmation {
		t.Errorf("SetComplexityScore with high score should require confirmation")
	}
	if !design.NeedsConfirmation() {
		t.Errorf("NeedsConfirmation should be true")
	}

	// Low complexity score
	lowScore, _ := valueobject.NewComplexityScore(50)
	design.SetComplexityScore(lowScore, 70)

	if design.RequireConfirmation {
		t.Errorf("SetComplexityScore with low score should not require confirmation")
	}
}

func TestDesignConfirmation(t *testing.T) {
	design := NewDesign("Content")

	// Require confirmation
	design.RequireConfirmation = true

	if design.CanProceedToTaskBreakdown() {
		t.Errorf("CanProceedToTaskBreakdown should be false when not confirmed")
	}

	// Confirm
	design.Confirm()
	if !design.IsConfirmed() {
		t.Errorf("IsConfirmed should be true")
	}
	if !design.CanProceedToTaskBreakdown() {
		t.Errorf("CanProceedToTaskBreakdown should be true after confirmation")
	}

	// Reject
	design.Reject()
	if design.IsConfirmed() {
		t.Errorf("IsConfirmed should be false after rejection")
	}
	if design.CanProceedToTaskBreakdown() {
		t.Errorf("CanProceedToTaskBreakdown should be false after rejection")
	}
}

func TestNewTask(t *testing.T) {
	deps := []uuid.UUID{uuid.New()}
	task := NewTask("Implement user login", deps, 1)

	if task.Description != "Implement user login" {
		t.Errorf("NewTask Description mismatch")
	}
	if task.Status != valueobject.TaskStatusPending {
		t.Errorf("NewTask Status should be Pending")
	}
	if task.Order != 1 {
		t.Errorf("NewTask Order = %d, expected 1", task.Order)
	}
	if !task.HasDependencies() {
		t.Errorf("NewTask should have dependencies")
	}
	if task.ID == uuid.Nil {
		t.Errorf("NewTask ID should not be nil")
	}
}

func TestTaskStatusTransitions(t *testing.T) {
	task := NewTask("Test task", nil, 1)

	// Start
	err := task.Start()
	if err != nil {
		t.Errorf("Start unexpected error: %v", err)
	}
	if !task.IsInProgress() {
		t.Errorf("Task should be InProgress after Start()")
	}

	// Start again should fail
	err = task.Start()
	if err == nil {
		t.Errorf("Start when already InProgress should fail")
	}

	// Complete
	result := valueobject.NewExecutionResult("Success output", true, 1000)
	err = task.Complete(result)
	if err != nil {
		t.Errorf("Complete unexpected error: %v", err)
	}
	if !task.IsCompleted() {
		t.Errorf("Task should be Completed after Complete()")
	}
	if task.ExecutionResult.Output != "Success output" {
		t.Errorf("ExecutionResult should be set")
	}
}

func TestTaskFailAndRetry(t *testing.T) {
	task := NewTask("Test task", nil, 1)

	// Must start before failing
	task.Start()

	// Fail
	err := task.Fail("Test failure", "Try again")
	if err != nil {
		t.Errorf("Fail unexpected error: %v", err)
	}
	if !task.IsFailed() {
		t.Errorf("Task should be Failed after Fail()")
	}
	if task.FailureReason != "Test failure" {
		t.Errorf("FailureReason should be set")
	}
	if task.RetryCount != 1 {
		t.Errorf("RetryCount should be 1 after first failure")
	}

	// Retry
	err = task.Retry(3)
	if err != nil {
		t.Errorf("Retry unexpected error: %v", err)
	}
	if !task.IsInProgress() {
		t.Errorf("Task should be InProgress after Retry()")
	}
	if task.FailureReason != "" {
		t.Errorf("FailureReason should be cleared after retry")
	}

	// Max retry limit - need to retry first, then fail again
	task.Fail("Second failure", "Try again") // RetryCount = 2
	task.Retry(3)                            // Back to InProgress
	task.Fail("Third failure", "Try again")  // RetryCount = 3
	task.Retry(3)                            // Back to InProgress (last retry)

	// Now fail again - RetryCount will be 4
	task.Fail("Fourth failure", "Try again") // RetryCount = 4

	err = task.Retry(3)
	if err == nil {
		t.Errorf("Retry when at max limit should fail")
	}
	if task.CanRetry(3) {
		t.Errorf("CanRetry should be false when retry count exceeds limit")
	}
}

func TestTaskSkip(t *testing.T) {
	task := NewTask("Test task", nil, 1)

	// Skip from pending
	err := task.Skip("User requested skip")
	if err != nil {
		t.Errorf("Skip from Pending unexpected error: %v", err)
	}
	if !task.IsSkipped() {
		t.Errorf("Task should be Skipped")
	}

	// Cannot skip completed task
	task2 := NewTask("Test task 2", nil, 2)
	task2.Start()
	task2.Complete(valueobject.NewExecutionResult("Success", true, 100))

	err = task2.Skip("Trying to skip completed")
	if err == nil {
		t.Errorf("Skip from Completed should fail")
	}
}

func TestNewExecution(t *testing.T) {
	execution := NewExecution("/path/to/worktree", "feature-branch")

	if execution.WorktreePath != "/path/to/worktree" {
		t.Errorf("NewExecution WorktreePath mismatch")
	}
	if execution.Branch.Name != "feature-branch" {
		t.Errorf("NewExecution Branch.Name mismatch")
	}
	if execution.Branch.IsDeprecated {
		t.Errorf("NewExecution Branch should not be deprecated")
	}
	if len(execution.CompletedTasks) != 0 {
		t.Errorf("NewExecution CompletedTasks should be empty")
	}
}

func TestExecutionTaskTracking(t *testing.T) {
	e := NewExecution("/path", "branch")
	taskID := uuid.New()

	// Set current task
	e.SetCurrentTask(taskID)
	if e.CurrentTaskID == nil || *e.CurrentTaskID != taskID {
		t.Errorf("SetCurrentTask should set CurrentTaskID")
	}

	// Mark completed
	e.MarkTaskCompleted(taskID)
	if e.CurrentTaskID != nil {
		t.Errorf("MarkTaskCompleted should clear CurrentTaskID")
	}
	if !e.HasCompletedTask(taskID) {
		t.Errorf("HasCompletedTask should be true")
	}

	// Set failed task
	taskID2 := uuid.New()
	e.SetFailedTask(taskID2, "Failure reason", "Suggestion")
	if e.FailedTask == nil {
		t.Errorf("SetFailedTask should set FailedTask")
	}
	if e.FailedTask.TaskID != taskID2.String() {
		t.Errorf("FailedTask.TaskID mismatch")
	}

	// Clear failed task
	e.ClearFailedTask()
	if e.FailedTask != nil {
		t.Errorf("ClearFailedTask should clear FailedTask")
	}
}

func TestExecutionFixTasks(t *testing.T) {
	e := NewExecution("/path", "branch")
	fixTaskID := uuid.New()

	e.AddFixTask(fixTaskID)
	if len(e.FixTasks) != 1 {
		t.Errorf("AddFixTask should add fix task")
	}

	e.ClearFixTasks()
	if len(e.FixTasks) != 0 {
		t.Errorf("ClearFixTasks should clear all fix tasks")
	}
}

func TestExecutionRollback(t *testing.T) {
	e := NewExecution("/path", "branch")

	e.RecordRollback(valueobject.StageExecution, valueobject.StageDesign, "User requested rollback", true)

	if len(e.RollbackHistory) != 1 {
		t.Errorf("RecordRollback should add rollback record")
	}
	if e.RollbackHistory[0].FromStage != valueobject.StageExecution {
		t.Errorf("Rollback FromStage mismatch")
	}
	if e.RollbackHistory[0].ToStage != valueobject.StageDesign {
		t.Errorf("Rollback ToStage mismatch")
	}
	if !e.RollbackHistory[0].UserInitiated {
		t.Errorf("Rollback should be user initiated")
	}
}

func TestExecutionBranchManagement(t *testing.T) {
	e := NewExecution("/path", "feature-branch")

	// Deprecate branch
	prNumber := 123
	e.DeprecateBranch("Rollback to design", &prNumber, "design")

	if !e.Branch.IsDeprecated {
		t.Errorf("Branch should be deprecated")
	}
	if len(e.DeprecatedBranches) != 1 {
		t.Errorf("DeprecatedBranches should have record")
	}

	// Set new branch
	e.SetNewBranch("new-feature-branch")
	if e.Branch.Name != "new-feature-branch" {
		t.Errorf("SetNewBranch should set new branch name")
	}
	if e.Branch.IsDeprecated {
		t.Errorf("New branch should not be deprecated")
	}
}

func TestNewAuditLog(t *testing.T) {
	sessionID := uuid.New()
	log := NewAuditLog(sessionID, "owner/repo", 123, "testuser", valueobject.ActorRoleAdmin, valueobject.OpStageTransition, "session", sessionID.String())

	if log.SessionID != sessionID {
		t.Errorf("AuditLog SessionID mismatch")
	}
	if log.Repository != "owner/repo" {
		t.Errorf("AuditLog Repository mismatch")
	}
	if log.IssueNumber != 123 {
		t.Errorf("AuditLog IssueNumber mismatch")
	}
	if log.Actor != "testuser" {
		t.Errorf("AuditLog Actor mismatch")
	}
	if log.ActorRole != valueobject.ActorRoleAdmin {
		t.Errorf("AuditLog ActorRole mismatch")
	}
	if log.Operation != valueobject.OpStageTransition {
		t.Errorf("AuditLog Operation mismatch")
	}
	if log.Result != valueobject.AuditResultSuccess {
		t.Errorf("NewAuditLog default result should be success")
	}
	if log.ID == uuid.Nil {
		t.Errorf("AuditLog ID should not be nil")
	}
}

func TestAuditLogResults(t *testing.T) {
	sessionID := uuid.New()
	log := NewAuditLog(sessionID, "owner/repo", 123, "user", valueobject.ActorRoleAdmin, valueobject.OpFileRead, "file", "test.go")

	// Mark success
	log.MarkSuccess()
	if !log.IsSuccess() {
		t.Errorf("IsSuccess should be true")
	}

	// Mark failed
	log.MarkFailed("File not found")
	if !log.IsFailed() {
		t.Errorf("IsFailed should be true")
	}
	if log.Error != "File not found" {
		t.Errorf("Error should be set")
	}

	// Mark denied
	log2 := NewAuditLog(sessionID, "owner/repo", 123, "user", valueobject.ActorRoleIssueAuthor, valueobject.OpApprovalDecision, "pr", "123")
	log2.MarkDenied("Not authorized")
	if !log2.IsDenied() {
		t.Errorf("IsDenied should be true")
	}
}

func TestAuditLogOutputTruncation(t *testing.T) {
	sessionID := uuid.New()
	log := NewAuditLog(sessionID, "owner/repo", 123, "user", valueobject.ActorRoleAdmin, valueobject.OpAgentCall, "agent", "claude")

	longOutput := "This is a very long output that should be truncated because it exceeds the maximum length setting"
	maxLength := 50

	log.SetOutput(longOutput, maxLength)

	if len(log.Output) > maxLength+3 { // +3 for "..."
		t.Errorf("SetOutput should truncate to maxLength + '...'")
	}
	if log.Output[len(log.Output)-3:] != "..." {
		t.Errorf("Truncated output should end with '...'")
	}

	// Short output should not be truncated
	shortOutput := "Short output"
	log.SetOutput(shortOutput, maxLength)
	if log.Output != shortOutput {
		t.Errorf("Short output should not be truncated")
	}
}

func TestNewRepository(t *testing.T) {
	repo := NewRepository("owner/repo")

	if repo.Name != "owner/repo" {
		t.Errorf("NewRepository Name mismatch")
	}
	if !repo.Enabled {
		t.Errorf("NewRepository should be enabled by default")
	}
	if repo.ID == uuid.Nil {
		t.Errorf("NewRepository ID should not be nil")
	}
}

func TestRepositoryValidate(t *testing.T) {
	tests := []struct {
		name     string
		repo     *Repository
		hasError bool
	}{
		{"valid", NewRepository("owner/repo"), false},
		{"empty_name", NewRepository(""), true},
		{"no_slash", NewRepository("ownerrepo"), true},
		{"too_short", NewRepository("o/r"), false}, // Minimum valid: "o/r" has slash and length 3
		{"only_slash", NewRepository("/"), true},   // Only slash is not valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.repo.Validate()
			if tt.hasError {
				if err == nil {
					t.Errorf("Repository(%s).Validate() expected error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Repository(%s).Validate() unexpected error: %v", tt.name, err)
				}
			}
		})
	}
}

func TestRepositoryEnableDisable(t *testing.T) {
	repo := NewRepository("owner/repo")

	repo.Disable()
	if repo.IsEnabled() {
		t.Errorf("Disable should set Enabled to false")
	}

	repo.Enable()
	if !repo.IsEnabled() {
		t.Errorf("Enable should set Enabled to true")
	}
}

func TestRepositoryConfig(t *testing.T) {
	repo := NewRepository("owner/repo")

	// Set config values
	repo.SetMaxConcurrency(5)
	repo.SetComplexityThreshold(80)
	repo.SetForceDesignConfirm(true)
	repo.SetDefaultModel("claude-opus")
	repo.SetTaskRetryLimit(5)

	if *repo.Config.MaxConcurrency != 5 {
		t.Errorf("MaxConcurrency mismatch")
	}
	if *repo.Config.ComplexityThreshold != 80 {
		t.Errorf("ComplexityThreshold mismatch")
	}
	if !*repo.Config.ForceDesignConfirm {
		t.Errorf("ForceDesignConfirm should be true")
	}
	if *repo.Config.DefaultModel != "claude-opus" {
		t.Errorf("DefaultModel mismatch")
	}
	if *repo.Config.TaskRetryLimit != 5 {
		t.Errorf("TaskRetryLimit mismatch")
	}
}

func TestRepositoryGetEffectiveConfig(t *testing.T) {
	repo := NewRepository("owner/repo")

	// Repository overrides
	repo.SetMaxConcurrency(5)
	repo.SetComplexityThreshold(80)

	// Global config
	globalConfig := RepoConfig{
		MaxConcurrency:      ptrInt(3),
		ComplexityThreshold: ptrInt(70),
		ForceDesignConfirm:  ptrBool(false),
		TaskRetryLimit:      ptrInt(3),
	}

	effective := repo.GetEffectiveConfig(globalConfig)

	// Repository values should override
	if *effective.MaxConcurrency != 5 {
		t.Errorf("Effective MaxConcurrency should use repo override")
	}
	if *effective.ComplexityThreshold != 80 {
		t.Errorf("Effective ComplexityThreshold should use repo override")
	}

	// Global values should be used when repo has no override
	if *effective.ForceDesignConfirm {
		t.Errorf("Effective ForceDesignConfirm should use global value")
	}
	if *effective.TaskRetryLimit != 3 {
		t.Errorf("Effective TaskRetryLimit should use global value")
	}
}

func ptrInt(v int) *int {
	return &v
}

func ptrBool(v bool) *bool {
	return &v
}

func TestConversationTurn(t *testing.T) {
	agentTurn := valueobject.NewConversationTurn("agent", "What is the requirement?")
	if !agentTurn.IsAgent() {
		t.Errorf("Agent turn IsAgent should be true")
	}
	if agentTurn.IsUser() {
		t.Errorf("Agent turn IsUser should be false")
	}

	userTurn := valueobject.NewConversationTurn("user", "The requirement is...")
	if !userTurn.IsUser() {
		t.Errorf("User turn IsUser should be true")
	}
	if userTurn.IsAgent() {
		t.Errorf("User turn IsAgent should be false")
	}
}

func TestDesignVersion(t *testing.T) {
	v1 := valueobject.NewDesignVersion(1, "Initial design", "initial")
	if !v1.IsInitial() {
		t.Errorf("Version 1 IsInitial should be true")
	}

	v2 := valueobject.NewDesignVersion(2, "Updated design", "rollback")
	if v2.IsInitial() {
		t.Errorf("Version 2 IsInitial should be false")
	}
}

func TestBranch(t *testing.T) {
	branch := valueobject.NewBranch("feature-branch")
	if !branch.IsActive() {
		t.Errorf("New branch should be active")
	}

	branch.Deprecate("Rollback to design")
	if branch.IsActive() {
		t.Errorf("Deprecated branch should not be active")
	}
	if !branch.IsDeprecated {
		t.Errorf("IsDeprecated should be true")
	}
	if branch.DeprecatedAt == nil {
		t.Errorf("DeprecatedAt should be set")
	}
}

func TestExecutionResult(t *testing.T) {
	result := valueobject.NewExecutionResult("Test output", true, 1000)

	if result.Output != "Test output" {
		t.Errorf("ExecutionResult Output mismatch")
	}
	if !result.Success {
		t.Errorf("ExecutionResult Success should be true")
	}
	if result.Duration != 1000 {
		t.Errorf("ExecutionResult Duration mismatch")
	}

	// Add test results
	result.AddTestResult("Test1", "passed", "")
	result.AddTestResult("Test2", "failed", "Assertion failed")

	if len(result.TestResults) != 2 {
		t.Errorf("AddTestResult should add test results")
	}
	if !result.HasTestFailures() {
		t.Errorf("HasTestFailures should be true")
	}

	// No failures
	result2 := valueobject.NewExecutionResult("Success", true, 500)
	result2.AddTestResult("Test1", "passed", "")
	if result2.HasTestFailures() {
		t.Errorf("HasTestFailures should be false with all passed tests")
	}
}

func TestActorRole(t *testing.T) {
	admin := valueobject.ActorRoleAdmin
	if !admin.IsValid() {
		t.Errorf("Admin role should be valid")
	}
	if !admin.CanAnswerClarification() {
		t.Errorf("Admin should be able to answer clarification")
	}
	if !admin.CanApprove() {
		t.Errorf("Admin should be able to approve")
	}

	author := valueobject.ActorRoleIssueAuthor
	if !author.IsValid() {
		t.Errorf("IssueAuthor role should be valid")
	}
	if !author.CanAnswerClarification() {
		t.Errorf("IssueAuthor should be able to answer clarification")
	}
	if author.CanApprove() {
		t.Errorf("IssueAuthor should not be able to approve")
	}

	invalid := valueobject.ActorRole("invalid")
	if invalid.IsValid() {
		t.Errorf("Invalid role should not be valid")
	}
}

func TestOperationType(t *testing.T) {
	validOps := []valueobject.OperationType{
		valueobject.OpSessionStart,
		valueobject.OpStageTransition,
		valueobject.OpAgentCall,
		valueobject.OpFileRead,
		valueobject.OpPRCreate,
	}

	for _, op := range validOps {
		if !op.IsValid() {
			t.Errorf("Operation %s should be valid", op)
		}
	}

	invalid := valueobject.OperationType("invalid_op")
	if invalid.IsValid() {
		t.Errorf("Invalid operation should not be valid")
	}
}

func TestAuditResult(t *testing.T) {
	success := valueobject.AuditResultSuccess
	if !success.IsValid() {
		t.Errorf("Success result should be valid")
	}

	failed := valueobject.AuditResultFailed
	if !failed.IsValid() {
		t.Errorf("Failed result should be valid")
	}

	denied := valueobject.AuditResultDenied
	if !denied.IsValid() {
		t.Errorf("Denied result should be valid")
	}

	invalid := valueobject.AuditResult("invalid")
	if invalid.IsValid() {
		t.Errorf("Invalid result should not be valid")
	}
}

func TestRollbackRecord(t *testing.T) {
	record := valueobject.NewRollbackRecord(
		valueobject.StageExecution,
		valueobject.StageDesign,
		"User requested rollback",
		true,
	)

	if record.FromStage != valueobject.StageExecution {
		t.Errorf("RollbackRecord FromStage mismatch")
	}
	if record.ToStage != valueobject.StageDesign {
		t.Errorf("RollbackRecord ToStage mismatch")
	}
	if record.Reason != "User requested rollback" {
		t.Errorf("RollbackRecord Reason mismatch")
	}
	if !record.UserInitiated {
		t.Errorf("RollbackRecord UserInitiated should be true")
	}
	if record.Timestamp.IsZero() {
		t.Errorf("RollbackRecord Timestamp should be set")
	}
}

package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// Issue represents a GitHub issue entity within the WorkSession aggregate.
// It captures the essential information from a GitHub issue for processing.
type Issue struct {
	ID         uuid.UUID `json:"id"`         // Unique identifier (internal)
	Number     int       `json:"number"`     // GitHub issue number
	Title      string    `json:"title"`      // Issue title
	Body       string    `json:"body"`       // Issue body/description
	Repository string    `json:"repository"` // Repository name (owner/repo format)
	Author     string    `json:"author"`     // Issue author (GitHub username)
	Labels     []string  `json:"labels"`     // Issue labels
	URL        string    `json:"url"`        // Full GitHub URL
	CreatedAt  time.Time `json:"createdAt"`  // When the issue was created (GitHub)
	UpdatedAt  time.Time `json:"updatedAt"`  // When the issue was last updated
}

// NewIssue creates a new Issue entity with the given attributes.
func NewIssue(number int, title, body, repository, author string) *Issue {
	return &Issue{
		ID:         uuid.New(),
		Number:     number,
		Title:      title,
		Body:       body,
		Repository: repository,
		Author:     author,
		Labels:     []string{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// NewIssueFromGitHub creates an Issue from GitHub webhook data.
func NewIssueFromGitHub(number int, title, body, repository, author string, labels []string, url string, createdAt time.Time) *Issue {
	return &Issue{
		ID:         uuid.New(),
		Number:     number,
		Title:      title,
		Body:       body,
		Repository: repository,
		Author:     author,
		Labels:     labels,
		URL:        url,
		CreatedAt:  createdAt,
		UpdatedAt:  time.Now(),
	}
}

// Validate validates the Issue entity.
func (i *Issue) Validate() error {
	if i.Number <= 0 {
		return errors.New(errors.ErrValidationFailed).WithDetail("issue number must be positive")
	}
	if i.Title == "" {
		return errors.New(errors.ErrValidationFailed).WithDetail("issue title cannot be empty")
	}
	if i.Repository == "" {
		return errors.New(errors.ErrValidationFailed).WithDetail("issue repository cannot be empty")
	}
	if i.Author == "" {
		return errors.New(errors.ErrValidationFailed).WithDetail("issue author cannot be empty")
	}
	return nil
}

// AddLabel adds a label to the issue.
func (i *Issue) AddLabel(label string) {
	for _, l := range i.Labels {
		if l == label {
			return // Already exists
		}
	}
	i.Labels = append(i.Labels, label)
	i.UpdatedAt = time.Now()
}

// RemoveLabel removes a label from the issue.
func (i *Issue) RemoveLabel(label string) {
	for idx, l := range i.Labels {
		if l == label {
			i.Labels = append(i.Labels[:idx], i.Labels[idx+1:]...)
			i.UpdatedAt = time.Now()
			return
		}
	}
}

// HasLabel checks if the issue has a specific label.
func (i *Issue) HasLabel(label string) bool {
	for _, l := range i.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// Clarification represents the clarification dialogue entity.
// It tracks the process of clarifying requirements through Q&A.
type Clarification struct {
	ConfirmedPoints   []string                       `json:"confirmedPoints"`   // Confirmed requirement points
	PendingQuestions  []string                       `json:"pendingQuestions"`  // Questions awaiting answers
	History           []valueobject.ConversationTurn `json:"history"`           // Q&A dialogue history
	Status            ClarificationStatus            `json:"status"`            // Current clarification status
	ClarityDimensions valueobject.ClarityDimensions  `json:"clarityDimensions"` // Clarity score details
}

// ClarificationStatus represents the status of clarification.
type ClarificationStatus string

const (
	ClarificationStatusInProgress ClarificationStatus = "in_progress"
	ClarificationStatusCompleted  ClarificationStatus = "completed"
)

// NewClarification creates a new Clarification entity.
func NewClarification() *Clarification {
	return &Clarification{
		ConfirmedPoints:  []string{},
		PendingQuestions: []string{},
		History:          []valueobject.ConversationTurn{},
		Status:           ClarificationStatusInProgress,
	}
}

// AddQuestion adds a new question from the agent.
func (c *Clarification) AddQuestion(question string) {
	c.PendingQuestions = append(c.PendingQuestions, question)
	c.History = append(c.History, valueobject.NewConversationTurn("agent", question))
}

// AnswerQuestion records an answer to a pending question.
func (c *Clarification) AnswerQuestion(question, answer string) error {
	// Find and remove the question from pending list
	for idx, q := range c.PendingQuestions {
		if q == question {
			c.PendingQuestions = append(c.PendingQuestions[:idx], c.PendingQuestions[idx+1:]...)
			c.History = append(c.History, valueobject.NewConversationTurn("user", answer))
			return nil
		}
	}
	return errors.New(errors.ErrValidationFailed).WithDetail("question not found in pending list")
}

// ConfirmPoint adds a confirmed requirement point.
func (c *Clarification) ConfirmPoint(point string) {
	for _, p := range c.ConfirmedPoints {
		if p == point {
			return // Already confirmed
		}
	}
	c.ConfirmedPoints = append(c.ConfirmedPoints, point)
}

// HasPendingQuestions returns true if there are unanswered questions.
func (c *Clarification) HasPendingQuestions() bool {
	return len(c.PendingQuestions) > 0
}

// CanComplete checks if clarification can be completed.
// Rules: all questions answered, at least one confirmed point, clarity score >= threshold.
func (c *Clarification) CanComplete(threshold int) bool {
	return !c.HasPendingQuestions() &&
		len(c.ConfirmedPoints) > 0 &&
		c.ClarityDimensions.CanEnterDesign(threshold)
}

// Complete marks the clarification as completed.
func (c *Clarification) Complete() {
	c.Status = ClarificationStatusCompleted
}

// IsCompleted returns true if clarification is completed.
func (c *Clarification) IsCompleted() bool {
	return c.Status == ClarificationStatusCompleted
}

// ClearPendingQuestions clears all pending questions.
// This is used during rollback to reset clarification state.
func (c *Clarification) ClearPendingQuestions() {
	c.PendingQuestions = []string{}
}

// SetClarityDimensions sets the clarity evaluation result.
func (c *Clarification) SetClarityDimensions(dimensions valueobject.ClarityDimensions) {
	c.ClarityDimensions = dimensions
}

// GetClarityScore returns the total clarity score.
func (c *Clarification) GetClarityScore() int {
	return c.ClarityDimensions.TotalScore()
}

// Design represents the design document entity with version management.
type Design struct {
	Versions            []valueobject.DesignVersion `json:"versions"`            // Design version history
	CurrentVersion      int                         `json:"currentVersion"`      // Current version number
	ComplexityScore     valueobject.ComplexityScore `json:"complexityScore"`     // Complexity evaluation
	RequireConfirmation bool                        `json:"requireConfirmation"` // Whether manual confirmation is needed
	Confirmed           bool                        `json:"confirmed"`           // Whether design is confirmed
}

// NewDesign creates a new Design entity with the initial version.
func NewDesign(content string) *Design {
	initialVersion := valueobject.NewDesignVersion(1, content, "initial")
	return &Design{
		Versions:            []valueobject.DesignVersion{initialVersion},
		CurrentVersion:      1,
		RequireConfirmation: false,
		Confirmed:           false,
	}
}

// AddVersion adds a new version of the design (for rollback or update).
func (d *Design) AddVersion(content, reason string) {
	newVersionNum := d.CurrentVersion + 1
	newVersion := valueobject.NewDesignVersion(newVersionNum, content, reason)
	d.Versions = append(d.Versions, newVersion)
	d.CurrentVersion = newVersionNum
	d.Confirmed = false // New version needs confirmation
}

// GetCurrentContent returns the content of the current version.
func (d *Design) GetCurrentContent() string {
	for _, v := range d.Versions {
		if v.Version == d.CurrentVersion {
			return v.Content
		}
	}
	return ""
}

// GetVersion returns a specific version's content.
func (d *Design) GetVersion(versionNum int) (valueobject.DesignVersion, error) {
	for _, v := range d.Versions {
		if v.Version == versionNum {
			return v, nil
		}
	}
	return valueobject.DesignVersion{}, errors.New(errors.ErrValidationFailed).WithDetail("version not found")
}

// SetComplexityScore sets the complexity evaluation result.
func (d *Design) SetComplexityScore(score valueobject.ComplexityScore, threshold int) {
	d.ComplexityScore = score
	d.RequireConfirmation = score.RequiresConfirmation(threshold)
}

// NeedsConfirmation returns true if the design requires manual confirmation.
func (d *Design) NeedsConfirmation() bool {
	return d.RequireConfirmation
}

// Confirm marks the design as confirmed by a user.
func (d *Design) Confirm() {
	d.Confirmed = true
}

// Reject marks the design as rejected, clearing confirmation.
func (d *Design) Reject() {
	d.Confirmed = false
}

// IsConfirmed returns true if the design is confirmed.
func (d *Design) IsConfirmed() bool {
	return d.Confirmed
}

// CanProceedToTaskBreakdown checks if design stage can proceed.
func (d *Design) CanProceedToTaskBreakdown() bool {
	// If confirmation is required, must be confirmed
	if d.RequireConfirmation {
		return d.Confirmed
	}
	// Otherwise can proceed automatically
	return true
}

// Task represents an individual executable task unit.
type Task struct {
	ID              uuid.UUID                   `json:"id"`              // Unique identifier
	Description     string                      `json:"description"`     // Task description
	Status          valueobject.TaskStatus      `json:"status"`          // Current status
	Dependencies    []uuid.UUID                 `json:"dependencies"`    // IDs of tasks this depends on
	ExecutionResult valueobject.ExecutionResult `json:"executionResult"` // Execution result
	RetryCount      int                         `json:"retryCount"`      // Number of retries
	FailureReason   string                      `json:"failureReason"`   // Reason for failure
	Suggestion      string                      `json:"suggestion"`      // Suggested fix
	Order           int                         `json:"order"`           // Execution order (for sorting)
}

// NewTask creates a new Task entity.
func NewTask(description string, dependencies []uuid.UUID, order int) *Task {
	return &Task{
		ID:           uuid.New(),
		Description:  description,
		Status:       valueobject.TaskStatusPending,
		Dependencies: dependencies,
		RetryCount:   0,
		Order:        order,
	}
}

// Start marks the task as in progress.
func (t *Task) Start() error {
	if !t.Status.CanStart() {
		return errors.New(errors.ErrInvalidTaskStatus).WithDetail(
			"cannot start task with status: " + t.Status.String(),
		)
	}
	t.Status = valueobject.TaskStatusInProgress
	return nil
}

// Complete marks the task as completed successfully.
func (t *Task) Complete(result valueobject.ExecutionResult) error {
	if !t.Status.CanComplete() {
		return errors.New(errors.ErrInvalidTaskStatus).WithDetail(
			"cannot complete task with status: " + t.Status.String(),
		)
	}
	t.Status = valueobject.TaskStatusCompleted
	t.ExecutionResult = result
	return nil
}

// Fail marks the task as failed.
func (t *Task) Fail(reason, suggestion string) error {
	if !t.Status.CanFail() {
		return errors.New(errors.ErrInvalidTaskStatus).WithDetail(
			"cannot fail task with status: " + t.Status.String(),
		)
	}
	t.Status = valueobject.TaskStatusFailed
	t.FailureReason = reason
	t.Suggestion = suggestion
	t.RetryCount++
	return nil
}

// Skip marks the task as skipped.
func (t *Task) Skip(reason string) error {
	if !t.Status.CanSkip() {
		return errors.New(errors.ErrInvalidTaskStatus).WithDetail(
			"cannot skip task with status: " + t.Status.String(),
		)
	}
	t.Status = valueobject.TaskStatusSkipped
	t.FailureReason = reason
	return nil
}

// Retry resets the task for retry (Failed -> InProgress).
func (t *Task) Retry(maxRetryLimit int) error {
	if !t.Status.CanRetry() {
		return errors.New(errors.ErrInvalidTaskStatus).WithDetail(
			"cannot retry task with status: " + t.Status.String(),
		)
	}
	if t.RetryCount >= maxRetryLimit {
		return errors.New(errors.ErrValidationFailed).WithDetail(
			"task has reached maximum retry limit",
		)
	}
	t.Status = valueobject.TaskStatusInProgress
	t.FailureReason = ""
	t.Suggestion = ""
	return nil
}

// CanRetry checks if the task can be retried.
func (t *Task) CanRetry(maxRetryLimit int) bool {
	return t.Status.CanRetry() && t.RetryCount < maxRetryLimit
}

// IsPending returns true if task is pending.
func (t *Task) IsPending() bool {
	return t.Status == valueobject.TaskStatusPending
}

// IsInProgress returns true if task is in progress.
func (t *Task) IsInProgress() bool {
	return t.Status == valueobject.TaskStatusInProgress
}

// IsCompleted returns true if task is completed.
func (t *Task) IsCompleted() bool {
	return t.Status == valueobject.TaskStatusCompleted
}

// IsFailed returns true if task is failed.
func (t *Task) IsFailed() bool {
	return t.Status == valueobject.TaskStatusFailed
}

// IsSkipped returns true if task is skipped.
func (t *Task) IsSkipped() bool {
	return t.Status == valueobject.TaskStatusSkipped
}

// HasDependencies returns true if the task has dependencies.
func (t *Task) HasDependencies() bool {
	return len(t.Dependencies) > 0
}

// Execution represents the execution phase state tracking entity.
type Execution struct {
	WorktreePath       string                         `json:"worktreePath"`       // Git worktree path
	Branch             valueobject.Branch             `json:"branch"`             // Current branch
	CompletedTasks     []uuid.UUID                    `json:"completedTasks"`     // IDs of completed tasks
	CurrentTaskID      *uuid.UUID                     `json:"currentTaskId"`      // ID of currently executing task
	FailedTask         *valueobject.FailedTask        `json:"failedTask"`         // Current failed task info
	FixTasks           []uuid.UUID                    `json:"fixTasks"`           // Additional fix task IDs (from PR rollback)
	RollbackHistory    []valueobject.RollbackRecord   `json:"rollbackHistory"`    // History of rollback operations
	DeprecatedBranches []valueobject.DeprecatedBranch `json:"deprecatedBranches"` // Branches deprecated during rollbacks
}

// NewExecution creates a new Execution entity.
func NewExecution(worktreePath, branchName string) *Execution {
	return &Execution{
		WorktreePath:       worktreePath,
		Branch:             valueobject.NewBranch(branchName),
		CompletedTasks:     []uuid.UUID{},
		FixTasks:           []uuid.UUID{},
		RollbackHistory:    []valueobject.RollbackRecord{},
		DeprecatedBranches: []valueobject.DeprecatedBranch{},
	}
}

// SetCurrentTask sets the currently executing task.
func (e *Execution) SetCurrentTask(taskID uuid.UUID) {
	e.CurrentTaskID = &taskID
}

// ClearCurrentTask clears the currently executing task.
func (e *Execution) ClearCurrentTask() {
	e.CurrentTaskID = nil
}

// MarkTaskCompleted marks a task as completed.
func (e *Execution) MarkTaskCompleted(taskID uuid.UUID) {
	e.CompletedTasks = append(e.CompletedTasks, taskID)
	e.ClearCurrentTask()
	e.FailedTask = nil
}

// SetFailedTask records the current failed task.
func (e *Execution) SetFailedTask(taskID uuid.UUID, reason, suggestion string) {
	e.FailedTask = &valueobject.FailedTask{
		TaskID:     taskID.String(),
		Reason:     reason,
		Suggestion: suggestion,
	}
}

// ClearFailedTask clears the failed task info.
func (e *Execution) ClearFailedTask() {
	e.FailedTask = nil
}

// AddFixTask adds a fix task (from PR rollback).
func (e *Execution) AddFixTask(taskID uuid.UUID) {
	e.FixTasks = append(e.FixTasks, taskID)
}

// ClearFixTasks clears all fix tasks.
func (e *Execution) ClearFixTasks() {
	e.FixTasks = []uuid.UUID{}
}

// HasCompletedTask checks if a task is in the completed list.
func (e *Execution) HasCompletedTask(taskID uuid.UUID) bool {
	for _, id := range e.CompletedTasks {
		if id == taskID {
			return true
		}
	}
	return false
}

// RecordRollback records a rollback operation.
func (e *Execution) RecordRollback(fromStage, toStage valueobject.Stage, reason string, userInitiated bool) {
	record := valueobject.NewRollbackRecord(fromStage, toStage, reason, userInitiated)
	e.RollbackHistory = append(e.RollbackHistory, record)
}

// DeprecateBranch marks the current branch as deprecated.
func (e *Execution) DeprecateBranch(reason string, prNumber *int, rollbackToStage string) {
	e.Branch.Deprecate(reason)
	deprecated := valueobject.NewDeprecatedBranch(e.Branch.Name, reason, prNumber, rollbackToStage)
	e.DeprecatedBranches = append(e.DeprecatedBranches, deprecated)
}

// SetNewBranch sets a new active branch.
func (e *Execution) SetNewBranch(branchName string) {
	e.Branch = valueobject.NewBranch(branchName)
}

// AuditLog represents an audit log entry entity.
// Audit logs are immutable and record all key operations in the system.
type AuditLog struct {
	ID           uuid.UUID                 `json:"id"`           // Unique identifier
	Timestamp    time.Time                 `json:"timestamp"`    // Operation timestamp
	SessionID    uuid.UUID                 `json:"sessionId"`    // Associated work session ID
	Repository   string                    `json:"repository"`   // Repository name
	IssueNumber  int                       `json:"issueNumber"`  // Issue number
	Actor        string                    `json:"actor"`        // Operator (GitHub username)
	ActorRole    valueobject.ActorRole     `json:"actorRole"`    // Operator role
	Operation    valueobject.OperationType `json:"operation"`    // Operation type
	ResourceType string                    `json:"resourceType"` // Type of resource operated on
	ResourceID   string                    `json:"resourceId"`   // Resource identifier
	Parameters   map[string]any            `json:"parameters"`   // Operation parameters
	Result       valueobject.AuditResult   `json:"result"`       // Operation result
	Duration     int                       `json:"duration"`     // Operation duration (ms)
	Output       string                    `json:"output"`       // Output summary (truncated if too long)
	Error        string                    `json:"error"`        // Error message (if failed)
}

// NewAuditLog creates a new audit log entry.
func NewAuditLog(
	sessionID uuid.UUID,
	repository string,
	issueNumber int,
	actor string,
	actorRole valueobject.ActorRole,
	operation valueobject.OperationType,
	resourceType string,
	resourceID string,
) *AuditLog {
	return &AuditLog{
		ID:           uuid.New(),
		Timestamp:    time.Now(),
		SessionID:    sessionID,
		Repository:   repository,
		IssueNumber:  issueNumber,
		Actor:        actor,
		ActorRole:    actorRole,
		Operation:    operation,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Parameters:   make(map[string]any),
		Result:       valueobject.AuditResultSuccess,
	}
}

// SetParameters sets the operation parameters.
func (a *AuditLog) SetParameters(params map[string]any) {
	a.Parameters = params
}

// SetResult sets the operation result.
func (a *AuditLog) SetResult(result valueobject.AuditResult) {
	a.Result = result
}

// SetDuration sets the operation duration.
func (a *AuditLog) SetDuration(durationMs int) {
	a.Duration = durationMs
}

// SetOutput sets the output summary.
func (a *AuditLog) SetOutput(output string, maxLength int) {
	if len(output) > maxLength {
		a.Output = output[:maxLength] + "..."
	} else {
		a.Output = output
	}
}

// SetError sets the error message.
func (a *AuditLog) SetError(errMsg string) {
	a.Error = errMsg
	a.Result = valueobject.AuditResultFailed
}

// MarkSuccess marks the operation as successful.
func (a *AuditLog) MarkSuccess() {
	a.Result = valueobject.AuditResultSuccess
}

// MarkFailed marks the operation as failed.
func (a *AuditLog) MarkFailed(errorMsg string) {
	a.Error = errorMsg
	a.Result = valueobject.AuditResultFailed
}

// MarkDenied marks the operation as denied (permission issue).
func (a *AuditLog) MarkDenied(reason string) {
	a.Error = reason
	a.Result = valueobject.AuditResultDenied
}

// IsSuccess returns true if the operation succeeded.
func (a *AuditLog) IsSuccess() bool {
	return a.Result == valueobject.AuditResultSuccess
}

// IsFailed returns true if the operation failed.
func (a *AuditLog) IsFailed() bool {
	return a.Result == valueobject.AuditResultFailed
}

// IsDenied returns true if the operation was denied.
func (a *AuditLog) IsDenied() bool {
	return a.Result == valueobject.AuditResultDenied
}

// Repository represents a repository configuration entity.
// It stores configuration overrides for specific repositories.
type Repository struct {
	ID               uuid.UUID                                  `json:"id"`               // Unique identifier
	Name             string                                     `json:"name"`             // Repository name (owner/repo format)
	Enabled          bool                                       `json:"enabled"`          // Whether the repository is enabled
	InstallationID   int64                                      `json:"installationId"`   // GitHub App Installation ID (0 if not installed)
	Config           RepoConfig                                 `json:"config"`           // Repository-specific configuration overrides
	ValidationConfig *valueobject.ExecutionValidationConfig     `json:"validationConfig"` // Validation configuration (optional)
	DetectedProject  *valueobject.DetectedProject               `json:"detectedProject"`  // Detected project info (optional)
}

// RepoConfig represents repository-specific configuration overrides.
type RepoConfig struct {
	MaxConcurrency      *int    `json:"maxConcurrency,omitempty"`      // Max concurrent sessions
	ComplexityThreshold *int    `json:"complexityThreshold,omitempty"` // Complexity threshold for design confirmation
	ForceDesignConfirm  *bool   `json:"forceDesignConfirm,omitempty"`  // Force design confirmation
	DefaultModel        *string `json:"defaultModel,omitempty"`        // Default AI model
	TaskRetryLimit      *int    `json:"taskRetryLimit,omitempty"`      // Max task retry count
}

// NewRepository creates a new Repository entity.
func NewRepository(name string) *Repository {
	return &Repository{
		ID:      uuid.New(),
		Name:    name,
		Enabled: true,
		Config:  RepoConfig{},
	}
}

// SetValidationConfig sets the validation configuration.
func (r *Repository) SetValidationConfig(config *valueobject.ExecutionValidationConfig) {
	r.ValidationConfig = config
}

// SetDetectedProject sets the detected project information.
func (r *Repository) SetDetectedProject(project *valueobject.DetectedProject) {
	r.DetectedProject = project
}

// Validate validates the Repository entity.
func (r *Repository) Validate() error {
	if r.Name == "" {
		return errors.New(errors.ErrValidationFailed).WithDetail("repository name cannot be empty")
	}
	// Validate owner/repo format: must contain "/" and have at least 3 characters (e.g., "a/b")
	if len(r.Name) < 3 || !strings.Contains(r.Name, "/") {
		return errors.New(errors.ErrValidationFailed).WithDetail("repository name must be in owner/repo format")
	}
	return nil
}

// Enable enables the repository.
func (r *Repository) Enable() {
	r.Enabled = true
}

// Disable disables the repository.
func (r *Repository) Disable() {
	r.Enabled = false
}

// IsEnabled returns true if the repository is enabled.
func (r *Repository) IsEnabled() bool {
	return r.Enabled
}

// SetConfig sets the repository configuration.
func (r *Repository) SetConfig(config RepoConfig) {
	r.Config = config
}

// SetMaxConcurrency sets the max concurrency override.
func (r *Repository) SetMaxConcurrency(value int) {
	r.Config.MaxConcurrency = &value
}

// SetComplexityThreshold sets the complexity threshold override.
func (r *Repository) SetComplexityThreshold(value int) {
	r.Config.ComplexityThreshold = &value
}

// SetForceDesignConfirm sets the force design confirm override.
func (r *Repository) SetForceDesignConfirm(value bool) {
	r.Config.ForceDesignConfirm = &value
}

// SetDefaultModel sets the default model override.
func (r *Repository) SetDefaultModel(value string) {
	r.Config.DefaultModel = &value
}

// SetTaskRetryLimit sets the task retry limit override.
func (r *Repository) SetTaskRetryLimit(value int) {
	r.Config.TaskRetryLimit = &value
}

// SetInstallationID sets the GitHub App Installation ID.
func (r *Repository) SetInstallationID(id int64) {
	r.InstallationID = id
}

// HasInstallation returns true if the repository has a GitHub App installation.
func (r *Repository) HasInstallation() bool {
	return r.InstallationID > 0
}

// GetEffectiveConfig merges repository config with global config.
// Repository config takes precedence over global config.
func (r *Repository) GetEffectiveConfig(globalConfig RepoConfig) RepoConfig {
	effective := globalConfig

	if r.Config.MaxConcurrency != nil {
		effective.MaxConcurrency = r.Config.MaxConcurrency
	}
	if r.Config.ComplexityThreshold != nil {
		effective.ComplexityThreshold = r.Config.ComplexityThreshold
	}
	if r.Config.ForceDesignConfirm != nil {
		effective.ForceDesignConfirm = r.Config.ForceDesignConfirm
	}
	if r.Config.DefaultModel != nil {
		effective.DefaultModel = r.Config.DefaultModel
	}
	if r.Config.TaskRetryLimit != nil {
		effective.TaskRetryLimit = r.Config.TaskRetryLimit
	}

	return effective
}

// ProcessResult constants for WebhookDelivery.
const (
	ProcessResultSuccess    = "success"    // Webhook processed successfully
	ProcessResultIgnored    = "ignored"    // Webhook ignored (e.g., unsupported event)
	ProcessResultError      = "error"      // Webhook processing failed
	ProcessResultProcessing = "processing" // Webhook is being processed
)

// WebhookDelivery represents a webhook delivery record for idempotency.
// It tracks GitHub webhook deliveries to prevent duplicate processing.
type WebhookDelivery struct {
	ID             uuid.UUID `json:"id"`             // Unique identifier
	DeliveryID     string    `json:"deliveryId"`     // X-GitHub-Delivery header (unique)
	EventType      string    `json:"eventType"`      // X-GitHub-Event header
	Repository     string    `json:"repository"`     // Repository full name
	PayloadHash    string    `json:"payloadHash"`    // SHA256 hash of payload (optional)
	Processed      bool      `json:"processed"`      // Whether the webhook was processed
	ProcessResult  string    `json:"processResult"`  // success, ignored, error
	ProcessMessage string    `json:"processMessage"` // Processing message or error
	CreatedAt      time.Time `json:"createdAt"`      // Record creation time
	ExpiresAt      time.Time `json:"expiresAt"`      // TTL expiration time
}

// IsValidProcessResult checks if the process result is valid.
func IsValidProcessResult(result string) bool {
	switch result {
	case ProcessResultSuccess, ProcessResultIgnored, ProcessResultError, ProcessResultProcessing:
		return true
	default:
		return false
	}
}

// NewWebhookDelivery creates a new WebhookDelivery entity.
func NewWebhookDelivery(deliveryID, eventType, repository string) *WebhookDelivery {
	return &WebhookDelivery{
		ID:         uuid.New(),
		DeliveryID: deliveryID,
		EventType:  eventType,
		Repository: repository,
		Processed:  false,
		CreatedAt:  time.Now(),
	}
}

// SetProcessed marks the delivery as processed.
func (d *WebhookDelivery) SetProcessed(result, message string) {
	d.Processed = true
	d.ProcessResult = result
	d.ProcessMessage = message
}

// SetExpiresAt sets the expiration time.
func (d *WebhookDelivery) SetExpiresAt(ttl time.Duration) {
	d.ExpiresAt = time.Now().Add(ttl)
}

// IsExpired returns true if the delivery record has expired.
// Returns false if ExpiresAt is not set (zero value).
func (d *WebhookDelivery) IsExpired() bool {
	if d.ExpiresAt.IsZero() {
		return false // No expiration time set, not considered expired
	}
	return time.Now().After(d.ExpiresAt)
}

// IsSuccess returns true if the webhook was processed successfully.
func (d *WebhookDelivery) IsSuccess() bool {
	return d.ProcessResult == ProcessResultSuccess
}

// IsError returns true if the webhook processing resulted in error.
func (d *WebhookDelivery) IsError() bool {
	return d.ProcessResult == ProcessResultError
}

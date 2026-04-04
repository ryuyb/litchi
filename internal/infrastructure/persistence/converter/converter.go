// Package converter provides conversion functions between domain models
// and persistence (GORM) models.
package converter

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/persistence/models"
)

// ============================================
// WorkSession Conversion
// ============================================

// WorkSessionToModel converts a domain WorkSession aggregate to a GORM model.
func WorkSessionToModel(session *aggregate.WorkSession) (*models.WorkSession, error) {
	if session == nil {
		return nil, nil
	}

	m := &models.WorkSession{
		ID:           session.ID,
		CurrentStage: session.CurrentStage.String(),
		Status:       string(session.SessionStatus),
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
	}

	// Convert Issue
	if session.Issue != nil {
		m.IssueID = session.Issue.ID
		m.Issue = IssueToModel(session.Issue)
	}

	// Convert Clarification
	if session.Clarification != nil {
		m.Clarification = ClarificationToModel(session.Clarification, session.ID)
	}

	// Convert Design
	if session.Design != nil {
		m.Design = DesignToModel(session.Design, session.ID)
	}

	// Convert Tasks
	if session.Tasks != nil {
		m.Tasks = TasksToModels(session.Tasks, session.ID)
	}

	// Convert Execution
	if session.Execution != nil {
		m.Execution = ExecutionToModel(session.Execution, session.ID)
	}

	return m, nil
}

// WorkSessionFromModel converts a GORM WorkSession model to a domain aggregate.
func WorkSessionFromModel(m *models.WorkSession) (*aggregate.WorkSession, error) {
	if m == nil {
		return nil, nil
	}

	// Parse stage
	stage, err := valueobject.Parse(m.CurrentStage)
	if err != nil {
		return nil, err
	}

	session := &aggregate.WorkSession{
		ID:             m.ID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
		CurrentStage:   stage,
		SessionStatus:  aggregate.SessionStatus(m.Status),
		PRRollbackCount: 0, // Will be set from Execution if available
	}

	// Convert Issue
	if m.Issue != nil {
		session.Issue = IssueFromModel(m.Issue)
	}

	// Convert Clarification
	if m.Clarification != nil {
		session.Clarification = ClarificationFromModel(m.Clarification)
	}

	// Convert Design
	if m.Design != nil {
		session.Design = DesignFromModel(m.Design, m.Design.Versions)
	}

	// Convert Tasks
	if m.Tasks != nil {
		session.Tasks = TasksFromModels(m.Tasks)
	}

	// Convert Execution
	if m.Execution != nil {
		session.Execution = ExecutionFromModel(m.Execution)
		// Extract PRRollbackCount from rollback history
		session.PRRollbackCount = countPRRollbacks(session.Execution.RollbackHistory)
	}

	return session, nil
}

// countPRRollbacks counts the number of rollbacks from PR stage.
func countPRRollbacks(history []valueobject.RollbackRecord) int {
	count := 0
	for _, r := range history {
		if r.FromStage == valueobject.StagePullRequest {
			count++
		}
	}
	return count
}

// ============================================
// Issue Conversion
// ============================================

// IssueToModel converts a domain Issue entity to a GORM model.
func IssueToModel(issue *entity.Issue) *models.Issue {
	if issue == nil {
		return nil
	}

	labelsJSON, _ := json.Marshal(issue.Labels)

	return &models.Issue{
		ID:         issue.ID,
		Number:     int64(issue.Number),
		Title:      issue.Title,
		Body:       issue.Body,
		Repository: issue.Repository,
		Author:     issue.Author,
		Labels:     labelsJSON,
		URL:        issue.URL,
		CreatedAt:  issue.CreatedAt,
	}
}

// IssueFromModel converts a GORM Issue model to a domain entity.
func IssueFromModel(m *models.Issue) *entity.Issue {
	if m == nil {
		return nil
	}

	// Unmarshal labels
	var labels []string
	if len(m.Labels) > 0 {
		json.Unmarshal(m.Labels, &labels)
	}
	if labels == nil {
		labels = []string{}
	}

	return &entity.Issue{
		ID:         m.ID,
		Number:     int(m.Number),
		Title:      m.Title,
		Body:       m.Body,
		Repository: m.Repository,
		Author:     m.Author,
		Labels:     labels,
		URL:        m.URL,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.CreatedAt, // Use CreatedAt as UpdatedAt fallback
	}
}

// ============================================
// Clarification Conversion
// ============================================

// ClarificationToModel converts a domain Clarification entity to a GORM model.
func ClarificationToModel(c *entity.Clarification, sessionID uuid.UUID) *models.Clarification {
	if c == nil {
		return nil
	}

	confirmedPointsJSON, _ := json.Marshal(c.ConfirmedPoints)
	pendingQuestionsJSON, _ := json.Marshal(c.PendingQuestions)
	historyJSON, _ := json.Marshal(c.History)
	clarityDimensionsJSON, _ := json.Marshal(c.ClarityDimensions)

	var clarityScore *int
	if c.ClarityDimensions.TotalScore() > 0 {
		score := c.ClarityDimensions.TotalScore()
		clarityScore = &score
	}

	return &models.Clarification{
		ID:                  uuid.New(),
		SessionID:           sessionID,
		ConfirmedPoints:     confirmedPointsJSON,
		PendingQuestions:    pendingQuestionsJSON,
		ConversationHistory: historyJSON,
		Status:              string(c.Status),
		ClarityScore:        clarityScore,
		ClarityDimensions:   clarityDimensionsJSON,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}

// ClarificationFromModel converts a GORM Clarification model to a domain entity.
func ClarificationFromModel(m *models.Clarification) *entity.Clarification {
	if m == nil {
		return nil
	}

	c := entity.NewClarification()

	// Unmarshal confirmed points
	if len(m.ConfirmedPoints) > 0 {
		json.Unmarshal(m.ConfirmedPoints, &c.ConfirmedPoints)
	}

	// Unmarshal pending questions
	if len(m.PendingQuestions) > 0 {
		json.Unmarshal(m.PendingQuestions, &c.PendingQuestions)
	}

	// Unmarshal conversation history
	if len(m.ConversationHistory) > 0 {
		json.Unmarshal(m.ConversationHistory, &c.History)
	}

	// Unmarshal clarity dimensions
	if len(m.ClarityDimensions) > 0 {
		json.Unmarshal(m.ClarityDimensions, &c.ClarityDimensions)
	}

	// Set status
	c.Status = entity.ClarificationStatus(m.Status)

	return c
}

// ============================================
// Design Conversion
// ============================================

// DesignToModel converts a domain Design entity to a GORM model.
func DesignToModel(d *entity.Design, sessionID uuid.UUID) *models.Design {
	if d == nil {
		return nil
	}

	var complexityScore *int
	if d.ComplexityScore.Value() > 0 {
		score := d.ComplexityScore.Value()
		complexityScore = &score
	}

	m := &models.Design{
		ID:                uuid.New(),
		SessionID:         sessionID,
		CurrentVersion:    d.CurrentVersion,
		ComplexityScore:   complexityScore,
		RequireConfirmation: d.RequireConfirmation,
		Confirmed:         d.Confirmed,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Convert versions
	if d.Versions != nil {
		m.Versions = DesignVersionsToModels(d.Versions, m.ID)
	}

	return m
}

// DesignFromModel converts a GORM Design model to a domain entity.
func DesignFromModel(m *models.Design, versionModels []models.DesignVersion) *entity.Design {
	if m == nil {
		return nil
	}

	d := &entity.Design{
		CurrentVersion:     m.CurrentVersion,
		RequireConfirmation: m.RequireConfirmation,
		Confirmed:          m.Confirmed,
	}

	// Set complexity score
	if m.ComplexityScore != nil {
		cs, err := valueobject.NewComplexityScore(*m.ComplexityScore)
		if err == nil {
			d.ComplexityScore = cs
		}
	}

	// Convert versions
	if versionModels != nil {
		d.Versions = DesignVersionsFromModels(versionModels)
	}

	return d
}

// DesignVersionsToModels converts domain design versions to GORM models.
func DesignVersionsToModels(versions []valueobject.DesignVersion, designID uuid.UUID) []models.DesignVersion {
	if versions == nil {
		return nil
	}

	result := make([]models.DesignVersion, len(versions))
	for i, v := range versions {
		result[i] = models.DesignVersion{
			ID:        uuid.New(),
			DesignID:  designID,
			Version:   v.Version,
			Content:   v.Content,
			Reason:    v.Reason,
			CreatedAt: v.CreatedAt,
		}
	}
	return result
}

// DesignVersionsFromModels converts GORM design version models to domain value objects.
func DesignVersionsFromModels(models []models.DesignVersion) []valueobject.DesignVersion {
	if models == nil {
		return nil
	}

	result := make([]valueobject.DesignVersion, len(models))
	for i, m := range models {
		result[i] = valueobject.DesignVersion{
			Version:   m.Version,
			Content:   m.Content,
			Reason:    m.Reason,
			CreatedAt: m.CreatedAt,
		}
	}
	return result
}

// ============================================
// Task Conversion
// ============================================

// TasksToModels converts a slice of domain Task entities to GORM models.
func TasksToModels(tasks []*entity.Task, sessionID uuid.UUID) []models.Task {
	if tasks == nil {
		return nil
	}

	result := make([]models.Task, len(tasks))
	for i, t := range tasks {
		result[i] = TaskToModel(t, sessionID)
	}
	return result
}

// TasksFromModels converts a slice of GORM Task models to domain entities.
func TasksFromModels(models []models.Task) []*entity.Task {
	if models == nil {
		return nil
	}

	result := make([]*entity.Task, len(models))
	for i, m := range models {
		result[i] = TaskFromModel(&m)
	}
	return result
}

// TaskToModel converts a domain Task entity to a GORM model.
func TaskToModel(t *entity.Task, sessionID uuid.UUID) models.Task {
	if t == nil {
		return models.Task{}
	}

	// Convert dependencies to many-to-many format
	depTasks := make([]models.Task, len(t.Dependencies))
	for i, depID := range t.Dependencies {
		depTasks[i] = models.Task{ID: depID}
	}

	return models.Task{
		ID:            t.ID,
		SessionID:     sessionID,
		Description:   t.Description,
		Status:        t.Status.String(),
		RetryCount:    t.RetryCount,
		FailureReason: t.FailureReason,
		Suggestion:    t.Suggestion,
		Seq:           t.Order,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Dependencies:  depTasks,
	}
}

// TaskFromModel converts a GORM Task model to a domain entity.
func TaskFromModel(m *models.Task) *entity.Task {
	if m == nil {
		return nil
	}

	// Parse status
	status, err := valueobject.ParseTaskStatus(m.Status)
	if err != nil {
		status = valueobject.TaskStatusPending
	}

	// Extract dependencies
	dependencies := make([]uuid.UUID, len(m.Dependencies))
	for i, dep := range m.Dependencies {
		dependencies[i] = dep.ID
	}

	return &entity.Task{
		ID:           m.ID,
		Description:  m.Description,
		Status:       status,
		Dependencies: dependencies,
		RetryCount:   m.RetryCount,
		FailureReason: m.FailureReason,
		Suggestion:   m.Suggestion,
		Order:        m.Seq,
	}
}

// ============================================
// Execution Conversion
// ============================================

// ExecutionToModel converts a domain Execution entity to a GORM model.
func ExecutionToModel(e *entity.Execution, sessionID uuid.UUID) *models.Execution {
	if e == nil {
		return nil
	}

	deprecatedBranchesJSON, _ := json.Marshal(e.DeprecatedBranches)
	failedTaskJSON, _ := json.Marshal(e.FailedTask)
	fixTasksJSON, _ := json.Marshal(e.FixTasks)
	rollbackHistoryJSON, _ := json.Marshal(e.RollbackHistory)

	var currentTaskID *uuid.UUID
	if e.CurrentTaskID != nil {
		currentTaskID = e.CurrentTaskID
	}

	m := &models.Execution{
		ID:                 uuid.New(),
		SessionID:          sessionID,
		WorktreePath:       e.WorktreePath,
		BranchName:         e.Branch.Name,
		BranchDeprecated:   e.Branch.IsDeprecated,
		DeprecatedBranches: deprecatedBranchesJSON,
		CurrentTaskID:      currentTaskID,
		FailedTask:         failedTaskJSON,
		FixTasks:           fixTasksJSON,
		RollbackHistory:    rollbackHistoryJSON,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Convert completed tasks
	if e.CompletedTasks != nil {
		m.CompletedTasks = make([]models.Task, len(e.CompletedTasks))
		for i, taskID := range e.CompletedTasks {
			m.CompletedTasks[i] = models.Task{ID: taskID}
		}
	}

	return m
}

// ExecutionFromModel converts a GORM Execution model to a domain entity.
func ExecutionFromModel(m *models.Execution) *entity.Execution {
	if m == nil {
		return nil
	}

	e := &entity.Execution{
		WorktreePath: m.WorktreePath,
		Branch:       valueobject.NewBranch(m.BranchName),
		CompletedTasks: make([]uuid.UUID, 0),
		FixTasks:     make([]uuid.UUID, 0),
		RollbackHistory: make([]valueobject.RollbackRecord, 0),
		DeprecatedBranches: make([]valueobject.DeprecatedBranch, 0),
	}

	// Set branch deprecated status
	if m.BranchDeprecated {
		e.Branch.Deprecate("")
	}

	// Set current task ID
	if m.CurrentTaskID != nil {
		e.CurrentTaskID = m.CurrentTaskID
	}

	// Unmarshal failed task
	if len(m.FailedTask) > 0 {
		json.Unmarshal(m.FailedTask, &e.FailedTask)
	}

	// Unmarshal fix tasks
	if len(m.FixTasks) > 0 {
		json.Unmarshal(m.FixTasks, &e.FixTasks)
	}

	// Unmarshal rollback history
	if len(m.RollbackHistory) > 0 {
		json.Unmarshal(m.RollbackHistory, &e.RollbackHistory)
	}

	// Unmarshal deprecated branches
	if len(m.DeprecatedBranches) > 0 {
		json.Unmarshal(m.DeprecatedBranches, &e.DeprecatedBranches)
	}

	// Extract completed task IDs
	if m.CompletedTasks != nil {
		e.CompletedTasks = make([]uuid.UUID, len(m.CompletedTasks))
		for i, task := range m.CompletedTasks {
			e.CompletedTasks[i] = task.ID
		}
	}

	return e
}
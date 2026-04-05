// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/event"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/infrastructure/git"
	"github.com/ryuyb/litchi/internal/infrastructure/github"
	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/ryuyb/litchi/internal/pkg/utils"
	"go.uber.org/zap"
)

// PRService handles the Pull Request phase of WorkSession.
// It manages PR creation, updates, conflict detection, and resolution.
//
// Core responsibilities:
// 1. PR creation - create PR from feature branch to base branch
// 2. PR updates - update PR content, handle new commits
// 3. Conflict handling - detect and help resolve merge conflicts
type PRService struct {
	sessionRepo       repository.WorkSessionRepository
	auditRepo         repository.AuditLogRepository
	ghPRService       *github.PullRequestService
	ghIssueService    *github.IssueService
	conflictDetector  git.ConflictDetector
	branchService     git.BranchService
	eventDispatcher   *event.Dispatcher
	config            *config.Config
	logger            *zap.Logger
}

// NewPRService creates a new PRService.
func NewPRService(
	sessionRepo repository.WorkSessionRepository,
	auditRepo repository.AuditLogRepository,
	ghPRService *github.PullRequestService,
	ghIssueService *github.IssueService,
	conflictDetector git.ConflictDetector,
	branchService git.BranchService,
	eventDispatcher *event.Dispatcher,
	config *config.Config,
	logger *zap.Logger,
) *PRService {
	return &PRService{
		sessionRepo:      sessionRepo,
		auditRepo:        auditRepo,
		ghPRService:      ghPRService,
		ghIssueService:   ghIssueService,
		conflictDetector: conflictDetector,
		branchService:    branchService,
		eventDispatcher:  eventDispatcher,
		config:           config,
		logger:           logger.Named("pr_service"),
	}
}

// CreatePR creates a Pull Request for the session.
// This method should be called after all tasks are completed.
//
// Steps:
// 1. Validate session is in PullRequest stage
// 2. Check all tasks are completed
// 3. Check for merge conflicts
// 4. Create PR on GitHub
// 5. Update session with PR number
// 6. Post PR link to issue
//
// Returns the created PR number.
func (s *PRService) CreatePR(
	ctx context.Context,
	sessionID uuid.UUID,
) (prNumber int, err error) {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return 0, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return 0, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate session is in PullRequest stage
	if session.GetCurrentStage() != valueobject.StagePullRequest {
		return 0, litchierrors.New(litchierrors.ErrInvalidStage).WithDetail(
			fmt.Sprintf("session is in %s stage, expected PullRequest", session.GetCurrentStage()),
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return 0, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Check if PR already exists
	if session.PRNumber != nil {
		return 0, litchierrors.New(litchierrors.ErrPRAlreadyExists).WithDetail(
			fmt.Sprintf("PR #%d already exists for this session", *session.PRNumber),
		)
	}

	// 5. Validate all tasks are completed
	if !session.AreAllTasksCompleted() {
		return 0, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"not all tasks are completed",
		)
	}

	// 6. Get execution context
	execution := session.GetExecution()
	if execution == nil {
		return 0, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"execution context not found",
		)
	}

	branchName := execution.Branch.Name

	// 7. Check for merge conflicts before creating PR
	conflicts, err := s.checkMergeConflicts(ctx, session, branchName)
	if err != nil {
		s.logger.Warn("failed to check merge conflicts, proceeding anyway",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
		// Continue even if conflict check fails - PR can still be created
	}
	if len(conflicts) > 0 {
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpPRConflictDetect, startTime, false,
			fmt.Sprintf("conflicts detected in %d files", len(conflicts)))
		return 0, litchierrors.New(litchierrors.ErrPRConflict).WithDetail(
			fmt.Sprintf("merge conflicts detected in %d files: %s",
				len(conflicts), strings.Join(conflicts, ", ")),
		)
	}

	// 8. Build PR title and body
	prTitle := s.buildPRTitle(session)
	prBody := s.buildPRBody(session)

	// 9. Get repository info
	owner := utils.ExtractOwner(session.Issue.Repository)
	repo := utils.ExtractRepo(session.Issue.Repository)
	baseBranch := s.config.Git.DefaultBaseBranch

	// 10. Create PR on GitHub
	prInfo, err := s.ghPRService.CreatePullRequest(ctx, owner, repo, &github.PRCreateRequest{
		Title:      prTitle,
		Body:       prBody,
		HeadBranch: branchName,
		BaseBranch: baseBranch,
		Draft:      false,
	})
	if err != nil {
		s.logger.Error("failed to create PR",
			zap.String("session_id", sessionID.String()),
			zap.String("branch", branchName),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpPRCreate, startTime, false, err.Error())
		return 0, litchierrors.Wrap(litchierrors.ErrPRCreateFailed, err)
	}

	prNumber = prInfo.Number

	// 11. Update session with PR number
	session.SetPRNumber(prNumber)

	// 12. Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return prNumber, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}

	// 13. Record audit log
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpPRCreate, startTime, true, fmt.Sprintf("PR #%d created", prNumber))

	// 14. Publish events
	s.publishEvents(ctx, session)

	// 15. Post PR link to issue
	if err := s.postPRLinkToIssue(ctx, session, prInfo); err != nil {
		s.logger.Warn("failed to post PR link to issue",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
	}

	s.logger.Info("PR created",
		zap.String("session_id", sessionID.String()),
		zap.Int("pr_number", prNumber),
		zap.String("branch", branchName),
		zap.String("base", baseBranch),
	)

	return prNumber, nil
}

// UpdatePR updates the Pull Request content.
// This is used when additional commits are pushed to the PR branch.
//
// Steps:
// 1. Validate PR exists
// 2. Update PR description with latest info
// 3. Check for new conflicts
//
// Returns error if update fails.
func (s *PRService) UpdatePR(
	ctx context.Context,
	sessionID uuid.UUID,
	reason string,
) error {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate PR exists
	if session.PRNumber == nil {
		return litchierrors.New(litchierrors.ErrPRNotFound).WithDetail(
			"no PR exists for this session",
		)
	}

	// 3. Validate session is active
	if !session.IsActive() {
		return litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"session is not active",
		)
	}

	// 4. Build updated PR body
	prBody := s.buildPRBody(session)
	if reason != "" {
		prBody += fmt.Sprintf("\n\n---\n**Update reason**: %s", reason)
	}

	// 5. Update PR on GitHub
	owner := utils.ExtractOwner(session.Issue.Repository)
	repo := utils.ExtractRepo(session.Issue.Repository)

	err = s.ghPRService.UpdatePullRequest(ctx, owner, repo, *session.PRNumber, &github.PRUpdateRequest{
		Body: &prBody,
	})
	if err != nil {
		s.logger.Error("failed to update PR",
			zap.String("session_id", sessionID.String()),
			zap.Int("pr_number", *session.PRNumber),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpPRUpdate, startTime, false, err.Error())
		return litchierrors.Wrap(litchierrors.ErrPRUpdateFailed, err)
	}

	// 6. Record audit log
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpPRUpdate, startTime, true, reason)

	// 7. Publish events
	s.publishEvents(ctx, session)

	s.logger.Info("PR updated",
		zap.String("session_id", sessionID.String()),
		zap.Int("pr_number", *session.PRNumber),
		zap.String("reason", reason),
	)

	return nil
}

// CheckConflicts checks for merge conflicts between PR branch and base branch.
//
// Steps:
// 1. Validate PR exists
// 2. Check for merge conflicts using conflict detector
// 3. Return list of conflicted files
//
// Returns list of conflicted file paths.
func (s *PRService) CheckConflicts(
	ctx context.Context,
	sessionID uuid.UUID,
) ([]string, error) {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate execution context exists
	execution := session.GetExecution()
	if execution == nil {
		return nil, litchierrors.New(litchierrors.ErrValidationFailed).WithDetail(
			"execution context not found",
		)
	}

	branchName := execution.Branch.Name

	// 3. Check conflicts
	conflicts, err := s.checkMergeConflicts(ctx, session, branchName)
	if err != nil {
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpPRConflictDetect, startTime, false, err.Error())
		return nil, err
	}

	// 4. Record audit log
	resultMsg := "no conflicts"
	if len(conflicts) > 0 {
		resultMsg = fmt.Sprintf("%d conflicts detected", len(conflicts))
	}
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpPRConflictDetect, startTime, true, resultMsg)

	s.logger.Info("conflict check completed",
		zap.String("session_id", sessionID.String()),
		zap.Int("conflicts", len(conflicts)),
	)

	return conflicts, nil
}

// GetPRStatus returns the current PR status.
func (s *PRService) GetPRStatus(
	ctx context.Context,
	sessionID uuid.UUID,
) (*PRStatus, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return nil, litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	status := &PRStatus{
		SessionID:    sessionID,
		CurrentStage: string(session.GetCurrentStage()),
		HasPR:        session.PRNumber != nil,
	}

	if session.PRNumber != nil {
		status.PRNumber = *session.PRNumber

		// Get PR info from GitHub
		owner := utils.ExtractOwner(session.Issue.Repository)
		repo := utils.ExtractRepo(session.Issue.Repository)

		prInfo, err := s.ghPRService.GetPullRequest(ctx, owner, repo, *session.PRNumber)
		if err != nil {
			s.logger.Warn("failed to get PR info from GitHub",
				zap.String("session_id", sessionID.String()),
				zap.Int("pr_number", *session.PRNumber),
				zap.Error(err),
			)
		} else {
			status.Title = prInfo.Title
			status.State = prInfo.State
			status.HeadBranch = prInfo.HeadBranch
			status.BaseBranch = prInfo.BaseBranch
			status.Mergeable = prInfo.Mergeable
			status.Merged = prInfo.Merged
			status.Draft = prInfo.Draft
			status.HTMLURL = prInfo.HTMLURL
			status.Commits = prInfo.Commits
			status.Additions = prInfo.Additions
			status.Deletions = prInfo.Deletions
			status.Changed = prInfo.Changed
		}

		// Get execution context for branch info
		if execution := session.GetExecution(); execution != nil {
			status.Branch = execution.Branch.Name
			status.WorktreePath = execution.WorktreePath
		}
	}

	return status, nil
}

// ClosePR closes the PR without merging.
// This is used when the PR needs to be cancelled.
func (s *PRService) ClosePR(
	ctx context.Context,
	sessionID uuid.UUID,
	reason string,
) error {
	startTime := time.Now()

	// 1. Get session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return litchierrors.Wrap(litchierrors.ErrDatabaseOperation, err)
	}
	if session == nil {
		return litchierrors.New(litchierrors.ErrSessionNotFound).WithDetail(
			fmt.Sprintf("session %s not found", sessionID),
		)
	}

	// 2. Validate PR exists
	if session.PRNumber == nil {
		return litchierrors.New(litchierrors.ErrPRNotFound).WithDetail(
			"no PR exists for this session",
		)
	}

	// 3. Close PR on GitHub
	owner := utils.ExtractOwner(session.Issue.Repository)
	repo := utils.ExtractRepo(session.Issue.Repository)

	err = s.ghPRService.ClosePullRequest(ctx, owner, repo, *session.PRNumber)
	if err != nil {
		s.logger.Error("failed to close PR",
			zap.String("session_id", sessionID.String()),
			zap.Int("pr_number", *session.PRNumber),
			zap.Error(err),
		)
		s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
			valueobject.OpPRClose, startTime, false, err.Error())
		return litchierrors.Wrap(litchierrors.ErrPRUpdateFailed, err)
	}

	// 4. Record audit log
	s.recordAuditLog(ctx, session, "system", valueobject.ActorRoleSystem,
		valueobject.OpPRClose, startTime, true, reason)

	// 5. Publish events
	s.publishEvents(ctx, session)

	s.logger.Info("PR closed",
		zap.String("session_id", sessionID.String()),
		zap.Int("pr_number", *session.PRNumber),
		zap.String("reason", reason),
	)

	return nil
}

// PRStatus represents the current status of a PR.
type PRStatus struct {
	SessionID    uuid.UUID `json:"sessionId"`
	CurrentStage string    `json:"currentStage"`
	HasPR        bool      `json:"hasPR"`
	PRNumber     int       `json:"prNumber,omitempty"`
	Title        string    `json:"title,omitempty"`
	State        string    `json:"state,omitempty"`
	HeadBranch   string    `json:"headBranch,omitempty"`
	BaseBranch   string    `json:"baseBranch,omitempty"`
	Mergeable    *bool     `json:"mergeable,omitempty"`
	Merged       bool      `json:"merged"`
	Draft        bool      `json:"draft"`
	HTMLURL      string    `json:"htmlUrl,omitempty"`
	Commits      int       `json:"commits,omitempty"`
	Additions    int       `json:"additions,omitempty"`
	Deletions    int       `json:"deletions,omitempty"`
	Changed      int       `json:"changed,omitempty"`
	Branch       string    `json:"branch,omitempty"`
	WorktreePath string    `json:"worktreePath,omitempty"`
}

// --- Internal Helper Methods ---

// checkMergeConflicts checks for merge conflicts between branch and base.
func (s *PRService) checkMergeConflicts(
	ctx context.Context,
	session *aggregate.WorkSession,
	branchName string,
) ([]string, error) {
	execution := session.GetExecution()
	if execution == nil {
		return nil, nil
	}

	worktreePath := execution.WorktreePath
	if worktreePath == "" {
		return nil, nil
	}

	baseBranch := s.config.Git.DefaultBaseBranch

	// Use conflict detector to check for conflicts
	conflictInfos, err := s.conflictDetector.DetectConflicts(ctx, worktreePath, branchName, baseBranch)
	if err != nil {
		s.logger.Warn("conflict detection failed",
			zap.String("worktree", worktreePath),
			zap.String("branch", branchName),
			zap.Error(err),
		)
		return nil, err
	}

	if len(conflictInfos) == 0 {
		return nil, nil
	}

	// Extract file paths
	conflicts := make([]string, len(conflictInfos))
	for i, ci := range conflictInfos {
		conflicts[i] = ci.FilePath
	}

	return conflicts, nil
}

// buildPRTitle builds the PR title from issue info.
func (s *PRService) buildPRTitle(session *aggregate.WorkSession) string {
	issue := session.GetIssue()
	if issue == nil {
		return "Implement changes"
	}

	return fmt.Sprintf("Resolve #%d: %s", issue.Number, issue.Title)
}

// buildPRBody builds the PR description from session context.
func (s *PRService) buildPRBody(session *aggregate.WorkSession) string {
	var body strings.Builder

	// Header
	issue := session.GetIssue()
	if issue != nil {
		body.WriteString(fmt.Sprintf("## Summary\n\n"))
		body.WriteString(fmt.Sprintf("Resolves #%d\n\n", issue.Number))
		body.WriteString(fmt.Sprintf("**Original Issue**: %s\n\n", issue.Title))
	}

	// Design summary
	if session.Design != nil {
		body.WriteString("## Design\n\n")
		// Get first 500 chars of design
		designContent := session.Design.GetCurrentContent()
		if len(designContent) > 500 {
			body.WriteString(designContent[:500] + "...\n\n")
		} else {
			body.WriteString(designContent + "\n\n")
		}
	}

	// Tasks summary
	tasks := session.GetTasks()
	if len(tasks) > 0 {
		body.WriteString("## Tasks Completed\n\n")
		for _, task := range tasks {
			status := "✅"
			if task.IsSkipped() {
				status = "⏭️"
			}
			body.WriteString(fmt.Sprintf("%s %s\n", status, task.Description))
		}
		body.WriteString("\n")
	}

	// Statistics
	body.WriteString("## Statistics\n\n")
	completed := 0
	skipped := 0
	for _, task := range tasks {
		if task.IsCompleted() {
			completed++
		} else if task.IsSkipped() {
			skipped++
		}
	}
	body.WriteString(fmt.Sprintf("- **Tasks Completed**: %d\n", completed))
	if skipped > 0 {
		body.WriteString(fmt.Sprintf("- **Tasks Skipped**: %d\n", skipped))
	}

	// Footer
	body.WriteString("\n---\n")
	body.WriteString("*This PR was automatically generated by Litchi.*\n")

	return body.String()
}

// postPRLinkToIssue posts the PR link as a comment on the issue.
func (s *PRService) postPRLinkToIssue(
	ctx context.Context,
	session *aggregate.WorkSession,
	prInfo *github.PRInfo,
) error {
	commentBody := fmt.Sprintf("## Pull Request Created\n\n")
	commentBody += fmt.Sprintf("**PR #%d**: %s\n\n", prInfo.Number, prInfo.HTMLURL)
	commentBody += fmt.Sprintf("**Branch**: `%s` → `%s`\n\n", prInfo.HeadBranch, prInfo.BaseBranch)
	commentBody += "Please review the changes and merge when ready.\n"

	owner := utils.ExtractOwner(session.Issue.Repository)
	repo := utils.ExtractRepo(session.Issue.Repository)

	_, err := s.ghIssueService.CreateComment(ctx, owner, repo, session.Issue.Number, commentBody)
	return err
}

// recordAuditLog records an audit log entry.
func (s *PRService) recordAuditLog(
	ctx context.Context,
	session *aggregate.WorkSession,
	actor string,
	actorRole valueobject.ActorRole,
	operation valueobject.OperationType,
	startTime time.Time,
	success bool,
	errMsg string,
) {
	if session == nil {
		return
	}

	auditLog := entity.NewAuditLog(
		session.ID,
		session.Issue.Repository,
		session.Issue.Number,
		actor,
		actorRole,
		operation,
		"pr",
		session.ID.String(),
	)

	auditLog.SetDuration(int(time.Since(startTime).Milliseconds()))

	if success {
		auditLog.MarkSuccess()
	} else if errMsg != "" {
		auditLog.MarkFailed(errMsg)
	}

	if err := s.auditRepo.Save(ctx, auditLog); err != nil {
		s.logger.Warn("failed to save audit log",
			zap.String("session_id", session.ID.String()),
			zap.Error(err),
		)
	}
}

// publishEvents publishes domain events from the session.
func (s *PRService) publishEvents(ctx context.Context, session *aggregate.WorkSession) {
	events := session.GetEvents()
	if len(events) == 0 {
		return
	}

	if err := s.eventDispatcher.DispatchBatch(ctx, events); err != nil {
		s.logger.Warn("failed to dispatch events",
			zap.String("session_id", session.ID.String()),
			zap.Int("event_count", len(events)),
			zap.Error(err),
		)
	}

	session.ClearEvents()
}
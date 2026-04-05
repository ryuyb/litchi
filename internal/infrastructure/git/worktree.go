package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// WorktreeService provides Git worktree management operations.
// This is the core service for enabling parallel session execution.
type WorktreeService interface {
	// CreateWorktree creates a new worktree for an existing branch.
	// The worktree allows isolated working directory for each session.
	CreateWorktree(ctx context.Context, mainRepoPath, worktreePath, branchName string) error

	// CreateWorktreeWithNewBranch creates a worktree with a new branch.
	// Useful when starting a new session with a fresh branch.
	CreateWorktreeWithNewBranch(ctx context.Context, mainRepoPath, worktreePath, branchName, startPoint string) error

	// DeleteWorktree removes a worktree.
	// Does not delete the associated branch.
	DeleteWorktree(ctx context.Context, mainRepoPath, worktreePath string) error

	// DeleteWorktreeAndBranch removes worktree and its associated branch.
	DeleteWorktreeAndBranch(ctx context.Context, mainRepoPath, worktreePath, branchName string) error

	// ListWorktrees returns all worktrees for a repository.
	ListWorktrees(ctx context.Context, mainRepoPath string) ([]WorktreeInfo, error)

	// WorktreeExists checks if a worktree exists at the given path.
	WorktreeExists(ctx context.Context, worktreePath string) bool

	// GetMainRepoPath returns the main repository path from a worktree.
	GetMainRepoPath(ctx context.Context, worktreePath string) (string, error)

	// ValidateWorktreeIsolation verifies worktree isolation.
	// Ensures the worktree has its own working directory and can operate independently.
	ValidateWorktreeIsolation(ctx context.Context, worktreePath string) error

	// PruneWorktrees removes stale worktree references.
	PruneWorktrees(ctx context.Context, mainRepoPath string) error
}

// WorktreeInfo contains worktree metadata.
type WorktreeInfo struct {
	Path       string // Worktree path
	Branch     string // Associated branch
	HEAD       string // Current HEAD commit SHA (short)
	IsLocked   bool   // Is worktree locked
	LockReason string // Lock reason (if locked)
	IsPrunable bool   // Can be pruned (if deleted externally)
}

// WorktreeServiceParams contains dependencies for creating a WorktreeService.
type WorktreeServiceParams struct {
	Executor    *CommandExecutor
	Client      GitClient
	BranchSvc   BranchService
	Logger      *zap.Logger
}

// worktreeServiceImpl implements WorktreeService.
type worktreeServiceImpl struct {
	executor  *CommandExecutor
	client    GitClient
	branchSvc BranchService
	logger    *zap.Logger
}

// NewWorktreeService creates a new WorktreeService.
func NewWorktreeService(p WorktreeServiceParams) WorktreeService {
	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &worktreeServiceImpl{
		executor:  p.Executor,
		client:    p.Client,
		branchSvc: p.BranchSvc,
		logger:    logger.Named("worktree-service"),
	}
}

// CreateWorktree creates a new worktree for an existing branch.
func (s *worktreeServiceImpl) CreateWorktree(ctx context.Context, mainRepoPath, worktreePath, branchName string) error {
	// Validate worktree path doesn't exist
	if _, err := os.Stat(worktreePath); err == nil {
		return errors.New(errors.ErrGitWorktreePathExists).
			WithDetail("worktree path already exists: " + worktreePath)
	}

	// Check if branch exists
	if !s.branchSvc.BranchExists(ctx, mainRepoPath, branchName) {
		return errors.New(errors.ErrGitBranchNotFound).
			WithDetail("branch not found: " + branchName)
	}

	// Create worktree
	_, err := s.executor.Exec(ctx, mainRepoPath, "worktree", "add", worktreePath, branchName)
	if err != nil {
		return errors.Wrap(errors.ErrGitWorktreeCreateFailed, err).
			WithDetail("failed to create worktree at " + worktreePath).
			WithContext("branch", branchName)
	}

	// Validate isolation
	if err := s.ValidateWorktreeIsolation(ctx, worktreePath); err != nil {
		// Clean up on validation failure
		if cleanErr := s.DeleteWorktree(ctx, mainRepoPath, worktreePath); cleanErr != nil {
			s.logger.Warn("failed to cleanup worktree after validation failure",
				zap.String("worktree", worktreePath),
				zap.Error(cleanErr),
			)
		}
		return err
	}

	s.logger.Info("worktree created",
		zap.String("main_repo", mainRepoPath),
		zap.String("worktree", worktreePath),
		zap.String("branch", branchName),
	)

	return nil
}

// CreateWorktreeWithNewBranch creates a worktree with a new branch.
func (s *worktreeServiceImpl) CreateWorktreeWithNewBranch(ctx context.Context, mainRepoPath, worktreePath, branchName, startPoint string) error {
	// Validate worktree path doesn't exist
	if _, err := os.Stat(worktreePath); err == nil {
		return errors.New(errors.ErrGitWorktreePathExists).
			WithDetail("worktree path already exists: " + worktreePath)
	}

	// Validate branch name
	if err := s.branchSvc.ValidateBranchName(branchName); err != nil {
		return err
	}

	// Check if branch already exists
	if s.branchSvc.BranchExists(ctx, mainRepoPath, branchName) {
		return errors.New(errors.ErrGitBranchExists).
			WithDetail("branch already exists: " + branchName)
	}

	// Build command arguments
	args := []string{"worktree", "add", "-b", branchName, worktreePath}
	if startPoint != "" {
		args = append(args, startPoint)
	}

	// Create worktree with new branch
	_, err := s.executor.Exec(ctx, mainRepoPath, args...)
	if err != nil {
		return errors.Wrap(errors.ErrGitWorktreeCreateFailed, err).
			WithDetail("failed to create worktree with branch at " + worktreePath).
			WithContext("branch", branchName)
	}

	// Validate isolation
	if err := s.ValidateWorktreeIsolation(ctx, worktreePath); err != nil {
		// Clean up on validation failure
		if cleanErr := s.DeleteWorktree(ctx, mainRepoPath, worktreePath); cleanErr != nil {
			s.logger.Warn("failed to cleanup worktree after validation failure",
				zap.String("worktree", worktreePath),
				zap.Error(cleanErr),
			)
		}
		if branchErr := s.branchSvc.DeleteBranchForce(ctx, mainRepoPath, branchName); branchErr != nil {
			s.logger.Warn("failed to cleanup branch after worktree validation failure",
				zap.String("branch", branchName),
				zap.Error(branchErr),
			)
		}
		return err
	}

	s.logger.Info("worktree created with new branch",
		zap.String("main_repo", mainRepoPath),
		zap.String("worktree", worktreePath),
		zap.String("branch", branchName),
		zap.String("start_point", startPoint),
	)

	return nil
}

// DeleteWorktree removes a worktree.
func (s *worktreeServiceImpl) DeleteWorktree(ctx context.Context, mainRepoPath, worktreePath string) error {
	// Check if worktree exists
	if !s.WorktreeExists(ctx, worktreePath) {
		return errors.New(errors.ErrGitWorktreeNotFound).
			WithDetail("worktree not found: " + worktreePath)
	}

	// Delete worktree
	_, err := s.executor.Exec(ctx, mainRepoPath, "worktree", "remove", "--force", worktreePath)
	if err != nil {
		return errors.Wrap(errors.ErrGitWorktreeDeleteFailed, err).
			WithDetail("failed to delete worktree: " + worktreePath)
	}

	s.logger.Info("worktree deleted",
		zap.String("main_repo", mainRepoPath),
		zap.String("worktree", worktreePath),
	)

	return nil
}

// DeleteWorktreeAndBranch removes worktree and its associated branch.
func (s *worktreeServiceImpl) DeleteWorktreeAndBranch(ctx context.Context, mainRepoPath, worktreePath, branchName string) error {
	// Delete worktree first
	if err := s.DeleteWorktree(ctx, mainRepoPath, worktreePath); err != nil {
		// If worktree doesn't exist, continue to try deleting branch
		if !errors.Is(err, errors.ErrGitWorktreeNotFound) {
			return err
		}
	}

	// Delete branch
	if err := s.branchSvc.DeleteBranchForce(ctx, mainRepoPath, branchName); err != nil {
		return err
	}

	s.logger.Info("worktree and branch deleted",
		zap.String("main_repo", mainRepoPath),
		zap.String("worktree", worktreePath),
		zap.String("branch", branchName),
	)

	return nil
}

// ListWorktrees returns all worktrees for a repository.
func (s *worktreeServiceImpl) ListWorktrees(ctx context.Context, mainRepoPath string) ([]WorktreeInfo, error) {
	// Get worktree list with porcelain format
	output, err := s.executor.ExecSimple(ctx, mainRepoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []WorktreeInfo
	var current WorktreeInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			// Empty line signals end of current worktree info
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 1 {
			continue
		}

		switch parts[0] {
		case "worktree":
			if len(parts) > 1 {
				current.Path = parts[1]
			}
		case "HEAD":
			if len(parts) > 1 {
				// Take short SHA
				if len(parts[1]) > 7 {
					current.HEAD = parts[1][:7]
				} else {
					current.HEAD = parts[1]
				}
			}
		case "branch":
			if len(parts) > 1 {
				// branch is refs/heads/branch-name
				branchRef := parts[1]
				if strings.HasPrefix(branchRef, "refs/heads/") {
					current.Branch = strings.TrimPrefix(branchRef, "refs/heads/")
				} else {
					current.Branch = branchRef
				}
			}
		case "locked":
			current.IsLocked = true
			if len(parts) > 1 {
				current.LockReason = parts[1]
			}
		case "prunable":
			current.IsPrunable = true
		}
	}

	// Don't forget the last worktree if no trailing empty line
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// WorktreeExists checks if a worktree exists at the given path.
func (s *worktreeServiceImpl) WorktreeExists(ctx context.Context, worktreePath string) bool {
	// Check if the path exists and is a Git repository (worktree)
	if _, err := os.Stat(worktreePath); err != nil {
		return false
	}

	// Verify it's a valid worktree by checking .git is a file (not directory)
	gitPath := filepath.Join(worktreePath, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}

	// Worktree has .git as a file, main repo has it as a directory
	return !info.IsDir()
}

// GetMainRepoPath returns the main repository path from a worktree.
func (s *worktreeServiceImpl) GetMainRepoPath(ctx context.Context, worktreePath string) (string, error) {
	// Use git rev-parse to get the common Git directory
	output, err := s.executor.ExecSimple(ctx, worktreePath, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", errors.Wrap(errors.ErrGitRepoNotFound, err).
			WithDetail("failed to get main repo path from worktree")
	}

	// The output is the path to .git directory of the main repo
	gitDir := output
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(worktreePath, gitDir)
	}

	// The main repo path is the parent of .git directory
	mainPath := filepath.Dir(gitDir)
	return mainPath, nil
}

// ValidateWorktreeIsolation verifies worktree isolation.
func (s *worktreeServiceImpl) ValidateWorktreeIsolation(ctx context.Context, worktreePath string) error {
	// 1. Check that worktree is a valid Git repository
	if !s.client.IsGitRepository(ctx, worktreePath) {
		return errors.New(errors.ErrGitWorktreeNotFound).
			WithDetail("worktree path is not a valid Git repository")
	}

	// 2. Check that .git is a file (worktree indicator)
	gitPath := filepath.Join(worktreePath, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return errors.New(errors.ErrGitWorktreeNotFound).
			WithDetail("worktree .git file not found")
	}
	if info.IsDir() {
		return errors.New(errors.ErrGitWorktreeNotFound).
			WithDetail("path is a main repository, not a worktree")
	}

	// 3. Check that worktree has independent HEAD
	branch, err := s.client.GetCurrentBranch(ctx, worktreePath)
	if err != nil {
		return errors.Wrap(errors.ErrGitOperationFailed, err).
			WithDetail("failed to get worktree branch")
	}

	s.logger.Debug("worktree isolation validated",
		zap.String("worktree", worktreePath),
		zap.String("branch", branch),
	)

	return nil
}

// PruneWorktrees removes stale worktree references.
func (s *worktreeServiceImpl) PruneWorktrees(ctx context.Context, mainRepoPath string) error {
	_, err := s.executor.Exec(ctx, mainRepoPath, "worktree", "prune")
	if err != nil {
		return errors.Wrap(errors.ErrGitOperationFailed, err).
			WithDetail("failed to prune worktrees")
	}

	s.logger.Info("worktrees pruned",
		zap.String("main_repo", mainRepoPath),
	)

	return nil
}

// GenerateWorktreePath generates a worktree path for a session.
func GenerateWorktreePath(basePath, repository string, issueNumber int) string {
	// Convert repository to directory-safe name
	repoSlug := strings.ReplaceAll(repository, "/", "-")
	return fmt.Sprintf("%s/%s-%d", basePath, repoSlug, issueNumber)
}
package git

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// CommitService provides Git commit and push operations.
type CommitService interface {
	// Add stages files for commit.
	// Supports both specific files and "." for all changes.
	Add(ctx context.Context, repoPath string, files []string) error

	// Commit creates a commit with the staged changes.
	Commit(ctx context.Context, repoPath string, message string, opts CommitOptions) error

	// Push pushes local commits to remote.
	Push(ctx context.Context, repoPath string, opts PushOptions) error

	// PushNewBranch pushes a new branch to remote with upstream tracking.
	PushNewBranch(ctx context.Context, repoPath, branchName string) error

	// GetStatus returns the current working tree status.
	GetStatus(ctx context.Context, repoPath string) (*GitStatus, error)

	// GetDiff returns the diff of staged/unstaged changes.
	GetDiff(ctx context.Context, repoPath string, staged bool) (string, error)

	// HasUncommittedChanges checks if there are uncommitted changes.
	HasUncommittedChanges(ctx context.Context, repoPath string) bool

	// GetLastCommit returns the last commit information.
	GetLastCommit(ctx context.Context, repoPath string) (*CommitInfo, error)

	// AmendLastCommit amends the last commit message or content.
	AmendLastCommit(ctx context.Context, repoPath string, message string) error
}

// CommitOptions defines options for creating a commit.
type CommitOptions struct {
	AuthorName  string // Author name (optional, uses config default)
	AuthorEmail string // Author email (optional, uses config default)
	AllowEmpty  bool   // Allow empty commits
	SignOff     bool   // Add Signed-off-by trailer
}

// PushOptions defines options for pushing.
type PushOptions struct {
	RemoteName  string // Remote name (default: "origin")
	BranchName  string // Branch to push (default: current)
	Force       bool   // Force push
	SetUpstream bool   // Set upstream tracking
}

// GitStatus represents working tree status.
type GitStatus struct {
	Staged    []FileStatus // Staged files
	Unstaged  []FileStatus // Unstaged (modified) files
	Untracked []string     // Untracked files
	IsClean   bool         // Working tree is clean
	Branch    string       // Current branch
	Ahead     int          // Commits ahead of upstream
	Behind    int          // Commits behind upstream
}

// FileStatus represents a file's Git status.
type FileStatus struct {
	Path   string // File path
	Status string // Status: added, modified, deleted, renamed
}

// CommitInfo contains commit metadata.
type CommitInfo struct {
	SHA       string    // Commit SHA
	Message   string    // Commit message
	Author    string    // Author name
	Email     string    // Author email
	Timestamp time.Time // Commit timestamp
}

// CommitServiceParams contains dependencies for creating a CommitService.
type CommitServiceParams struct {
	Executor *CommandExecutor
	Client   GitClient
	Logger   *zap.Logger
}

// commitServiceImpl implements CommitService.
type commitServiceImpl struct {
	executor *CommandExecutor
	client   GitClient
	logger   *zap.Logger
}

// NewCommitService creates a new CommitService.
func NewCommitService(p CommitServiceParams) CommitService {
	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &commitServiceImpl{
		executor: p.Executor,
		client:   p.Client,
		logger:   logger.Named("commit-service"),
	}
}

// Add stages files for commit.
func (s *commitServiceImpl) Add(ctx context.Context, repoPath string, files []string) error {
	args := []string{"add"}
	args = append(args, files...)

	_, err := s.executor.Exec(ctx, repoPath, args...)
	if err != nil {
		return errors.Wrap(errors.ErrGitAddFailed, err).
			WithDetail("failed to stage files")
	}

	s.logger.Debug("files staged",
		zap.String("repo", repoPath),
		zap.Strings("files", files),
	)

	return nil
}

// Commit creates a commit with the staged changes.
func (s *commitServiceImpl) Commit(ctx context.Context, repoPath string, message string, opts CommitOptions) error {
	args := []string{"commit", "-m", message}

	if opts.AllowEmpty {
		args = append(args, "--allow-empty")
	}

	if opts.SignOff {
		args = append(args, "--signoff")
	}

	if opts.AuthorName != "" && opts.AuthorEmail != "" {
		author := fmt.Sprintf("%s <%s>", opts.AuthorName, opts.AuthorEmail)
		args = append(args, "--author", author)
	}

	_, err := s.executor.Exec(ctx, repoPath, args...)
	if err != nil {
		return errors.Wrap(errors.ErrGitCommitFailed, err).
			WithDetail("failed to create commit")
	}

	s.logger.Info("commit created",
		zap.String("repo", repoPath),
		zap.String("message", message),
	)

	return nil
}

// Push pushes local commits to remote.
func (s *commitServiceImpl) Push(ctx context.Context, repoPath string, opts PushOptions) error {
	args := []string{"push"}

	if opts.RemoteName == "" {
		opts.RemoteName = "origin"
	}
	args = append(args, opts.RemoteName)

	if opts.BranchName != "" {
		args = append(args, opts.BranchName)
	}

	if opts.Force {
		args = append(args, "--force")
	}

	if opts.SetUpstream {
		args = append(args, "--set-upstream")
	}

	_, err := s.executor.Exec(ctx, repoPath, args...)
	if err != nil {
		return errors.Wrap(errors.ErrGitPushFailed, err).
			WithDetail("failed to push to remote")
	}

	s.logger.Info("pushed to remote",
		zap.String("repo", repoPath),
		zap.String("remote", opts.RemoteName),
		zap.String("branch", opts.BranchName),
	)

	return nil
}

// PushNewBranch pushes a new branch to remote with upstream tracking.
func (s *commitServiceImpl) PushNewBranch(ctx context.Context, repoPath, branchName string) error {
	// Check for uncommitted changes
	if s.HasUncommittedChanges(ctx, repoPath) {
		// Not blocking, just log warning
		s.logger.Warn("pushing with uncommitted changes",
			zap.String("repo", repoPath),
			zap.String("branch", branchName),
		)
	}

	// Push with upstream tracking
	opts := PushOptions{
		RemoteName:  "origin",
		BranchName:  branchName,
		SetUpstream: true,
	}

	return s.Push(ctx, repoPath, opts)
}

// GetStatus returns the current working tree status.
func (s *commitServiceImpl) GetStatus(ctx context.Context, repoPath string) (*GitStatus, error) {
	// Get porcelain status
	output, err := s.executor.ExecSimple(ctx, repoPath, "status", "--porcelain=v1", "--branch")
	if err != nil {
		return nil, err
	}

	status := &GitStatus{
		IsClean: true,
	}

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// First line with --branch contains branch info
		if i == 0 && strings.HasPrefix(line, "## ") {
			branchInfo := strings.TrimPrefix(line, "## ")
			// Parse branch info: "branch_name...origin/branch_name [ahead X, behind Y]"
			parts := strings.SplitN(branchInfo, "...", 2)
			if len(parts) > 0 {
				status.Branch = strings.TrimSpace(parts[0])
			}
			// Parse ahead/behind
			if strings.Contains(branchInfo, "ahead ") {
				fmt.Sscanf(branchInfo, "%*s ahead %d", &status.Ahead)
			}
			if strings.Contains(branchInfo, "behind ") {
				fmt.Sscanf(branchInfo, "%*s behind %d", &status.Behind)
			}
			continue
		}

		// Parse file status
		if len(line) < 3 {
			continue
		}

		x := line[0] // Index status
		y := line[1] // Worktree status
		path := strings.TrimSpace(line[2:])

		// Staged files (X is not space or ?)
		if x != ' ' && x != '?' {
			status.Staged = append(status.Staged, FileStatus{
				Path:   path,
				Status: statusCodeToString(x),
			})
			status.IsClean = false
		}

		// Unstaged/untracked files
		if y != ' ' || x == '?' {
			if x == '?' && y == '?' {
				// Untracked file
				status.Untracked = append(status.Untracked, path)
			} else if y != ' ' {
				// Modified file
				status.Unstaged = append(status.Unstaged, FileStatus{
					Path:   path,
					Status: statusCodeToString(y),
				})
			}
			status.IsClean = false
		}
	}

	return status, nil
}

// GetDiff returns the diff of staged/unstaged changes.
func (s *commitServiceImpl) GetDiff(ctx context.Context, repoPath string, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--staged")
	}

	output, err := s.executor.ExecSimple(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}

	return output, nil
}

// HasUncommittedChanges checks if there are uncommitted changes.
func (s *commitServiceImpl) HasUncommittedChanges(ctx context.Context, repoPath string) bool {
	output, err := s.executor.ExecSimple(ctx, repoPath, "status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(output) != ""
}

// GetLastCommit returns the last commit information.
func (s *commitServiceImpl) GetLastCommit(ctx context.Context, repoPath string) (*CommitInfo, error) {
	// Get commit info with format: SHA|author|email|timestamp|message
	output, err := s.executor.ExecSimple(ctx, repoPath,
		"log", "-1", "--format=%H|%an|%ae|%aI|%s",
	)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(output, "|", 5)
	if len(parts) < 5 {
		return nil, errors.New(errors.ErrGitOperationFailed).
			WithDetail("failed to parse commit info")
	}

	timestamp, err := time.Parse(time.RFC3339, parts[3])
	if err != nil {
		s.logger.Warn("failed to parse commit timestamp, using current time",
			zap.String("timestamp", parts[3]),
			zap.Error(err),
		)
		timestamp = time.Now()
	}

	return &CommitInfo{
		SHA:       parts[0],
		Author:    parts[1],
		Email:     parts[2],
		Timestamp: timestamp,
		Message:   parts[4],
	}, nil
}

// AmendLastCommit amends the last commit message or content.
func (s *commitServiceImpl) AmendLastCommit(ctx context.Context, repoPath string, message string) error {
	_, err := s.executor.Exec(ctx, repoPath, "commit", "--amend", "-m", message)
	if err != nil {
		return errors.Wrap(errors.ErrGitCommitFailed, err).
			WithDetail("failed to amend commit")
	}

	s.logger.Info("commit amended",
		zap.String("repo", repoPath),
	)

	return nil
}

// statusCodeToString converts Git status code to human-readable string.
func statusCodeToString(code byte) string {
	switch code {
	case 'M':
		return "modified"
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case 'U':
		return "unmerged"
	default:
		return "unknown"
	}
}
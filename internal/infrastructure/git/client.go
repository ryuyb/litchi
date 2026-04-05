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

// GitClient provides core Git repository operations.
type GitClient interface {
	// OpenRepository opens a Git repository at the given path.
	// Returns error if path is not a valid Git repository.
	OpenRepository(ctx context.Context, path string) (*Repository, error)

	// CloneRepository clones a remote repository to local path.
	CloneRepository(ctx context.Context, remoteURL, localPath string, opts CloneOptions) error

	// IsGitRepository checks if the path is a valid Git repository.
	IsGitRepository(ctx context.Context, path string) bool

	// GetRemoteURL returns the remote URL for the given remote name.
	GetRemoteURL(ctx context.Context, repoPath, remoteName string) (string, error)

	// GetCurrentBranch returns the current branch name.
	GetCurrentBranch(ctx context.Context, repoPath string) (string, error)

	// Fetch updates remote refs from the remote repository.
	Fetch(ctx context.Context, repoPath string, remoteName string) error

	// Pull fetches and merges the remote branch.
	Pull(ctx context.Context, repoPath string) error

	// Init initializes a new Git repository.
	Init(ctx context.Context, path string) error

	// SetConfig sets a Git configuration value.
	SetConfig(ctx context.Context, repoPath, key, value string) error

	// GetConfig gets a Git configuration value.
	GetConfig(ctx context.Context, repoPath, key string) (string, error)
}

// Repository represents an opened Git repository.
type Repository struct {
	Path         string // Local repository path
	WorktreePath string // Worktree path (if different from main repo)
	RemoteURL    string // Remote URL (owner/repo format)
	IsWorktree   bool   // Whether this is a worktree
}

// CloneOptions defines options for cloning a repository.
type CloneOptions struct {
	Branch       string // Branch to clone (optional, default: default branch)
	Depth        int    // Shallow clone depth (0 = full clone)
	SingleBranch bool   // Clone only the specified branch
}

// GitClientParams contains dependencies for creating a GitClient.
type GitClientParams struct {
	Executor *CommandExecutor
	Logger   *zap.Logger
}

// gitClientImpl implements GitClient using command-line execution.
type gitClientImpl struct {
	executor *CommandExecutor
	logger   *zap.Logger
}

// NewGitClient creates a new GitClient.
func NewGitClient(p GitClientParams) GitClient {
	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &gitClientImpl{
		executor: p.Executor,
		logger:   logger.Named("git-client"),
	}
}

// OpenRepository opens a Git repository at the given path.
func (c *gitClientImpl) OpenRepository(ctx context.Context, path string) (*Repository, error) {
	// Check if path is a Git repository
	if !c.IsGitRepository(ctx, path) {
		return nil, errors.New(errors.ErrGitRepoNotFound).
			WithDetail("path is not a valid Git repository: " + path)
	}

	repo := &Repository{
		Path: path,
	}

	// Check if this is a worktree
	if c.isWorktree(path) {
		repo.IsWorktree = true
		mainPath, err := c.GetMainRepoPath(ctx, path)
		if err == nil {
			repo.WorktreePath = path
			repo.Path = mainPath
		}
	}

	// Get remote URL
	remoteURL, err := c.GetRemoteURL(ctx, path, "origin")
	if err == nil {
		repo.RemoteURL = remoteURL
	}

	return repo, nil
}

// CloneRepository clones a remote repository to local path.
func (c *gitClientImpl) CloneRepository(ctx context.Context, remoteURL, localPath string, opts CloneOptions) error {
	args := []string{"clone"}

	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}

	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}

	if opts.SingleBranch {
		args = append(args, "--single-branch")
	}

	args = append(args, remoteURL, localPath)

	_, err := c.executor.Exec(ctx, "", args...)
	if err != nil {
		return errors.Wrap(errors.ErrGitCloneFailed, err).
			WithDetail("failed to clone repository: " + remoteURL)
	}

	c.logger.Info("repository cloned",
		zap.String("remote", remoteURL),
		zap.String("path", localPath),
	)

	return nil
}

// IsGitRepository checks if the path is a valid Git repository.
func (c *gitClientImpl) IsGitRepository(ctx context.Context, path string) bool {
	// Check for .git directory or file (worktree uses a file)
	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); err == nil {
		return true
	}

	// Also try git rev-parse to confirm it's a valid repo
	_, err := c.executor.ExecSimple(ctx, path, "rev-parse", "--git-dir")
	return err == nil
}

// GetRemoteURL returns the remote URL for the given remote name.
func (c *gitClientImpl) GetRemoteURL(ctx context.Context, repoPath, remoteName string) (string, error) {
	output, err := c.executor.ExecSimple(ctx, repoPath, "remote", "get-url", remoteName)
	if err != nil {
		return "", err
	}

	// Convert SSH URL to owner/repo format if needed
	url := output
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.TrimPrefix(url, "git@github.com:")
		url = strings.TrimSuffix(url, ".git")
	} else if strings.HasPrefix(url, "https://github.com/") {
		url = strings.TrimPrefix(url, "https://github.com/")
		url = strings.TrimSuffix(url, ".git")
	}

	return url, nil
}

// GetCurrentBranch returns the current branch name.
func (c *gitClientImpl) GetCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	output, err := c.executor.ExecSimple(ctx, repoPath, "branch", "--show-current")
	if err != nil {
		return "", err
	}

	if output == "" {
		// Detached HEAD - get commit SHA instead
		output, err = c.executor.ExecSimple(ctx, repoPath, "rev-parse", "--short", "HEAD")
		if err != nil {
			return "", err
		}
	}

	return output, nil
}

// Fetch updates remote refs from the remote repository.
func (c *gitClientImpl) Fetch(ctx context.Context, repoPath string, remoteName string) error {
	args := []string{"fetch", remoteName}
	_, err := c.executor.Exec(ctx, repoPath, args...)
	if err != nil {
		return errors.Wrap(errors.ErrGitFetchFailed, err).
			WithDetail("failed to fetch from remote: " + remoteName)
	}
	return nil
}

// Pull fetches and merges the remote branch.
func (c *gitClientImpl) Pull(ctx context.Context, repoPath string) error {
	_, err := c.executor.Exec(ctx, repoPath, "pull")
	if err != nil {
		return errors.Wrap(errors.ErrGitOperationFailed, err).
			WithDetail("failed to pull from remote")
	}
	return nil
}

// Init initializes a new Git repository.
func (c *gitClientImpl) Init(ctx context.Context, path string) error {
	_, err := c.executor.Exec(ctx, path, "init")
	if err != nil {
		return errors.Wrap(errors.ErrGitOperationFailed, err).
			WithDetail("failed to initialize repository at: " + path)
	}
	return nil
}

// SetConfig sets a Git configuration value.
func (c *gitClientImpl) SetConfig(ctx context.Context, repoPath, key, value string) error {
	_, err := c.executor.Exec(ctx, repoPath, "config", key, value)
	return err
}

// GetConfig gets a Git configuration value.
func (c *gitClientImpl) GetConfig(ctx context.Context, repoPath, key string) (string, error) {
	return c.executor.ExecSimple(ctx, repoPath, "config", "--get", key)
}

// isWorktree checks if the path is a Git worktree.
func (c *gitClientImpl) isWorktree(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	// Worktree has .git as a file, not a directory
	return !info.IsDir()
}

// GetMainRepoPath returns the main repository path from a worktree.
func (c *gitClientImpl) GetMainRepoPath(ctx context.Context, worktreePath string) (string, error) {
	// Use git rev-parse to get the common Git directory
	output, err := c.executor.ExecSimple(ctx, worktreePath, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}

	// The output is relative to the worktree's .git directory
	// For a worktree, it points to the main repo's .git directory
	gitDir := output
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(worktreePath, gitDir)
	}

	// The main repo path is the parent of .git directory
	mainPath := filepath.Dir(gitDir)
	return mainPath, nil
}
package git

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// BranchService provides Git branch management operations.
type BranchService interface {
	// CreateBranch creates a new branch from the current HEAD.
	// Returns error if branch already exists.
	CreateBranch(ctx context.Context, repoPath, branchName string) error

	// CreateBranchFromRef creates a new branch from a specific reference.
	CreateBranchFromRef(ctx context.Context, repoPath, branchName, startPoint string) error

	// SwitchBranch switches to an existing branch.
	SwitchBranch(ctx context.Context, repoPath, branchName string) error

	// DeleteBranch deletes a local branch.
	// Returns error if branch has unmerged changes or is current branch.
	DeleteBranch(ctx context.Context, repoPath, branchName string) error

	// DeleteBranchForce deletes a branch even with unmerged changes.
	DeleteBranchForce(ctx context.Context, repoPath, branchName string) error

	// ListBranches returns all local branches.
	ListBranches(ctx context.Context, repoPath string) ([]BranchInfo, error)

	// BranchExists checks if a branch exists.
	BranchExists(ctx context.Context, repoPath, branchName string) bool

	// ValidateBranchName validates branch naming conventions.
	// Returns error if branch name doesn't follow Git rules.
	ValidateBranchName(branchName string) error

	// GenerateBranchName generates a valid branch name from issue info.
	// Format: issue-{number}-{slug} (e.g., "issue-123-fix-login-bug")
	GenerateBranchName(issueNumber int, title string) string
}

// BranchInfo contains branch metadata.
type BranchInfo struct {
	Name      string // Branch name
	IsCurrent bool   // Is this the current branch
	IsDefault bool   // Is this the default/main branch
	Upstream  string // Upstream branch name (if tracking)
	LastCommit string // Last commit SHA (short)
}

// BranchServiceParams contains dependencies for creating a BranchService.
type BranchServiceParams struct {
	Executor *CommandExecutor
	Client   GitClient
	Logger   *zap.Logger
}

// branchServiceImpl implements BranchService.
type branchServiceImpl struct {
	executor *CommandExecutor
	client   GitClient
	logger   *zap.Logger
}

// NewBranchService creates a new BranchService.
func NewBranchService(p BranchServiceParams) BranchService {
	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &branchServiceImpl{
		executor: p.Executor,
		client:   p.Client,
		logger:   logger.Named("branch-service"),
	}
}

// CreateBranch creates a new branch from the current HEAD.
func (s *branchServiceImpl) CreateBranch(ctx context.Context, repoPath, branchName string) error {
	// Validate branch name
	if err := s.ValidateBranchName(branchName); err != nil {
		return err
	}

	// Check if branch already exists
	if s.BranchExists(ctx, repoPath, branchName) {
		return errors.New(errors.ErrGitBranchExists).
			WithDetail("branch already exists: " + branchName)
	}

	// Create branch
	_, err := s.executor.Exec(ctx, repoPath, "branch", branchName)
	if err != nil {
		return errors.Wrap(errors.ErrGitBranchCreateFailed, err).
			WithDetail("failed to create branch: " + branchName)
	}

	s.logger.Info("branch created",
		zap.String("repo", repoPath),
		zap.String("branch", branchName),
	)

	return nil
}

// CreateBranchFromRef creates a new branch from a specific reference.
func (s *branchServiceImpl) CreateBranchFromRef(ctx context.Context, repoPath, branchName, startPoint string) error {
	// Validate branch name
	if err := s.ValidateBranchName(branchName); err != nil {
		return err
	}

	// Check if branch already exists
	if s.BranchExists(ctx, repoPath, branchName) {
		return errors.New(errors.ErrGitBranchExists).
			WithDetail("branch already exists: " + branchName)
	}

	// Create branch from reference
	_, err := s.executor.Exec(ctx, repoPath, "branch", branchName, startPoint)
	if err != nil {
		return errors.Wrap(errors.ErrGitBranchCreateFailed, err).
			WithDetail("failed to create branch " + branchName + " from " + startPoint)
	}

	s.logger.Info("branch created from ref",
		zap.String("repo", repoPath),
		zap.String("branch", branchName),
		zap.String("start_point", startPoint),
	)

	return nil
}

// SwitchBranch switches to an existing branch.
func (s *branchServiceImpl) SwitchBranch(ctx context.Context, repoPath, branchName string) error {
	// Check if branch exists
	if !s.BranchExists(ctx, repoPath, branchName) {
		return errors.New(errors.ErrGitBranchNotFound).
			WithDetail("branch not found: " + branchName)
	}

	// Switch branch
	_, err := s.executor.Exec(ctx, repoPath, "checkout", branchName)
	if err != nil {
		return errors.Wrap(errors.ErrGitBranchSwitchFailed, err).
			WithDetail("failed to switch to branch: " + branchName)
	}

	s.logger.Info("switched to branch",
		zap.String("repo", repoPath),
		zap.String("branch", branchName),
	)

	return nil
}

// DeleteBranch deletes a local branch.
func (s *branchServiceImpl) DeleteBranch(ctx context.Context, repoPath, branchName string) error {
	// Check if branch exists
	if !s.BranchExists(ctx, repoPath, branchName) {
		return errors.New(errors.ErrGitBranchNotFound).
			WithDetail("branch not found: " + branchName)
	}

	// Delete branch
	_, err := s.executor.Exec(ctx, repoPath, "branch", "-d", branchName)
	if err != nil {
		return errors.Wrap(errors.ErrGitBranchDeleteFailed, err).
			WithDetail("failed to delete branch: " + branchName)
	}

	s.logger.Info("branch deleted",
		zap.String("repo", repoPath),
		zap.String("branch", branchName),
	)

	return nil
}

// DeleteBranchForce deletes a branch even with unmerged changes.
func (s *branchServiceImpl) DeleteBranchForce(ctx context.Context, repoPath, branchName string) error {
	// Check if branch exists
	if !s.BranchExists(ctx, repoPath, branchName) {
		return errors.New(errors.ErrGitBranchNotFound).
			WithDetail("branch not found: " + branchName)
	}

	// Force delete branch
	_, err := s.executor.Exec(ctx, repoPath, "branch", "-D", branchName)
	if err != nil {
		return errors.Wrap(errors.ErrGitBranchDeleteFailed, err).
			WithDetail("failed to force delete branch: " + branchName)
	}

	s.logger.Info("branch force deleted",
		zap.String("repo", repoPath),
		zap.String("branch", branchName),
	)

	return nil
}

// ListBranches returns all local branches.
func (s *branchServiceImpl) ListBranches(ctx context.Context, repoPath string) ([]BranchInfo, error) {
	// Get branch list with format: %(refname:short)|%(HEAD)|%(upstream:short)|%(objectname:short)
	output, err := s.executor.ExecSimple(ctx, repoPath,
		"for-each-ref",
		"--sort=-committerdate",
		"--format=%(refname:short)|%(HEAD)|%(upstream:short)|%(objectname:short)",
		"refs/heads/",
	)
	if err != nil {
		return nil, err
	}

	var branches []BranchInfo
	lines := strings.Split(output, "\n")

	// Get current branch to mark it
	currentBranch, _ := s.client.GetCurrentBranch(ctx, repoPath)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		name := parts[0]
		isCurrent := parts[1] == "*" || name == currentBranch
		upstream := parts[2]
		lastCommit := parts[3]

		// Check if this is the default branch (main or master)
		isDefault := name == "main" || name == "master"

		branches = append(branches, BranchInfo{
			Name:       name,
			IsCurrent:  isCurrent,
			IsDefault:  isDefault,
			Upstream:   upstream,
			LastCommit: lastCommit,
		})
	}

	return branches, nil
}

// BranchExists checks if a branch exists.
func (s *branchServiceImpl) BranchExists(ctx context.Context, repoPath, branchName string) bool {
	// Use rev-parse to check if branch exists
	_, err := s.executor.ExecSimple(ctx, repoPath, "rev-parse", "--verify", "refs/heads/"+branchName)
	return err == nil
}

// ValidateBranchName validates branch naming conventions.
func (s *branchServiceImpl) ValidateBranchName(branchName string) error {
	if branchName == "" {
		return errors.New(errors.ErrGitBranchNameInvalid).
			WithDetail("branch name cannot be empty")
	}

	// Git branch naming rules:
	// - Cannot start with '.' or '-'
	// - Cannot contain '..', '~', '^', ':', '?', '*', '[', '\\'
	// - Cannot contain control characters
	// - Cannot end with '/'
	// - Cannot end with '.lock'
	invalidPatterns := []string{"..", "~", "^", ":", "?", "*", "[", "\\", "//", "@{"}
	for _, pattern := range invalidPatterns {
		if strings.Contains(branchName, pattern) {
			return errors.New(errors.ErrGitBranchNameInvalid).
				WithDetail("branch name contains invalid pattern: " + pattern)
		}
	}

	if strings.HasPrefix(branchName, ".") || strings.HasPrefix(branchName, "-") {
		return errors.New(errors.ErrGitBranchNameInvalid).
			WithDetail("branch name cannot start with '.' or '-'")
	}

	if strings.HasSuffix(branchName, "/") || strings.HasSuffix(branchName, ".lock") {
		return errors.New(errors.ErrGitBranchNameInvalid).
			WithDetail("branch name cannot end with '/' or '.lock'")
	}

	// Check for consecutive slashes
	if strings.Contains(branchName, "//") {
		return errors.New(errors.ErrGitBranchNameInvalid).
			WithDetail("branch name cannot contain consecutive slashes")
	}

	// Check for control characters
	for _, r := range branchName {
		if r < 32 || r == 127 {
			return errors.New(errors.ErrGitBranchNameInvalid).
				WithDetail("branch name contains control characters")
		}
	}

	return nil
}

// GenerateBranchName generates a valid branch name from issue info.
func (s *branchServiceImpl) GenerateBranchName(issueNumber int, title string) string {
	// Convert title to slug
	slug := titleToSlug(title)

	// Limit slug length
	if len(slug) > 50 {
		slug = slug[:50]
		// Don't end with hyphen
		for strings.HasSuffix(slug, "-") {
			slug = slug[:len(slug)-1]
		}
	}

	return fmt.Sprintf("issue-%d-%s", issueNumber, slug)
}

// titleToSlug converts a title to a URL-friendly slug.
func titleToSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()

	// Remove consecutive hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	if slug == "" {
		slug = "feature"
	}

	return slug
}

// ParseBranchName extracts issue number from branch name.
// Returns 0 if no issue number is found.
func ParseBranchName(branchName string) int {
	// Match pattern: issue-{number}-{slug}
	re := regexp.MustCompile(`^issue-(\d+)-`)
	matches := re.FindStringSubmatch(branchName)
	if len(matches) > 1 {
		num, _ := strconv.Atoi(matches[1])
		return num
	}
	return 0
}
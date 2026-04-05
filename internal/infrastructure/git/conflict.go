package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// ConflictDetector provides merge conflict detection and analysis.
type ConflictDetector interface {
	// DetectConflicts checks for potential merge conflicts.
	// Returns list of files that would conflict.
	DetectConflicts(ctx context.Context, repoPath, sourceBranch, targetBranch string) ([]ConflictInfo, error)

	// HasConflicts checks if there are active merge conflicts.
	HasConflicts(ctx context.Context, repoPath string) bool

	// GetConflictedFiles returns files with active merge conflicts.
	GetConflictedFiles(ctx context.Context, repoPath string) ([]string, error)

	// AbortMerge aborts an in-progress merge.
	AbortMerge(ctx context.Context, repoPath string) error

	// GetMergeStatus returns current merge/rebase status.
	GetMergeStatus(ctx context.Context, repoPath string) (*MergeStatus, error)
}

// ConflictInfo contains conflict metadata.
type ConflictInfo struct {
	FilePath     string // File with conflict
	ConflictType string // Type: modify/modify, delete/modify, etc.
	OurChanges   string // Our side changes summary (optional)
	TheirChanges string // Their side changes summary (optional)
}

// MergeStatus represents current merge/rebase status.
type MergeStatus struct {
	InMerge        bool     // Merge in progress
	InRebase       bool     // Rebase in progress
	ConflictedFiles []string // Files with conflicts
	CurrentBranch  string   // Current branch
	MergeBranch    string   // Branch being merged
}

// ConflictDetectorParams contains dependencies for creating a ConflictDetector.
type ConflictDetectorParams struct {
	Executor *CommandExecutor
	Client   GitClient
	Logger   *zap.Logger
}

// conflictDetectorImpl implements ConflictDetector.
type conflictDetectorImpl struct {
	executor *CommandExecutor
	client   GitClient
	logger   *zap.Logger
}

// NewConflictDetector creates a new ConflictDetector.
func NewConflictDetector(p ConflictDetectorParams) ConflictDetector {
	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &conflictDetectorImpl{
		executor: p.Executor,
		client:   p.Client,
		logger:   logger.Named("conflict-detector"),
	}
}

// DetectConflicts checks for potential merge conflicts.
func (d *conflictDetectorImpl) DetectConflicts(ctx context.Context, repoPath, sourceBranch, targetBranch string) ([]ConflictInfo, error) {
	// Fetch latest from remote
	if err := d.client.Fetch(ctx, repoPath, "origin"); err != nil {
		d.logger.Warn("failed to fetch, using local state for conflict detection", zap.Error(err))
	}

	// Try merge with --no-commit to detect conflicts
	_, err := d.executor.Exec(ctx, repoPath, "merge", "origin/"+targetBranch, "--no-commit", "--no-ff")
	if err != nil {
		// Check if it's a conflict error
		if d.HasConflicts(ctx, repoPath) {
			conflictedFiles, err := d.GetConflictedFiles(ctx, repoPath)
			if err != nil {
				d.logger.Warn("failed to get conflicted files", zap.Error(err))
			}

			conflicts := make([]ConflictInfo, len(conflictedFiles))
			for i, file := range conflictedFiles {
				conflicts[i] = ConflictInfo{
					FilePath:     file,
					ConflictType: "modify/modify", // Simplified
				}
			}

			// Abort the test merge
			if abortErr := d.AbortMerge(ctx, repoPath); abortErr != nil {
				d.logger.Warn("failed to abort merge after conflict detection", zap.Error(abortErr))
			}

			d.logger.Warn("merge conflicts detected",
				zap.String("source", sourceBranch),
				zap.String("target", targetBranch),
				zap.Int("conflicts", len(conflicts)),
			)

			return conflicts, nil
		}

		// Not a conflict error, abort and return
		if abortErr := d.AbortMerge(ctx, repoPath); abortErr != nil {
			d.logger.Warn("failed to abort merge", zap.Error(abortErr))
		}
		return nil, errors.Wrap(errors.ErrGitOperationFailed, err).
			WithDetail("merge test failed")
	}

	// No conflicts, abort the test merge
	if abortErr := d.AbortMerge(ctx, repoPath); abortErr != nil {
		d.logger.Warn("failed to abort merge after clean test", zap.Error(abortErr))
	}

	d.logger.Info("no merge conflicts detected",
		zap.String("source", sourceBranch),
		zap.String("target", targetBranch),
	)

	return nil, nil
}

// HasConflicts checks if there are active merge conflicts.
func (d *conflictDetectorImpl) HasConflicts(ctx context.Context, repoPath string) bool {
	// Check for unmerged files
	output, err := d.executor.ExecSimple(ctx, repoPath, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return false
	}
	return strings.TrimSpace(output) != ""
}

// GetConflictedFiles returns files with active merge conflicts.
func (d *conflictDetectorImpl) GetConflictedFiles(ctx context.Context, repoPath string) ([]string, error) {
	output, err := d.executor.ExecSimple(ctx, repoPath, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return nil, nil
	}

	files := strings.Split(strings.TrimSpace(output), "\n")
	result := make([]string, 0, len(files))
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f != "" {
			result = append(result, f)
		}
	}

	return result, nil
}

// AbortMerge aborts an in-progress merge.
func (d *conflictDetectorImpl) AbortMerge(ctx context.Context, repoPath string) error {
	_, err := d.executor.Exec(ctx, repoPath, "merge", "--abort")
	if err != nil {
		// Try rebase abort if merge abort fails
		_, err2 := d.executor.Exec(ctx, repoPath, "rebase", "--abort")
		if err2 != nil {
			return errors.Wrap(errors.ErrGitOperationFailed, err).
				WithDetail("failed to abort merge/rebase")
		}
	}

	d.logger.Debug("merge/rebase aborted",
		zap.String("repo", repoPath),
	)

	return nil
}

// GetMergeStatus returns current merge/rebase status.
func (d *conflictDetectorImpl) GetMergeStatus(ctx context.Context, repoPath string) (*MergeStatus, error) {
	status := &MergeStatus{}

	// Check current branch
	branch, err := d.client.GetCurrentBranch(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	status.CurrentBranch = branch

	// Check for merge state
	// Git stores merge state in .git/MERGE_HEAD
	_, err = d.executor.ExecSimple(ctx, repoPath, "rev-parse", "--verify", "MERGE_HEAD")
	if err == nil {
		status.InMerge = true
		// Get merge branch from MERGE_MSG file
		mergeMsgPath := filepath.Join(repoPath, ".git", "MERGE_MSG")
		if data, readErr := os.ReadFile(mergeMsgPath); readErr == nil {
			output := string(data)
			if strings.Contains(output, "Merge branch '") {
				// Extract branch name from merge message
				parts := strings.Split(output, "'")
				if len(parts) > 1 {
					status.MergeBranch = parts[1]
				}
			}
		}
	}

	// Check for rebase state
	_, err = d.executor.ExecSimple(ctx, repoPath, "rev-parse", "--verify", "REBASE_HEAD")
	if err == nil {
		status.InRebase = true
	}

	// Get conflicted files
	conflictedFiles, err := d.GetConflictedFiles(ctx, repoPath)
	if err == nil {
		status.ConflictedFiles = conflictedFiles
	}

	return status, nil
}

// ResolveConflict marks a file as resolved.
func (d *conflictDetectorImpl) ResolveConflict(ctx context.Context, repoPath, filePath string) error {
	_, err := d.executor.Exec(ctx, repoPath, "add", filePath)
	if err != nil {
		return errors.Wrap(errors.ErrGitOperationFailed, err).
			WithDetail("failed to mark file as resolved: " + filePath)
	}

	d.logger.Info("conflict resolved",
		zap.String("repo", repoPath),
		zap.String("file", filePath),
	)

	return nil
}
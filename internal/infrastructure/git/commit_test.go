package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupCommitTestRepo creates a repository for commit testing.
func setupCommitTestRepo(t *testing.T) (string, *CommandExecutor, *commitServiceImpl) {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "commit-test-repo")

	// Initialize Git repository
	err := os.MkdirAll(repoPath, 0755)
	require.NoError(t, err)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	err = cmd.Run()
	require.NoError(t, err)

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = repoPath
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoPath
	err = cmd.Run()
	require.NoError(t, err)

	// Create initial commit
	testFile := filepath.Join(repoPath, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository\n"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoPath
	err = cmd.Run()
	require.NoError(t, err)

	// Create executor and services
	executor := NewCommandExecutor(CommandExecutorParams{
		GitBinary: "git",
		Logger:    zap.NewNop(),
	})

	client := NewGitClient(GitClientParams{
		Executor: executor,
		Logger:   zap.NewNop(),
	})

	commitSvc := NewCommitService(CommitServiceParams{
		Executor: executor,
		Client:   client,
		Logger:   zap.NewNop(),
	}).(*commitServiceImpl)

	return repoPath, executor, commitSvc
}

func TestCommitService_Add(t *testing.T) {
	repoPath, _, commitSvc := setupCommitTestRepo(t)
	ctx := context.Background()

	// Create a new file
	testFile := filepath.Join(repoPath, "new-file.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Add the file
	err = commitSvc.Add(ctx, repoPath, []string{"new-file.txt"})
	assert.NoError(t, err)

	// Verify file is staged
	status, err := commitSvc.GetStatus(ctx, repoPath)
	require.NoError(t, err)
	assert.Len(t, status.Staged, 1)
	assert.Equal(t, "new-file.txt", status.Staged[0].Path)
}

func TestCommitService_Commit(t *testing.T) {
	repoPath, _, commitSvc := setupCommitTestRepo(t)
	ctx := context.Background()

	// Create and stage a file
	testFile := filepath.Join(repoPath, "commit-test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	err = commitSvc.Add(ctx, repoPath, []string{"commit-test.txt"})
	require.NoError(t, err)

	// Commit
	err = commitSvc.Commit(ctx, repoPath, "Test commit", CommitOptions{})
	assert.NoError(t, err)

	// Verify commit was created
	lastCommit, err := commitSvc.GetLastCommit(ctx, repoPath)
	require.NoError(t, err)
	assert.Equal(t, "Test commit", lastCommit.Message)
}

func TestCommitService_CommitWithSignOff(t *testing.T) {
	repoPath, _, commitSvc := setupCommitTestRepo(t)
	ctx := context.Background()

	// Create and stage a file
	testFile := filepath.Join(repoPath, "signoff-test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	err = commitSvc.Add(ctx, repoPath, []string{"signoff-test.txt"})
	require.NoError(t, err)

	// Commit with sign-off
	err = commitSvc.Commit(ctx, repoPath, "Test commit with sign-off", CommitOptions{SignOff: true})
	assert.NoError(t, err)

	// Verify commit was created (we can't easily verify sign-off without parsing the commit body)
	lastCommit, err := commitSvc.GetLastCommit(ctx, repoPath)
	require.NoError(t, err)
	assert.Equal(t, "Test commit with sign-off", lastCommit.Message)
}

func TestCommitService_GetStatus(t *testing.T) {
	repoPath, _, commitSvc := setupCommitTestRepo(t)
	ctx := context.Background()

	// Initially clean
	status, err := commitSvc.GetStatus(ctx, repoPath)
	require.NoError(t, err)
	assert.True(t, status.IsClean)

	// Create a new file
	testFile := filepath.Join(repoPath, "status-test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Check status with untracked file
	status, err = commitSvc.GetStatus(ctx, repoPath)
	require.NoError(t, err)
	assert.False(t, status.IsClean)
	assert.Len(t, status.Untracked, 1)

	// Stage the file
	err = commitSvc.Add(ctx, repoPath, []string{"status-test.txt"})
	require.NoError(t, err)

	// Check status with staged file
	status, err = commitSvc.GetStatus(ctx, repoPath)
	require.NoError(t, err)
	assert.False(t, status.IsClean)
	assert.Len(t, status.Staged, 1)

	// Modify the staged file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	// Check status with both staged and unstaged
	status, err = commitSvc.GetStatus(ctx, repoPath)
	require.NoError(t, err)
	assert.False(t, status.IsClean)
	assert.Len(t, status.Staged, 1)
	assert.Len(t, status.Unstaged, 1)
}

func TestCommitService_HasUncommittedChanges(t *testing.T) {
	repoPath, _, commitSvc := setupCommitTestRepo(t)
	ctx := context.Background()

	// Initially no uncommitted changes
	assert.False(t, commitSvc.HasUncommittedChanges(ctx, repoPath))

	// Create a new file
	testFile := filepath.Join(repoPath, "uncommitted-test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Now has uncommitted changes
	assert.True(t, commitSvc.HasUncommittedChanges(ctx, repoPath))
}

func TestCommitService_GetLastCommit(t *testing.T) {
	repoPath, _, commitSvc := setupCommitTestRepo(t)
	ctx := context.Background()

	// Get initial commit
	commit, err := commitSvc.GetLastCommit(ctx, repoPath)
	require.NoError(t, err)
	assert.Equal(t, "Initial commit", commit.Message)
	assert.NotEmpty(t, commit.SHA)
	assert.Equal(t, "Test User", commit.Author)
	assert.Equal(t, "test@test.com", commit.Email)
}

func TestCommitService_AmendLastCommit(t *testing.T) {
	repoPath, _, commitSvc := setupCommitTestRepo(t)
	ctx := context.Background()

	// Create and commit a file
	testFile := filepath.Join(repoPath, "amend-test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	err = commitSvc.Add(ctx, repoPath, []string{"amend-test.txt"})
	require.NoError(t, err)

	err = commitSvc.Commit(ctx, repoPath, "Original message", CommitOptions{})
	require.NoError(t, err)

	// Amend the commit
	err = commitSvc.AmendLastCommit(ctx, repoPath, "Amended message")
	assert.NoError(t, err)

	// Verify the message was changed
	commit, err := commitSvc.GetLastCommit(ctx, repoPath)
	require.NoError(t, err)
	assert.Equal(t, "Amended message", commit.Message)
}

func TestCommitService_GetDiff(t *testing.T) {
	repoPath, _, commitSvc := setupCommitTestRepo(t)
	ctx := context.Background()

	// Modify a file
	testFile := filepath.Join(repoPath, "README.md")
	err := os.WriteFile(testFile, []byte("# Modified\n\nNew content\n"), 0644)
	require.NoError(t, err)

	// Get unstaged diff
	diff, err := commitSvc.GetDiff(ctx, repoPath, false)
	require.NoError(t, err)
	assert.Contains(t, diff, "New content")

	// Stage the file
	err = commitSvc.Add(ctx, repoPath, []string{"README.md"})
	require.NoError(t, err)

	// Get staged diff
	diff, err = commitSvc.GetDiff(ctx, repoPath, true)
	require.NoError(t, err)
	assert.Contains(t, diff, "New content")
}

func TestStatusCodeToString(t *testing.T) {
	tests := []struct {
		code byte
		want string
	}{
		{'M', "modified"},
		{'A', "added"},
		{'D', "deleted"},
		{'R', "renamed"},
		{'C', "copied"},
		{'U', "unmerged"},
		{'X', "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := statusCodeToString(tt.code)
			assert.Equal(t, tt.want, got)
		})
	}
}
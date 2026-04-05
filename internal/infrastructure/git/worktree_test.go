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

// setupWorktreeTestRepo creates a main repository for worktree testing.
func setupWorktreeTestRepo(t *testing.T) (string, *CommandExecutor, *worktreeServiceImpl) {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "main-repo")

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
	err = os.WriteFile(testFile, []byte("# Main Repository\n"), 0644)
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

	branchSvc := NewBranchService(BranchServiceParams{
		Executor: executor,
		Client:   client,
		Logger:   zap.NewNop(),
	})

	worktreeSvc := NewWorktreeService(WorktreeServiceParams{
		Executor:  executor,
		Client:    client,
		BranchSvc: branchSvc,
		Logger:    zap.NewNop(),
	}).(*worktreeServiceImpl)

	return repoPath, executor, worktreeSvc
}

func TestWorktreeService_CreateWorktreeWithNewBranch(t *testing.T) {
	mainRepoPath, _, worktreeSvc := setupWorktreeTestRepo(t)
	ctx := context.Background()

	// Create worktree path
	tmpDir := filepath.Dir(mainRepoPath)
	worktreePath := filepath.Join(tmpDir, "worktree-1")

	// Create worktree with new branch
	err := worktreeSvc.CreateWorktreeWithNewBranch(ctx, mainRepoPath, worktreePath, "issue-123-test-feature", "main")
	require.NoError(t, err)

	// Verify worktree exists
	assert.True(t, worktreeSvc.WorktreeExists(ctx, worktreePath))

	// Verify worktree isolation
	err = worktreeSvc.ValidateWorktreeIsolation(ctx, worktreePath)
	assert.NoError(t, err)

	// Verify branch was created
	branch, err := worktreeSvc.client.GetCurrentBranch(ctx, worktreePath)
	assert.NoError(t, err)
	assert.Equal(t, "issue-123-test-feature", branch)

	// Clean up
	err = worktreeSvc.DeleteWorktree(ctx, mainRepoPath, worktreePath)
	assert.NoError(t, err)
}

func TestWorktreeService_DeleteWorktree(t *testing.T) {
	mainRepoPath, _, worktreeSvc := setupWorktreeTestRepo(t)
	ctx := context.Background()

	// Create a worktree first
	tmpDir := filepath.Dir(mainRepoPath)
	worktreePath := filepath.Join(tmpDir, "worktree-to-delete")

	err := worktreeSvc.CreateWorktreeWithNewBranch(ctx, mainRepoPath, worktreePath, "branch-to-delete", "main")
	require.NoError(t, err)

	// Delete the worktree
	err = worktreeSvc.DeleteWorktree(ctx, mainRepoPath, worktreePath)
	assert.NoError(t, err)

	// Verify worktree no longer exists
	assert.False(t, worktreeSvc.WorktreeExists(ctx, worktreePath))

	// Verify the branch still exists (DeleteWorktree shouldn't delete branch)
	// Need to check through ListBranches or BranchExists
}

func TestWorktreeService_DeleteWorktreeAndBranch(t *testing.T) {
	mainRepoPath, _, worktreeSvc := setupWorktreeTestRepo(t)
	ctx := context.Background()

	// Create a worktree first
	tmpDir := filepath.Dir(mainRepoPath)
	worktreePath := filepath.Join(tmpDir, "worktree-and-branch-delete")

	branchName := "branch-and-worktree-delete"
	err := worktreeSvc.CreateWorktreeWithNewBranch(ctx, mainRepoPath, worktreePath, branchName, "main")
	require.NoError(t, err)

	// Delete both worktree and branch
	err = worktreeSvc.DeleteWorktreeAndBranch(ctx, mainRepoPath, worktreePath, branchName)
	assert.NoError(t, err)

	// Verify worktree no longer exists
	assert.False(t, worktreeSvc.WorktreeExists(ctx, worktreePath))
}

func TestWorktreeService_ListWorktrees(t *testing.T) {
	mainRepoPath, _, worktreeSvc := setupWorktreeTestRepo(t)
	ctx := context.Background()

	// Initially should have one worktree (the main repo itself as a worktree entry)
	worktrees, err := worktreeSvc.ListWorktrees(ctx, mainRepoPath)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(worktrees), 1)

	// Create additional worktrees
	tmpDir := filepath.Dir(mainRepoPath)
	worktreePath1 := filepath.Join(tmpDir, "worktree-1")
	worktreePath2 := filepath.Join(tmpDir, "worktree-2")

	err = worktreeSvc.CreateWorktreeWithNewBranch(ctx, mainRepoPath, worktreePath1, "feature-1", "main")
	require.NoError(t, err)

	err = worktreeSvc.CreateWorktreeWithNewBranch(ctx, mainRepoPath, worktreePath2, "feature-2", "main")
	require.NoError(t, err)

	// List worktrees again
	worktrees, err = worktreeSvc.ListWorktrees(ctx, mainRepoPath)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(worktrees), 3) // main + 2 created

	// Verify branches are correct
	branchNames := make(map[string]bool)
	for _, wt := range worktrees {
		if wt.Branch != "" {
			branchNames[wt.Branch] = true
		}
	}
	assert.True(t, branchNames["feature-1"])
	assert.True(t, branchNames["feature-2"])
}

func TestWorktreeService_WorktreeExists(t *testing.T) {
	mainRepoPath, _, worktreeSvc := setupWorktreeTestRepo(t)
	ctx := context.Background()

	// Test non-existing worktree
	tmpDir := filepath.Dir(mainRepoPath)
	nonExistingPath := filepath.Join(tmpDir, "non-existing")
	assert.False(t, worktreeSvc.WorktreeExists(ctx, nonExistingPath))

	// Create a worktree
	worktreePath := filepath.Join(tmpDir, "existing-worktree")
	err := worktreeSvc.CreateWorktreeWithNewBranch(ctx, mainRepoPath, worktreePath, "test-branch", "main")
	require.NoError(t, err)

	// Test existing worktree
	assert.True(t, worktreeSvc.WorktreeExists(ctx, worktreePath))
}

func TestWorktreeService_GetMainRepoPath(t *testing.T) {
	mainRepoPath, _, worktreeSvc := setupWorktreeTestRepo(t)
	ctx := context.Background()

	// Create a worktree
	tmpDir := filepath.Dir(mainRepoPath)
	worktreePath := filepath.Join(tmpDir, "worktree-for-path-test")

	err := worktreeSvc.CreateWorktreeWithNewBranch(ctx, mainRepoPath, worktreePath, "path-test-branch", "main")
	require.NoError(t, err)

	// Get main repo path from worktree
	mainPath, err := worktreeSvc.GetMainRepoPath(ctx, worktreePath)
	assert.NoError(t, err)

	// Verify the path points to the main repo
	// Note: paths might differ slightly due to symlinks, so we compare the resolved paths
	resolvedMain, _ := filepath.EvalSymlinks(mainRepoPath)
	resolvedResult, _ := filepath.EvalSymlinks(mainPath)
	assert.Equal(t, resolvedMain, resolvedResult)
}

func TestWorktreeService_ValidateWorktreeIsolation(t *testing.T) {
	mainRepoPath, _, worktreeSvc := setupWorktreeTestRepo(t)
	ctx := context.Background()

	// Create a worktree
	tmpDir := filepath.Dir(mainRepoPath)
	worktreePath := filepath.Join(tmpDir, "isolation-test-worktree")

	err := worktreeSvc.CreateWorktreeWithNewBranch(ctx, mainRepoPath, worktreePath, "isolation-test", "main")
	require.NoError(t, err)

	// Validate isolation
	err = worktreeSvc.ValidateWorktreeIsolation(ctx, worktreePath)
	assert.NoError(t, err)

	// Test that main repo fails isolation check (it's not a worktree)
	err = worktreeSvc.ValidateWorktreeIsolation(ctx, mainRepoPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a worktree")
}

func TestWorktreeService_GenerateWorktreePath(t *testing.T) {
	tests := []struct {
		name        string
		basePath    string
		repository  string
		issueNumber int
		want        string
	}{
		{
			name:        "simple path",
			basePath:    "/var/litchi/worktrees",
			repository:  "owner/repo",
			issueNumber: 123,
			want:        "/var/litchi/worktrees/owner-repo-123",
		},
		{
			name:        "complex repository name",
			basePath:    "/tmp/worktrees",
			repository:  "organization/project-name",
			issueNumber: 456,
			want:        "/tmp/worktrees/organization-project-name-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWorktreePath(tt.basePath, tt.repository, tt.issueNumber)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWorktreeService_PruneWorktrees(t *testing.T) {
	mainRepoPath, _, worktreeSvc := setupWorktreeTestRepo(t)
	ctx := context.Background()

	// Create a worktree
	tmpDir := filepath.Dir(mainRepoPath)
	worktreePath := filepath.Join(tmpDir, "prune-test-worktree")

	err := worktreeSvc.CreateWorktreeWithNewBranch(ctx, mainRepoPath, worktreePath, "prune-test", "main")
	require.NoError(t, err)

	// Manually remove the worktree directory (simulate external deletion)
	os.RemoveAll(worktreePath)

	// Prune worktrees
	err = worktreeSvc.PruneWorktrees(ctx, mainRepoPath)
	assert.NoError(t, err)

	// Verify the worktree is no longer listed
	worktrees, err := worktreeSvc.ListWorktrees(ctx, mainRepoPath)
	require.NoError(t, err)

	for _, wt := range worktrees {
		assert.NotEqual(t, worktreePath, wt.Path)
	}
}
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

// setupTestRepo creates a temporary Git repository for testing.
func setupTestRepo(t *testing.T) (string, *CommandExecutor, *branchServiceImpl) {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

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

	branchSvc := NewBranchService(BranchServiceParams{
		Executor: executor,
		Client:   client,
		Logger:   zap.NewNop(),
	}).(*branchServiceImpl)

	return repoPath, executor, branchSvc
}

func TestBranchService_ValidateBranchName(t *testing.T) {
	branchSvc := &branchServiceImpl{}

	tests := []struct {
		name       string
		branchName string
		wantErr    bool
	}{
		{
			name:       "valid simple name",
			branchName: "feature-123",
			wantErr:    false,
		},
		{
			name:       "valid issue branch name",
			branchName: "issue-123-fix-login-bug",
			wantErr:    false,
		},
		{
			name:       "empty name",
			branchName: "",
			wantErr:    true,
		},
		{
			name:       "starts with dot",
			branchName: ".feature",
			wantErr:    true,
		},
		{
			name:       "starts with dash",
			branchName: "-feature",
			wantErr:    true,
		},
		{
			name:       "contains double dot",
			branchName: "feature..test",
			wantErr:    true,
		},
		{
			name:       "contains tilde",
			branchName: "feature~test",
			wantErr:    true,
		},
		{
			name:       "ends with slash",
			branchName: "feature/",
			wantErr:    true,
		},
		{
			name:       "ends with .lock",
			branchName: "feature.lock",
			wantErr:    true,
		},
		{
			name:       "contains consecutive slashes",
			branchName: "feature//test",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := branchSvc.ValidateBranchName(tt.branchName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBranchService_GenerateBranchName(t *testing.T) {
	branchSvc := &branchServiceImpl{}

	tests := []struct {
		name        string
		issueNumber int
		title       string
		want        string
	}{
		{
			name:        "simple title",
			issueNumber: 123,
			title:       "Fix login bug",
			want:        "issue-123-fix-login-bug",
		},
		{
			name:        "title with special characters",
			issueNumber: 456,
			title:       "Add new feature: user authentication",
			want:        "issue-456-add-new-feature-user-authentication",
		},
		{
			name:        "title with multiple spaces",
			issueNumber: 789,
			title:       "Update   README   file",
			want:        "issue-789-update-readme-file",
		},
		{
			name:        "title with uppercase",
			issueNumber: 100,
			title:       "Add API Endpoints",
			want:        "issue-100-add-api-endpoints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := branchSvc.GenerateBranchName(tt.issueNumber, tt.title)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBranchService_CreateBranch(t *testing.T) {
	repoPath, _, branchSvc := setupTestRepo(t)
	ctx := context.Background()

	// Test creating a new branch
	err := branchSvc.CreateBranch(ctx, repoPath, "test-branch")
	assert.NoError(t, err)

	// Verify branch exists
	exists := branchSvc.BranchExists(ctx, repoPath, "test-branch")
	assert.True(t, exists)

	// Test creating duplicate branch
	err = branchSvc.CreateBranch(ctx, repoPath, "test-branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestBranchService_BranchExists(t *testing.T) {
	repoPath, _, branchSvc := setupTestRepo(t)
	ctx := context.Background()

	// Create a branch
	err := branchSvc.CreateBranch(ctx, repoPath, "existing-branch")
	require.NoError(t, err)

	tests := []struct {
		name       string
		branchName string
		want       bool
	}{
		{
			name:       "existing branch",
			branchName: "existing-branch",
			want:       true,
		},
		{
			name:       "non-existing branch",
			branchName: "non-existing-branch",
			want:       false,
		},
		{
			name:       "main branch (created by git init)",
			branchName: "main",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := branchSvc.BranchExists(ctx, repoPath, tt.branchName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBranchService_DeleteBranch(t *testing.T) {
	repoPath, _, branchSvc := setupTestRepo(t)
	ctx := context.Background()

	// Create a branch
	err := branchSvc.CreateBranch(ctx, repoPath, "branch-to-delete")
	require.NoError(t, err)

	// Delete the branch
	err = branchSvc.DeleteBranch(ctx, repoPath, "branch-to-delete")
	assert.NoError(t, err)

	// Verify branch no longer exists
	exists := branchSvc.BranchExists(ctx, repoPath, "branch-to-delete")
	assert.False(t, exists)

	// Test deleting non-existing branch
	err = branchSvc.DeleteBranch(ctx, repoPath, "non-existing-branch")
	assert.Error(t, err)
}

func TestBranchService_ListBranches(t *testing.T) {
	repoPath, _, branchSvc := setupTestRepo(t)
	ctx := context.Background()

	// Create some branches
	err := branchSvc.CreateBranch(ctx, repoPath, "feature-1")
	require.NoError(t, err)
	err = branchSvc.CreateBranch(ctx, repoPath, "feature-2")
	require.NoError(t, err)

	// List branches
	branches, err := branchSvc.ListBranches(ctx, repoPath)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(branches), 3) // master + 2 created branches

	// Verify branch names are present
	branchNames := make([]string, len(branches))
	for i, b := range branches {
		branchNames[i] = b.Name
	}
	assert.Contains(t, branchNames, "main")
	assert.Contains(t, branchNames, "feature-1")
	assert.Contains(t, branchNames, "feature-2")
}

func TestBranchService_ParseBranchName(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		want       int
	}{
		{
			name:       "valid issue branch",
			branchName: "issue-123-fix-login-bug",
			want:       123,
		},
		{
			name:       "valid issue branch with complex slug",
			branchName: "issue-456-update-user-profile-page",
			want:       456,
		},
		{
			name:       "non-issue branch",
			branchName: "feature-test",
			want:       0,
		},
		{
			name:       "main branch",
			branchName: "main",
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseBranchName(tt.branchName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTitleToSlug(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "simple title",
			title: "Fix login bug",
			want:  "fix-login-bug",
		},
		{
			name:  "title with special characters",
			title: "Add @mentions feature! (v2)",
			want:  "add-mentions-feature-v2",
		},
		{
			name:  "title with multiple spaces",
			title: "Update   API   endpoints",
			want:  "update-api-endpoints",
		},
		{
			name:  "title with consecutive hyphens",
			title: "Fix --help option",
			want:  "fix-help-option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titleToSlug(tt.title)
			assert.Equal(t, tt.want, got)
		})
	}
}
// Package service provides application services for the Litchi system.
package service

import (
	"context"
	"errors"
	"testing"

	litchierrors "github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Test Helpers ---

func newTestAuthService(permissionAPI PermissionAPI) *AuthService {
	return NewAuthServiceWithAPI(permissionAPI, zap.NewNop())
}

// --- CheckRepoPermission Tests ---

func TestAuthService_CheckRepoPermission_AdminPermission(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(&PermissionResult{Permission: "admin"}, nil)

	authService := newTestAuthService(mockAPI)
	hasPermission, err := authService.CheckRepoPermission(ctx, "owner/repo", "testuser")

	require.NoError(t, err)
	assert.True(t, hasPermission, "admin permission should return true")
}

func TestAuthService_CheckRepoPermission_MaintainPermission(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(&PermissionResult{Permission: "maintain"}, nil)

	authService := newTestAuthService(mockAPI)
	hasPermission, err := authService.CheckRepoPermission(ctx, "owner/repo", "testuser")

	require.NoError(t, err)
	assert.True(t, hasPermission, "maintain permission should return true")
}

func TestAuthService_CheckRepoPermission_WritePermission(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(&PermissionResult{Permission: "write"}, nil)

	authService := newTestAuthService(mockAPI)
	hasPermission, err := authService.CheckRepoPermission(ctx, "owner/repo", "testuser")

	require.NoError(t, err)
	assert.False(t, hasPermission, "write permission should return false")
}

func TestAuthService_CheckRepoPermission_ReadPermission(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(&PermissionResult{Permission: "read"}, nil)

	authService := newTestAuthService(mockAPI)
	hasPermission, err := authService.CheckRepoPermission(ctx, "owner/repo", "testuser")

	require.NoError(t, err)
	assert.False(t, hasPermission, "read permission should return false")
}

func TestAuthService_CheckRepoPermission_NonePermission(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(&PermissionResult{Permission: "none"}, nil)

	authService := newTestAuthService(mockAPI)
	hasPermission, err := authService.CheckRepoPermission(ctx, "owner/repo", "testuser")

	require.NoError(t, err)
	assert.False(t, hasPermission, "none permission should return false")
}

func TestAuthService_CheckRepoPermission_APIError(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(nil, litchierrors.New(litchierrors.ErrGitHubAPIError).WithDetail("API error"))

	authService := newTestAuthService(mockAPI)
	hasPermission, err := authService.CheckRepoPermission(ctx, "owner/repo", "testuser")

	require.Error(t, err)
	assert.False(t, hasPermission)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrGitHubAPIError))
}

// --- GetPermissionLevel Tests ---

func TestAuthService_GetPermissionLevel_Success(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(&PermissionResult{Permission: "admin"}, nil)

	authService := newTestAuthService(mockAPI)
	permission, err := authService.GetPermissionLevel(ctx, "owner/repo", "testuser")

	require.NoError(t, err)
	assert.Equal(t, "admin", permission)
}

func TestAuthService_GetPermissionLevel_Maintain(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(&PermissionResult{Permission: "maintain"}, nil)

	authService := newTestAuthService(mockAPI)
	permission, err := authService.GetPermissionLevel(ctx, "owner/repo", "testuser")

	require.NoError(t, err)
	assert.Equal(t, "maintain", permission)
}

func TestAuthService_GetPermissionLevel_APIError(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(nil, litchierrors.New(litchierrors.ErrGitHubAuthFailed).WithDetail("auth error"))

	authService := newTestAuthService(mockAPI)
	permission, err := authService.GetPermissionLevel(ctx, "owner/repo", "testuser")

	require.Error(t, err)
	assert.Empty(t, permission)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrGitHubAuthFailed))
}

// --- ValidateActorPermission Tests ---

func TestAuthService_ValidateActorPermission_ViewerOperation(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	authService := newTestAuthService(mockAPI)

	// Viewer operations should pass without any API call
	err := authService.ValidateActorPermission(ctx, "owner/repo", "testuser", "view_progress")
	require.NoError(t, err)

	err = authService.ValidateActorPermission(ctx, "owner/repo", "testuser", "create_issue")
	require.NoError(t, err)

	err = authService.ValidateActorPermission(ctx, "owner/repo", "testuser", "trigger_agent")
	require.NoError(t, err)
}

func TestAuthService_ValidateActorPermission_AdminOperation_HasAdminPermission(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(&PermissionResult{Permission: "admin"}, nil)

	authService := newTestAuthService(mockAPI)
	err := authService.ValidateActorPermission(ctx, "owner/repo", "testuser", "admin_force")

	require.NoError(t, err)
}

func TestAuthService_ValidateActorPermission_AdminOperation_HasMaintainPermission(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(&PermissionResult{Permission: "maintain"}, nil)

	authService := newTestAuthService(mockAPI)
	err := authService.ValidateActorPermission(ctx, "owner/repo", "testuser", "admin_force")

	require.NoError(t, err)
}

func TestAuthService_ValidateActorPermission_AdminOperation_NoPermission(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(&PermissionResult{Permission: "write"}, nil)

	authService := newTestAuthService(mockAPI)
	err := authService.ValidateActorPermission(ctx, "owner/repo", "testuser", "admin_force")

	require.Error(t, err)
	assert.True(t, litchierrors.Is(err, litchierrors.ErrPermissionDenied))
}

func TestAuthService_ValidateActorPermission_AdminOperation_APIError(t *testing.T) {
	ctx := context.Background()
	mockAPI := NewMockPermissionAPI(t)

	mockAPI.EXPECT().GetPermissionLevel(ctx, "owner", "repo", "testuser").
		Return(nil, errors.New("API error"))

	authService := newTestAuthService(mockAPI)
	err := authService.ValidateActorPermission(ctx, "owner/repo", "testuser", "admin_force")

	require.Error(t, err)
}

// --- IsAdminOrMaintain Tests ---

func TestIsAdminOrMaintain(t *testing.T) {
	assert.True(t, IsAdminOrMaintain("admin"))
	assert.True(t, IsAdminOrMaintain("maintain"))
	assert.False(t, IsAdminOrMaintain("write"))
	assert.False(t, IsAdminOrMaintain("read"))
	assert.False(t, IsAdminOrMaintain("none"))
	assert.False(t, IsAdminOrMaintain(""))
}

// --- getRequiredRole Tests ---

func TestAuthService_GetRequiredRole(t *testing.T) {
	authService := newTestAuthService(nil)

	// Viewer operations
	assert.Equal(t, "viewer", authService.getRequiredRole("view_progress"))
	assert.Equal(t, "viewer", authService.getRequiredRole("create_issue"))
	assert.Equal(t, "viewer", authService.getRequiredRole("trigger_agent"))

	// Admin operations
	assert.Equal(t, "repo_admin", authService.getRequiredRole("admin_force"))
	assert.Equal(t, "repo_admin", authService.getRequiredRole("resume"))
	assert.Equal(t, "repo_admin", authService.getRequiredRole("rollback"))
	assert.Equal(t, "repo_admin", authService.getRequiredRole("unknown_operation"))
}
// Package service provides application services for the Litchi system.
package service

import (
	"context"

	"github.com/ryuyb/litchi/internal/infrastructure/github"
	"github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/ryuyb/litchi/internal/pkg/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// PermissionResult contains the result of a permission check.
type PermissionResult struct {
	Permission string
}

// PermissionAPI defines the interface for checking repository permissions via GitHub API.
// This interface allows mocking the GitHub API in tests.
type PermissionAPI interface {
	// GetPermissionLevel fetches the permission level for a user on a repository.
	// Returns the permission result or an error.
	GetPermissionLevel(ctx context.Context, owner, repo, username string) (*PermissionResult, error)
}

// GitHubClientProvider defines the interface for obtaining GitHub clients.
// This allows AuthService to be decoupled from concrete ClientManager implementation.
type GitHubClientProvider interface {
	// GetClient returns a GitHub client for the specified repository.
	GetClient(ctx context.Context, repoName string) (*github.Client, error)
}

// githubPermissionAPI implements PermissionAPI using real GitHub API calls.
type githubPermissionAPI struct {
	provider GitHubClientProvider
}

// GetPermissionLevel fetches the permission level using the GitHub API.
func (a *githubPermissionAPI) GetPermissionLevel(ctx context.Context, owner, repo, username string) (*PermissionResult, error) {
	repoName := owner + "/" + repo
	client, err := a.provider.GetClient(ctx, repoName)
	if err != nil {
		return nil, errors.Wrap(errors.ErrGitHubAuthFailed, err).
			WithDetail("failed to get GitHub client for repository: " + repoName)
	}

	permissionLevel, resp, err := client.GitHub().Repositories.GetPermissionLevel(ctx, owner, repo, username)
	if err != nil {
		return nil, errors.Wrap(errors.ErrGitHubAPIError, err).
			WithDetail("failed to get permission level for user: " + username)
	}

	// Update rate limit stats from response
	if resp != nil {
		client.RateLimiter().UpdateFromResponse(resp)
	}

	return &PermissionResult{
		Permission: permissionLevel.GetPermission(),
	}, nil
}

// AuthService provides permission checking for GitHub repositories.
// It verifies if users have admin or maintain permissions on repositories.
type AuthService struct {
	permissionAPI PermissionAPI
	logger        *zap.Logger
}

// AuthServiceParams holds dependencies for AuthService.
type AuthServiceParams struct {
	fx.In

	ClientManager *github.ClientManager
	Logger        *zap.Logger
}

// NewAuthService creates a new AuthService.
func NewAuthService(p AuthServiceParams) *AuthService {
	provider := p.ClientManager
	return &AuthService{
		permissionAPI: &githubPermissionAPI{provider: provider},
		logger:        p.Logger.Named("auth_service"),
	}
}

// NewAuthServiceWithAPI creates an AuthService with a custom PermissionAPI (for testing).
func NewAuthServiceWithAPI(permissionAPI PermissionAPI, logger *zap.Logger) *AuthService {
	return &AuthService{
		permissionAPI: permissionAPI,
		logger:        logger.Named("auth_service"),
	}
}

// CheckRepoPermission checks if a user has admin or maintain permission on a repository.
// Returns true if the user has admin or maintain permission, false otherwise.
// The repoName should be in "owner/repo" format.
func (s *AuthService) CheckRepoPermission(ctx context.Context, repoName, username string) (bool, error) {
	// Extract owner and repo from the full name
	owner := utils.ExtractOwner(repoName)
	repo := utils.ExtractRepo(repoName)

	// Get permission level for the user
	result, err := s.permissionAPI.GetPermissionLevel(ctx, owner, repo, username)
	if err != nil {
		return false, err
	}

	// Check if permission is admin or maintain
	hasAdminPermission := result.Permission == "admin" || result.Permission == "maintain"

	s.logger.Debug("permission check completed",
		zap.String("repo", repoName),
		zap.String("username", username),
		zap.String("permission", result.Permission),
		zap.Bool("has_admin_permission", hasAdminPermission),
	)

	return hasAdminPermission, nil
}

// ValidateActorPermission validates if an actor has permission to perform an operation.
// For operations requiring repository admin permission, it checks the GitHub permission.
// The repoName should be in "owner/repo" format.
func (s *AuthService) ValidateActorPermission(ctx context.Context, repoName, username, operation string) error {
	// Determine required permission level for the operation
	requiredRole := s.getRequiredRole(operation)

	// If no special permission required, allow
	if requiredRole == "viewer" {
		return nil
	}

	// For admin operations, check repository permission
	if requiredRole == "repo_admin" {
		hasPermission, err := s.CheckRepoPermission(ctx, repoName, username)
		if err != nil {
			return err
		}
		if !hasPermission {
			return errors.New(errors.ErrPermissionDenied).
				WithDetail("operation '" + operation + "' requires repository admin or maintain permission")
		}
	}

	return nil
}

// getRequiredRole returns the required role level for an operation.
// Following architecture.md section 12.1 permission matrix:
// - viewer: operations available to anyone (view_progress, create_issue)
// - repo_admin: operations requiring admin/maintain permission
func (s *AuthService) getRequiredRole(operation string) string {
	// Operations available to all users (Issue authors can perform these)
	viewerOperations := []string{
		"view_progress",
		"create_issue",
		"trigger_agent", // Anyone can trigger, but non-admins will enter waiting state
	}

	for _, op := range viewerOperations {
		if op == operation {
			return "viewer"
		}
	}

	// All other operations require repository admin permission
	return "repo_admin"
}

// GetPermissionLevel returns the raw permission level for a user on a repository.
// This is useful for displaying the actual permission level rather than just boolean.
// Returns the permission level string (e.g., "admin", "maintain", "write", "read", "none").
func (s *AuthService) GetPermissionLevel(ctx context.Context, repoName, username string) (string, error) {
	owner := utils.ExtractOwner(repoName)
	repo := utils.ExtractRepo(repoName)

	result, err := s.permissionAPI.GetPermissionLevel(ctx, owner, repo, username)
	if err != nil {
		return "", err
	}

	return result.Permission, nil
}

// IsAdminOrMaintain checks if the given permission level qualifies as repository admin.
func IsAdminOrMaintain(permission string) bool {
	return permission == "admin" || permission == "maintain"
}
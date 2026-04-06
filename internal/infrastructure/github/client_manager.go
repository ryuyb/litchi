// Package github provides GitHub API integration with client management.
package github

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/ryuyb/litchi/internal/pkg/health"
	"github.com/ryuyb/litchi/internal/pkg/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// ClientManager manages GitHub API clients with support for both PAT and GitHub App authentication.
// For PAT mode, it provides a shared client for all repositories.
// For GitHub App mode, it manages installation tokens per repository.
type ClientManager struct {
	strategy    AuthStrategy
	repoRepo    repository.RepositoryRepository
	rateLimiter *RateLimiter
	retryPolicy valueobject.RetryPolicy
	logger      *zap.Logger

	// PAT mode: shared client
	sharedClient *Client

	// App mode: JWT transport and singleflight for token fetching
	jwtTransport *JWTTransport
	tokenFetchMu sync.Map // singleflight: stores chan struct{} per installationID to prevent duplicate fetches
}

// ClientManagerParams contains dependencies for creating a ClientManager.
type ClientManagerParams struct {
	fx.In

	Config   *config.Config
	RepoRepo repository.RepositoryRepository
	Logger   *zap.Logger
}

// NewClientManager creates a new ClientManager.
func NewClientManager(p ClientManagerParams) (*ClientManager, error) {
	if p.Config == nil || p.Logger == nil {
		return nil, fmt.Errorf("config and logger are required")
	}

	ghConfig := &p.Config.GitHub

	// Create authentication strategy
	strategy, err := NewAuthStrategyFromConfig(ghConfig, p.Logger)
	if err != nil {
		return nil, err
	}

	// Create rate limiter
	rateLimiter := NewRateLimiter(RateLimiterParams{
		Config: &p.Config.Failure.RateLimit,
		Logger: p.Logger,
	})

	cm := &ClientManager{
		strategy:    strategy,
		repoRepo:    p.RepoRepo,
		rateLimiter: rateLimiter,
		retryPolicy: valueobject.DefaultRetryPolicy,
		logger:      p.Logger.Named("github_client_manager"),
	}

	// For PAT mode, create a shared client upfront
	if !strategy.SupportsRepositoryPerClient() {
		client, err := strategy.CreateClient(context.Background())
		if err != nil {
			return nil, errors.Wrap(errors.ErrGitHubAuthFailed, err).
				WithDetail("failed to create PAT client")
		}

		cm.sharedClient = &Client{
			client:      client,
			logger:      p.Logger.Named("github"),
			rateLimiter: rateLimiter,
			retryPolicy: valueobject.DefaultRetryPolicy,
		}
	} else {
		// For App mode, get JWT transport from strategy
		appStrategy, ok := strategy.(*GitHubAppAuthStrategy)
		if !ok {
			return nil, errors.New(errors.ErrGitHubAuthFailed).
				WithDetail("expected GitHubAppAuthStrategy for App mode")
		}
		cm.jwtTransport = NewJWTTransport(appStrategy.GetAppID(), appStrategy.GetPrivateKey())
	}

	return cm, nil
}

// GetClient returns a GitHub client for the specified repository.
// For PAT mode, returns the shared client.
// For App mode, returns a client with the appropriate installation token.
func (cm *ClientManager) GetClient(ctx context.Context, repoName string) (*Client, error) {
	// PAT mode: return shared client
	if cm.sharedClient != nil {
		return cm.sharedClient, nil
	}

	// GitHub App mode: get client for the repository's installation
	return cm.getAppClient(ctx, repoName)
}

// GetClientForInstallation returns a GitHub client for the specified installation ID.
// This is useful when the installation ID is known (e.g., from webhook payload).
func (cm *ClientManager) GetClientForInstallation(ctx context.Context, installationID int64) (*Client, error) {
	// PAT mode: return shared client
	if cm.sharedClient != nil {
		return cm.sharedClient, nil
	}

	if installationID <= 0 {
		return nil, errors.New(errors.ErrGitHubAuthFailed).
			WithDetail("invalid installation ID")
	}

	return cm.createAppClient(ctx, installationID)
}

// GetSharedClient returns the shared client (for PAT mode) or nil.
func (cm *ClientManager) GetSharedClient() *Client {
	return cm.sharedClient
}

// GetAuthType returns the current authentication type.
func (cm *ClientManager) GetAuthType() AuthType {
	return cm.strategy.GetAuthType()
}

// getAppClient creates a client for the specified repository using GitHub App authentication.
func (cm *ClientManager) getAppClient(ctx context.Context, repoName string) (*Client, error) {
	// Get repository from database to find installation ID
	repo, err := cm.repoRepo.FindByName(ctx, repoName)
	if err != nil {
		return nil, errors.Wrap(errors.ErrDatabaseOperation, err)
	}

	var installationID int64

	if repo != nil && repo.HasInstallation() {
		installationID = repo.InstallationID
	} else {
		// Try to find installation via API
		installationID, err = cm.findInstallationForRepo(ctx, repoName)
		if err != nil {
			return nil, errors.New(errors.ErrGitHubAuthFailed).
				WithDetail(fmt.Sprintf("no installation found for repository %s", repoName))
		}

		// Update repository with installation ID
		if repo == nil {
			repo = entity.NewRepository(repoName)
		}
		repo.SetInstallationID(installationID)
		if err := cm.repoRepo.Save(ctx, repo); err != nil {
			cm.logger.Warn("failed to save installation ID",
				zap.String("repo", repoName),
				zap.Error(err),
			)
		}
	}

	return cm.createAppClient(ctx, installationID)
}

// createAppClient creates a GitHub client with an installation token.
func (cm *ClientManager) createAppClient(ctx context.Context, installationID int64) (*Client, error) {
	appStrategy, ok := cm.strategy.(*GitHubAppAuthStrategy)
	if !ok {
		return nil, errors.New(errors.ErrGitHubAuthFailed).
			WithDetail("expected GitHubAppAuthStrategy for App mode")
	}

	// Check cache first
	cachedToken := appStrategy.GetTokenCache().Get(installationID)
	if cachedToken != nil {
		return cm.createClientWithToken(cachedToken.Token), nil
	}

	// Fetch new installation token with singleflight to prevent duplicate concurrent fetches
	token, err := cm.fetchInstallationTokenSingleflight(ctx, installationID, appStrategy)
	if err != nil {
		return nil, err
	}

	return cm.createClientWithToken(token), nil
}

// fetchInstallationTokenSingleflight fetches an installation token with singleflight protection.
// This prevents concurrent requests for the same installation ID from making duplicate API calls.
func (cm *ClientManager) fetchInstallationTokenSingleflight(ctx context.Context, installationID int64, appStrategy *GitHubAppAuthStrategy) (string, error) {
	// Check if there's already an ongoing fetch for this installation
	chI, loaded := cm.tokenFetchMu.LoadOrStore(installationID, make(chan struct{}))
	ch := chI.(chan struct{})

	if loaded {
		// Wait for the ongoing fetch to complete
		select {
		case <-ch:
			// Fetch completed, check cache again
			cachedToken := appStrategy.GetTokenCache().Get(installationID)
			if cachedToken != nil {
				return cachedToken.Token, nil
			}
			// Cache still empty after waiting - this shouldn't happen normally
			// Return error to let caller retry or handle appropriately
			return "", errors.New(errors.ErrGitHubAuthFailed).
				WithDetail("token not available after concurrent fetch completed")
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	// We're the first to fetch, do the actual fetch
	defer func() {
		close(ch)
		cm.tokenFetchMu.Delete(installationID)
	}()

	return cm.fetchInstallationToken(ctx, installationID, appStrategy)
}

// fetchInstallationToken fetches a new installation token from GitHub.
func (cm *ClientManager) fetchInstallationToken(ctx context.Context, installationID int64, appStrategy *GitHubAppAuthStrategy) (string, error) {
	// Create JWT client for App API calls
	jwtClient := github.NewClient(&http.Client{
		Transport: cm.jwtTransport,
	})

	// Create installation token
	token, resp, err := jwtClient.Apps.CreateInstallationToken(ctx, installationID, nil)
	if err != nil {
		return "", errors.Wrap(errors.ErrGitHubAuthFailed, err).
			WithDetail("failed to create installation token")
	}
	cm.rateLimiter.UpdateFromResponse(resp)

	// Cache the token
	appStrategy.GetTokenCache().Set(installationID, &InstallationToken{
		Token:     token.GetToken(),
		ExpiresAt: token.GetExpiresAt().Time,
	})

	cm.logger.Debug("fetched new installation token",
		zap.Int64("installation_id", installationID),
		zap.Time("expires_at", token.GetExpiresAt().Time),
	)

	return token.GetToken(), nil
}

// findInstallationForRepo finds the installation ID for a repository.
func (cm *ClientManager) findInstallationForRepo(ctx context.Context, repoName string) (int64, error) {
	owner, repo := utils.ExtractOwner(repoName), utils.ExtractRepo(repoName)

	jwtClient := github.NewClient(&http.Client{
		Transport: cm.jwtTransport,
	})

	installation, resp, err := jwtClient.Apps.FindRepositoryInstallation(ctx, owner, repo)
	if err != nil {
		return 0, errors.Wrap(errors.ErrGitHubAPIError, err).
			WithDetail(fmt.Sprintf("installation not found for %s", repoName))
	}
	cm.rateLimiter.UpdateFromResponse(resp)

	if installation == nil || installation.ID == nil {
		return 0, errors.New(errors.ErrGitHubAPIError).
			WithDetail(fmt.Sprintf("installation not found for %s", repoName))
	}

	return installation.GetID(), nil
}

// createClientWithToken creates a GitHub client with the given token.
func (cm *ClientManager) createClientWithToken(token string) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)

	return &Client{
		client:      client,
		logger:      cm.logger.Named("github"),
		rateLimiter: cm.rateLimiter,
		retryPolicy: cm.retryPolicy,
	}
}

// Check performs health check for GitHub API connectivity.
func (cm *ClientManager) Check(ctx context.Context) health.CheckResult {
	start := time.Now()

	var err error
	if cm.sharedClient != nil {
		err = cm.sharedClient.Ping(ctx)
	} else {
		_, err = cm.pingApp(ctx)
	}

	latency := time.Since(start)

	result := health.CheckResult{
		Name:      "github",
		LatencyMs: int(latency.Milliseconds()),
	}

	if err != nil {
		result.Status = "fail"
		result.Error = err.Error()
		result.Message = "GitHub API connection failed"
		cm.logger.Error("github health check failed", zap.Error(err))
	} else {
		result.Status = "pass"
		result.Message = fmt.Sprintf("API connection OK (auth: %s)", cm.strategy.GetAuthType())
		result.Details = map[string]any{
			"auth_type":            cm.strategy.GetAuthType(),
			"rate_limit_remaining": cm.rateLimiter.GetRemaining(),
		}
	}

	return result
}

// Name returns the health check component name.
func (cm *ClientManager) Name() string {
	return "github"
}

// pingApp verifies the GitHub App can authenticate with the API.
func (cm *ClientManager) pingApp(ctx context.Context) (*github.App, error) {
	jwtClient := github.NewClient(&http.Client{
		Transport: cm.jwtTransport,
	})

	app, resp, err := jwtClient.Apps.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	cm.rateLimiter.UpdateFromResponse(resp)

	return app, nil
}

// RefreshInstallationToken forces a refresh of the installation token for a given installation ID.
func (cm *ClientManager) RefreshInstallationToken(ctx context.Context, installationID int64) error {
	if cm.sharedClient != nil {
		// PAT mode: no token to refresh
		return nil
	}

	appStrategy, ok := cm.strategy.(*GitHubAppAuthStrategy)
	if !ok {
		return errors.New(errors.ErrGitHubAuthFailed).
			WithDetail("expected GitHubAppAuthStrategy for App mode")
	}
	appStrategy.GetTokenCache().Delete(installationID)

	_, err := cm.fetchInstallationToken(ctx, installationID, appStrategy)
	return err
}

// ClearInstallationTokens clears cached tokens for a given installation ID.
// This should be called when an installation is deleted or suspended.
func (cm *ClientManager) ClearInstallationTokens(installationID int64) {
	if cm.sharedClient != nil {
		// PAT mode: no tokens to clear
		return
	}

	appStrategy, ok := cm.strategy.(*GitHubAppAuthStrategy)
	if !ok {
		return
	}
	appStrategy.GetTokenCache().Delete(installationID)
	cm.logger.Debug("cleared installation token cache",
		zap.Int64("installation_id", installationID),
	)
}

// RateLimiter returns the rate limiter for external access.
func (cm *ClientManager) RateLimiter() *RateLimiter {
	return cm.rateLimiter
}
// Package github provides GitHub API integration with authentication strategies.
package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// AuthType represents the authentication type.
type AuthType string

const (
	AuthTypePAT       AuthType = "pat"
	AuthTypeGitHubApp AuthType = "github_app"
)

// AuthStrategy defines the interface for GitHub authentication strategies.
type AuthStrategy interface {
	// GetAuthType returns the authentication type.
	GetAuthType() AuthType

	// SupportsRepositoryPerClient indicates if the strategy requires per-repository clients.
	// PAT mode returns false (shared client), App mode returns true (per-installation client).
	SupportsRepositoryPerClient() bool

	// CreateClient creates a GitHub client for this authentication strategy.
	// For PAT: creates an OAuth2 client with the token; always succeeds if token is non-empty.
	// For GitHub App: creates a JWT client for App-level API calls (not installation tokens);
	// may fail if private key is invalid or JWT generation fails.
	CreateClient(ctx context.Context) (*github.Client, error)
}

// PATAuthStrategy implements authentication using a Personal Access Token.
type PATAuthStrategy struct {
	token  string
	logger *zap.Logger
}

// NewPATAuthStrategy creates a new PAT authentication strategy.
func NewPATAuthStrategy(token string, logger *zap.Logger) *PATAuthStrategy {
	return &PATAuthStrategy{
		token:  token,
		logger: logger,
	}
}

func (s *PATAuthStrategy) GetAuthType() AuthType                     { return AuthTypePAT }
func (s *PATAuthStrategy) SupportsRepositoryPerClient() bool         { return false }
func (s *PATAuthStrategy) GetToken() string                          { return s.token }

// CreateClient creates a GitHub client using the PAT.
func (s *PATAuthStrategy) CreateClient(ctx context.Context) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: s.token})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}

// GitHubAppAuthStrategy implements authentication using a GitHub App.
type GitHubAppAuthStrategy struct {
	appID      int64
	privateKey *rsa.PrivateKey
	tokenCache *InstallationTokenCache
	logger     *zap.Logger
}

// NewGitHubAppAuthStrategy creates a new GitHub App authentication strategy.
func NewGitHubAppAuthStrategy(appID int64, privateKey []byte, logger *zap.Logger) (*GitHubAppAuthStrategy, error) {
	key, err := parsePrivateKey(privateKey)
	if err != nil {
		return nil, errors.Wrap(errors.ErrGitHubAuthFailed, err).
			WithDetail("failed to parse private key")
	}

	return &GitHubAppAuthStrategy{
		appID:      appID,
		privateKey: key,
		tokenCache: NewInstallationTokenCache(),
		logger:     logger,
	}, nil
}

func (s *GitHubAppAuthStrategy) GetAuthType() AuthType                     { return AuthTypeGitHubApp }
func (s *GitHubAppAuthStrategy) SupportsRepositoryPerClient() bool         { return true }
func (s *GitHubAppAuthStrategy) GetAppID() int64                          { return s.appID }
func (s *GitHubAppAuthStrategy) GetPrivateKey() *rsa.PrivateKey           { return s.privateKey }
func (s *GitHubAppAuthStrategy) GetTokenCache() *InstallationTokenCache   { return s.tokenCache }

// CreateClient creates a GitHub client with JWT authentication for App-level API calls.
// This is used for operations like fetching installation tokens, not for repository operations.
func (s *GitHubAppAuthStrategy) CreateClient(ctx context.Context) (*github.Client, error) {
	jwtTransport := NewJWTTransport(s.appID, s.privateKey)
	return github.NewClient(&http.Client{Transport: jwtTransport}), nil
}

// parsePrivateKey parses a PEM-encoded RSA private key.
func parsePrivateKey(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		key, ok := keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		return key, nil
	}

	return key, nil
}

// InstallationToken represents a cached installation token.
type InstallationToken struct {
	Token     string
	ExpiresAt time.Time
}

// IsExpired checks if the token is expired (with 5-minute buffer).
func (t *InstallationToken) IsExpired() bool {
	if t == nil {
		return true
	}
	// Consider expired 5 minutes before actual expiration
	return time.Now().Add(5 * time.Minute).After(t.ExpiresAt)
}

// InstallationTokenCache caches installation tokens by installation ID.
type InstallationTokenCache struct {
	mu     sync.RWMutex
	tokens map[int64]*InstallationToken
}

// NewInstallationTokenCache creates a new token cache.
func NewInstallationTokenCache() *InstallationTokenCache {
	return &InstallationTokenCache{
		tokens: make(map[int64]*InstallationToken),
	}
}

// Get retrieves a cached token, returns nil if not found or expired.
func (c *InstallationTokenCache) Get(installationID int64) *InstallationToken {
	c.mu.RLock()
	defer c.mu.RUnlock()

	token, ok := c.tokens[installationID]
	if !ok {
		return nil
	}

	if token.IsExpired() {
		return nil
	}

	return token
}

// Set stores a token in the cache.
func (c *InstallationTokenCache) Set(installationID int64, token *InstallationToken) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokens[installationID] = token
}

// Delete removes a token from the cache.
func (c *InstallationTokenCache) Delete(installationID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.tokens, installationID)
}

// NewAuthStrategyFromConfig creates an authentication strategy based on configuration.
func NewAuthStrategyFromConfig(cfg *config.GitHubConfig, logger *zap.Logger) (AuthStrategy, error) {
	// Check if values are real or unresolved env placeholders
	hasPAT := cfg.Token != "" && !config.IsEnvPlaceholder(cfg.Token)
	hasApp := cfg.AppID != "" && !config.IsEnvPlaceholder(cfg.AppID) &&
		cfg.PrivateKeyPath != "" && !config.IsEnvPlaceholder(cfg.PrivateKeyPath)

	if hasApp {
		// GitHub App authentication takes precedence
		privateKey, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, errors.Wrap(errors.ErrGitHubAuthFailed, err).
				WithDetail(fmt.Sprintf("failed to read private key file: %s", cfg.PrivateKeyPath))
		}

		appID, err := strconv.ParseInt(cfg.AppID, 10, 64)
		if err != nil {
			return nil, errors.Wrap(errors.ErrGitHubAuthFailed, err).
				WithDetail("invalid app_id format")
		}

		strategy, err := NewGitHubAppAuthStrategy(appID, privateKey, logger)
		if err != nil {
			return nil, err
		}

		logger.Info("using GitHub App authentication",
			zap.Int64("app_id", appID),
		)

		return strategy, nil
	}

	if hasPAT {
		logger.Info("using PAT authentication")
		return NewPATAuthStrategy(cfg.Token, logger), nil
	}

	return nil, errors.New(errors.ErrGitHubAuthFailed).
		WithDetail("no authentication method configured")
}
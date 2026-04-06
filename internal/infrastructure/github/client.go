// Package github provides GitHub API integration with rate limiting and retry logic.
package github

import (
	"context"
	stderrors "errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/pkg/errors"
	"github.com/ryuyb/litchi/internal/pkg/health"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// ClientParams contains dependencies for creating a GitHubClient.
type ClientParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

// Client wraps go-github client with rate limit handling and retry logic.
type Client struct {
	client      *github.Client
	logger      *zap.Logger
	rateLimiter *RateLimiter
	retryPolicy valueobject.RetryPolicy
	config      *config.GitHubConfig
}

// NewClient creates a GitHub client with authentication.
// Supports both Personal Access Token and GitHub App authentication.
func NewClient(p ClientParams) (*Client, error) {
	if p.Config == nil || p.Logger == nil {
		return nil, fmt.Errorf("config and logger are required")
	}

	ghConfig := &p.Config.GitHub
	var client *github.Client

	// Authentication strategy
	if ghConfig.AppID != "" && ghConfig.PrivateKeyPath != "" {
		// GitHub App authentication (reserved for future use)
		// TODO: Implement GitHub App authentication when needed
		p.Logger.Warn("GitHub App authentication configured but not yet implemented, falling back to token authentication",
			zap.String("app_id", ghConfig.AppID),
		)
	}

	// Personal Access Token authentication
	if ghConfig.Token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: ghConfig.Token},
		)
		tc := oauth2.NewClient(context.Background(), ts)
		client = github.NewClient(tc)
	} else {
		return nil, errors.New(errors.ErrGitHubAuthFailed).
			WithDetail("no authentication method configured")
	}

	// Create rate limiter using config
	rateLimitCfg := &p.Config.Failure.RateLimit
	rateLimiter := NewRateLimiter(RateLimiterParams{
		Config: rateLimitCfg,
		Logger: p.Logger,
	})

	// Use default retry policy
	retryPolicy := valueobject.DefaultRetryPolicy

	return &Client{
		client:      client,
		logger:      p.Logger.Named("github"),
		rateLimiter: rateLimiter,
		retryPolicy: retryPolicy,
		config:      ghConfig,
	}, nil
}

// GitHub returns the underlying go-github client for direct API access.
// Use with caution - rate limiting and retry logic are not applied.
func (c *Client) GitHub() *github.Client {
	return c.client
}

// RateLimiter returns the rate limiter for external inspection.
func (c *Client) RateLimiter() *RateLimiter {
	return c.rateLimiter
}

// Ping tests the GitHub API connection.
func (c *Client) Ping(ctx context.Context) error {
	// Get authenticated user to test connection
	_, resp, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return c.wrapError(err)
	}
	c.rateLimiter.UpdateFromResponse(resp)
	return nil
}

// Name returns the health check component name.
func (c *Client) Name() string {
	return "github"
}

// Check performs the health check for GitHub API.
func (c *Client) Check(ctx context.Context) health.CheckResult {
	start := time.Now()

	err := c.Ping(ctx)
	latency := time.Since(start)

	result := health.CheckResult{
		Name:      c.Name(),
		LatencyMs: int(latency.Milliseconds()),
	}

	if err != nil {
		result.Status = "fail"
		result.Error = err.Error()
		result.Message = "GitHub API connection failed"
		c.logger.Error("github health check failed", zap.Error(err))
	} else {
		result.Status = "pass"
		result.Message = "API connection OK"

		// Add rate limit info if available
		remaining := c.rateLimiter.GetRemaining()
		if remaining > 0 {
			result.Details = map[string]any{
				"rate_limit_remaining": remaining,
			}
		}
	}

	return result
}

// executeWithRetry wraps API calls with rate limit handling and retry logic.
// Returns the result from the operation directly, eliminating the need for
// external variable assignment in the callback.
func executeWithRetry[T any](c *Client, ctx context.Context, op func() (T, *github.Response, error)) (T, error) {
	var lastErr error
	var zero T

	for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
		// Check rate limit before execution
		if err := c.rateLimiter.WaitForRateLimit(ctx); err != nil {
			return zero, err
		}

		result, resp, err := op()
		if err == nil {
			// Update rate limit stats from response headers
			c.rateLimiter.UpdateFromResponse(resp)
			return result, nil
		}

		// Update rate limit stats even on error
		c.rateLimiter.UpdateFromResponse(resp)

		// Check if rate limited (403 with rate limit headers)
		if resp != nil && resp.StatusCode == 403 {
			if c.rateLimiter.IsRateLimited(resp) {
				if c.rateLimiter.WaitEnabled() {
					c.logger.Warn("rate limit hit, waiting for reset",
						zap.Int("attempt", attempt),
					)
					if err := c.rateLimiter.WaitForReset(ctx, resp); err != nil {
						return zero, err
					}
					continue // Retry after wait
				}
				return zero, errors.New(errors.ErrGitHubAPIRateLimit).
					WithDetail("rate limit exceeded, wait disabled")
			}
		}

		// Check if retryable
		if !c.isRetryableError(err, resp) {
			return zero, c.wrapError(err)
		}

		lastErr = err
		backoff := c.retryPolicy.BackoffConfig.CalculateDelay(attempt)
		c.logger.Warn("API call failed, retrying",
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.Duration("backoff", backoff),
		)

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(backoff):
		}
	}

	return zero, errors.Wrap(errors.ErrGitHubAPIError, lastErr).
		WithDetail("max retries exceeded")
}

// executeWithRetryResponse wraps API calls that only return a response (no result data).
// This is useful for delete/remove operations that don't return a meaningful result.
func executeWithRetryResponse(c *Client, ctx context.Context, op func() (*github.Response, error)) error {
	_, err := executeWithRetry(c, ctx, func() (struct{}, *github.Response, error) {
		resp, err := op()
		return struct{}{}, resp, err
	})
	return err
}

// isRetryableError determines if an error is retryable.
func (c *Client) isRetryableError(err error, resp *github.Response) bool {
	// Context errors are not retryable
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	if resp == nil {
		// Network errors are retryable
		return true
	}

	// HTTP status codes that are retryable
	switch resp.StatusCode {
	case 429: // Too many requests
		return true
	case 500, 502, 503, 504: // Server errors
		return true
	default:
		return false
	}
}

// wrapError wraps a GitHub API error into a Litchi error.
func (c *Client) wrapError(err error) error {
	if err == nil {
		return nil
	}

	// Check for specific error types
	var ghErr *github.ErrorResponse
	if stderrors.As(err, &ghErr) {
		switch ghErr.Response.StatusCode {
		case 401:
			return errors.Wrap(errors.ErrGitHubAuthFailed, err)
		case 403:
			// Check if rate limited by looking at the response
			if ghErr.Response != nil {
				remaining := ghErr.Response.Header.Get("X-RateLimit-Remaining")
				if remaining == "0" {
					return errors.Wrap(errors.ErrGitHubAPIRateLimit, err)
				}
			}
			return errors.Wrap(errors.ErrGitHubAPIError, err).
				WithDetail("forbidden")
		case 404:
			return errors.Wrap(errors.ErrGitHubAPIError, err).
				WithDetail("not found")
		default:
			return errors.Wrap(errors.ErrGitHubAPIError, err).
				WithDetail(fmt.Sprintf("HTTP %d", ghErr.Response.StatusCode))
		}
	}

	return errors.Wrap(errors.ErrGitHubAPIError, err)
}

// RateLimiter handles GitHub API rate limit with wait-and-retry strategy.
type RateLimiter struct {
	enabled         bool
	waitEnabled     bool
	maxWait         time.Duration
	notifyThreshold int
	logger          *zap.Logger

	// Runtime state (updated from response headers)
	remaining int
	resetTime time.Time
	mutex     sync.RWMutex
}

// RateLimiterParams contains dependencies for creating a RateLimiter.
type RateLimiterParams struct {
	Config *config.RateLimitConfig
	Logger *zap.Logger
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(p RateLimiterParams) *RateLimiter {
	cfg := p.Config
	if cfg == nil {
		cfg = &config.RateLimitConfig{
			Enabled:         true,
			WaitEnabled:     true,
			MaxWaitDuration: "30m",
			NotifyThreshold: 10,
		}
	}

	maxWait, err := time.ParseDuration(cfg.MaxWaitDuration)
	if err != nil || maxWait == 0 {
		if err != nil {
			p.Logger.Warn("invalid max_wait_duration, using default",
				zap.String("config", cfg.MaxWaitDuration),
				zap.Error(err),
			)
		}
		maxWait = 30 * time.Minute
	}

	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &RateLimiter{
		enabled:         cfg.Enabled,
		waitEnabled:     cfg.WaitEnabled,
		maxWait:         maxWait,
		notifyThreshold: cfg.NotifyThreshold,
		logger:          logger.Named("rate_limiter"),
	}
}

// WaitEnabled returns whether waiting is enabled.
func (r *RateLimiter) WaitEnabled() bool {
	return r.waitEnabled
}

// WaitForRateLimit checks if we should wait before making a request.
// Logs warning if remaining calls are below threshold.
func (r *RateLimiter) WaitForRateLimit(ctx context.Context) error {
	if !r.enabled {
		return nil
	}

	r.mutex.RLock()
	remaining := r.remaining
	resetTime := r.resetTime
	r.mutex.RUnlock()

	// Notify if below threshold
	if r.notifyThreshold > 0 && remaining < r.notifyThreshold && remaining > 0 {
		r.logger.Warn("approaching rate limit",
			zap.Int("remaining", remaining),
			zap.Int("threshold", r.notifyThreshold),
			zap.Time("reset_time", resetTime),
		)
	}

	return nil
}

// IsRateLimited checks if the response indicates a rate limit error.
func (r *RateLimiter) IsRateLimited(resp *github.Response) bool {
	if resp == nil || resp.Response == nil {
		return false
	}

	// Check for rate limit exceeded
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	if remaining == "0" {
		return true
	}

	// Check for 403 with rate limit condition
	if resp.StatusCode == 403 && remaining != "" {
		rem, _ := strconv.Atoi(remaining)
		return rem == 0
	}

	return false
}

// WaitForReset waits for rate limit reset when we hit the limit.
func (r *RateLimiter) WaitForReset(ctx context.Context, resp *github.Response) error {
	if resp == nil {
		return errors.New(errors.ErrGitHubAPIRateLimit).
			WithDetail("no response for rate limit check")
	}

	resetTime := resp.Header.Get("X-RateLimit-Reset")
	if resetTime == "" {
		return errors.New(errors.ErrGitHubAPIRateLimit).
			WithDetail("missing rate limit reset header")
	}

	resetTimestamp, err := strconv.ParseInt(resetTime, 10, 64)
	if err != nil {
		return errors.Wrap(errors.ErrGitHubAPIRateLimit, err).
			WithDetail("invalid rate limit reset header")
	}

	waitDuration := time.Until(time.Unix(resetTimestamp, 0))
	if waitDuration <= 0 {
		// Already reset
		return nil
	}

	if waitDuration > r.maxWait {
		return errors.New(errors.ErrGitHubAPIRateLimit).
			WithDetail(fmt.Sprintf("wait duration %v exceeds max wait %v", waitDuration, r.maxWait))
	}

	r.logger.Info("waiting for rate limit reset",
		zap.Duration("wait_duration", waitDuration),
	)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitDuration):
		return nil
	}
}

// UpdateFromResponse updates rate limit state from API response headers.
func (r *RateLimiter) UpdateFromResponse(resp *github.Response) {
	if resp == nil {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	remaining := resp.Header.Get("X-RateLimit-Remaining")
	resetTime := resp.Header.Get("X-RateLimit-Reset")

	if remaining != "" {
		r.remaining, _ = strconv.Atoi(remaining)
	}

	if resetTime != "" {
		timestamp, _ := strconv.ParseInt(resetTime, 10, 64)
		r.resetTime = time.Unix(timestamp, 0)
	}
}

// GetRemaining returns the current remaining rate limit.
func (r *RateLimiter) GetRemaining() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.remaining
}

// GetResetTime returns the current rate limit reset time.
func (r *RateLimiter) GetResetTime() time.Time {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.resetTime
}
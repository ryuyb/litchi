package config

import (
	"errors"
	"fmt"
	"time"
)

// Config holds all configuration for the Litchi application.
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	GitHub     GitHubConfig     `mapstructure:"github"`
	Git        GitConfig        `mapstructure:"git"`
	Webhook    WebhookConfig    `mapstructure:"webhook"`
	Agent      AgentConfig      `mapstructure:"agent"`
	Clarity    ClarityConfig    `mapstructure:"clarity"`
	Complexity ComplexityConfig `mapstructure:"complexity"`
	Audit      AuditConfig      `mapstructure:"audit"`
	Failure    FailureConfig    `mapstructure:"failure"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Redis      RedisConfig      `mapstructure:"redis"`
}

// Validate validates all configuration fields.
func (c *Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}
	if err := c.GitHub.Validate(); err != nil {
		return fmt.Errorf("github config: %w", err)
	}
	if err := c.Git.Validate(); err != nil {
		return fmt.Errorf("git config: %w", err)
	}
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}
	return nil
}

type ServerConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Mode            string `mapstructure:"mode"`            // debug, release, test
	Version         string `mapstructure:"version"`
	EnableSwaggerUI bool   `mapstructure:"enable_swagger"` // Enable Swagger UI endpoint
}

func (c *ServerConfig) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	if c.Mode != "debug" && c.Mode != "release" && c.Mode != "test" {
		return errors.New("mode must be one of: debug, release, test")
	}
	return nil
}

type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Name            string `mapstructure:"name"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime string `mapstructure:"conn_max_lifetime"`  // connection max lifetime, e.g. "1h"
	ConnMaxIdleTime string `mapstructure:"conn_max_idle_time"` // connection max idle time, e.g. "10m"
	AutoMigrate     bool   `mapstructure:"auto_migrate"`       // run migrations on startup
}

func (c *DatabaseConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Name == "" {
		return errors.New("name is required")
	}
	if c.User == "" {
		return errors.New("user is required")
	}
	if c.Password == "" {
		return errors.New("password is required (set DB_PASSWORD environment variable)")
	}
	if c.Port < 1 || c.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	return nil
}

type GitHubConfig struct {
	Token          string `mapstructure:"token"`
	WebhookSecret  string `mapstructure:"webhook_secret"`
	AppID          string `mapstructure:"app_id"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
}

func (c *GitHubConfig) Validate() error {
	if c.Token == "" {
		return errors.New("token is required (set GITHUB_TOKEN environment variable)")
	}
	if c.WebhookSecret == "" {
		return errors.New("webhook_secret is required (set GITHUB_WEBHOOK_SECRET environment variable)")
	}
	return nil
}

// GitConfig holds Git-related configuration.
type GitConfig struct {
	WorktreeBasePath    string `mapstructure:"worktree_base_path"`    // Base path for worktrees (default: /var/litchi/worktrees)
	WorktreeAutoClean   bool   `mapstructure:"worktree_auto_clean"`   // Auto-clean worktrees on session end
	BranchNamingPattern string `mapstructure:"branch_naming_pattern"` // Branch naming pattern (default: issue-{number}-{slug})
	DefaultBaseBranch   string `mapstructure:"default_base_branch"`   // Default base branch (default: main)
	CommitSignOff       bool   `mapstructure:"commit_sign_off"`       // Add Signed-off-by trailer
	GitBinaryPath       string `mapstructure:"git_binary_path"`       // Git binary path (default: git)
	CommandTimeout      string `mapstructure:"command_timeout"`       // Git command timeout (default: 5m)
}

// Validate validates Git configuration.
func (c *GitConfig) Validate() error {
	// Set defaults if empty
	if c.WorktreeBasePath == "" {
		c.WorktreeBasePath = "/var/litchi/worktrees"
	}
	if c.DefaultBaseBranch == "" {
		c.DefaultBaseBranch = "main"
	}
	if c.BranchNamingPattern == "" {
		c.BranchNamingPattern = "issue-{number}-{slug}"
	}
	if c.GitBinaryPath == "" {
		c.GitBinaryPath = "git"
	}
	if c.CommandTimeout == "" {
		c.CommandTimeout = "5m"
	}

	// Validate command timeout format (default already set above)
	if _, err := time.ParseDuration(c.CommandTimeout); err != nil {
		return fmt.Errorf("invalid git.command_timeout: %w", err)
	}

	return nil
}

type WebhookConfig struct {
	Idempotency IdempotencyConfig `mapstructure:"idempotency"`
}

type IdempotencyConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	TTL             string `mapstructure:"ttl"`
	AutoCleanup     bool   `mapstructure:"auto_cleanup"`
	CleanupInterval string `mapstructure:"cleanup_interval"`
}

type AgentConfig struct {
	Type            string `mapstructure:"type"`
	MaxConcurrency  int    `mapstructure:"max_concurrency"`
	TaskRetryLimit  int    `mapstructure:"task_retry_limit"`
	ApprovalTimeout string `mapstructure:"approval_timeout"`
}

type ClarityConfig struct {
	Threshold             int `mapstructure:"threshold"`
	AutoProceedThreshold  int `mapstructure:"auto_proceed_threshold"`
	ForceClarifyThreshold int `mapstructure:"force_clarify_threshold"`
}

type ComplexityConfig struct {
	Threshold          int  `mapstructure:"threshold"`
	ForceDesignConfirm bool `mapstructure:"force_design_confirm"`
}

type AuditConfig struct {
	Enabled             bool     `mapstructure:"enabled"`
	RetentionDays       int      `mapstructure:"retention_days"`
	MaxOutputLength     int      `mapstructure:"max_output_length"`
	SensitiveOperations []string `mapstructure:"sensitive_operations"`
}

type FailureConfig struct {
	Retry           RetryConfig           `mapstructure:"retry"`
	RateLimit       RateLimitConfig       `mapstructure:"rate_limit"`
	Timeout         TimeoutConfig         `mapstructure:"timeout"`
	Queue           QueueConfig           `mapstructure:"queue"`
	TestEnvironment TestEnvironmentConfig `mapstructure:"test_environment"`
}

type RetryConfig struct {
	MaxRetries        int      `mapstructure:"max_retries"`
	InitialBackoff    string   `mapstructure:"initial_backoff"`
	MaxBackoff        string   `mapstructure:"max_backoff"`
	BackoffMultiplier float64  `mapstructure:"backoff_multiplier"`
	RetryableErrors   []string `mapstructure:"retryable_errors"`
}

type RateLimitConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	WaitEnabled     bool   `mapstructure:"wait_enabled"`
	MaxWaitDuration string `mapstructure:"max_wait_duration"`
	NotifyThreshold int    `mapstructure:"notify_threshold"`
}

type TimeoutConfig struct {
	ClarificationAgent string `mapstructure:"clarification_agent"`
	DesignAnalysis     string `mapstructure:"design_analysis"`
	DesignGeneration   string `mapstructure:"design_generation"`
	TaskBreakdown      string `mapstructure:"task_breakdown"`
	TaskExecution      string `mapstructure:"task_execution"`
	TestRun            string `mapstructure:"test_run"`
	PRCreation         string `mapstructure:"pr_creation"`
	ApprovalWait       string `mapstructure:"approval_wait"`
	SessionMaxDuration string `mapstructure:"session_max_duration"`
}

type QueueConfig struct {
	MaxLength       int    `mapstructure:"max_length"`
	PriorityEnabled bool   `mapstructure:"priority_enabled"`
	TimeoutOnQueue  string `mapstructure:"timeout_on_queue"`
}

type TestEnvironmentConfig struct {
	SkipIfNoTests     bool   `mapstructure:"skip_if_no_tests"`
	SkipIfUnavailable bool   `mapstructure:"skip_if_unavailable"`
	CheckInterval     string `mapstructure:"check_interval"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // json, console
	Output string `mapstructure:"output"` // stdout, stderr, file path
}

type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
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
	Middleware MiddlewareConfig `mapstructure:"middleware"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Session    SessionConfig    `mapstructure:"session"`

	// env holds the current environment (dev, uat, prod, etc.)
	env Environment
}

// Environment returns the current environment.
func (c *Config) Environment() Environment {
	return c.env
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
	if err := c.Middleware.Validate(); err != nil {
		return fmt.Errorf("middleware config: %w", err)
	}
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config: %w", err)
	}
	if err := c.Session.Validate(); err != nil {
		return fmt.Errorf("session config: %w", err)
	}
	return nil
}

// Clone creates a deep copy of the configuration.
func (c *Config) Clone() *Config {
	clone := &Config{
		Server:     c.Server,
		Database:   c.Database,
		GitHub:     c.GitHub,
		Git:        c.Git,
		Webhook:    c.Webhook,
		Agent:      c.Agent,
		Clarity:    c.Clarity,
		Complexity: c.Complexity,
		Audit:      c.Audit,
		Failure:    c.Failure,
		Logging:    c.Logging,
		Middleware: c.Middleware,
		Redis:      c.Redis,
	}

	// Deep copy slices
	if len(c.Audit.SensitiveOperations) > 0 {
		clone.Audit.SensitiveOperations = make([]string, len(c.Audit.SensitiveOperations))
		copy(clone.Audit.SensitiveOperations, c.Audit.SensitiveOperations)
	}
	if len(c.Failure.Retry.RetryableErrors) > 0 {
		clone.Failure.Retry.RetryableErrors = make([]string, len(c.Failure.Retry.RetryableErrors))
		copy(clone.Failure.Retry.RetryableErrors, c.Failure.Retry.RetryableErrors)
	}
	// Deep copy middleware slices
	if len(c.Middleware.CORS.AllowOrigins) > 0 {
		clone.Middleware.CORS.AllowOrigins = make([]string, len(c.Middleware.CORS.AllowOrigins))
		copy(clone.Middleware.CORS.AllowOrigins, c.Middleware.CORS.AllowOrigins)
	}
	if len(c.Middleware.CORS.AllowMethods) > 0 {
		clone.Middleware.CORS.AllowMethods = make([]string, len(c.Middleware.CORS.AllowMethods))
		copy(clone.Middleware.CORS.AllowMethods, c.Middleware.CORS.AllowMethods)
	}
	if len(c.Middleware.CORS.AllowHeaders) > 0 {
		clone.Middleware.CORS.AllowHeaders = make([]string, len(c.Middleware.CORS.AllowHeaders))
		copy(clone.Middleware.CORS.AllowHeaders, c.Middleware.CORS.AllowHeaders)
	}
	if len(c.Middleware.CORS.ExposeHeaders) > 0 {
		clone.Middleware.CORS.ExposeHeaders = make([]string, len(c.Middleware.CORS.ExposeHeaders))
		copy(clone.Middleware.CORS.ExposeHeaders, c.Middleware.CORS.ExposeHeaders)
	}

	return clone
}

type ServerConfig struct {
	Host            string          `mapstructure:"host"`
	Port            int             `mapstructure:"port"`
	Mode            string          `mapstructure:"mode"`            // debug, release, test
	Version         string          `mapstructure:"version"`
	EnableSwaggerUI bool            `mapstructure:"enable_swagger"` // Enable Swagger UI endpoint
	WebSocket       *WebSocketConfig `mapstructure:"websocket"`      // WebSocket configuration (optional)
}

func (c *ServerConfig) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	if c.Mode != "debug" && c.Mode != "release" && c.Mode != "test" {
		return errors.New("mode must be one of: debug, release, test")
	}
	if c.WebSocket != nil {
		if err := c.WebSocket.Validate(); err != nil {
			return fmt.Errorf("websocket config: %w", err)
		}
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

// Validate validates GitHub configuration.
// Requires webhook_secret and either token (PAT) or app_id with private_key_path (GitHub App).
func (c *GitHubConfig) Validate() error {
	// Webhook secret is always required (check if it's a real value, not an env placeholder)
	webhookSecret := c.WebhookSecret
	if IsEnvPlaceholder(webhookSecret) {
		webhookSecret = ""
	}
	if webhookSecret == "" {
		return errors.New("webhook_secret is required (set GITHUB_WEBHOOK_SECRET environment variable)")
	}

	// Check authentication method
	// A value like "${GITHUB_TOKEN}" means the env var wasn't set, so we treat it as empty
	hasPAT := c.Token != "" && !IsEnvPlaceholder(c.Token)
	hasApp := c.AppID != "" && !IsEnvPlaceholder(c.AppID) &&
		c.PrivateKeyPath != "" && !IsEnvPlaceholder(c.PrivateKeyPath)

	if !hasPAT && !hasApp {
		return errors.New("either token or app_id with private_key_path is required")
	}

	// If GitHub App is configured, validate private key file exists
	if hasApp {
		if _, err := os.Stat(c.PrivateKeyPath); os.IsNotExist(err) {
			return fmt.Errorf("private key file not found: %s", c.PrivateKeyPath)
		}
	}

	return nil
}

// IsEnvPlaceholder checks if the value is an unresolved environment variable placeholder.
// Returns true for values like "${GITHUB_TOKEN}" that weren't expanded by Viper.
func IsEnvPlaceholder(value string) bool {
	return len(value) > 3 && value[0] == '$' && value[1] == '{' && value[len(value)-1] == '}'
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

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level      string             `mapstructure:"level"`
	Outputs    []OutputConfig     `mapstructure:"outputs"`
	Encoder    EncoderConfig      `mapstructure:"encoder"`
	Caller     CallerConfig       `mapstructure:"caller"`
	Stacktrace StacktraceConfig   `mapstructure:"stacktrace"`
}

// OutputConfig defines a single output target configuration.
type OutputConfig struct {
	Type     string              `mapstructure:"type"`     // console | file
	Format   string              `mapstructure:"format"`   // json | console (per output)
	Path     string              `mapstructure:"path"`     // file path (required for file type)
	Rotation RotationConfig      `mapstructure:"rotation"` // file rotation config
	Console  ConsoleOutputConfig `mapstructure:"console"`  // console-specific config
}

// RotationConfig holds file rotation configuration using lumberjack.
type RotationConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	MaxSize    int  `mapstructure:"max_size"`    // MB, default 100
	MaxBackups int  `mapstructure:"max_backups"` // number of old files to retain
	MaxAge     int  `mapstructure:"max_age"`     // days to retain old files
	Compress   bool `mapstructure:"compress"`    // compress rotated files
	LocalTime  bool `mapstructure:"local_time"`  // use local time for timestamps
}

// ConsoleOutputConfig holds console output configuration.
type ConsoleOutputConfig struct {
	Stream string `mapstructure:"stream"` // stdout | stderr
	Color  bool   `mapstructure:"color"`  // enable colored output
}

// EncoderConfig holds encoder configuration.
type EncoderConfig struct {
	TimeFormat     string `mapstructure:"time_format"`     // iso8601 | epoch | epochMillis | custom
	DurationFormat string `mapstructure:"duration_format"` // string | seconds | nanos
	LevelFormat    string `mapstructure:"level_format"`    // lowercase | uppercase | capitalColor
}

// CallerConfig controls caller information display.
type CallerConfig struct {
	Enabled bool `mapstructure:"enabled"` // add caller (file:line) to logs
	Skip    int  `mapstructure:"skip"`    // stack frames to skip
}

// StacktraceConfig controls stacktrace capture.
type StacktraceConfig struct {
	Enabled bool   `mapstructure:"enabled"` // enable stacktrace
	Level   string `mapstructure:"level"`   // minimum level for stacktrace (default: error)
}

// Validate validates logging configuration.
func (c *LoggingConfig) Validate() error {
	// Validate level
	validLevels := []string{"debug", "info", "warn", "error", "dpanic", "panic", "fatal"}
	if c.Level != "" && !slices.Contains(validLevels, strings.ToLower(c.Level)) {
		return fmt.Errorf("invalid log level: %s, valid: %v", c.Level, validLevels)
	}

	// Validate outputs
	for i, output := range c.Outputs {
		if err := output.Validate(); err != nil {
			return fmt.Errorf("outputs[%d]: %w", i, err)
		}
	}

	// Validate stacktrace level
	if c.Stacktrace.Enabled && c.Stacktrace.Level != "" {
		if !slices.Contains(validLevels, strings.ToLower(c.Stacktrace.Level)) {
			return fmt.Errorf("invalid stacktrace level: %s", c.Stacktrace.Level)
		}
	}

	return nil
}

// Validate validates output configuration.
func (c *OutputConfig) Validate() error {
	validTypes := []string{"console", "file"}
	if !slices.Contains(validTypes, c.Type) {
		return fmt.Errorf("invalid output type: %s, valid: %v", c.Type, validTypes)
	}

	validFormats := []string{"json", "console"}
	if c.Format != "" && !slices.Contains(validFormats, strings.ToLower(c.Format)) {
		return fmt.Errorf("invalid format: %s, valid: %v", c.Format, validFormats)
	}

	if c.Type == "file" {
		if c.Path == "" {
			return errors.New("file output requires path")
		}
		if err := c.Rotation.Validate(); err != nil {
			return fmt.Errorf("rotation: %w", err)
		}
	}

	if c.Type == "console" {
		validStreams := []string{"stdout", "stderr"}
		if c.Console.Stream != "" && !slices.Contains(validStreams, c.Console.Stream) {
			return fmt.Errorf("invalid console stream: %s, valid: %v", c.Console.Stream, validStreams)
		}
	}

	return nil
}

// Validate validates rotation configuration.
func (c *RotationConfig) Validate() error {
	if c.Enabled {
		if c.MaxSize < 0 {
			return errors.New("max_size must be positive")
		}
		if c.MaxBackups < 0 {
			return errors.New("max_backups must be non-negative")
		}
		if c.MaxAge < 0 {
			return errors.New("max_age must be non-negative")
		}
	}
	return nil
}

type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// WebSocketConfig holds WebSocket connection configuration.
type WebSocketConfig struct {
	PingInterval time.Duration `mapstructure:"ping_interval"` // Ping interval for keep-alive (default: 30s)
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`  // Read timeout (default: 60s)
	WriteTimeout time.Duration `mapstructure:"write_timeout"` // Write timeout (default: 10s)
}

// Validate validates WebSocket configuration.
func (c *WebSocketConfig) Validate() error {
	if c.PingInterval > 0 && c.PingInterval < time.Second {
		return errors.New("ping_interval must be at least 1s")
	}
	if c.ReadTimeout > 0 && c.PingInterval > 0 && c.ReadTimeout < c.PingInterval {
		return errors.New("read_timeout must be greater than ping_interval")
	}
	if c.WriteTimeout > 0 && c.WriteTimeout < time.Second {
		return errors.New("write_timeout must be at least 1s")
	}
	return nil
}

// MiddlewareConfig holds HTTP middleware configuration.
type MiddlewareConfig struct {
	Recover   RecoverMiddlewareConfig   `mapstructure:"recover"`
	RequestID RequestIDMiddlewareConfig `mapstructure:"request_id"`
	CORS      CORSMiddlewareConfig      `mapstructure:"cors"`
	Limiter   LimiterMiddlewareConfig   `mapstructure:"limiter"`
	Compress  CompressMiddlewareConfig  `mapstructure:"compress"`
	CSRF      CSRFMiddlewareConfig      `mapstructure:"csrf"`
}

// SessionConfig holds session configuration.
type SessionConfig struct {
	IdleTimeout string `mapstructure:"idle_timeout"` // Session idle timeout (default: 24h)
}

// Validate validates session configuration.
func (c *SessionConfig) Validate() error {
	if c.IdleTimeout != "" {
		if _, err := time.ParseDuration(c.IdleTimeout); err != nil {
			return fmt.Errorf("session.idle_timeout: %w", err)
		}
	}
	return nil
}

// GetIdleTimeout returns the parsed idle timeout duration with default.
func (c *SessionConfig) GetIdleTimeout() time.Duration {
	if c.IdleTimeout == "" {
		return 24 * time.Hour
	}
	d, err := time.ParseDuration(c.IdleTimeout)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

// CSRFMiddlewareConfig holds CSRF middleware configuration.
type CSRFMiddlewareConfig struct {
	Enabled       bool     `mapstructure:"enabled"`
	CookieName    string   `mapstructure:"cookie_name"`     // CSRF cookie name (default: csrf_)
	IdleTimeout   string   `mapstructure:"idle_timeout"`    // Token idle timeout (default: 30m)
	ExcludedPaths []string `mapstructure:"excluded_paths"`  // Paths excluded from CSRF check
	TrustedOrigins []string `mapstructure:"trusted_origins"` // Trusted origins for unsafe requests
}

// Validate validates middleware configuration.
func (c *MiddlewareConfig) Validate() error {
	if err := c.Limiter.Validate(); err != nil {
		return fmt.Errorf("limiter: %w", err)
	}
	if err := c.Compress.Validate(); err != nil {
		return fmt.Errorf("compress: %w", err)
	}
	return nil
}

// RecoverMiddlewareConfig holds recover middleware configuration.
type RecoverMiddlewareConfig struct {
	Enabled          bool `mapstructure:"enabled"`
	EnableStackTrace bool `mapstructure:"enable_stack_trace"`
}

// RequestIDMiddlewareConfig holds request ID middleware configuration.
type RequestIDMiddlewareConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Header  string `mapstructure:"header"`
}

// CORSMiddlewareConfig holds CORS middleware configuration.
type CORSMiddlewareConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	ExposeHeaders    []string `mapstructure:"expose_headers"`
	MaxAge           int      `mapstructure:"max_age"`
}

// LimiterMiddlewareConfig holds rate limiter middleware configuration.
type LimiterMiddlewareConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Max        int    `mapstructure:"max"`
	Expiration string `mapstructure:"expiration"` // Duration string like "1m", "30s"
}

// Validate validates limiter middleware configuration.
func (c *LimiterMiddlewareConfig) Validate() error {
	if c.Enabled && c.Expiration != "" {
		if _, err := time.ParseDuration(c.Expiration); err != nil {
			return fmt.Errorf("limiter.expiration: %w", err)
		}
	}
	return nil
}

// CompressMiddlewareConfig holds compress middleware configuration.
type CompressMiddlewareConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Level   int  `mapstructure:"level"` // Compression level (-1 to 2)
}

// Validate validates compress middleware configuration.
func (c *CompressMiddlewareConfig) Validate() error {
	if c.Enabled && (c.Level < -1 || c.Level > 2) {
		return fmt.Errorf("compress.level must be between -1 and 2 (got %d)", c.Level)
	}
	return nil
}

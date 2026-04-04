package config

// Config holds all configuration for the Litchi application.
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	GitHub     GitHubConfig     `mapstructure:"github"`
	Webhook    WebhookConfig    `mapstructure:"webhook"`
	Agent      AgentConfig      `mapstructure:"agent"`
	Clarity    ClarityConfig    `mapstructure:"clarity"`
	Complexity ComplexityConfig `mapstructure:"complexity"`
	Audit      AuditConfig      `mapstructure:"audit"`
	Failure    FailureConfig    `mapstructure:"failure"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Redis      RedisConfig      `mapstructure:"redis"`
}

type ServerConfig struct {
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Mode    string `mapstructure:"mode"` // debug, release, test
	Version string `mapstructure:"version"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Name         string `mapstructure:"name"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type GitHubConfig struct {
	Token           string `mapstructure:"token"`
	WebhookSecret   string `mapstructure:"webhook_secret"`
	AppID           string `mapstructure:"app_id"`
	PrivateKeyPath  string `mapstructure:"private_key_path"`
}

type WebhookConfig struct {
	Idempotency IdempotencyConfig `mapstructure:"idempotency"`
}

type IdempotencyConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	TTL           string `mapstructure:"ttl"`
	AutoCleanup   bool   `mapstructure:"auto_cleanup"`
	CleanupInterval string `mapstructure:"cleanup_interval"`
}

type AgentConfig struct {
	Type             string `mapstructure:"type"`
	MaxConcurrency   int    `mapstructure:"max_concurrency"`
	TaskRetryLimit   int    `mapstructure:"task_retry_limit"`
	ApprovalTimeout  string `mapstructure:"approval_timeout"`
}

type ClarityConfig struct {
	Threshold            int `mapstructure:"threshold"`
	AutoProceedThreshold int `mapstructure:"auto_proceed_threshold"`
	ForceClarifyThreshold int `mapstructure:"force_clarify_threshold"`
}

type ComplexityConfig struct {
	Threshold          int  `mapstructure:"threshold"`
	ForceDesignConfirm bool `mapstructure:"force_design_confirm"`
}

type AuditConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	RetentionDays     int      `mapstructure:"retention_days"`
	MaxOutputLength   int      `mapstructure:"max_output_length"`
	SensitiveOperations []string `mapstructure:"sensitive_operations"`
}

type FailureConfig struct {
	Retry          RetryConfig          `mapstructure:"retry"`
	RateLimit      RateLimitConfig      `mapstructure:"rate_limit"`
	Timeout        TimeoutConfig        `mapstructure:"timeout"`
	Queue          QueueConfig          `mapstructure:"queue"`
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
	ClarificationAgent  string `mapstructure:"clarification_agent"`
	DesignAnalysis      string `mapstructure:"design_analysis"`
	DesignGeneration    string `mapstructure:"design_generation"`
	TaskBreakdown       string `mapstructure:"task_breakdown"`
	TaskExecution       string `mapstructure:"task_execution"`
	TestRun             string `mapstructure:"test_run"`
	PRCreation          string `mapstructure:"pr_creation"`
	ApprovalWait        string `mapstructure:"approval_wait"`
	SessionMaxDuration  string `mapstructure:"session_max_duration"`
}

type QueueConfig struct {
	MaxLength        int  `mapstructure:"max_length"`
	PriorityEnabled  bool `mapstructure:"priority_enabled"`
	TimeoutOnQueue   string `mapstructure:"timeout_on_queue"`
}

type TestEnvironmentConfig struct {
	SkipIfNoTests      bool   `mapstructure:"skip_if_no_tests"`
	SkipIfUnavailable  bool   `mapstructure:"skip_if_unavailable"`
	CheckInterval      string `mapstructure:"check_interval"`
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
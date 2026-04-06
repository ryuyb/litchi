package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Environment represents the deployment environment.
type Environment string

const (
	EnvDev  Environment = "dev"
	EnvUAT  Environment = "uat"
	EnvProd Environment = "prod"
)

// LoadOptions specifies options for loading configuration.
type LoadOptions struct {
	// ConfigPath is the explicit config file path (optional).
	// If empty, uses default search paths.
	ConfigPath string
	// Env is the environment for config file selection (optional).
	// If empty, detects from LITCHI_ENV/GO_ENV/ENV environment variables.
	Env Environment
}

// detectEnvironment detects the current environment from options or environment variables.
// Priority: opts.Env > LITCHI_ENV > GO_ENV > ENV > default (dev).
func detectEnvironment(optsEnv Environment) Environment {
	if optsEnv != "" {
		return optsEnv
	}
	if env := os.Getenv("LITCHI_ENV"); env != "" {
		return Environment(strings.ToLower(env))
	}
	if env := os.Getenv("GO_ENV"); env != "" {
		return Environment(strings.ToLower(env))
	}
	if env := os.Getenv("ENV"); env != "" {
		return Environment(strings.ToLower(env))
	}
	return EnvDev
}

// getConfigDir returns the config directory path from options or environment.
// Priority: opts.ConfigPath's directory > LITCHI_CONFIG_DIR > LITCHI_CONFIG_PATH's directory > default.
func getConfigDir(optsConfigPath string) string {
	if optsConfigPath != "" {
		return filepath.Dir(optsConfigPath)
	}
	// LITCHI_CONFIG_DIR takes precedence
	if dir := os.Getenv("LITCHI_CONFIG_DIR"); dir != "" {
		return dir
	}
	// If LITCHI_CONFIG_PATH is set, use its directory
	if path := os.Getenv("LITCHI_CONFIG_PATH"); path != "" {
		return filepath.Dir(path)
	}
	return "./config"
}

// isConfigFileNotFound checks if the error is a config file not found error.
func isConfigFileNotFound(err error) bool {
	if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
		return true
	}
	// Also check for os.ErrNotExist for file system errors
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	return false
}

// NewConfigWithOptions creates a Config instance with explicit options.
// This is the primary config loading function, allowing direct control over
// config path and environment without relying on environment variables.
func NewConfigWithOptions(opts LoadOptions) (*Config, error) {
	v := viper.New()

	env := detectEnvironment(opts.Env)
	configDir := getConfigDir(opts.ConfigPath)

	// Set config file details
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// If explicit config path provided, use it directly
	if opts.ConfigPath != "" {
		v.SetConfigFile(opts.ConfigPath)
	} else {
		v.AddConfigPath(configDir)
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/litchi")
	}

	// Enable environment variable override
	v.AutomaticEnv()
	v.SetEnvPrefix("LITCHI")

	// Bind specific env vars
	// Errors are intentionally ignored - if binding fails, values fallback to config file.
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("github.token", "GITHUB_TOKEN")
	_ = v.BindEnv("github.webhook_secret", "GITHUB_WEBHOOK_SECRET")
	_ = v.BindEnv("github.app_id", "GITHUB_APP_ID")
	_ = v.BindEnv("github.private_key_path", "GITHUB_PRIVATE_KEY_PATH")

	// Set defaults
	setDefaults(v)

	// Step 1: Read base config file (config.yaml)
	if err := v.ReadInConfig(); err != nil {
		if !isConfigFileNotFound(err) {
			return nil, fmt.Errorf("config file read error: %w", err)
		}
		// Config file not found is acceptable - defaults and env vars will be used
	}

	// Step 2: Read environment-specific config file (config.{env}.yaml)
	// Skip if explicit config path was provided (user specified exact file)
	if opts.ConfigPath == "" {
		envConfigFile := fmt.Sprintf("config.%s", env)
		v.SetConfigName(envConfigFile)
		if err := v.MergeInConfig(); err != nil {
			if !isConfigFileNotFound(err) {
				return nil, fmt.Errorf("environment config file read error: %w", err)
			}
			// Environment config file not found is acceptable
		}

		// Step 3: Read local override config file (config.local.yaml)
		// This is for developer-specific overrides, should be in .gitignore
		v.SetConfigName("config.local")
		if err := v.MergeInConfig(); err != nil {
			if !isConfigFileNotFound(err) {
				return nil, fmt.Errorf("local config file read error: %w", err)
			}
			// Local config is optional
		}
	}

	// Unmarshal to struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set env in config for reference
	cfg.env = env

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for configuration.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.version", "dev")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.name", "litchi")
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", "1h")
	v.SetDefault("database.conn_max_idle_time", "10m")

	// Webhook defaults
	v.SetDefault("webhook.idempotency.enabled", true)
	v.SetDefault("webhook.idempotency.ttl", "24h")
	v.SetDefault("webhook.idempotency.auto_cleanup", true)
	v.SetDefault("webhook.idempotency.cleanup_interval", "1h")

	// Git defaults
	v.SetDefault("git.worktree_base_path", "/var/litchi/worktrees")
	v.SetDefault("git.worktree_auto_clean", true)
	v.SetDefault("git.branch_naming_pattern", "issue-{number}-{slug}")
	v.SetDefault("git.default_base_branch", "main")
	v.SetDefault("git.commit_sign_off", true)
	v.SetDefault("git.git_binary_path", "git")
	v.SetDefault("git.command_timeout", "5m")

	// Agent defaults
	v.SetDefault("agent.type", "claude-code")
	v.SetDefault("agent.max_concurrency", 3)
	v.SetDefault("agent.task_retry_limit", 3)
	v.SetDefault("agent.approval_timeout", "24h")

	// Clarity defaults
	v.SetDefault("clarity.threshold", 60)
	v.SetDefault("clarity.auto_proceed_threshold", 80)
	v.SetDefault("clarity.force_clarify_threshold", 40)

	// Complexity defaults
	v.SetDefault("complexity.threshold", 70)
	v.SetDefault("complexity.force_design_confirm", false)

	// Audit defaults
	v.SetDefault("audit.enabled", true)
	v.SetDefault("audit.retention_days", 90)
	v.SetDefault("audit.max_output_length", 1000)

	// Failure defaults
	v.SetDefault("failure.retry.max_retries", 3)
	v.SetDefault("failure.retry.initial_backoff", "5s")
	v.SetDefault("failure.retry.max_backoff", "60s")
	v.SetDefault("failure.retry.backoff_multiplier", 2.0)

	v.SetDefault("failure.rate_limit.enabled", true)
	v.SetDefault("failure.rate_limit.wait_enabled", true)
	v.SetDefault("failure.rate_limit.max_wait_duration", "30m")
	v.SetDefault("failure.rate_limit.notify_threshold", 10)

	v.SetDefault("failure.timeout.clarification_agent", "5m")
	v.SetDefault("failure.timeout.design_analysis", "10m")
	v.SetDefault("failure.timeout.design_generation", "15m")
	v.SetDefault("failure.timeout.task_breakdown", "10m")
	v.SetDefault("failure.timeout.task_execution", "30m")
	v.SetDefault("failure.timeout.test_run", "10m")
	v.SetDefault("failure.timeout.pr_creation", "5m")
	v.SetDefault("failure.timeout.approval_wait", "24h")
	v.SetDefault("failure.timeout.session_max_duration", "72h")

	v.SetDefault("failure.queue.max_length", 10)
	v.SetDefault("failure.queue.priority_enabled", true)
	v.SetDefault("failure.queue.timeout_on_queue", "1h")

	v.SetDefault("failure.test_environment.skip_if_no_tests", true)
	v.SetDefault("failure.test_environment.skip_if_unavailable", false)
	v.SetDefault("failure.test_environment.check_interval", "5m")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.outputs", []map[string]any{
		{"type": "console", "format": "json"},
	})
	v.SetDefault("logging.encoder.time_format", "iso8601")
	v.SetDefault("logging.encoder.duration_format", "string")
	v.SetDefault("logging.encoder.level_format", "lowercase")
	v.SetDefault("logging.caller.enabled", false)
	v.SetDefault("logging.caller.skip", 0)
	v.SetDefault("logging.stacktrace.enabled", true)
	v.SetDefault("logging.stacktrace.level", "error")

	// Redis defaults
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
}

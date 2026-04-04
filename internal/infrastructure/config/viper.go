package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "config",
		Provides: []string{"*config.Config"},
		Invokes:  []string{},
		Depends:  []string{},
	})
}

// Module provides the config module for Fx.
var Module = fx.Module("config",
	fx.Provide(NewConfig),
)

// NewConfig creates a Config instance from configuration file and environment variables.
func NewConfig() (*Config, error) {
	v := viper.New()

	// Set config file details
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/litchi")

	// Enable environment variable override
	v.AutomaticEnv()
	v.SetEnvPrefix("LITCHI")

	// Bind specific env vars.
	// Errors are intentionally ignored - if binding fails, values fallback to config file.
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("github.token", "GITHUB_TOKEN")
	_ = v.BindEnv("github.webhook_secret", "GITHUB_WEBHOOK_SECRET")
	_ = v.BindEnv("github.app_id", "GITHUB_APP_ID")
	_ = v.BindEnv("github.private_key_path", "GITHUB_PRIVATE_KEY_PATH")

	// Set defaults
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if errors.Is(err, viper.ConfigFileNotFoundError{}) {
			// No config file found, use defaults and env vars
		} else {
			return nil, fmt.Errorf("config file read error: %w", err)
		}
	}

	// Unmarshal to struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
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

	// Webhook defaults
	v.SetDefault("webhook.idempotency.enabled", true)
	v.SetDefault("webhook.idempotency.ttl", "24h")
	v.SetDefault("webhook.idempotency.auto_cleanup", true)
	v.SetDefault("webhook.idempotency.cleanup_interval", "1h")

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
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")

	// Redis defaults
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)
}

// GetConfigPath returns the config file path from environment or default.
func GetConfigPath() string {
	if path := os.Getenv("LITCHI_CONFIG_PATH"); path != "" {
		return path
	}
	return "./config.yaml"
}
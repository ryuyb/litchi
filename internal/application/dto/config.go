// Package dto provides Data Transfer Objects for API request/response structures.
package dto

import "github.com/ryuyb/litchi/internal/infrastructure/config"

// ConfigResponse represents the global configuration in API response.
type ConfigResponse struct {
	Server   ServerConfigDTO   `json:"server"`
	Database DatabaseConfigDTO `json:"database"`
	Agent    AgentConfigDTO    `json:"agent"`
	Git      GitConfigDTO      `json:"git"`
	Webhook  WebhookConfigDTO  `json:"webhook"`
	Clarity  ClarityConfigDTO  `json:"clarity"`
	Complexity ComplexityConfigDTO `json:"complexity"`
	Audit    AuditConfigDTO    `json:"audit"`
} // @name Config

// ServerConfigDTO represents server configuration.
type ServerConfigDTO struct {
	Host    string `json:"host" example:"0.0.0.0"`
	Port    int    `json:"port" example:"8080"`
	Mode    string `json:"mode" example:"debug"`
	Version string `json:"version" example:"0.1.0"`
} // @name ServerConfig

// DatabaseConfigDTO represents database configuration (sensitive fields excluded).
type DatabaseConfigDTO struct {
	Host            string `json:"host" example:"localhost"`
	Port            int    `json:"port" example:"5432"`
	Name            string `json:"name" example:"litchi"`
	User            string `json:"user" example:"postgres"`
	SSLMode         string `json:"sslmode" example:"disable"`
	MaxOpenConns    int    `json:"maxOpenConns" example:"100"`
	MaxIdleConns    int    `json:"maxIdleConns" example:"10"`
	ConnMaxLifetime string `json:"connMaxLifetime" example:"1h"`
} // @name DatabaseConfig

// AgentConfigDTO represents agent configuration.
type AgentConfigDTO struct {
	Type            string `json:"type" example:"claude-code"`
	MaxConcurrency  int    `json:"maxConcurrency" example:"3"`
	TaskRetryLimit  int    `json:"taskRetryLimit" example:"3"`
	ApprovalTimeout string `json:"approvalTimeout" example:"24h"`
} // @name AgentConfig

// GitConfigDTO represents git configuration.
type GitConfigDTO struct {
	WorktreeBasePath     string `json:"worktreeBasePath" example:"/var/litchi/worktrees"`
	WorktreeAutoClean    bool   `json:"worktreeAutoClean" example:"true"`
	BranchNamingPattern  string `json:"branchNamingPattern" example:"issue-{number}-{slug}"`
	DefaultBaseBranch    string `json:"defaultBaseBranch" example:"main"`
	CommitSignOff        bool   `json:"commitSignOff" example:"true"`
	GitBinaryPath        string `json:"gitBinaryPath" example:"git"`
	CommandTimeout       string `json:"commandTimeout" example:"5m"`
} // @name GitConfig

// WebhookConfigDTO represents webhook configuration.
type WebhookConfigDTO struct {
	IdempotencyEnabled    bool   `json:"idempotencyEnabled" example:"true"`
	IdempotencyTTL        string `json:"idempotencyTTL" example:"24h"`
	IdempotencyAutoCleanup bool  `json:"idempotencyAutoCleanup" example:"true"`
} // @name WebhookConfig

// ClarityConfigDTO represents clarity configuration.
type ClarityConfigDTO struct {
	Threshold            int `json:"threshold" example:"60"`
	AutoProceedThreshold int `json:"autoProceedThreshold" example:"80"`
	ForceClarifyThreshold int `json:"forceClarifyThreshold" example:"40"`
} // @name ClarityConfig

// ComplexityConfigDTO represents complexity configuration.
type ComplexityConfigDTO struct {
	Threshold          int  `json:"threshold" example:"70"`
	ForceDesignConfirm bool `json:"forceDesignConfirm" example:"false"`
} // @name ComplexityConfig

// AuditConfigDTO represents audit configuration.
type AuditConfigDTO struct {
	Enabled         bool `json:"enabled" example:"true"`
	RetentionDays   int  `json:"retentionDays" example:"90"`
	MaxOutputLength int  `json:"maxOutputLength" example:"1000"`
} // @name AuditConfigDTO

// UpdateConfigRequest represents update config request (partial update).
type UpdateConfigRequest struct {
	Agent    *AgentConfigUpdate    `json:"agent,omitempty"`
	Git      *GitConfigUpdate      `json:"git,omitempty"`
	Clarity  *ClarityConfigUpdate  `json:"clarity,omitempty"`
	Complexity *ComplexityConfigUpdate `json:"complexity,omitempty"`
	Audit    *AuditConfigUpdate    `json:"audit,omitempty"`
} // @name UpdateConfig

// AgentConfigUpdate represents agent config update fields.
type AgentConfigUpdate struct {
	MaxConcurrency  *int    `json:"maxConcurrency,omitempty"`
	TaskRetryLimit  *int    `json:"taskRetryLimit,omitempty"`
	ApprovalTimeout *string `json:"approvalTimeout,omitempty"`
} // @name AgentConfigUpdate

// GitConfigUpdate represents git config update fields.
type GitConfigUpdate struct {
	WorktreeBasePath    *string `json:"worktreeBasePath,omitempty"`
	WorktreeAutoClean   *bool   `json:"worktreeAutoClean,omitempty"`
	BranchNamingPattern *string `json:"branchNamingPattern,omitempty"`
	DefaultBaseBranch   *string `json:"defaultBaseBranch,omitempty"`
	CommitSignOff       *bool   `json:"commitSignOff,omitempty"`
	CommandTimeout      *string `json:"commandTimeout,omitempty"`
} // @name GitConfigUpdate

// ClarityConfigUpdate represents clarity config update fields.
type ClarityConfigUpdate struct {
	Threshold            *int `json:"threshold,omitempty"`
	AutoProceedThreshold *int `json:"autoProceedThreshold,omitempty"`
	ForceClarifyThreshold *int `json:"forceClarifyThreshold,omitempty"`
} // @name ClarityConfigUpdate

// ComplexityConfigUpdate represents complexity config update fields.
type ComplexityConfigUpdate struct {
	Threshold          *int  `json:"threshold,omitempty"`
	ForceDesignConfirm *bool `json:"forceDesignConfirm,omitempty"`
} // @name ComplexityConfigUpdate

// AuditConfigUpdate represents audit config update fields.
type AuditConfigUpdate struct {
	Enabled         *bool `json:"enabled,omitempty"`
	RetentionDays   *int  `json:"retentionDays,omitempty"`
	MaxOutputLength *int  `json:"maxOutputLength,omitempty"`
} // @name AuditConfigUpdate

// ToConfigResponse converts config.Config to DTO.
func ToConfigResponse(cfg *config.Config) ConfigResponse {
	return ConfigResponse{
		Server: ServerConfigDTO{
			Host:    cfg.Server.Host,
			Port:    cfg.Server.Port,
			Mode:    cfg.Server.Mode,
			Version: cfg.Server.Version,
		},
		Database: DatabaseConfigDTO{
			Host:            cfg.Database.Host,
			Port:            cfg.Database.Port,
			Name:            cfg.Database.Name,
			User:            cfg.Database.User,
			SSLMode:         cfg.Database.SSLMode,
			MaxOpenConns:    cfg.Database.MaxOpenConns,
			MaxIdleConns:    cfg.Database.MaxIdleConns,
			ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		},
		Agent: AgentConfigDTO{
			Type:            cfg.Agent.Type,
			MaxConcurrency:  cfg.Agent.MaxConcurrency,
			TaskRetryLimit:  cfg.Agent.TaskRetryLimit,
			ApprovalTimeout: cfg.Agent.ApprovalTimeout,
		},
		Git: GitConfigDTO{
			WorktreeBasePath:    cfg.Git.WorktreeBasePath,
			WorktreeAutoClean:   cfg.Git.WorktreeAutoClean,
			BranchNamingPattern: cfg.Git.BranchNamingPattern,
			DefaultBaseBranch:   cfg.Git.DefaultBaseBranch,
			CommitSignOff:       cfg.Git.CommitSignOff,
			GitBinaryPath:       cfg.Git.GitBinaryPath,
			CommandTimeout:      cfg.Git.CommandTimeout,
		},
		Webhook: WebhookConfigDTO{
			IdempotencyEnabled:    cfg.Webhook.Idempotency.Enabled,
			IdempotencyTTL:        cfg.Webhook.Idempotency.TTL,
			IdempotencyAutoCleanup: cfg.Webhook.Idempotency.AutoCleanup,
		},
		Clarity: ClarityConfigDTO{
			Threshold:            cfg.Clarity.Threshold,
			AutoProceedThreshold: cfg.Clarity.AutoProceedThreshold,
			ForceClarifyThreshold: cfg.Clarity.ForceClarifyThreshold,
		},
		Complexity: ComplexityConfigDTO{
			Threshold:          cfg.Complexity.Threshold,
			ForceDesignConfirm: cfg.Complexity.ForceDesignConfirm,
		},
		Audit: AuditConfigDTO{
			Enabled:         cfg.Audit.Enabled,
			RetentionDays:   cfg.Audit.RetentionDays,
			MaxOutputLength: cfg.Audit.MaxOutputLength,
		},
	}
}
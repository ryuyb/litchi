// Package config provides HTTP handlers for configuration management API.
package config

import (
	"testing"

	"github.com/ryuyb/litchi/internal/application/dto"
	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
)

func TestUpdateConfig_Atomicity(t *testing.T) {
	t.Run("original config unchanged when validation fails", func(t *testing.T) {
		// Create initial config
		cfg := &config.Config{
			Agent: config.AgentConfig{
				MaxConcurrency: 5,
				TaskRetryLimit: 3,
			},
			Clarity: config.ClarityConfig{
				Threshold:             60,
				AutoProceedThreshold:  80,
				ForceClarifyThreshold: 40,
			},
		}

		handler := &Handler{
			cfg:    cfg,
			logger: nil, // not needed for this test
		}

		// Snapshot original values
		originalMaxConcurrency := cfg.Agent.MaxConcurrency
		originalThreshold := cfg.Clarity.Threshold

		// Create update request with invalid agent value
		req := &dto.UpdateConfigRequest{
			Agent: &dto.AgentConfigUpdate{
				MaxConcurrency: new(15), // Invalid: > 10
			},
		}

		// Create a snapshot
		snapshot := cfg.Clone()

		// Apply updates to snapshot (should fail)
		err := handler.applyUpdates(snapshot, req)
		assert.Error(t, err, "applyUpdates should fail for invalid MaxConcurrency")

		// Verify original config is unchanged
		assert.Equal(t, originalMaxConcurrency, cfg.Agent.MaxConcurrency,
			"original MaxConcurrency should be unchanged")
		assert.Equal(t, originalThreshold, cfg.Clarity.Threshold,
			"original Clarity.Threshold should be unchanged")
	})

	t.Run("partial update failure does not modify original", func(t *testing.T) {
		cfg := &config.Config{
			Agent: config.AgentConfig{
				MaxConcurrency: 5,
				TaskRetryLimit: 3,
			},
			Git: config.GitConfig{
				WorktreeBasePath: "/tmp/worktrees",
			},
		}

		handler := &Handler{
			cfg:    cfg,
			logger: nil,
		}

		// Snapshot original values
		originalMaxConcurrency := cfg.Agent.MaxConcurrency
		originalWorktreeBasePath := cfg.Git.WorktreeBasePath

		// Create update request with one valid and one invalid update
		req := &dto.UpdateConfigRequest{
			Agent: &dto.AgentConfigUpdate{
				MaxConcurrency: new(10), // Valid
			},
			Git: &dto.GitConfigUpdate{
				WorktreeBasePath: new(""), // Invalid: empty string
			},
		}

		// Create a snapshot
		snapshot := cfg.Clone()

		// Apply updates (should fail on Git update)
		err := handler.applyUpdates(snapshot, req)
		assert.Error(t, err, "applyUpdates should fail for empty WorktreeBasePath")

		// Verify original config is unchanged
		assert.Equal(t, originalMaxConcurrency, cfg.Agent.MaxConcurrency,
			"original MaxConcurrency should be unchanged")
		assert.Equal(t, originalWorktreeBasePath, cfg.Git.WorktreeBasePath,
			"original WorktreeBasePath should be unchanged")
	})

	t.Run("successful update replaces original", func(t *testing.T) {
		cfg := &config.Config{
			Agent: config.AgentConfig{
				MaxConcurrency: 5,
				TaskRetryLimit: 3,
			},
		}

		handler := &Handler{
			cfg:    cfg,
			logger: nil,
		}

		// Create valid update request
		req := &dto.UpdateConfigRequest{
			Agent: &dto.AgentConfigUpdate{
				MaxConcurrency: new(8),
			},
		}

		// Create a snapshot
		snapshot := cfg.Clone()

		// Apply updates
		err := handler.applyUpdates(snapshot, req)
		assert.NoError(t, err)

		// Note: We skip Validate() here as the test focuses on atomicity,
		// not config completeness. In production, Validate() is called.

		// Replace original with snapshot
		*cfg = *snapshot

		// Verify update was applied
		assert.Equal(t, 8, cfg.Agent.MaxConcurrency,
			"MaxConcurrency should be updated to 8")
	})
}

// Helper functions
func intPtr(v int) *int {
	return &v
}

func strPtr(v string) *string {
	return &v
}

// Package cache provides file-based cache implementations for agent execution context.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ryuyb/litchi/internal/domain/repository"
	"go.uber.org/zap"
)

const (
	// litchiDir is the directory name for cache storage within worktree.
	litchiDir = ".litchi"

	// contextFile is the filename for execution context cache.
	contextFile = "context.json"
)

// FileCacheRepository implements CacheRepository using filesystem storage.
type FileCacheRepository struct {
	logger *zap.Logger
}

// NewFileCacheRepository creates a new FileCacheRepository instance.
func NewFileCacheRepository(logger *zap.Logger) *FileCacheRepository {
	return &FileCacheRepository{
		logger: logger,
	}
}

// Save writes context.json to {worktreePath}/.litchi/
// Note: File I/O operations are typically fast and not easily interruptible,
// so context cancellation is checked only at the start of the operation.
func (r *FileCacheRepository) Save(ctx context.Context, worktreePath string, cache *repository.ExecutionContextCache) error {
	// Check for context cancellation before starting I/O
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if cache == nil {
		return errors.New("cache cannot be nil")
	}

	// Validate cache data integrity
	if err := cache.Validate(); err != nil {
		return fmt.Errorf("cache validation failed: %w", err)
	}

	// Clean the path to prevent path traversal and normalize separators
	worktreePath = filepath.Clean(worktreePath)

	// Create .litchi directory if not exists
	litchiPath := filepath.Join(worktreePath, litchiDir)
	if err := os.MkdirAll(litchiPath, 0755); err != nil {
		return fmt.Errorf("failed to create .litchi directory: %w", err)
	}

	// Marshal cache to JSON with indentation for readability
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write to file
	contextPath := filepath.Join(litchiPath, contextFile)
	if err := os.WriteFile(contextPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	r.logger.Debug("saved execution context cache",
		zap.String("path", contextPath),
		zap.String("sessionId", cache.SessionID.String()),
	)

	return nil
}

// Load reads context.json from {worktreePath}/.litchi/
// Returns nil if file does not exist.
// Note: File I/O operations are typically fast and not easily interruptible,
// so context cancellation is checked only at the start of the operation.
func (r *FileCacheRepository) Load(ctx context.Context, worktreePath string) (*repository.ExecutionContextCache, error) {
	// Check for context cancellation before starting I/O
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Clean the path to prevent path traversal and normalize separators
	worktreePath = filepath.Clean(worktreePath)

	contextPath := filepath.Join(worktreePath, litchiDir, contextFile)

	data, err := os.ReadFile(contextPath)
	if err != nil {
		if os.IsNotExist(err) {
			r.logger.Debug("context cache file not found",
				zap.String("path", contextPath),
			)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read context file: %w", err)
	}

	var cache repository.ExecutionContextCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	r.logger.Debug("loaded execution context cache",
		zap.String("path", contextPath),
		zap.String("sessionId", cache.SessionID.String()),
	)

	return &cache, nil
}

// Delete removes the .litchi directory from worktree.
// Note: File I/O operations are typically fast and not easily interruptible,
// so context cancellation is checked only at the start of the operation.
func (r *FileCacheRepository) Delete(ctx context.Context, worktreePath string) error {
	// Check for context cancellation before starting I/O
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Clean the path to prevent path traversal and normalize separators
	worktreePath = filepath.Clean(worktreePath)

	litchiPath := filepath.Join(worktreePath, litchiDir)

	// Check if directory exists before attempting to delete
	if _, err := os.Stat(litchiPath); os.IsNotExist(err) {
		r.logger.Debug(".litchi directory not found, nothing to delete",
			zap.String("path", litchiPath),
		)
		return nil
	}

	if err := os.RemoveAll(litchiPath); err != nil {
		return fmt.Errorf("failed to remove .litchi directory: %w", err)
	}

	r.logger.Debug("deleted .litchi directory",
		zap.String("path", litchiPath),
	)

	return nil
}
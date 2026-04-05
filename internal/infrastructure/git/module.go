// Package git provides Git operations via Fx module.
package git

import (
	"time"

	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/ryuyb/litchi/internal/pkg/fxutil"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func init() {
	fxutil.RegisterModule(fxutil.ModuleInfo{
		Name:     "git",
		Provides: []string{"*CommandExecutor", "GitClient", "BranchService", "WorktreeService", "CommitService", "ConflictDetector"},
		Invokes:  []string{},
		Depends:  []string{"*config.Config", "*zap.Logger"},
	})
}

// Module provides Git operations via Fx.
var Module = fx.Module("git",
	fx.Provide(
		NewCommandExecutorFromConfig,
		NewGitClientFromDeps,
		NewBranchServiceFromDeps,
		NewWorktreeServiceFromDeps,
		NewCommitServiceFromDeps,
		NewConflictDetectorFromDeps,
	),
)

// CommandExecutorConfigParams contains dependencies for creating a CommandExecutor from config.
type CommandExecutorConfigParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

// NewCommandExecutorFromConfig creates a new command executor with config.
func NewCommandExecutorFromConfig(p CommandExecutorConfigParams) *CommandExecutor {
	timeout := 5 * time.Minute
	if p.Config.Git.CommandTimeout != "" {
		if parsed, err := time.ParseDuration(p.Config.Git.CommandTimeout); err == nil {
			timeout = parsed
		}
	}

	return NewCommandExecutor(CommandExecutorParams{
		GitBinary: p.Config.Git.GitBinaryPath,
		Timeout:   timeout,
		Logger:    p.Logger,
	})
}

// GitClientDeps contains dependencies for creating a GitClient.
type GitClientDeps struct {
	fx.In

	Executor *CommandExecutor
	Logger   *zap.Logger
}

// NewGitClientFromDeps creates a new GitClient with dependencies.
func NewGitClientFromDeps(p GitClientDeps) GitClient {
	return NewGitClient(GitClientParams{
		Executor: p.Executor,
		Logger:   p.Logger,
	})
}

// BranchServiceDeps contains dependencies for creating a BranchService.
type BranchServiceDeps struct {
	fx.In

	Executor *CommandExecutor
	Client   GitClient
	Logger   *zap.Logger
}

// NewBranchServiceFromDeps creates a new BranchService with dependencies.
func NewBranchServiceFromDeps(p BranchServiceDeps) BranchService {
	return NewBranchService(BranchServiceParams{
		Executor: p.Executor,
		Client:   p.Client,
		Logger:   p.Logger,
	})
}

// WorktreeServiceDeps contains dependencies for creating a WorktreeService.
type WorktreeServiceDeps struct {
	fx.In

	Executor  *CommandExecutor
	Client    GitClient
	BranchSvc BranchService
	Logger    *zap.Logger
}

// NewWorktreeServiceFromDeps creates a new WorktreeService with dependencies.
func NewWorktreeServiceFromDeps(p WorktreeServiceDeps) WorktreeService {
	return NewWorktreeService(WorktreeServiceParams{
		Executor:  p.Executor,
		Client:    p.Client,
		BranchSvc: p.BranchSvc,
		Logger:    p.Logger,
	})
}

// CommitServiceDeps contains dependencies for creating a CommitService.
type CommitServiceDeps struct {
	fx.In

	Executor *CommandExecutor
	Client   GitClient
	Logger   *zap.Logger
}

// NewCommitServiceFromDeps creates a new CommitService with dependencies.
func NewCommitServiceFromDeps(p CommitServiceDeps) CommitService {
	return NewCommitService(CommitServiceParams{
		Executor: p.Executor,
		Client:   p.Client,
		Logger:   p.Logger,
	})
}

// ConflictDetectorDeps contains dependencies for creating a ConflictDetector.
type ConflictDetectorDeps struct {
	fx.In

	Executor *CommandExecutor
	Client   GitClient
	Logger   *zap.Logger
}

// NewConflictDetectorFromDeps creates a new ConflictDetector with dependencies.
func NewConflictDetectorFromDeps(p ConflictDetectorDeps) ConflictDetector {
	return NewConflictDetector(ConflictDetectorParams{
		Executor: p.Executor,
		Client:   p.Client,
		Logger:   p.Logger,
	})
}
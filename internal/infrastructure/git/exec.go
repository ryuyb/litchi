// Package git provides Git operations using command-line execution.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// CommandExecutor wraps Git command execution with timeout and error handling.
type CommandExecutor struct {
	gitBinary string
	timeout   time.Duration
	logger    *zap.Logger
}

// CommandExecutorParams contains dependencies for creating a CommandExecutor.
type CommandExecutorParams struct {
	GitBinary string
	Timeout   time.Duration
	Logger    *zap.Logger
}

// NewCommandExecutor creates a new command executor.
func NewCommandExecutor(p CommandExecutorParams) *CommandExecutor {
	gitBinary := p.GitBinary
	if gitBinary == "" {
		gitBinary = "git"
	}

	timeout := p.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &CommandExecutor{
		gitBinary: gitBinary,
		timeout:   timeout,
		logger:    logger.Named("git-exec"),
	}
}

// ExecResult contains the result of a Git command execution.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Exec executes a Git command in the specified working directory.
func (e *CommandExecutor) Exec(ctx context.Context, workDir string, args ...string) (*ExecResult, error) {
	return e.ExecWithTimeout(ctx, workDir, e.timeout, args...)
}

// ExecWithTimeout executes a Git command with a specific timeout.
func (e *CommandExecutor) ExecWithTimeout(ctx context.Context, workDir string, timeout time.Duration, args ...string) (*ExecResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.gitBinary, args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	e.logger.Debug("executing git command",
		zap.String("workdir", workDir),
		zap.Strings("args", args),
		zap.Duration("timeout", timeout),
	)

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	result := &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	// Log the result
	e.logger.Debug("git command completed",
		zap.Strings("args", args),
		zap.Duration("duration", duration),
		zap.Int("exit_code", result.ExitCode),
	)

	// Handle context timeout
	if ctx.Err() == context.DeadlineExceeded {
		return nil, errors.New(errors.ErrGitCommandFailed).
			WithDetail(fmt.Sprintf("command timeout after %v", timeout)).
			WithContext("command", strings.Join(args, " "))
	}

	// Handle command error
	if err != nil {
		return result, e.wrapCommandError(err, args, result)
	}

	return result, nil
}

// ExecSimple executes a Git command and returns only stdout (for simple operations).
func (e *CommandExecutor) ExecSimple(ctx context.Context, workDir string, args ...string) (string, error) {
	result, err := e.Exec(ctx, workDir, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// wrapCommandError wraps a command execution error into a Litchi error.
func (e *CommandExecutor) wrapCommandError(err error, args []string, result *ExecResult) error {
	cmdStr := strings.Join(args, " ")
	stderr := strings.TrimSpace(result.Stderr)

	// Check for authentication errors
	if strings.Contains(stderr, "Authentication failed") ||
		strings.Contains(stderr, "Permission denied") ||
		strings.Contains(stderr, "could not read Username") ||
		strings.Contains(stderr, "Invalid username or password") {
		return errors.Wrap(errors.ErrGitAuthentication, err).
			WithDetail(stderr).
			WithContext("command", cmdStr)
	}

	// Check for branch-related errors
	if strings.Contains(stderr, "already exists") {
		if args[0] == "branch" {
			return errors.Wrap(errors.ErrGitBranchExists, err).
				WithDetail(stderr).
				WithContext("command", cmdStr)
		}
		if args[0] == "worktree" {
			return errors.Wrap(errors.ErrGitWorktreePathExists, err).
				WithDetail(stderr).
				WithContext("command", cmdStr)
		}
	}

	// Check for not found errors
	if strings.Contains(stderr, "not found") || strings.Contains(stderr, "does not exist") {
		if args[0] == "branch" {
			return errors.Wrap(errors.ErrGitBranchNotFound, err).
				WithDetail(stderr).
				WithContext("command", cmdStr)
		}
		if args[0] == "worktree" {
			return errors.Wrap(errors.ErrGitWorktreeNotFound, err).
				WithDetail(stderr).
				WithContext("command", cmdStr)
		}
		return errors.Wrap(errors.ErrGitRepoNotFound, err).
			WithDetail(stderr).
			WithContext("command", cmdStr)
	}

	// Check for merge conflicts
	if strings.Contains(stderr, "CONFLICT") || strings.Contains(stderr, "Merge conflict") {
		return errors.Wrap(errors.ErrGitMergeConflict, err).
			WithDetail(stderr).
			WithContext("command", cmdStr)
	}

	// Check for worktree locked
	if strings.Contains(stderr, "is locked") {
		return errors.Wrap(errors.ErrGitWorktreeLocked, err).
			WithDetail(stderr).
			WithContext("command", cmdStr)
	}

	// Default: wrap as generic Git command error
	return errors.Wrap(errors.ErrGitCommandFailed, err).
		WithDetail(stderr).
		WithContext("command", cmdStr).
		WithContext("exit_code", result.ExitCode)
}

// SetTimeout updates the default command timeout.
func (e *CommandExecutor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}

// SetGitBinary updates the Git binary path.
func (e *CommandExecutor) SetGitBinary(gitBinary string) {
	e.gitBinary = gitBinary
}
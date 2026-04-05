// Package command provides command execution utilities for validation.
package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Executor executes validation commands with timeout and error handling.
type Executor struct {
	logger  *zap.Logger
	timeout time.Duration
}

// ExecutorParams contains dependencies for Executor.
type ExecutorParams struct {
	Logger  *zap.Logger
	Timeout time.Duration
}

// NewExecutor creates a new command executor.
func NewExecutor(p ExecutorParams) *Executor {
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Executor{
		logger:  logger.Named("command-exec"),
		timeout: timeout,
	}
}

// ExecResult contains the result of a command execution.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration int64 // milliseconds
}

// Exec executes a command in the specified working directory.
func (e *Executor) Exec(ctx context.Context, workDir string, cmd string, args []string, env map[string]string, timeoutSeconds int) (*ExecResult, error) {
	// Security: validate command to prevent injection
	if err := validateCommand(cmd); err != nil {
		return nil, fmt.Errorf("invalid command: %w", err)
	}

	// Determine timeout
	execTimeout := e.timeout
	if timeoutSeconds > 0 {
		execTimeout = time.Duration(timeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	// Build command
	execCmd := exec.CommandContext(ctx, cmd, args...)
	execCmd.Dir = workDir

	// Set environment variables
	if len(env) > 0 {
		execCmd.Env = os.Environ()
		for k, v := range env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	e.logger.Debug("executing command",
		zap.String("workdir", workDir),
		zap.String("command", cmd),
		zap.Strings("args", args),
		zap.Duration("timeout", execTimeout),
	)

	startTime := time.Now()
	err := execCmd.Run()
	duration := time.Since(startTime).Milliseconds()

	result := &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
		Duration: duration,
	}

	if execCmd.ProcessState != nil {
		result.ExitCode = execCmd.ProcessState.ExitCode()
	}

	e.logger.Debug("command completed",
		zap.String("command", cmd),
		zap.Int64("duration_ms", duration),
		zap.Int("exit_code", result.ExitCode),
	)

	// Handle timeout - explicitly kill the process
	if ctx.Err() == context.DeadlineExceeded {
		if execCmd.Process != nil {
			execCmd.Process.Kill()
		}
		return nil, fmt.Errorf("command timeout after %v: %s", execTimeout, strings.Join(args, " "))
	}

	// Handle command error - validation tools may fail (exit code != 0)
	// This is expected for lint/test failures, so we return result with error info
	if err != nil {
		// Don't wrap as error, just return result with exit code
		// The validator will handle the exit code appropriately
		return result, nil
	}

	return result, nil
}

// ExecWithOutput executes a command and returns combined stdout+stderr.
func (e *Executor) ExecWithOutput(ctx context.Context, workDir string, cmd string, args []string, env map[string]string, timeoutSeconds int) (string, int64, error) {
	result, err := e.Exec(ctx, workDir, cmd, args, env, timeoutSeconds)
	if err != nil {
		return "", 0, err
	}

	// Combine stdout and stderr for tools that output to both
	output := result.Stdout
	if result.Stderr != "" {
		output = output + "\n" + result.Stderr
	}

	return strings.TrimSpace(output), result.Duration, nil
}

// CheckFileExists checks if a file exists in the working directory.
func (e *Executor) CheckFileExists(workDir string, filename string) bool {
	path := filepath.Join(workDir, filename)
	_, err := os.Stat(path)
	return err == nil
}

// FindFiles finds files matching a pattern in the working directory.
// Skips common large directories like .git, node_modules, vendor.
func (e *Executor) FindFiles(workDir string, pattern string) ([]string, error) {
	// Directories to skip for performance
	skipDirs := map[string]bool{
		".git":         true,
		".svn":         true,
		".hg":          true,
		"node_modules": true,
		"vendor":       true,
		"venv":         true,
		".venv":        true,
		"__pycache__":  true,
		".idea":        true,
		".vscode":      true,
		"dist":         true,
		"build":        true,
		"target":       true,
		"bin":          true,
		"pkg":          true,
	}

	matches := []string{}
	err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		matched, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return err
		}
		if matched {
			relPath, err := filepath.Rel(workDir, path)
			if err != nil {
				return err
			}
			matches = append(matches, relPath)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find files failed: %w", err)
	}
	return matches, nil
}

// ReadFile reads a file content from the working directory.
func (e *Executor) ReadFile(workDir string, filename string) ([]byte, error) {
	path := filepath.Join(workDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	return content, nil
}

// SetTimeout updates the default command timeout.
func (e *Executor) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}

// validateCommand validates the command string for security.
// Prevents command injection by disallowing shell metacharacters.
func validateCommand(cmd string) error {
	if cmd == "" {
		return fmt.Errorf("command cannot be empty")
	}
	// Prevent command injection - disallow shell metacharacters
	dangerousChars := ";&|`$()<>\\"
	if strings.ContainsAny(cmd, dangerousChars) {
		return fmt.Errorf("command contains dangerous characters: %s", dangerousChars)
	}
	return nil
}
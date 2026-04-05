package claude

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// ProcessExecutor executes Claude Code processes.
type ProcessExecutor struct {
	logger       *zap.Logger
	runningProcs map[uuid.UUID]*RunningProcess
	mu           sync.RWMutex
}

// RunningProcess represents a running process.
type RunningProcess struct {
	SessionID   uuid.UUID
	Cmd         *exec.Cmd
	CancelFunc  context.CancelFunc
	StartTime   time.Time
	Stdout      *bytes.Buffer
	Stderr      *bytes.Buffer
	Done        chan struct{}
	WorktreeDir string
}

// ProcessResult is the result of process execution.
type ProcessResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// NewProcessExecutor creates a new process executor.
func NewProcessExecutor(logger *zap.Logger) *ProcessExecutor {
	return &ProcessExecutor{
		logger:       logger.Named("claude-process"),
		runningProcs: make(map[uuid.UUID]*RunningProcess),
	}
}

// Execute executes the command and manages the process lifecycle.
func (e *ProcessExecutor) Execute(ctx context.Context, cmd *ClaudeCommand, sessionID uuid.UUID) (*ProcessResult, error) {
	// Ensure timeout is set (default to 30 minutes if zero or negative)
	timeout := cmd.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	// Create child context for timeout control
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build exec.Cmd
	execCmd := exec.CommandContext(execCtx, cmd.Binary, cmd.Args...)
	execCmd.Dir = cmd.WorkDir

	// Set environment variables
	execCmd.Env = os.Environ()
	for k, v := range cmd.Env {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set process group ID for killing process tree on cancel (platform-specific)
	execCmd.SysProcAttr = &syscall.SysProcAttr{}
	setProcessGroupAttr(execCmd.SysProcAttr)

	// Create output buffers
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	// Use MultiWriter for simultaneous buffer and stream handling
	stdoutHandler := NewStreamHandler(e.logger, "stdout")
	stderrHandler := NewStreamHandler(e.logger, "stderr")

	execCmd.Stdout = io.MultiWriter(stdoutBuf, stdoutHandler)
	execCmd.Stderr = io.MultiWriter(stderrBuf, stderrHandler)

	// Record running process
	proc := &RunningProcess{
		SessionID:   sessionID,
		Cmd:         execCmd,
		CancelFunc:  cancel,
		StartTime:   time.Now(),
		Stdout:      stdoutBuf,
		Stderr:      stderrBuf,
		Done:        make(chan struct{}),
		WorktreeDir: cmd.WorkDir,
	}

	e.mu.Lock()
	e.runningProcs[sessionID] = proc
	e.mu.Unlock()

	// Start process
	startTime := time.Now()
	err := execCmd.Start()
	if err != nil {
		e.cleanupProcess(sessionID)
		return nil, errors.Wrap(errors.ErrAgentProcessCrash, err).
			WithDetail("failed to start claude process")
	}

	e.logger.Info("claude process started",
		zap.String("sessionId", sessionID.String()),
		zap.String("command", cmd.String()),
		zap.String("workDir", cmd.WorkDir),
	)

	// Wait for process completion
	waitErr := execCmd.Wait()
	duration := time.Since(startTime)

	close(proc.Done)
	e.cleanupProcess(sessionID)

	// Build result
	result := &ProcessResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		Duration: duration,
	}

	if execCmd.ProcessState != nil {
		result.ExitCode = execCmd.ProcessState.ExitCode()
	}

	// Handle timeout
	if execCtx.Err() == context.DeadlineExceeded {
		e.logger.Warn("claude process timeout",
			zap.String("sessionId", sessionID.String()),
			zap.Duration("timeout", cmd.Timeout),
		)
		return result, errors.New(errors.ErrAgentTimeout).
			WithDetail(fmt.Sprintf("process timeout after %v", cmd.Timeout)).
			WithContext("sessionId", sessionID.String())
	}

	// Handle context cancellation (not timeout)
	if ctx.Err() == context.Canceled {
		e.logger.Info("claude process cancelled",
			zap.String("sessionId", sessionID.String()),
		)
		return result, errors.New(errors.ErrAgentExecutionFail).
			WithDetail("process cancelled by user").
			WithContext("sessionId", sessionID.String())
	}

	// Handle process crash
	if waitErr != nil {
		e.logger.Error("claude process crashed",
			zap.String("sessionId", sessionID.String()),
			zap.Error(waitErr),
			zap.Int("exitCode", result.ExitCode),
		)
		return result, errors.Wrap(errors.ErrAgentProcessCrash, waitErr).
			WithDetail(result.Stderr).
			WithContext("exitCode", result.ExitCode)
	}

	e.logger.Info("claude process completed",
		zap.String("sessionId", sessionID.String()),
		zap.Duration("duration", duration),
		zap.Int("exitCode", result.ExitCode),
	)

	return result, nil
}

// Cancel cancels a running process.
func (e *ProcessExecutor) Cancel(sessionID uuid.UUID) error {
	e.mu.Lock()
	proc, exists := e.runningProcs[sessionID]
	e.mu.Unlock()

	if !exists {
		return nil // Process doesn't exist or already completed
	}

	// Call cancel function
	proc.CancelFunc()

	// Kill process group (including all child processes) - platform-specific
	if proc.Cmd.Process != nil {
		// Send SIGTERM to process group (or just the process on Windows)
		if err := killProcessGroup(proc.Cmd.Process.Pid, syscall.SIGTERM); err != nil {
			e.logger.Warn("failed to send SIGTERM to process",
				zap.String("sessionId", sessionID.String()),
				zap.Error(err),
				zap.String("os", runtime.GOOS),
			)
		}
	}

	// Wait for process to end (max 5 seconds)
	select {
	case <-proc.Done:
		e.logger.Info("claude process cancelled",
			zap.String("sessionId", sessionID.String()),
		)
	case <-time.After(5 * time.Second):
		// Force kill
		if proc.Cmd.Process != nil {
			if err := killProcessGroup(proc.Cmd.Process.Pid, syscall.SIGKILL); err != nil {
				e.logger.Warn("failed to force kill process",
					zap.String("sessionId", sessionID.String()),
					zap.Error(err),
					zap.String("os", runtime.GOOS),
				)
			}
		}
		e.logger.Warn("claude process force killed after timeout",
			zap.String("sessionId", sessionID.String()),
		)
	}

	e.cleanupProcess(sessionID)
	return nil
}

// IsRunning checks if a process is running.
func (e *ProcessExecutor) IsRunning(sessionID uuid.UUID) bool {
	e.mu.RLock()
	_, exists := e.runningProcs[sessionID]
	e.mu.RUnlock()
	return exists
}

// GetProcess returns the running process info.
func (e *ProcessExecutor) GetProcess(sessionID uuid.UUID) (*RunningProcess, bool) {
	e.mu.RLock()
	proc, exists := e.runningProcs[sessionID]
	e.mu.RUnlock()
	return proc, exists
}

// cleanupProcess removes process record.
func (e *ProcessExecutor) cleanupProcess(sessionID uuid.UUID) {
	e.mu.Lock()
	delete(e.runningProcs, sessionID)
	e.mu.Unlock()
}

// Shutdown shuts down all running processes.
func (e *ProcessExecutor) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	procs := make([]*RunningProcess, 0, len(e.runningProcs))
	for _, proc := range e.runningProcs {
		procs = append(procs, proc)
	}
	e.runningProcs = make(map[uuid.UUID]*RunningProcess)
	e.mu.Unlock()

	if len(procs) == 0 {
		return nil
	}

	e.logger.Info("shutting down all claude processes", zap.Int("count", len(procs)))

	// Cancel all processes
	for _, proc := range procs {
		proc.CancelFunc()
		if proc.Cmd.Process != nil {
			killProcessGroup(proc.Cmd.Process.Pid, syscall.SIGTERM)
		}
	}

	// Wait for all processes to end
	deadline := time.After(10 * time.Second)
	for _, proc := range procs {
		select {
		case <-proc.Done:
		case <-deadline:
			if proc.Cmd.Process != nil {
				killProcessGroup(proc.Cmd.Process.Pid, syscall.SIGKILL)
			}
		}
	}

	e.logger.Info("all claude processes shutdown complete")
	return nil
}

// ListRunning returns all running session IDs.
func (e *ProcessExecutor) ListRunning() []uuid.UUID {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(e.runningProcs))
	for id := range e.runningProcs {
		ids = append(ids, id)
	}
	return ids
}
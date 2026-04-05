package claude

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/repository"
	"github.com/ryuyb/litchi/internal/domain/service"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
	"github.com/ryuyb/litchi/internal/infrastructure/agent/parser"
	"github.com/ryuyb/litchi/internal/infrastructure/agent/permission"
	"github.com/ryuyb/litchi/internal/infrastructure/agent/retry"
	"github.com/ryuyb/litchi/internal/pkg/errors"
	"go.uber.org/zap"
)

// ClaudeCodeAgent implements AgentRunner interface using Claude Code CLI.
type ClaudeCodeAgent struct {
	commandBuilder *CommandBuilder
	processExecutor *ProcessExecutor
	outputParser   parser.OutputParser
	permissionCtrl permission.PermissionController
	retryHandler   retry.RetryHandler
	cacheRepo      repository.CacheRepository
	logger         *zap.Logger

	// Track running sessions
	runningSessions map[uuid.UUID]*SessionState
	mu              sync.RWMutex
}

// SessionState tracks the state of a running session.
type SessionState struct {
	SessionID uuid.UUID
	StartTime time.Time
	Stage     service.AgentStage
	Status    string // idle, running, paused, cancelled, completed, failed
	Progress  float64
	Message   string
}

// ClaudeCodeAgentParams contains dependencies for ClaudeCodeAgent.
type ClaudeCodeAgentParams struct {
	ClaudeBinary   string
	OutputParser   parser.OutputParser
	PermissionCtrl permission.PermissionController
	RetryHandler   retry.RetryHandler
	CacheRepo      repository.CacheRepository
	Logger         *zap.Logger
}

// NewClaudeCodeAgent creates a new ClaudeCodeAgent.
func NewClaudeCodeAgent(params ClaudeCodeAgentParams) *ClaudeCodeAgent {
	return &ClaudeCodeAgent{
		commandBuilder:  NewCommandBuilder(params.ClaudeBinary),
		processExecutor: NewProcessExecutor(params.Logger),
		outputParser:    params.OutputParser,
		permissionCtrl:  params.PermissionCtrl,
		retryHandler:    params.RetryHandler,
		cacheRepo:       params.CacheRepo,
		logger:          params.Logger.Named("claude-agent"),
		runningSessions: make(map[uuid.UUID]*SessionState),
	}
}

// Execute executes an Agent task.
func (a *ClaudeCodeAgent) Execute(ctx context.Context, req *service.AgentRequest) (*service.AgentResponse, error) {
	// Validate request
	if err := a.ValidateRequest(req); err != nil {
		return nil, err
	}

	// Check if already running
	if a.IsRunning(req.SessionID) {
		return nil, errors.New(errors.ErrAgentAlreadyRunning).
			WithContext("sessionId", req.SessionID.String())
	}

	// Get allowed tools based on stage
	if len(req.AllowedTools) == 0 {
		req.AllowedTools = a.permissionCtrl.GetAllowedTools(req.Stage)
	}

	// Build command
	cmd := a.commandBuilder.BuildCommand(req)

	// Create session state
	state := &SessionState{
		SessionID:  req.SessionID,
		StartTime:  time.Now(),
		Stage:      req.Stage,
		Status:     "running",
		Progress:   0.0,
		Message:    "Executing agent task",
	}

	a.mu.Lock()
	a.runningSessions[req.SessionID] = state
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.runningSessions, req.SessionID)
		a.mu.Unlock()
	}()

	// Execute process
	result, err := a.processExecutor.Execute(ctx, cmd, req.SessionID)
	if err != nil {
		return a.buildErrorResponse(req, result, err), err
	}

	// Parse output
	response, err := a.outputParser.Parse(result.Stdout, req.Stage)
	if err != nil {
		a.logger.Warn("failed to parse output, returning raw",
			zap.String("sessionId", req.SessionID.String()),
			zap.Error(err),
		)
		response = &service.AgentResponse{
			SessionID: req.SessionID,
			Stage:     req.Stage,
			Success:   result.ExitCode == 0,
			Output:    result.Stdout,
			Duration:  result.Duration,
		}
	}

	response.SessionID = req.SessionID
	response.Stage = req.Stage
	response.Duration = result.Duration

	// Update session state
	state.Status = "completed"
	state.Progress = 100.0
	state.Message = "Execution completed"

	return response, nil
}

// ExecuteWithRetry executes a task with automatic retry.
func (a *ClaudeCodeAgent) ExecuteWithRetry(ctx context.Context, req *service.AgentRequest, policy valueobject.RetryPolicy) (*service.AgentResponse, error) {
	return a.retryHandler.ExecuteWithRetry(ctx, req, policy, a.Execute)
}

// ValidateRequest validates the request parameters.
func (a *ClaudeCodeAgent) ValidateRequest(req *service.AgentRequest) error {
	if req == nil {
		return errors.New(errors.ErrAgentInvalidRequest).WithDetail("request is nil")
	}
	return req.Validate()
}

// PrepareContext prepares execution context from cache.
func (a *ClaudeCodeAgent) PrepareContext(ctx context.Context, sessionID uuid.UUID, worktreePath string) (*service.AgentContext, error) {
	cache, err := a.cacheRepo.Load(ctx, worktreePath)
	if err != nil {
		a.logger.Warn("failed to load cache, using empty context",
			zap.String("sessionId", sessionID.String()),
			zap.Error(err),
		)
		return &service.AgentContext{}, nil
	}

	// Convert cache to context
	return a.cacheToContext(cache), nil
}

// SaveContext saves execution context to cache.
func (a *ClaudeCodeAgent) SaveContext(ctx context.Context, worktreePath string, cache *service.AgentContextCache) error {
	// Convert domain cache to infrastructure cache
	infraCache := a.domainToInfraCache(cache)
	return a.cacheRepo.Save(ctx, worktreePath, infraCache)
}

// Cancel cancels a running execution.
func (a *ClaudeCodeAgent) Cancel(sessionID uuid.UUID) error {
	if !a.IsRunning(sessionID) {
		return errors.New(errors.ErrAgentNotRunning).
			WithContext("sessionId", sessionID.String())
	}

	// Cancel process
	if err := a.processExecutor.Cancel(sessionID); err != nil {
		return err
	}

	// Update session state
	a.mu.Lock()
	if state, exists := a.runningSessions[sessionID]; exists {
		state.Status = "cancelled"
		state.Message = "Cancelled by user"
	}
	a.mu.Unlock()

	return nil
}

// GetStatus retrieves the current execution status.
func (a *ClaudeCodeAgent) GetStatus(sessionID uuid.UUID) (*service.AgentStatus, error) {
	a.mu.RLock()
	state, exists := a.runningSessions[sessionID]
	a.mu.RUnlock()

	if !exists {
		return &service.AgentStatus{
			SessionID: sessionID,
			Status:    "idle",
		}, nil
	}

	return &service.AgentStatus{
		SessionID:    state.SessionID,
		Status:       state.Status,
		CurrentStage: state.Stage,
		StartTime:    state.StartTime,
		Progress:     state.Progress,
		Message:      state.Message,
	}, nil
}

// IsRunning checks if Agent is executing for a session.
func (a *ClaudeCodeAgent) IsRunning(sessionID uuid.UUID) bool {
	return a.processExecutor.IsRunning(sessionID)
}

// Shutdown gracefully shuts down the executor.
func (a *ClaudeCodeAgent) Shutdown(ctx context.Context) error {
	a.logger.Info("shutting down claude agent")

	// Cancel all running sessions
	a.mu.Lock()
	for _, state := range a.runningSessions {
		state.Status = "cancelled"
		state.Message = "Server shutdown"
	}
	a.runningSessions = make(map[uuid.UUID]*SessionState)
	a.mu.Unlock()

	// Shutdown process executor
	return a.processExecutor.Shutdown(ctx)
}

// buildErrorResponse builds an error response from process result.
func (a *ClaudeCodeAgent) buildErrorResponse(req *service.AgentRequest, result *ProcessResult, execErr error) *service.AgentResponse {
	response := &service.AgentResponse{
		SessionID: req.SessionID,
		Stage:     req.Stage,
		Success:   false,
		Error: &service.AgentErrorInfo{
			Code:      errors.GetCode(execErr),
			Category:  a.categorizeError(execErr),
			Message:   execErr.Error(),
			Retryable: a.isRetryable(execErr),
		},
	}

	// Safely handle potentially nil result
	if result != nil {
		response.Output = result.Stdout
		response.Duration = result.Duration
		response.Error.Detail = result.Stderr
	}

	return response
}

// categorizeError categorizes an error by type.
func (a *ClaudeCodeAgent) categorizeError(err error) string {
	if errors.Is(err, errors.ErrAgentTimeout) {
		return "timeout"
	}
	if errors.Is(err, errors.ErrAgentProcessCrash) {
		return "process"
	}
	if errors.Is(err, errors.ErrAgentPermissionDenied) {
		return "permission"
	}
	return "execution"
}

// isRetryable determines if an error is retryable.
func (a *ClaudeCodeAgent) isRetryable(err error) bool {
	// Timeout and process crashes are generally retryable
	if errors.Is(err, errors.ErrAgentTimeout) || errors.Is(err, errors.ErrAgentProcessCrash) {
		return true
	}
	// Permission errors are not retryable without user intervention
	if errors.Is(err, errors.ErrAgentPermissionDenied) {
		return false
	}
	return false
}

// cacheToContext converts ExecutionContextCache to AgentContext.
func (a *ClaudeCodeAgent) cacheToContext(cache *repository.ExecutionContextCache) *service.AgentContext {
	if cache == nil {
		return &service.AgentContext{}
	}

	ctx := &service.AgentContext{}

	if cache.Clarification != nil {
		ctx.ClarifiedPoints = cache.Clarification.ConfirmedPoints
	}

	if cache.Execution != nil {
		ctx.Branch = cache.Execution.Branch
	}

	if len(cache.Tasks) > 0 {
		ctx.Tasks = make([]service.TaskContext, len(cache.Tasks))
		for i, task := range cache.Tasks {
			ctx.Tasks[i] = service.TaskContext{
				ID:     task.ID,
				Status: task.Status,
			}
		}
	}

	return ctx
}

// domainToInfraCache converts AgentContextCache to ExecutionContextCache.
func (a *ClaudeCodeAgent) domainToInfraCache(cache *service.AgentContextCache) *repository.ExecutionContextCache {
	if cache == nil {
		return nil
	}

	infraCache := &repository.ExecutionContextCache{
		SessionID:    cache.SessionID,
		CurrentStage: cache.CurrentStage,
		Status:       cache.Status,
		UpdatedAt:     cache.UpdatedAt,
	}

	if cache.PauseReason != nil {
		infraCache.PauseReason = cache.PauseReason
	}

	if len(cache.ClarifiedPoints) > 0 {
		infraCache.Clarification = &repository.ClarificationCache{
			ConfirmedPoints: cache.ClarifiedPoints,
		}
	}

	if cache.DesignVersion > 0 || cache.ComplexityScore != nil {
		infraCache.Design = &repository.DesignCache{
			CurrentVersion:  cache.DesignVersion,
			ComplexityScore: cache.ComplexityScore,
		}
	}

	if cache.CurrentTaskID != nil || len(cache.CompletedTaskIDs) > 0 || cache.Branch != "" {
		infraCache.Execution = &repository.ExecutionCache{
			CurrentTaskID:    cache.CurrentTaskID,
			CompletedTaskIDs: cache.CompletedTaskIDs,
			Branch:           cache.Branch,
			WorktreePath:     cache.WorktreePath,
		}
	}

	return infraCache
}
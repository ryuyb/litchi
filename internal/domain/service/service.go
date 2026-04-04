package service

import (
	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/aggregate"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// --- T2.4.1 ComplexityEvaluator ---

// CodebaseInfo provides contextual information about the existing codebase
// for complexity evaluation. This helps the evaluator assess the impact
// of proposed design changes.
type CodebaseInfo struct {
	AffectedModules   []string          // List of modules that will be affected
	ExistingPatterns  map[string]string // Key patterns/conventions in existing code
	RecentChanges     []string          // Recent commit descriptions for context
	ProjectSize       int               // Approximate number of files in project
	TechStack         []string          // Technologies used (e.g., "go", "react", "postgres")
	SensitiveAreas    []string          // Areas requiring careful handling (e.g., "auth", "payment")
}

// ComplexityWeights defines custom weights for complexity evaluation dimensions.
// Default weights: CodeChange=30%, Modules=25%, Breaking=25%, Testing=20%
type ComplexityWeights struct {
	CodeChangeWeight    int // Weight for estimated code change (0-100, default 30)
	ModulesWeight       int // Weight for affected modules (0-100, default 25)
	BreakingChangeWeight int // Weight for breaking changes (0-100, default 25)
	TestingWeight       int // Weight for test coverage difficulty (0-100, default 20)
}

// DefaultComplexityWeights returns the default weight configuration.
func DefaultComplexityWeights() ComplexityWeights {
	return ComplexityWeights{
		CodeChangeWeight:    30,
		ModulesWeight:       25,
		BreakingChangeWeight: 25,
		TestingWeight:       20,
	}
}

// Validate ensures weights sum to 100 and each is in valid range.
func (w ComplexityWeights) Validate() bool {
	sum := w.CodeChangeWeight + w.ModulesWeight + w.BreakingChangeWeight + w.TestingWeight
	return sum == 100 &&
		w.CodeChangeWeight >= 0 && w.CodeChangeWeight <= 100 &&
		w.ModulesWeight >= 0 && w.ModulesWeight <= 100 &&
		w.BreakingChangeWeight >= 0 && w.BreakingChangeWeight <= 100 &&
		w.TestingWeight >= 0 && w.TestingWeight <= 100
}

// ComplexityEvaluator evaluates the complexity of a design.
// This is a domain service interface - implementation will be in infrastructure layer
// (typically calling an Agent to perform the actual evaluation).
//
// The evaluator analyzes the design content and codebase context to produce
// a ComplexityScore with dimension breakdowns.
type ComplexityEvaluator interface {
	// Evaluate analyzes a design and returns a complexity score.
	// The design content and codebase information are used for evaluation.
	// Returns ComplexityScore with dimension breakdowns.
	//
	// Parameters:
	// - design: the design entity containing content to evaluate
	// - codebaseInfo: contextual information about the existing codebase
	// - weights: custom weights for dimension scoring (optional, uses defaults if not provided)
	//
	// Returns:
	// - ComplexityScore: the calculated complexity score
	// - ComplexityDimensions: dimension breakdown for transparency
	// - error: evaluation failure
	Evaluate(design *entity.Design, codebaseInfo *CodebaseInfo, weights *ComplexityWeights) (
		valueobject.ComplexityScore, valueobject.ComplexityDimensions, error)

	// EvaluateWithDefaultWeights evaluates using default weight configuration.
	// Convenience method for common use cases.
	EvaluateWithDefaultWeights(design *entity.Design, codebaseInfo *CodebaseInfo) (
		valueobject.ComplexityScore, valueobject.ComplexityDimensions, error)

	// GetThreshold returns the complexity threshold for requiring manual confirmation.
	// Designs with score >= threshold need user confirmation before task breakdown.
	GetThreshold() int

	// SetThreshold configures the complexity threshold.
	SetThreshold(threshold int) error
}

// --- T2.4.2 StageTransitionService ---

// TransitionContext provides context for stage transition decisions.
// Contains thresholds and configuration needed for precondition validation.
type TransitionContext struct {
	ClarityThreshold          int // Minimum clarity score to enter design stage
	ComplexityThreshold       int // Complexity threshold for design confirmation
	ForceDesignConfirm        bool // Force design confirmation regardless of complexity
	TaskRetryLimit            int // Maximum retry count for failed tasks
	AllowPRRollback           bool // Allow rollback from PR stage
	MaxPRRollbackCount        int // Maximum PR rollback count
}

// DefaultTransitionContext returns default transition configuration.
func DefaultTransitionContext() TransitionContext {
	return TransitionContext{
		ClarityThreshold:          60,
		ComplexityThreshold:       70,
		ForceDesignConfirm:        false,
		TaskRetryLimit:            3,
		AllowPRRollback:           true,
		MaxPRRollbackCount:        3,
	}
}

// StageTransitionService handles stage transitions and rollback operations.
// This is a domain service that validates transition preconditions and
// coordinates the transition process.
//
// Note: The actual transition logic is implemented in WorkSession aggregate root.
// This service provides:
// - Precondition validation (using external configuration/thresholds)
// - Transition decision support (can/cannot transition)
// - Validation of rollback eligibility
type StageTransitionService interface {
	// CanTransition checks if a forward transition is allowed.
	// Validates both the stage sequence and business preconditions.
	//
	// Preconditions checked:
	// - Clarification → Design: clarity score >= threshold, no pending questions
	// - Design → TaskBreakdown: design confirmed (if required)
	// - TaskBreakdown → Execution: tasks defined
	// - Execution → PullRequest: all tasks completed/skipped
	// - PullRequest → Completed: PR created
	CanTransition(session *aggregate.WorkSession, target valueobject.Stage, ctx TransitionContext) bool

	// GetTransitionError returns the reason why transition cannot proceed.
	// Useful for providing user feedback on blocked transitions.
	GetTransitionError(session *aggregate.WorkSession, target valueobject.Stage, ctx TransitionContext) error

	// CanRollback checks if rollback to a target stage is allowed.
	// Validates rollback rules and additional constraints.
	//
	// Additional constraints:
	// - PR rollback count must not exceed MaxPRRollbackCount
	// - AllowPRRollback must be true for PR stage rollback
	CanRollback(session *aggregate.WorkSession, target valueobject.Stage, ctx TransitionContext) bool

	// GetRollbackError returns the reason why rollback cannot proceed.
	GetRollbackError(session *aggregate.WorkSession, target valueobject.Stage, ctx TransitionContext) error

	// ValidateTransitionPreconditions validates stage-specific preconditions.
	// Returns detailed error if preconditions are not met.
	ValidateTransitionPreconditions(session *aggregate.WorkSession, target valueobject.Stage, ctx TransitionContext) error

	// ValidateRollbackPreconditions validates rollback-specific preconditions.
	ValidateRollbackPreconditions(session *aggregate.WorkSession, target valueobject.Stage, ctx TransitionContext) error

	// GetAllowedRollbackTargets returns all valid rollback targets for a session.
	GetAllowedRollbackTargets(session *aggregate.WorkSession, ctx TransitionContext) []valueobject.Stage
}

// --- T2.4.3 TaskScheduler ---

// TaskScheduler manages task execution order and dependency resolution.
// This is a domain service that provides task scheduling intelligence.
//
// Note: Basic dependency checking is implemented in WorkSession aggregate.
// This service provides:
// - Execution order calculation (topological sort)
// - Parallel task identification
// - Dependency graph analysis
// - Execution plan generation
type TaskScheduler interface {
	// GetExecutionOrder returns tasks in valid execution order.
	// Uses topological sort based on dependency graph.
	// Tasks with satisfied dependencies come before dependent tasks.
	GetExecutionOrder(tasks []*entity.Task) ([]*entity.Task, error)

	// GetNextExecutable returns the next task that can be executed.
	// Considers dependencies and task status.
	// Returns nil if no task can be executed (blocked or all done).
	GetNextExecutable(tasks []*entity.Task, completedIDs []uuid.UUID, maxRetryLimit int) *entity.Task

	// GetParallelTasks returns tasks that can be executed in parallel.
	// Tasks with no dependencies or all dependencies satisfied can run together.
	GetParallelTasks(tasks []*entity.Task, completedIDs []uuid.UUID) []*entity.Task

	// GetBlockedTasks returns tasks blocked by incomplete dependencies.
	// Useful for progress tracking and user feedback.
	GetBlockedTasks(tasks []*entity.Task, completedIDs []uuid.UUID) []*entity.Task

	// GetDependencyGraph returns the dependency relationship map.
	// Maps each task ID to its dependent task IDs (reverse dependency).
	GetDependencyGraph(tasks []*entity.Task) map[uuid.UUID][]uuid.UUID

	// CanRetryTask checks if a failed task can be retried.
	// Validates retry limit and dependency status.
	CanRetryTask(task *entity.Task, completedIDs []uuid.UUID, maxRetryLimit int) bool

	// GetExecutionPlan generates a complete execution plan.
	// Returns phases of parallel-executable task groups.
	GetExecutionPlan(tasks []*entity.Task) ([][]*entity.Task, error)

	// ValidateDependencies checks if all dependency references are valid.
	// Returns error if circular dependency or invalid reference exists.
	ValidateDependencies(tasks []*entity.Task) error
}
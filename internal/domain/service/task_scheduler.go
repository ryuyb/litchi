package service

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/pkg/errors"
)

// DefaultTaskScheduler provides default implementation of TaskScheduler.
// It implements pure algorithmic scheduling logic without external dependencies.
type DefaultTaskScheduler struct{}

// NewDefaultTaskScheduler creates a new DefaultTaskScheduler instance.
func NewDefaultTaskScheduler() *DefaultTaskScheduler {
	return &DefaultTaskScheduler{}
}

// GetExecutionOrder returns tasks in valid execution order using topological sort.
// Tasks with satisfied dependencies come before dependent tasks.
func (s *DefaultTaskScheduler) GetExecutionOrder(tasks []*entity.Task) ([]*entity.Task, error) {
	if len(tasks) == 0 {
		return []*entity.Task{}, nil
	}

	// Validate dependencies first
	if err := s.ValidateDependencies(tasks); err != nil {
		return nil, err
	}

	// Build reverse dependency graph: task.ID -> tasks that depend on it
	// This allows O(1) lookup of dependents during topological sort
	reverseGraph := s.GetDependencyGraph(tasks)
	inDegree := s.buildInDegree(tasks)
	taskMap := s.buildTaskMap(tasks)

	// Kahn's algorithm for topological sort:
	// 1. Find all tasks with no dependencies (in-degree = 0)
	// 2. Add them to result, remove their edges
	// 3. Repeat until all tasks processed
	result := make([]*entity.Task, 0, len(tasks))
	queue := make([]*entity.Task, 0, len(tasks)) // Pre-allocate max possible capacity
	queue = append(queue, s.findZeroInDegreeTasks(tasks, inDegree)...)

	for len(queue) > 0 {
		// Take task with zero in-degree
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree for tasks that depend on current (O(k) where k = number of dependents)
		dependents := reverseGraph[current.ID]
		for _, depTaskID := range dependents {
			inDegree[depTaskID]--
			if inDegree[depTaskID] == 0 {
				queue = append(queue, taskMap[depTaskID])
			}
		}
	}

	// If result length != tasks length, there's a cycle (should have been caught by ValidateDependencies)
	if len(result) != len(tasks) {
		return nil, errors.New(errors.ErrValidationFailed).WithDetail(
			"circular dependency detected in task list",
		)
	}

	return result, nil
}

// GetNextExecutable returns the next task that can be executed.
// Considers dependencies (must be satisfied) and task status (pending or retryable).
func (s *DefaultTaskScheduler) GetNextExecutable(tasks []*entity.Task, completedIDs []uuid.UUID, maxRetryLimit int) *entity.Task {
	completedSet := s.buildIDSet(completedIDs)

	// Priority: Pending tasks first, then retryable failed tasks
	// Use Order field for sequencing among candidates

	// First pass: find pending tasks with satisfied dependencies
	pendingCandidates := make([]*entity.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.IsPending() && s.areDependenciesSatisfied(task, completedSet) {
			pendingCandidates = append(pendingCandidates, task)
		}
	}
	if len(pendingCandidates) > 0 {
		return s.selectByOrder(pendingCandidates)
	}

	// Second pass: find retryable failed tasks
	retryCandidates := make([]*entity.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.IsFailed() && task.CanRetry(maxRetryLimit) && s.areDependenciesSatisfied(task, completedSet) {
			retryCandidates = append(retryCandidates, task)
		}
	}
	if len(retryCandidates) > 0 {
		return s.selectByOrder(retryCandidates)
	}

	return nil
}

// GetParallelTasks returns tasks that can be executed in parallel.
// All pending tasks with satisfied dependencies (or no dependencies) can run together.
func (s *DefaultTaskScheduler) GetParallelTasks(tasks []*entity.Task, completedIDs []uuid.UUID) []*entity.Task {
	completedSet := s.buildIDSet(completedIDs)
	parallelTasks := make([]*entity.Task, 0, len(tasks))

	for _, task := range tasks {
		// Only pending tasks can be added to parallel execution
		if task.IsPending() && s.areDependenciesSatisfied(task, completedSet) {
			parallelTasks = append(parallelTasks, task)
		}
	}

	return parallelTasks
}

// GetBlockedTasks returns tasks blocked by incomplete dependencies.
func (s *DefaultTaskScheduler) GetBlockedTasks(tasks []*entity.Task, completedIDs []uuid.UUID) []*entity.Task {
	completedSet := s.buildIDSet(completedIDs)
	blockedTasks := make([]*entity.Task, 0, len(tasks))

	for _, task := range tasks {
		// Pending or failed-retryable tasks with unsatisfied dependencies are blocked
		if (task.IsPending() || task.IsFailed()) && !s.areDependenciesSatisfied(task, completedSet) {
			blockedTasks = append(blockedTasks, task)
		}
	}

	return blockedTasks
}

// GetDependencyGraph returns reverse dependency map (task -> tasks that depend on it).
func (s *DefaultTaskScheduler) GetDependencyGraph(tasks []*entity.Task) map[uuid.UUID][]uuid.UUID {
	graph := make(map[uuid.UUID][]uuid.UUID, len(tasks))

	for _, task := range tasks {
		// Initialize entry for each task
		if graph[task.ID] == nil {
			graph[task.ID] = []uuid.UUID{}
		}

		// For each dependency, add this task as dependent
		for _, depID := range task.Dependencies {
			if graph[depID] == nil {
				graph[depID] = []uuid.UUID{}
			}
			graph[depID] = append(graph[depID], task.ID)
		}
	}

	return graph
}

// CanRetryTask checks if a failed task can be retried.
func (s *DefaultTaskScheduler) CanRetryTask(task *entity.Task, completedIDs []uuid.UUID, maxRetryLimit int) bool {
	if task == nil {
		return false
	}

	// Must be failed status
	if !task.IsFailed() {
		return false
	}

	// Retry count must be under limit
	if task.RetryCount >= maxRetryLimit {
		return false
	}

	// Dependencies must be satisfied
	completedSet := s.buildIDSet(completedIDs)
	return s.areDependenciesSatisfied(task, completedSet)
}

// GetExecutionPlan generates execution phases (groups of parallel-executable tasks).
// Phase 0 contains tasks with no dependencies.
// Phase N contains tasks whose dependencies are all in phases 0 to N-1.
func (s *DefaultTaskScheduler) GetExecutionPlan(tasks []*entity.Task) ([][]*entity.Task, error) {
	if len(tasks) == 0 {
		return [][]*entity.Task{}, nil
	}

	// Validate dependencies first
	if err := s.ValidateDependencies(tasks); err != nil {
		return nil, err
	}

	taskMap := s.buildTaskMap(tasks)
	phaseMap := make(map[uuid.UUID]int) // task ID -> phase number
	phases := make([][]*entity.Task, 0)

	// Calculate phase for each task
	// Phase = max(phase of dependencies) + 1
	// Tasks with no dependencies have phase 0

	for _, task := range tasks {
		phase := s.calculateTaskPhase(task, phaseMap, taskMap)
		phaseMap[task.ID] = phase

		// Ensure phases slice has enough capacity
		for len(phases) <= phase {
			phases = append(phases, []*entity.Task{})
		}
		phases[phase] = append(phases[phase], task)
	}

	return phases, nil
}

// ValidateDependencies checks for circular dependencies and invalid references.
func (s *DefaultTaskScheduler) ValidateDependencies(tasks []*entity.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	taskMap := s.buildTaskMap(tasks)

	// Check for invalid references
	for _, task := range tasks {
		for _, depID := range task.Dependencies {
			if taskMap[depID] == nil {
				return errors.New(errors.ErrValidationFailed).WithDetail(
					fmt.Sprintf("task %s references non-existent dependency %s", task.ID, depID),
				)
			}
			// Self-dependency is invalid
			if depID == task.ID {
				return errors.New(errors.ErrValidationFailed).WithDetail(
					fmt.Sprintf("task %s has self-dependency", task.ID),
				)
			}
		}
	}

	// Check for circular dependencies using DFS
	visited := make(map[uuid.UUID]bool)
	inProgress := make(map[uuid.UUID]bool)

	for _, task := range tasks {
		if !visited[task.ID] {
			if s.hasCircularDependency(task, taskMap, visited, inProgress) {
				return errors.New(errors.ErrValidationFailed).WithDetail(
					fmt.Sprintf("circular dependency detected involving task %s", task.ID),
				)
			}
		}
	}

	return nil
}

// --- Helper methods ---

// buildTaskMap creates a map for quick task lookup by ID.
func (s *DefaultTaskScheduler) buildTaskMap(tasks []*entity.Task) map[uuid.UUID]*entity.Task {
	taskMap := make(map[uuid.UUID]*entity.Task)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}
	return taskMap
}

// buildInDegree creates an in-degree map for topological sort.
func (s *DefaultTaskScheduler) buildInDegree(tasks []*entity.Task) map[uuid.UUID]int {
	inDegree := make(map[uuid.UUID]int, len(tasks))
	for _, task := range tasks {
		inDegree[task.ID] = len(task.Dependencies)
	}
	return inDegree
}

// buildIDSet creates a set from UUID slice for O(1) lookup.
func (s *DefaultTaskScheduler) buildIDSet(ids []uuid.UUID) map[uuid.UUID]bool {
	set := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// findZeroInDegreeTasks finds tasks with zero in-degree.
func (s *DefaultTaskScheduler) findZeroInDegreeTasks(tasks []*entity.Task, inDegree map[uuid.UUID]int) []*entity.Task {
	result := make([]*entity.Task, 0, len(tasks)) // Pre-allocate max possible capacity
	for _, task := range tasks {
		if inDegree[task.ID] == 0 {
			result = append(result, task)
		}
	}
	// Sort by Order field
	return s.sortByOrder(result)
}

// areDependenciesSatisfied checks if all task dependencies are in completed set.
func (s *DefaultTaskScheduler) areDependenciesSatisfied(task *entity.Task, completedSet map[uuid.UUID]bool) bool {
	if len(task.Dependencies) == 0 {
		return true
	}
	for _, depID := range task.Dependencies {
		if !completedSet[depID] {
			return false
		}
	}
	return true
}

// selectByOrder selects the task with lowest Order value.
func (s *DefaultTaskScheduler) selectByOrder(tasks []*entity.Task) *entity.Task {
	if len(tasks) == 0 {
		return nil
	}
	result := tasks[0]
	for _, task := range tasks[1:] {
		if task.Order < result.Order {
			result = task
		}
	}
	return result
}

// sortByOrder sorts tasks by Order field (ascending).
func (s *DefaultTaskScheduler) sortByOrder(tasks []*entity.Task) []*entity.Task {
	result := make([]*entity.Task, len(tasks))
	copy(result, tasks)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Order < result[j].Order
	})
	return result
}

// hasCircularDependency detects circular dependencies using DFS.
func (s *DefaultTaskScheduler) hasCircularDependency(task *entity.Task, taskMap map[uuid.UUID]*entity.Task, visited, inProgress map[uuid.UUID]bool) bool {
	if inProgress[task.ID] {
		return true // Circular dependency found
	}
	if visited[task.ID] {
		return false // Already processed, no cycle
	}

	inProgress[task.ID] = true

	for _, depID := range task.Dependencies {
		depTask := taskMap[depID]
		if depTask != nil && s.hasCircularDependency(depTask, taskMap, visited, inProgress) {
			return true
		}
	}

	inProgress[task.ID] = false
	visited[task.ID] = true
	return false
}

// calculateTaskPhase calculates the execution phase for a task.
// Phase = max(phase of all dependencies) + 1.
// Uses recursion with memoization via phaseMap.
func (s *DefaultTaskScheduler) calculateTaskPhase(task *entity.Task, phaseMap map[uuid.UUID]int, taskMap map[uuid.UUID]*entity.Task) int {
	// If already calculated, return cached value
	if phase, ok := phaseMap[task.ID]; ok {
		return phase
	}

	// No dependencies -> phase 0
	if len(task.Dependencies) == 0 {
		return 0
	}

	// Calculate max phase of dependencies
	maxDepPhase := 0
	for _, depID := range task.Dependencies {
		depTask := taskMap[depID]
		if depTask != nil {
			depPhase := s.calculateTaskPhase(depTask, phaseMap, taskMap)
			if depPhase > maxDepPhase {
				maxDepPhase = depPhase
			}
		}
	}

	return maxDepPhase + 1
}
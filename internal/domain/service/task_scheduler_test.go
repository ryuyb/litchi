package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ryuyb/litchi/internal/domain/entity"
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

func TestTaskScheduler_GetExecutionOrder(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	// Create tasks with dependencies
	task1ID := uuid.New()
	task2ID := uuid.New()
	task3ID := uuid.New()
	task4ID := uuid.New()

	task1 := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	task1.ID = task1ID

	task2 := entity.NewTask("Task 2", []uuid.UUID{task1ID}, 2)
	task2.ID = task2ID

	task3 := entity.NewTask("Task 3", []uuid.UUID{task1ID}, 3)
	task3.ID = task3ID

	task4 := entity.NewTask("Task 4", []uuid.UUID{task2ID, task3ID}, 4)
	task4.ID = task4ID

	tasks := []*entity.Task{task4, task2, task3, task1} // Intentionally unordered

	order, err := scheduler.GetExecutionOrder(tasks)
	if err != nil {
		t.Fatalf("GetExecutionOrder failed: %v", err)
	}

	// Verify order: Task1 must come before Task2 and Task3
	// Task2 and Task3 must come before Task4
	if len(order) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(order))
	}

	// Find positions
	task1Pos := findTaskPosition(order, task1ID)
	task2Pos := findTaskPosition(order, task2ID)
	task3Pos := findTaskPosition(order, task3ID)
	task4Pos := findTaskPosition(order, task4ID)

	if task1Pos > task2Pos {
		t.Error("Task1 should come before Task2")
	}
	if task1Pos > task3Pos {
		t.Error("Task1 should come before Task3")
	}
	if task2Pos > task4Pos {
		t.Error("Task2 should come before Task4")
	}
	if task3Pos > task4Pos {
		t.Error("Task3 should come before Task4")
	}
}

func TestTaskScheduler_GetExecutionOrder_CircularDependency(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	task1ID := uuid.New()
	task2ID := uuid.New()

	task1 := entity.NewTask("Task 1", []uuid.UUID{task2ID}, 1)
	task1.ID = task1ID

	task2 := entity.NewTask("Task 2", []uuid.UUID{task1ID}, 2)
	task2.ID = task2ID

	tasks := []*entity.Task{task1, task2}

	_, err := scheduler.GetExecutionOrder(tasks)
	if err == nil {
		t.Error("expected error for circular dependency")
	}
}

func TestTaskScheduler_GetNextExecutable(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	task1ID := uuid.New()
	task2ID := uuid.New()

	task1 := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	task1.ID = task1ID

	task2 := entity.NewTask("Task 2", []uuid.UUID{task1ID}, 2)
	task2.ID = task2ID

	tasks := []*entity.Task{task1, task2}

	// No completed tasks -> Task1 should be next (no dependencies, pending)
	next := scheduler.GetNextExecutable(tasks, []uuid.UUID{}, 3)
	if next == nil || next.ID != task1ID {
		t.Error("expected Task1 to be next executable")
	}

	// Mark Task1 as completed - Task2 should be next
	task1.Start()
	task1.Complete(valueobject.ExecutionResult{})

	next = scheduler.GetNextExecutable(tasks, []uuid.UUID{task1ID}, 3)
	if next == nil || next.ID != task2ID {
		t.Error("expected Task2 to be next executable after Task1 completed")
	}
}

func TestTaskScheduler_GetParallelTasks(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	task1ID := uuid.New()
	task2ID := uuid.New()
	task3ID := uuid.New()
	task4ID := uuid.New()

	task1 := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	task1.ID = task1ID

	task2 := entity.NewTask("Task 2", []uuid.UUID{}, 2)
	task2.ID = task2ID

	task3 := entity.NewTask("Task 3", []uuid.UUID{task1ID, task2ID}, 3)
	task3.ID = task3ID

	task4 := entity.NewTask("Task 4", []uuid.UUID{task3ID}, 4)
	task4.ID = task4ID

	tasks := []*entity.Task{task1, task2, task3, task4}

	// Phase 0: Task1 and Task2 can run in parallel (both pending, no dependencies)
	parallel := scheduler.GetParallelTasks(tasks, []uuid.UUID{})
	if len(parallel) != 2 {
		t.Errorf("expected 2 parallel tasks, got %d", len(parallel))
	}

	// Mark Task1 and Task2 as completed
	task1.Start()
	task1.Complete(valueobject.ExecutionResult{})
	task2.Start()
	task2.Complete(valueobject.ExecutionResult{})

	// Phase 1: Task3 can run now (dependencies satisfied)
	parallel = scheduler.GetParallelTasks(tasks, []uuid.UUID{task1ID, task2ID})
	if len(parallel) != 1 || parallel[0].ID != task3ID {
		t.Errorf("expected Task3 to be next parallel task, got %d tasks", len(parallel))
	}
}

func TestTaskScheduler_GetExecutionPlan(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	task1ID := uuid.New()
	task2ID := uuid.New()
	task3ID := uuid.New()
	task4ID := uuid.New()

	task1 := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	task1.ID = task1ID

	task2 := entity.NewTask("Task 2", []uuid.UUID{}, 2)
	task2.ID = task2ID

	task3 := entity.NewTask("Task 3", []uuid.UUID{task1ID, task2ID}, 3)
	task3.ID = task3ID

	task4 := entity.NewTask("Task 4", []uuid.UUID{task3ID}, 4)
	task4.ID = task4ID

	tasks := []*entity.Task{task1, task2, task3, task4}

	plan, err := scheduler.GetExecutionPlan(tasks)
	if err != nil {
		t.Fatalf("GetExecutionPlan failed: %v", err)
	}

	// Should have 3 phases:
	// Phase 0: Task1, Task2
	// Phase 1: Task3
	// Phase 2: Task4
	if len(plan) != 3 {
		t.Errorf("expected 3 phases, got %d", len(plan))
	}

	if len(plan[0]) != 2 {
		t.Errorf("expected 2 tasks in phase 0, got %d", len(plan[0]))
	}
	if len(plan[1]) != 1 {
		t.Errorf("expected 1 task in phase 1, got %d", len(plan[1]))
	}
	if len(plan[2]) != 1 {
		t.Errorf("expected 1 task in phase 2, got %d", len(plan[2]))
	}
}

func TestTaskScheduler_GetBlockedTasks(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	task1ID := uuid.New()
	task2ID := uuid.New()

	task1 := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	task1.ID = task1ID

	task2 := entity.NewTask("Task 2", []uuid.UUID{task1ID}, 2)
	task2.ID = task2ID

	tasks := []*entity.Task{task1, task2}

	// Without Task1 completed, Task2 is blocked
	blocked := scheduler.GetBlockedTasks(tasks, []uuid.UUID{})
	if len(blocked) != 1 || blocked[0].ID != task2ID {
		t.Errorf("expected Task2 to be blocked")
	}

	// With Task1 completed, no tasks blocked
	blocked = scheduler.GetBlockedTasks(tasks, []uuid.UUID{task1ID})
	if len(blocked) != 0 {
		t.Errorf("expected no blocked tasks after Task1 completed")
	}
}

func TestTaskScheduler_CanRetryTask(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	task1ID := uuid.New()
	task2ID := uuid.New()

	task1 := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	task1.ID = task1ID

	task2 := entity.NewTask("Task 2", []uuid.UUID{task1ID}, 2)
	task2.ID = task2ID

	// Mark Task2 as failed
	task2.Start()
	task2.Fail("error", "try again")

	// Task2 cannot retry because Task1 not completed (dependency not satisfied)
	if scheduler.CanRetryTask(task2, []uuid.UUID{}, 3) {
		t.Error("Task2 should not be retryable with unsatisfied dependencies")
	}

	// Mark Task1 as completed
	task1.Start()
	task1.Complete(valueobject.ExecutionResult{})

	// Task2 can retry with Task1 completed
	if !scheduler.CanRetryTask(task2, []uuid.UUID{task1ID}, 3) {
		t.Error("Task2 should be retryable with satisfied dependencies")
	}

	// Mark Task2 as retried multiple times (simulating retries)
	task2.Retry(3)                         // Reset to InProgress
	task2.Fail("error again", "try again") // Fail again, retry count = 2
	task2.Retry(3)                         // Reset to InProgress
	task2.Fail("error again", "try again") // Fail again, retry count = 3

	if scheduler.CanRetryTask(task2, []uuid.UUID{task1ID}, 3) {
		t.Error("Task2 should not be retryable after reaching limit")
	}
}

func TestTaskScheduler_ValidateDependencies(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	task1ID := uuid.New()
	task2ID := uuid.New()
	nonexistentID := uuid.New()

	task1 := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	task1.ID = task1ID

	task2 := entity.NewTask("Task 2", []uuid.UUID{nonexistentID}, 2)
	task2.ID = task2ID

	tasks := []*entity.Task{task1, task2}

	// Should fail due to nonexistent dependency
	err := scheduler.ValidateDependencies(tasks)
	if err == nil {
		t.Error("expected error for nonexistent dependency")
	}

	// Self-dependency test
	task3 := entity.NewTask("Task 3", []uuid.UUID{task1ID}, 3)
	task3.ID = task1ID                        // Same ID as task1
	task3.Dependencies = []uuid.UUID{task1ID} // Self-dependency

	err = scheduler.ValidateDependencies([]*entity.Task{task3})
	if err == nil {
		t.Error("expected error for self-dependency")
	}
}

func TestTaskScheduler_GetDependencyGraph(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	task1ID := uuid.New()
	task2ID := uuid.New()
	task3ID := uuid.New()

	task1 := entity.NewTask("Task 1", []uuid.UUID{}, 1)
	task1.ID = task1ID

	task2 := entity.NewTask("Task 2", []uuid.UUID{task1ID}, 2)
	task2.ID = task2ID

	task3 := entity.NewTask("Task 3", []uuid.UUID{task1ID, task2ID}, 3)
	task3.ID = task3ID

	tasks := []*entity.Task{task1, task2, task3}

	graph := scheduler.GetDependencyGraph(tasks)

	// Task1 should have Task2 and Task3 depending on it
	if len(graph[task1ID]) != 2 {
		t.Errorf("expected 2 dependents for Task1, got %d", len(graph[task1ID]))
	}

	// Task2 should have Task3 depending on it
	if len(graph[task2ID]) != 1 {
		t.Errorf("expected 1 dependent for Task2, got %d", len(graph[task2ID]))
	}

	// Task3 has no dependents
	if len(graph[task3ID]) != 0 {
		t.Errorf("expected 0 dependents for Task3, got %d", len(graph[task3ID]))
	}
}

// Helper function to find task position in ordered list
func findTaskPosition(tasks []*entity.Task, id uuid.UUID) int {
	for i, task := range tasks {
		if task.ID == id {
			return i
		}
	}
	return -1
}

// Test empty tasks scenarios
func TestTaskScheduler_EmptyTasks(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	order, err := scheduler.GetExecutionOrder([]*entity.Task{})
	if err != nil || len(order) != 0 {
		t.Error("empty tasks should return empty order")
	}

	plan, err := scheduler.GetExecutionPlan([]*entity.Task{})
	if err != nil || len(plan) != 0 {
		t.Error("empty tasks should return empty plan")
	}

	err = scheduler.ValidateDependencies([]*entity.Task{})
	if err != nil {
		t.Error("empty tasks should pass validation")
	}

	next := scheduler.GetNextExecutable([]*entity.Task{}, []uuid.UUID{}, 3)
	if next != nil {
		t.Error("empty tasks should return nil for next executable")
	}

	parallel := scheduler.GetParallelTasks([]*entity.Task{}, []uuid.UUID{})
	if len(parallel) != 0 {
		t.Error("empty tasks should return empty parallel list")
	}
}

// Test retry with failed task status
func TestTaskScheduler_RetryFailedTask(t *testing.T) {
	scheduler := NewDefaultTaskScheduler()

	task := entity.NewTask("Task", []uuid.UUID{}, 1)

	// Pending task cannot retry (must be failed)
	if scheduler.CanRetryTask(task, []uuid.UUID{}, 3) {
		t.Error("pending task should not be retryable")
	}

	// Start and fail the task
	task.Start()
	task.Fail("error", "suggestion")

	// Now can retry (failed status, no dependencies, retry count < limit)
	if !scheduler.CanRetryTask(task, []uuid.UUID{}, 3) {
		t.Error("failed task should be retryable")
	}

	// Complete the task - should not be retryable
	task2 := entity.NewTask("Task2", []uuid.UUID{}, 2)
	task2.Start()
	task2.Complete(valueobject.ExecutionResult{})
	if scheduler.CanRetryTask(task2, []uuid.UUID{}, 3) {
		t.Error("completed task should not be retryable")
	}
}

package task_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
	"opspilot/task-api/internal/task"
)

func TestMemoryRepository_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := task.NewMemoryRepository()

	// Initial list should be empty slice, not nil
	tasks, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing tasks: %v", err)
	}
	if tasks == nil || len(tasks) != 0 {
		t.Fatalf("expected empty non-nil slice, got: %v", tasks)
	}

	// Not found checks
	_, err = repo.GetByID(ctx, "non-existent")
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	err = repo.Delete(ctx, "non-existent")
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	_, err = repo.Update(ctx, task.Task{ID: "non-existent"})
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	// Create task 1
	now := time.Now().UTC()
	t1 := task.Task{
		ID:          "id-1",
		Title:       "Task One",
		Description: "First task",
		Status:      task.StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created1, err := repo.Create(ctx, t1)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	if created1.ID != "id-1" || created1.Title != "Task One" {
		t.Fatalf("unexpected task created: %+v", created1)
	}

	// Create task 2
	t2 := task.Task{
		ID:          "id-2",
		Title:       "Task Two",
		Description: "Second task",
		Status:      task.StatusInProgress,
		CreatedAt:   now.Add(1 * time.Minute),
		UpdatedAt:   now.Add(1 * time.Minute),
	}
	_, err = repo.Create(ctx, t2)
	if err != nil {
		t.Fatalf("failed to create task 2: %v", err)
	}

	// GetByID
	got, err := repo.GetByID(ctx, "id-1")
	if err != nil {
		t.Fatalf("unexpected error getting task: %v", err)
	}
	if got.Title != "Task One" {
		t.Errorf("expected title 'Task One', got %q", got.Title)
	}

	// List (should be ordered newest first, so id-2 then id-1)
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(list))
	}
	if list[0].ID != "id-2" || list[1].ID != "id-1" {
		t.Errorf("expected tasks ordered newest first [id-2, id-1], got [%s, %s]", list[0].ID, list[1].ID)
	}

	// Update task
	updateData := task.Task{
		ID:          "id-1",
		Title:       "Task One Updated",
		Description: "Updated description",
		Status:      task.StatusCompleted,
		UpdatedAt:   now.Add(2 * time.Minute),
	}
	updated, err := repo.Update(ctx, updateData)
	if err != nil {
		t.Fatalf("unexpected error updating: %v", err)
	}
	if updated.Title != "Task One Updated" || updated.Status != task.StatusCompleted {
		t.Errorf("unexpected updated data: %+v", updated)
	}
	// Check that CreatedAt was preserved
	if !updated.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt to be preserved (%v), got: %v", now, updated.CreatedAt)
	}

	// Delete task
	err = repo.Delete(ctx, "id-1")
	if err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}

	// Verify deleted
	_, err = repo.GetByID(ctx, "id-1")
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound after deletion, got: %v", err)
	}

	// List should now only have 1
	listAfterDelete, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing: %v", err)
	}
	if len(listAfterDelete) != 1 || listAfterDelete[0].ID != "id-2" {
		t.Errorf("expected 1 task (id-2), got %d tasks", len(listAfterDelete))
	}
}

func TestMemoryRepository_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	repo := task.NewMemoryRepository()
	_, err := repo.Create(ctx, task.Task{ID: "any"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled on Create, got %v", err)
	}

	_, err = repo.GetByID(ctx, "any")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled on GetByID, got %v", err)
	}

	_, err = repo.List(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled on List, got %v", err)
	}

	_, err = repo.Update(ctx, task.Task{ID: "any"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled on Update, got %v", err)
	}

	err = repo.Delete(ctx, "any")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled on Delete, got %v", err)
	}
}

func TestMemoryRepository_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := task.NewMemoryRepository()

	const workers = 50
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		id := fmt.Sprintf("concurrent-task-%d", i)
		go func(taskID string) {
			defer wg.Done()

			// Create
			_, err := repo.Create(ctx, task.Task{
				ID:        taskID,
				Title:     "Concurrent Task",
				Status:    task.StatusPending,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			})
			if err != nil {
				t.Errorf("concurrent create error: %v", err)
				return
			}

			// Read
			_, err = repo.GetByID(ctx, taskID)
			if err != nil && !errors.Is(err, task.ErrNotFound) {
				t.Errorf("concurrent get error: %v", err)
				return
			}

			// List
			_, err = repo.List(ctx)
			if err != nil {
				t.Errorf("concurrent list error: %v", err)
				return
			}

			// Update
			_, _ = repo.Update(ctx, task.Task{
				ID:        taskID,
				Title:     "Concurrent Task Updated",
				Status:    task.StatusCompleted,
				UpdatedAt: time.Now().UTC(),
			})

			// Delete
			_ = repo.Delete(ctx, taskID)
		}(id)
	}

	wg.Wait()
}

func TestMemoryRepository_DuplicateID(t *testing.T) {
	ctx := context.Background()
	repo := task.NewMemoryRepository()

	t1 := task.Task{
		ID:        "duplicate-id",
		Title:     "Initial Task",
		Status:    task.StatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err := repo.Create(ctx, t1)
	if err != nil {
		t.Fatalf("unexpected error creating first task: %v", err)
	}

	// Attempt to create second task with identical ID
	t2 := task.Task{
		ID:        "duplicate-id",
		Title:     "Conflicting Task",
		Status:    task.StatusInProgress,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err = repo.Create(ctx, t2)
	if !errors.Is(err, task.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists on duplicate ID creation, got: %v", err)
	}
}

func TestMemoryRepository_TenantIsolation(t *testing.T) {
	repo := task.NewMemoryRepository()

	ctxA := auth.WithUser(context.Background(), auth.Claims{TenantID: "tenant-A", Role: auth.RoleAdmin})
	ctxB := auth.WithUser(context.Background(), auth.Claims{TenantID: "tenant-B", Role: auth.RoleAdmin})

	// 1. Tenant A creates a task
	taskA := task.Task{
		ID:        "task-of-a",
		Title:     "Confidential Task A",
		Status:    task.StatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	created, err := repo.Create(ctxA, taskA)
	if err != nil {
		t.Fatalf("unexpected error creating task under tenant A: %v", err)
	}
	if created.TenantID != "tenant-A" {
		t.Errorf("expected task.TenantID to be 'tenant-A', got %q", created.TenantID)
	}

	// 2. Tenant A can read its task
	fetchedA, err := repo.GetByID(ctxA, "task-of-a")
	if err != nil || fetchedA.ID != "task-of-a" {
		t.Fatalf("tenant A failed to retrieve its own task: %v", err)
	}

	// 3. Tenant B CANNOT read Tenant A's task (must return ErrNotFound)
	_, err = repo.GetByID(ctxB, "task-of-a")
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-tenant GetByID, got %v", err)
	}

	// 4. Tenant B List must NOT include Tenant A's task
	listB, err := repo.List(ctxB)
	if err != nil {
		t.Fatalf("unexpected error listing for tenant B: %v", err)
	}
	if len(listB) != 0 {
		t.Errorf("tenant B should see 0 tasks, got %d", len(listB))
	}

	// Tenant A List sees 1 task
	listA, err := repo.List(ctxA)
	if err != nil || len(listA) != 1 {
		t.Fatalf("expected tenant A to list 1 task, got len=%d, err=%v", len(listA), err)
	}

	// 5. Tenant B CANNOT update Tenant A's task
	taskAUpdate := fetchedA
	taskAUpdate.Title = "Hijacked Title"
	_, err = repo.Update(ctxB, taskAUpdate)
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound when tenant B tries to update tenant A task, got %v", err)
	}

	// 6. Tenant B CANNOT delete Tenant A's task
	err = repo.Delete(ctxB, "task-of-a")
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound when tenant B tries to delete tenant A task, got %v", err)
	}

	// Verify task still exists for Tenant A
	stillThere, err := repo.GetByID(ctxA, "task-of-a")
	if err != nil || stillThere.Title != "Confidential Task A" {
		t.Errorf("task should remain unchanged for tenant A, got: %+v, err: %v", stillThere, err)
	}
}

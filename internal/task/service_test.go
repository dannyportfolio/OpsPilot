package task_test

import (
	"context"
	"errors"
	"testing"

	"opspilot/task-api/internal/task"
)

func TestService_CreateTask(t *testing.T) {
	ctx := context.Background()
	repo := task.NewMemoryRepository()
	svc := task.NewService(repo)

	// Valid creation with minimal fields
	req := task.CreateTaskRequest{
		Title: "Service Task 1",
	}

	created, err := svc.CreateTask(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error creating task: %v", err)
	}

	if created.ID == "" {
		t.Error("expected non-empty ID")
	}
	if !task.IsValidID(created.ID) {
		t.Errorf("expected valid UUID, got: %q", created.ID)
	}
	if created.Status != task.StatusPending {
		t.Errorf("expected default status 'pending', got: %q", created.Status)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Error("expected timestamps to be set")
	}

	// Validation failure
	invalidReq := task.CreateTaskRequest{
		Title: "",
	}
	_, err = svc.CreateTask(ctx, invalidReq)
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestService_GetTask(t *testing.T) {
	ctx := context.Background()
	repo := task.NewMemoryRepository()
	svc := task.NewService(repo)

	// Create a task
	created, err := svc.CreateTask(ctx, task.CreateTaskRequest{Title: "Fetchable"})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Successful get
	fetched, err := svc.GetTask(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, fetched.ID)
	}

	// Not found (valid UUID)
	_, err = svc.GetTask(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	// Invalid UUID format
	_, err = svc.GetTask(ctx, "invalid-uuid")
	if err == nil {
		t.Error("expected validation error for invalid UUID format")
	}

	// Empty ID
	_, err = svc.GetTask(ctx, "   ")
	if err == nil {
		t.Error("expected validation error for empty ID")
	}
}

func TestService_ListTasks(t *testing.T) {
	ctx := context.Background()
	repo := task.NewMemoryRepository()
	svc := task.NewService(repo)

	// Initially empty list
	tasks, err := svc.ListTasks(ctx)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}

	// Create tasks
	_, _ = svc.CreateTask(ctx, task.CreateTaskRequest{Title: "Task 1"})
	_, _ = svc.CreateTask(ctx, task.CreateTaskRequest{Title: "Task 2"})

	tasks, err = svc.ListTasks(ctx)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestService_UpdateTask(t *testing.T) {
	ctx := context.Background()
	repo := task.NewMemoryRepository()
	svc := task.NewService(repo)

	created, err := svc.CreateTask(ctx, task.CreateTaskRequest{Title: "Before Update"})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Successful update
	updateReq := task.UpdateTaskRequest{
		Title:       "After Update",
		Description: "Updated details",
		Status:      task.StatusCompleted,
	}

	updated, err := svc.UpdateTask(ctx, created.ID, updateReq)
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}
	if updated.Title != "After Update" || updated.Status != task.StatusCompleted {
		t.Errorf("unexpected updated fields: %+v", updated)
	}

	// Update with invalid UUID
	_, err = svc.UpdateTask(ctx, "bad-id", updateReq)
	if err == nil {
		t.Error("expected error for invalid UUID")
	}

	// Update not found
	_, err = svc.UpdateTask(ctx, "00000000-0000-0000-0000-000000000000", updateReq)
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	// Update with invalid body
	_, err = svc.UpdateTask(ctx, created.ID, task.UpdateTaskRequest{Title: ""})
	if err == nil {
		t.Error("expected validation error for empty title")
	}
}

func TestService_DeleteTask(t *testing.T) {
	ctx := context.Background()
	repo := task.NewMemoryRepository()
	svc := task.NewService(repo)

	created, err := svc.CreateTask(ctx, task.CreateTaskRequest{Title: "To Delete"})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Successful delete
	err = svc.DeleteTask(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	// Delete already deleted -> ErrNotFound
	err = svc.DeleteTask(ctx, created.ID)
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	// Delete invalid UUID format
	err = svc.DeleteTask(ctx, "not-a-uuid")
	if err == nil {
		t.Error("expected validation error for invalid UUID")
	}

	// Delete empty ID
	err = svc.DeleteTask(ctx, "")
	if err == nil {
		t.Error("expected validation error for empty ID")
	}
}

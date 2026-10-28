package task_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"opspilot/task-api/internal/platform/database"
	"opspilot/task-api/internal/task"
)

func setupPostgresTest(t *testing.T) (*sql.DB, *task.PostgresRepository) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/opspilot?sslmode=disable"
	}

	db, err := database.Connect(ctx, url)
	if err != nil {
		t.Skipf("skipping PostgreSQL integration tests (cannot connect: %v)", err)
		return nil, nil
	}

	// Apply migrations
	migrationsDir := "../../migrations"
	if _, err := os.Stat(migrationsDir); err != nil {
		migrationsDir = "../../../migrations"
	}

	if err := database.MigrateUp(ctx, db, migrationsDir); err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}

	// Clean up tasks table for isolated testing
	_, _ = db.ExecContext(ctx, "DELETE FROM tasks WHERE tenant_id IN ('default', 'tenant_a', 'tenant_b')")

	repo := task.NewPostgresRepository(db)
	return db, repo
}

func TestPostgresRepository_CRUD(t *testing.T) {
	db, repo := setupPostgresTest(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Initial list should be empty
	tasks, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}

	// 2. Create task
	now := time.Now().UTC().Truncate(time.Microsecond)
	id, err := task.GenerateID()
	if err != nil {
		t.Fatalf("failed to generate ID: %v", err)
	}

	input := task.Task{
		ID:          id,
		Title:       "Postgres Task 1",
		Description: "Database persisted task",
		Status:      task.StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if created.ID != id || created.Title != "Postgres Task 1" {
		t.Errorf("created task mismatch: %+v", created)
	}

	// 3. Duplicate ID rejection
	_, err = repo.Create(ctx, input)
	if !errors.Is(err, task.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists on duplicate ID, got: %v", err)
	}

	// 4. Get by ID
	fetched, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if fetched.ID != id || fetched.Description != "Database persisted task" {
		t.Errorf("fetched task mismatch: %+v", fetched)
	}

	// 5. Update task
	updateTime := time.Now().UTC().Truncate(time.Microsecond)
	toUpdate := task.Task{
		ID:          id,
		Title:       "Postgres Task Updated",
		Description: "Updated desc",
		Status:      task.StatusCompleted,
		UpdatedAt:   updateTime,
	}

	updated, err := repo.Update(ctx, toUpdate)
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}
	if updated.Title != "Postgres Task Updated" || updated.Status != task.StatusCompleted {
		t.Errorf("updated task mismatch: %+v", updated)
	}

	// 6. Delete task
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	// 7. Get after delete -> ErrNotFound
	_, err = repo.GetByID(ctx, id)
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound for deleted task, got: %v", err)
	}

	// 8. Delete again -> ErrNotFound
	if err := repo.Delete(ctx, id); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("expected ErrNotFound on redundant delete, got: %v", err)
	}
}

func TestPostgresRepository_TenantIsolation(t *testing.T) {
	db, _ := setupPostgresTest(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx := context.Background()

	repoA := task.NewPostgresRepositoryWithTenant(db, "tenant_a")
	repoB := task.NewPostgresRepositoryWithTenant(db, "tenant_b")

	// Tenant A creates task
	id, _ := task.GenerateID()
	now := time.Now().UTC()
	_, err := repoA.Create(ctx, task.Task{
		ID:        id,
		Title:     "Tenant A Confidential Task",
		Status:    task.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Tenant A create failed: %v", err)
	}

	// Tenant B tries to get Tenant A's task -> must return ErrNotFound
	_, err = repoB.GetByID(ctx, id)
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("Tenant B was able to access Tenant A's task! got: %v", err)
	}

	// Tenant B tries to update Tenant A's task -> must return ErrNotFound
	_, err = repoB.Update(ctx, task.Task{
		ID:        id,
		Title:     "Tenant B Mutation",
		Status:    task.StatusCompleted,
		UpdatedAt: now,
	})
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("Tenant B was able to update Tenant A's task! got: %v", err)
	}

	// Tenant B tries to delete Tenant A's task -> must return ErrNotFound
	err = repoB.Delete(ctx, id)
	if !errors.Is(err, task.ErrNotFound) {
		t.Errorf("Tenant B was able to delete Tenant A's task! got: %v", err)
	}

	// Tenant B lists tasks -> must not see Tenant A's task
	tasksB, err := repoB.List(ctx)
	if err != nil {
		t.Fatalf("Tenant B list failed: %v", err)
	}
	if len(tasksB) != 0 {
		t.Errorf("Tenant B listed %d tasks, expected 0", len(tasksB))
	}
}

func TestPostgresRepository_ContextCancellation(t *testing.T) {
	db, repo := setupPostgresTest(t)
	if db == nil {
		return
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	id, _ := task.GenerateID()
	_, err := repo.Create(ctx, task.Task{ID: id, Title: "Canceled"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

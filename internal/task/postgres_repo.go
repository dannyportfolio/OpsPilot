package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"opspilot/task-api/internal/auth"
)

// PostgresRepository provides a PostgreSQL persistence adapter conforming to task.Repository.
type PostgresRepository struct {
	db       *sql.DB
	tenantID string
}

// NewPostgresRepository initializes a PostgresRepository with the default tenant context.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return NewPostgresRepositoryWithTenant(db, "default")
}

// NewPostgresRepositoryWithTenant initializes a PostgresRepository with an explicit tenant ID.
func NewPostgresRepositoryWithTenant(db *sql.DB, tenantID string) *PostgresRepository {
	if strings.TrimSpace(tenantID) == "" {
		tenantID = "default"
	}
	return &PostgresRepository{
		db:       db,
		tenantID: tenantID,
	}
}

func (r *PostgresRepository) effectiveTenant(ctx context.Context) string {
	if tenant, ok := auth.TenantFromContext(ctx); ok && tenant != "" {
		return tenant
	}
	return r.tenantID
}

// Create inserts a new task record into PostgreSQL using parameterized SQL.
func (r *PostgresRepository) Create(ctx context.Context, t Task) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}

	tenantID := r.effectiveTenant(ctx)
	if t.TenantID != "" {
		tenantID = t.TenantID
	}

	query := `
	INSERT INTO tasks (id, title, description, status, tenant_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id, title, description, status, created_at, updated_at;
	`

	var created Task
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(
		ctx,
		query,
		t.ID,
		t.Title,
		t.Description,
		string(t.Status),
		tenantID,
		t.CreatedAt,
		t.UpdatedAt,
	).Scan(
		&created.ID,
		&created.Title,
		&created.Description,
		&created.Status,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		// PostgreSQL duplicate key violation error code 23505
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return Task{}, ErrAlreadyExists
		}
		return Task{}, fmt.Errorf("failed to insert task: %w", err)
	}

	created.CreatedAt = createdAt.UTC()
	created.UpdatedAt = updatedAt.UTC()

	return created, nil
}

// GetByID retrieves a task by ID within the current tenant boundary.
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}

	query := `
	SELECT id, title, description, status, created_at, updated_at
	FROM tasks
	WHERE id = $1 AND tenant_id = $2;
	`

	var t Task
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id, r.effectiveTenant(ctx)).Scan(
		&t.ID,
		&t.Title,
		&t.Description,
		&t.Status,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, fmt.Errorf("failed to query task by id: %w", err)
	}

	t.CreatedAt = createdAt.UTC()
	t.UpdatedAt = updatedAt.UTC()

	return t, nil
}

// List returns all tasks within the tenant boundary ordered newest first.
func (r *PostgresRepository) List(ctx context.Context) ([]Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	query := `
	SELECT id, title, description, status, created_at, updated_at
	FROM tasks
	WHERE tenant_id = $1
	ORDER BY created_at DESC;
	`

	rows, err := r.db.QueryContext(ctx, query, r.effectiveTenant(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks list: %w", err)
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var t Task
		var createdAt, updatedAt time.Time

		if err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.Description,
			&t.Status,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task row: %w", err)
		}

		t.CreatedAt = createdAt.UTC()
		t.UpdatedAt = updatedAt.UTC()
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during task row iteration: %w", err)
	}

	return tasks, nil
}

// Update modifies an existing task within the tenant boundary.
func (r *PostgresRepository) Update(ctx context.Context, t Task) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}

	query := `
	UPDATE tasks
	SET title = $1, description = $2, status = $3, updated_at = $4
	WHERE id = $5 AND tenant_id = $6
	RETURNING id, title, description, status, created_at, updated_at;
	`

	var updated Task
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(
		ctx,
		query,
		t.Title,
		t.Description,
		string(t.Status),
		t.UpdatedAt,
		t.ID,
		r.effectiveTenant(ctx),
	).Scan(
		&updated.ID,
		&updated.Title,
		&updated.Description,
		&updated.Status,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, fmt.Errorf("failed to update task: %w", err)
	}

	updated.CreatedAt = createdAt.UTC()
	updated.UpdatedAt = updatedAt.UTC()

	return updated, nil
}

// Delete removes a task by ID within the tenant boundary.
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	query := `
	DELETE FROM tasks
	WHERE id = $1 AND tenant_id = $2;
	`

	res, err := r.db.ExecContext(ctx, query, id, r.effectiveTenant(ctx))
	if err != nil {
		return fmt.Errorf("failed to execute delete task: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

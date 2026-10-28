package task

import "context"

// Repository defines the contract for task data persistence and retrieval.
// Decoupled from concrete storage mechanisms to allow in-memory, DynamoDB,
// or SQL database backends.
type Repository interface {
	// Create stores a new task.
	Create(ctx context.Context, t Task) (Task, error)

	// GetByID fetches a task by its unique ID. Returns ErrNotFound if not found.
	GetByID(ctx context.Context, id string) (Task, error)

	// List returns all tasks, optionally sorted by creation timestamp descending.
	List(ctx context.Context) ([]Task, error)

	// Update modifies an existing task. Returns ErrNotFound if the task does not exist.
	Update(ctx context.Context, t Task) (Task, error)

	// Delete removes a task by ID. Returns ErrNotFound if the task does not exist.
	Delete(ctx context.Context, id string) error
}

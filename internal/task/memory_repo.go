package task

import (
	"context"
	"sort"
	"sync"

	"opspilot/task-api/internal/auth"
)

// MemoryRepository provides a thread-safe, in-memory implementation of the Repository interface.
// Note: In-memory data is volatile and is reset when the process terminates.
type MemoryRepository struct {
	mu    sync.RWMutex
	tasks map[string]Task
}

// NewMemoryRepository initializes and returns a ready-to-use MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		tasks: make(map[string]Task),
	}
}

// Create stores a new task in memory.
func (r *MemoryRepository) Create(ctx context.Context, t Task) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[t.ID]; exists {
		return Task{}, ErrAlreadyExists
	}

	// Defensive copy
	item := t
	if item.TenantID == "" {
		if tenant, ok := auth.TenantFromContext(ctx); ok && tenant != "" {
			item.TenantID = tenant
		} else {
			item.TenantID = "default"
		}
	}
	r.tasks[item.ID] = item

	return item, nil
}

// GetByID retrieves a task by its unique ID. Returns ErrNotFound if it does not exist or belongs to another tenant.
func (r *MemoryRepository) GetByID(ctx context.Context, id string) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return Task{}, ErrNotFound
	}

	if tenant, ok := auth.TenantFromContext(ctx); ok && tenant != "" {
		if task.TenantID != tenant {
			return Task{}, ErrNotFound
		}
	}

	return task, nil
}

// List returns all tasks, ordered by CreatedAt descending.
// If an authenticated tenant is in context, only tasks belonging to that tenant are returned.
func (r *MemoryRepository) List(ctx context.Context) ([]Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	tenant, hasTenant := auth.TenantFromContext(ctx)

	result := make([]Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		if hasTenant && tenant != "" && t.TenantID != tenant {
			continue
		}
		result = append(result, t)
	}

	// Sort newest first
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

// Update modifies an existing task in memory. Returns ErrNotFound if the task does not exist or belongs to another tenant.
func (r *MemoryRepository) Update(ctx context.Context, t Task) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.tasks[t.ID]
	if !exists {
		return Task{}, ErrNotFound
	}

	if tenant, ok := auth.TenantFromContext(ctx); ok && tenant != "" {
		if existing.TenantID != tenant {
			return Task{}, ErrNotFound
		}
	}

	// Preserve initial creation time and tenant ID if not supplied
	item := t
	if item.CreatedAt.IsZero() {
		item.CreatedAt = existing.CreatedAt
	}
	if item.TenantID == "" {
		item.TenantID = existing.TenantID
	}

	r.tasks[item.ID] = item
	return item, nil
}

// Delete removes a task by its ID. Returns ErrNotFound if it does not exist or belongs to another tenant.
func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.tasks[id]
	if !exists {
		return ErrNotFound
	}

	if tenant, ok := auth.TenantFromContext(ctx); ok && tenant != "" {
		if existing.TenantID != tenant {
			return ErrNotFound
		}
	}

	delete(r.tasks, id)
	return nil
}

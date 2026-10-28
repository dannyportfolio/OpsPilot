package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"opspilot/task-api/internal/auth"
)

// Service defines application use cases for task management.
type Service interface {
	CreateTask(ctx context.Context, req CreateTaskRequest) (Task, error)
	GetTask(ctx context.Context, id string) (Task, error)
	ListTasks(ctx context.Context) ([]Task, error)
	UpdateTask(ctx context.Context, id string, req UpdateTaskRequest) (Task, error)
	DeleteTask(ctx context.Context, id string) error
}

type taskService struct {
	repo Repository
}

// NewService constructs a ready-to-use task application service.
func NewService(repo Repository) Service {
	return &taskService{repo: repo}
}

// CreateTask validates input, generates a secure UUID, sets initial timestamps, and persists the task.
func (s *taskService) CreateTask(ctx context.Context, req CreateTaskRequest) (Task, error) {
	if err := req.Validate(); err != nil {
		return Task{}, err
	}

	id, err := GenerateID()
	if err != nil {
		return Task{}, fmt.Errorf("failed to generate task identifier: %w", err)
	}

	now := time.Now().UTC()
	status := StatusPending
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}

	tenantID := "default"
	if tID, ok := auth.TenantFromContext(ctx); ok && tID != "" {
		tenantID = tID
	}

	t := Task{
		ID:          id,
		TenantID:    tenantID,
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.Create(ctx, t)
}

// GetTask validates the identifier format and retrieves the task from storage.
func (s *taskService) GetTask(ctx context.Context, id string) (Task, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return Task{}, &ValidationError{
			Errors: []FieldError{{Field: "id", Message: "task id is required"}},
		}
	}

	if !IsValidID(trimmedID) {
		return Task{}, &ValidationError{
			Errors: []FieldError{{Field: "id", Message: "task id must be a valid RFC 4122 UUID"}},
		}
	}

	return s.repo.GetByID(ctx, trimmedID)
}

// ListTasks retrieves all tasks from storage ordered newest first.
func (s *taskService) ListTasks(ctx context.Context) ([]Task, error) {
	return s.repo.List(ctx)
}

// UpdateTask validates input and identifier, and applies modifications.
func (s *taskService) UpdateTask(ctx context.Context, id string, req UpdateTaskRequest) (Task, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return Task{}, &ValidationError{
			Errors: []FieldError{{Field: "id", Message: "task id is required"}},
		}
	}

	if err := req.Validate(); err != nil {
		return Task{}, err
	}

	if !IsValidID(trimmedID) {
		return Task{}, &ValidationError{
			Errors: []FieldError{{Field: "id", Message: "task id must be a valid RFC 4122 UUID"}},
		}
	}

	now := time.Now().UTC()
	t := Task{
		ID:          trimmedID,
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Status:      req.Status,
		UpdatedAt:   now,
	}

	return s.repo.Update(ctx, t)
}

// DeleteTask validates the identifier and removes the task from storage.
func (s *taskService) DeleteTask(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return &ValidationError{
			Errors: []FieldError{{Field: "id", Message: "task id is required"}},
		}
	}

	if !IsValidID(trimmedID) {
		return &ValidationError{
			Errors: []FieldError{{Field: "id", Message: "task id must be a valid RFC 4122 UUID"}},
		}
	}

	return s.repo.Delete(ctx, trimmedID)
}

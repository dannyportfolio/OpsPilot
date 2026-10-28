package task

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Common domain errors.
var (
	ErrNotFound      = errors.New("task not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrAlreadyExists = errors.New("task already exists")
)

// Status represents the operational state of a task.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
)

// IsValid checks if the status matches recognized enum values.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted:
		return true
	default:
		return false
	}
}

// Task represents an item of work managed by the system.
type Task struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FieldError represents a specific input validation failure on a field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError aggregates one or more field validation errors.
type ValidationError struct {
	Errors []FieldError `json:"errors"`
}

func (v *ValidationError) Error() string {
	if len(v.Errors) == 0 {
		return "validation failed"
	}
	var messages []string
	for _, fe := range v.Errors {
		messages = append(messages, fmt.Sprintf("%s: %s", fe.Field, fe.Message))
	}
	return strings.Join(messages, "; ")
}

// CreateTaskRequest contains parameters required to create a new task.
type CreateTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Status      *Status `json:"status,omitempty"`
}

// Validate checks the payload for correctness and returns a ValidationError if any constraints fail.
func (r *CreateTaskRequest) Validate() error {
	var errs []FieldError

	trimmedTitle := strings.TrimSpace(r.Title)
	if trimmedTitle == "" {
		errs = append(errs, FieldError{Field: "title", Message: "title is required and cannot be empty"})
	} else if len(trimmedTitle) > 200 {
		errs = append(errs, FieldError{Field: "title", Message: "title cannot exceed 200 characters"})
	}

	if len(r.Description) > 2000 {
		errs = append(errs, FieldError{Field: "description", Message: "description cannot exceed 2000 characters"})
	}

	if r.Status != nil && *r.Status != "" {
		if !r.Status.IsValid() {
			errs = append(errs, FieldError{
				Field:   "status",
				Message: "status must be one of: pending, in_progress, completed",
			})
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

// UpdateTaskRequest contains parameters required to update an existing task.
type UpdateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      Status `json:"status"`
}

// Validate checks the payload for correctness and returns a ValidationError if any constraints fail.
func (r *UpdateTaskRequest) Validate() error {
	var errs []FieldError

	trimmedTitle := strings.TrimSpace(r.Title)
	if trimmedTitle == "" {
		errs = append(errs, FieldError{Field: "title", Message: "title is required and cannot be empty"})
	} else if len(trimmedTitle) > 200 {
		errs = append(errs, FieldError{Field: "title", Message: "title cannot exceed 200 characters"})
	}

	if len(r.Description) > 2000 {
		errs = append(errs, FieldError{Field: "description", Message: "description cannot exceed 2000 characters"})
	}

	if !r.Status.IsValid() {
		errs = append(errs, FieldError{
			Field:   "status",
			Message: "status must be one of: pending, in_progress, completed",
		})
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

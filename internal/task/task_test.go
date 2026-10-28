package task_test

import (
	"errors"
	"strings"
	"testing"

	"opspilot/task-api/internal/task"
)

func ptrStatus(s task.Status) *task.Status {
	return &s
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status task.Status
		want   bool
	}{
		{"pending is valid", task.StatusPending, true},
		{"in_progress is valid", task.StatusInProgress, true},
		{"completed is valid", task.StatusCompleted, true},
		{"empty is invalid", "", false},
		{"unknown is invalid", "invalid_status", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.status.IsValid(); got != tc.want {
				t.Errorf("Status.IsValid() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCreateTaskRequest_Validate(t *testing.T) {
	longDescription := strings.Repeat("a", 2001)
	validDescription := strings.Repeat("a", 2000)
	longTitle := strings.Repeat("t", 201)

	tests := []struct {
		name       string
		req        task.CreateTaskRequest
		wantErr    bool
		errorField string
	}{
		{
			name: "valid request minimal",
			req: task.CreateTaskRequest{
				Title: "Write Go code",
			},
			wantErr: false,
		},
		{
			name: "valid request complete",
			req: task.CreateTaskRequest{
				Title:       "Deploy Serverless API",
				Description: "Deploy to AWS Lambda with API Gateway",
				Status:      ptrStatus(task.StatusInProgress),
			},
			wantErr: false,
		},
		{
			name: "valid request with boundary description",
			req: task.CreateTaskRequest{
				Title:       "Test boundary",
				Description: validDescription,
			},
			wantErr: false,
		},
		{
			name: "missing title",
			req: task.CreateTaskRequest{
				Title: "",
			},
			wantErr:    true,
			errorField: "title",
		},
		{
			name: "whitespace title",
			req: task.CreateTaskRequest{
				Title: "   \t \n ",
			},
			wantErr:    true,
			errorField: "title",
		},
		{
			name: "title exceeds 200 chars",
			req: task.CreateTaskRequest{
				Title: longTitle,
			},
			wantErr:    true,
			errorField: "title",
		},
		{
			name: "description exceeds 2000 chars",
			req: task.CreateTaskRequest{
				Title:       "Valid Title",
				Description: longDescription,
			},
			wantErr:    true,
			errorField: "description",
		},
		{
			name: "invalid status enum",
			req: task.CreateTaskRequest{
				Title:  "Valid Title",
				Status: ptrStatus("archive"),
			},
			wantErr:    true,
			errorField: "status",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr && tc.errorField != "" {
				var valErr *task.ValidationError
				if !errors.As(err, &valErr) {
					t.Fatalf("expected *task.ValidationError, got %T", err)
				}
				found := false
				for _, fe := range valErr.Errors {
					if fe.Field == tc.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected validation error on field %q, got: %v", tc.errorField, valErr.Errors)
				}
			}
		})
	}
}

func TestUpdateTaskRequest_Validate(t *testing.T) {
	tests := []struct {
		name       string
		req        task.UpdateTaskRequest
		wantErr    bool
		errorField string
	}{
		{
			name: "valid update",
			req: task.UpdateTaskRequest{
				Title:       "Updated Task",
				Description: "Updated details",
				Status:      task.StatusCompleted,
			},
			wantErr: false,
		},
		{
			name: "missing status in update",
			req: task.UpdateTaskRequest{
				Title: "Updated Task",
			},
			wantErr:    true,
			errorField: "status",
		},
		{
			name: "invalid status in update",
			req: task.UpdateTaskRequest{
				Title:  "Updated Task",
				Status: "deleted",
			},
			wantErr:    true,
			errorField: "status",
		},
		{
			name: "empty title in update",
			req: task.UpdateTaskRequest{
				Title:  "",
				Status: task.StatusCompleted,
			},
			wantErr:    true,
			errorField: "title",
		},
		{
			name: "long description in update",
			req: task.UpdateTaskRequest{
				Title:       "Updated Task",
				Description: strings.Repeat("x", 2001),
				Status:      task.StatusCompleted,
			},
			wantErr:    true,
			errorField: "description",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr && tc.errorField != "" {
				var valErr *task.ValidationError
				if !errors.As(err, &valErr) {
					t.Fatalf("expected *task.ValidationError, got %T", err)
				}
				found := false
				for _, fe := range valErr.Errors {
					if fe.Field == tc.errorField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected validation error on field %q, got: %v", tc.errorField, valErr.Errors)
				}
			}
		})
	}
}

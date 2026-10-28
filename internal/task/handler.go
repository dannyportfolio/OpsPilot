package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"opspilot/task-api/internal/platform/response"
)

// Handler provides HTTP handlers for managing tasks.
type Handler struct {
	svc Service
}

// NewHandler constructs a new Handler delegating to the application service.
func NewHandler(svc Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

// Health handles GET /health.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Create handles POST /tasks.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("malformed JSON body: %v", err))
		return
	}

	created, err := h.svc.CreateTask(r.Context(), req)
	if err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			response.Error(w, http.StatusBadRequest, "validation failed", valErr.Errors)
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to store task")
		return
	}

	w.Header().Set("Location", "/tasks/"+created.ID)
	response.JSON(w, http.StatusCreated, created)
}

// List handles GET /tasks.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.ListTasks(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve tasks")
		return
	}

	// Ensure empty slice serializes to [] rather than null
	if tasks == nil {
		tasks = []Task{}
	}

	response.JSON(w, http.StatusOK, tasks)
}

// GetByID handles GET /tasks/{id}.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	t, err := h.svc.GetTask(r.Context(), id)
	if err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			response.Error(w, http.StatusBadRequest, "validation failed", valErr.Errors)
			return
		}
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, fmt.Sprintf("task with id %q not found", id))
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to retrieve task")
		return
	}

	response.JSON(w, http.StatusOK, t)
}

// Update handles PUT /tasks/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdateTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, fmt.Sprintf("malformed JSON body: %v", err))
		return
	}

	updated, err := h.svc.UpdateTask(r.Context(), id, req)
	if err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			response.Error(w, http.StatusBadRequest, "validation failed", valErr.Errors)
			return
		}
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, fmt.Sprintf("task with id %q not found", id))
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	response.JSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /tasks/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.DeleteTask(r.Context(), id); err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			response.Error(w, http.StatusBadRequest, "validation failed", valErr.Errors)
			return
		}
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, fmt.Sprintf("task with id %q not found", id))
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// decodeJSON strictly decodes incoming JSON payloads.
// It enforces 1 MB body boundary, rejects unknown fields, empty payloads, and multiple/trailing data.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("request body is empty")
	}
	defer r.Body.Close()

	// 1 MB payload limit bound
	limitedReader := http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(limitedReader)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	// Ensure there are no additional JSON tokens
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body contains multiple JSON values or extra data")
	}

	return nil
}

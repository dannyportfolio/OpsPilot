package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"opspilot/task-api/internal/platform/response"
)

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := map[string]string{"message": "hello"}

	response.JSON(rec, http.StatusOK, payload)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %q", contentType)
	}

	var got map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if got["message"] != "hello" {
		t.Errorf("expected payload message 'hello', got %q", got["message"])
	}
}

func TestError(t *testing.T) {
	t.Run("error without details", func(t *testing.T) {
		rec := httptest.NewRecorder()
		response.Error(rec, http.StatusBadRequest, "invalid request")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}

		if got.Error != "invalid request" {
			t.Errorf("expected error message 'invalid request', got %q", got.Error)
		}
		if got.Details != nil {
			t.Errorf("expected details to be nil, got %v", got.Details)
		}
	})

	t.Run("error with details", func(t *testing.T) {
		rec := httptest.NewRecorder()
		details := []string{"field1 required", "field2 invalid"}
		response.Error(rec, http.StatusUnprocessableEntity, "validation failed", details)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode error response: %v", err)
		}

		if got.Error != "validation failed" {
			t.Errorf("expected error message 'validation failed', got %q", got.Error)
		}

		detSlice, ok := got.Details.([]any)
		if !ok || len(detSlice) != 2 {
			t.Fatalf("expected 2 details items, got %v", got.Details)
		}
	})
}

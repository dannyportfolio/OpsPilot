package response

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents a standardized JSON error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details any    `json:"details,omitempty"`
}

// JSON serializes data as JSON and sets the given HTTP status code and Content-Type header.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// If encoding fails, we cannot rewrite the header if already written,
			// but we can log or handle if necessary.
			_ = err
		}
	}
}

// Error writes a standardized JSON error response.
func Error(w http.ResponseWriter, status int, message string, details ...any) {
	resp := ErrorResponse{
		Error: message,
	}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	JSON(w, status, resp)
}

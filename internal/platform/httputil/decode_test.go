package httputil_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"opspilot/task-api/internal/platform/httputil"
)

type samplePayload struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestDecodeJSON(t *testing.T) {
	t.Run("valid single JSON object", func(t *testing.T) {
		body := `{"name":"OpsPilot","age":1}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		rec := httptest.NewRecorder()

		var got samplePayload
		err := httputil.DecodeJSON(rec, req, &got, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "OpsPilot" || got.Age != 1 {
			t.Errorf("unexpected decoded content: %+v", got)
		}
	})

	t.Run("disallow unknown fields", func(t *testing.T) {
		body := `{"name":"OpsPilot","extra":"field"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		rec := httptest.NewRecorder()

		var got samplePayload
		err := httputil.DecodeJSON(rec, req, &got, 0)
		if err == nil {
			t.Fatal("expected error for unknown fields")
		}
	})

	t.Run("reject multiple JSON objects", func(t *testing.T) {
		body := `{"name":"first"}{"name":"second"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		rec := httptest.NewRecorder()

		var got samplePayload
		err := httputil.DecodeJSON(rec, req, &got, 0)
		if err == nil {
			t.Fatal("expected error for multiple JSON values")
		}
	})

	t.Run("body exceeds size limit", func(t *testing.T) {
		largeBody := `{"name":"` + strings.Repeat("a", 100) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(largeBody))
		rec := httptest.NewRecorder()

		var got samplePayload
		err := httputil.DecodeJSON(rec, req, &got, 50) // limit 50 bytes
		if err == nil {
			t.Fatal("expected error for payload exceeding limit")
		}
	})

	t.Run("nil request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Body = nil
		rec := httptest.NewRecorder()

		var got samplePayload
		err := httputil.DecodeJSON(rec, req, &got, 0)
		if err == nil {
			t.Fatal("expected error for nil body")
		}
	})

	t.Run("malformed json syntax", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{not-json}`))
		rec := httptest.NewRecorder()

		var got samplePayload
		err := httputil.DecodeJSON(rec, req, &got, 0)
		if err == nil {
			t.Fatal("expected error for malformed json")
		}
	})
}

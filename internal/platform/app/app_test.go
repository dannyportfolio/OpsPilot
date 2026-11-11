package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opspilot/task-api/internal/platform/app"
	"opspilot/task-api/internal/platform/config"
)

func TestAppBuild_InMemory(t *testing.T) {
	cfg := &config.Config{
		AppEnv:        "test",
		Port:          "8080",
		StorageDriver: "memory",
		LogLevel:      "INFO",
		JWTSecret:     "test-secret-at-least-32-bytes-long",
		TokenTTL:      1 * time.Hour,
	}

	application, err := app.Build(context.Background(), cfg, app.BuildOptions{})
	if err != nil {
		t.Fatalf("failed to build app: %v", err)
	}

	if application.Handler == nil {
		t.Fatal("expected non-nil handler")
	}

	// Verify unauthenticated /health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	application.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /health, got %d", rec.Code)
	}

	// Verify secured /tasks endpoint rejects unauthenticated request
	reqTasks := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	recTasks := httptest.NewRecorder()
	application.Handler.ServeHTTP(recTasks, reqTasks)

	if recTasks.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for /tasks, got %d", recTasks.Code)
	}
}

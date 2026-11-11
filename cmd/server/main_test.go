package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
	"opspilot/task-api/internal/platform/app"
	"opspilot/task-api/internal/platform/config"
)

func TestServerAssembly_RoutesAndSecurity(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "memory")
	t.Setenv("AUTH_JWT_SECRET", "test-secret-at-least-32-bytes-long")
	t.Setenv("SEED_DEV_DATA", "true")

	cfg := config.Load()
	application, err := app.Build(context.Background(), cfg, app.BuildOptions{})
	if err != nil {
		t.Fatalf("failed to build app: %v", err)
	}

	// 1. Health check is accessible at root
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	application.Handler.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /health, got %d", recHealth.Code)
	}

	// 2. Unauthenticated request to /tasks is rejected with 401
	reqTasks := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	recTasks := httptest.NewRecorder()
	application.Handler.ServeHTTP(recTasks, reqTasks)
	if recTasks.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for /tasks without auth, got %d", recTasks.Code)
	}

	// 3. Login through mounted authRouter works
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	recLogin := httptest.NewRecorder()
	application.Handler.ServeHTTP(recLogin, loginReq)
	if recLogin.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for empty login payload, got %d", recLogin.Code)
	}
}

func TestTokenService_InvalidSecretStartupFailure(t *testing.T) {
	// Secret shorter than 32 bytes must return error
	_, err := auth.NewTokenService("too-short-secret-under-32-bytes", "opspilot", "opspilot-api", 1*time.Hour)
	if err == nil {
		t.Fatal("expected error when JWT secret is shorter than 32 bytes")
	}
}

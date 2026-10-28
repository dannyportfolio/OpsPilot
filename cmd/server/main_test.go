package main_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
	"opspilot/task-api/internal/platform/config"
	"opspilot/task-api/internal/task"
)

func TestServerAssembly_RoutesAndSecurity(t *testing.T) {
	cfg := config.Load()

	tokenSvc, err := auth.NewTokenService(cfg.JWTSecret, "opspilot", "opspilot-api", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("failed to initialize token service: %v", err)
	}

	revStore := auth.NewMemoryRevocationStore()
	rateLimiter := auth.NewMemoryRateLimiter(5, 1*time.Minute, 1*time.Minute)
	userRepo := auth.NewMemoryUserRepository()
	authSvc := auth.NewService(userRepo, tokenSvc, revStore, rateLimiter, nil)

	// Seed admin
	_, err = authSvc.CreateUser(context.Background(), "default-tenant", "admin@opspilot.internal", "AdminSecure123!", auth.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to seed admin user: %v", err)
	}

	taskRepo := task.NewMemoryRepository()
	taskSvc := task.NewService(taskRepo)

	authRouter := auth.NewRouter(authSvc, tokenSvc, revStore, nil)
	taskRouter := task.NewSecuredRouter(taskSvc, tokenSvc, revStore, nil)

	rootMux := http.NewServeMux()
	rootMux.Handle("/auth/", authRouter)
	rootMux.Handle("/", taskRouter)

	// 1. Health check is accessible at root
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	rootMux.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Errorf("expected 200 OK for /health, got %d", recHealth.Code)
	}

	// 2. Unauthenticated request to /tasks is rejected with 401
	reqTasks := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	recTasks := httptest.NewRecorder()
	rootMux.ServeHTTP(recTasks, reqTasks)
	if recTasks.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for /tasks without auth, got %d", recTasks.Code)
	}

	// 3. Login through mounted authRouter works
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	// (Without body should return 400 Bad Request, proving route is reached)
	recLogin := httptest.NewRecorder()
	rootMux.ServeHTTP(recLogin, loginReq)
	if recLogin.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for empty login payload, got %d", recLogin.Code)
	}
}

func TestTokenService_InvalidSecretStartupFailure(t *testing.T) {
	// Secret shorter than 16 bytes must return error
	_, err := auth.NewTokenService("too-short", "opspilot", "opspilot-api", 1*time.Hour)
	if err == nil {
		t.Fatal("expected error when JWT secret is shorter than 16 bytes")
	}
}

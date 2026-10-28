package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
)

func setupAuthRouter() (http.Handler, auth.Service, auth.TokenService, auth.RevocationStore) {
	repo := auth.NewMemoryUserRepository()
	tokenSvc, _ := auth.NewTokenService("test-secret-at-least-32-bytes-long", "opspilot", "opspilot-api", 1*time.Hour)
	revStore := auth.NewMemoryRevocationStore()
	limiter := auth.NewMemoryRateLimiter(5, 1*time.Minute, 1*time.Minute)

	svc := auth.NewService(repo, tokenSvc, revStore, limiter, nil)
	router := auth.NewRouter(svc, tokenSvc, revStore, nil)

	return router, svc, tokenSvc, revStore
}

func TestAuthHandler_Login(t *testing.T) {
	router, svc, _, _ := setupAuthRouter()
	ctx := context.Background()

	// Seed user
	_, err := svc.CreateUser(ctx, "tenant-1", "user@example.com", "Password123!", auth.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	tests := []struct {
		name         string
		method       string
		body         string
		expectedCode int
	}{
		{
			name:         "Success",
			method:       http.MethodPost,
			body:         `{"email":"user@example.com","password":"Password123!"}`,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Invalid Password",
			method:       http.MethodPost,
			body:         `{"email":"user@example.com","password":"WrongPassword!"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Unknown Email",
			method:       http.MethodPost,
			body:         `{"email":"ghost@example.com","password":"Password123!"}`,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Missing Password",
			method:       http.MethodPost,
			body:         `{"email":"user@example.com"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Malformed JSON",
			method:       http.MethodPost,
			body:         `{bad-json}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Unknown Fields",
			method:       http.MethodPost,
			body:         `{"email":"user@example.com","password":"Password123!","extra":"field"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Method Not Allowed",
			method:       http.MethodGet,
			body:         "",
			expectedCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/auth/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d, body: %s", tt.expectedCode, rec.Code, rec.Body.String())
			}

			if tt.expectedCode == http.StatusOK {
				var resp auth.LoginResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Token == "" || resp.TokenType != "Bearer" {
					t.Errorf("invalid login response: %+v", resp)
				}
			}
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	router, svc, tokenSvc, _ := setupAuthRouter()
	ctx := context.Background()

	_, _ = svc.CreateUser(ctx, "tenant-1", "user@example.com", "Password123!", auth.RoleAdmin)
	loginResp, _ := svc.Login(ctx, auth.LoginRequest{Email: "user@example.com", Password: "Password123!"}, "127.0.0.1")

	// 1. Successful logout
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body: %s", rec.Code, rec.Body.String())
	}

	// 2. Subsequent call with same token must fail with 401 (token revoked)
	req2 := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req2.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec2 := httptest.NewRecorder()

	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for revoked token, got %d", rec2.Code)
	}

	// 3. Logout without token must fail with 401
	req3 := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec3 := httptest.NewRecorder()

	router.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for missing token, got %d", rec3.Code)
	}

	_ = tokenSvc
}

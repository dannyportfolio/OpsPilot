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

func TestAuthHandler_RateLimit_SpoofedXFF(t *testing.T) {
	router, svc, _, _ := setupAuthRouter()
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, "tenant-1", "victim@example.com", "Password123!", auth.RoleMember)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// An untrusted external caller (RemoteAddr: 198.51.100.1:54321) attempts brute force
	// by spoofing different X-Forwarded-For headers on each request.
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"victim@example.com","password":"WrongPassword"}`))
		req.RemoteAddr = "198.51.100.1:54321"
		req.Header.Set("X-Forwarded-For", "203.0.113."+string(rune('1'+i)))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401 Unauthorized, got %d", i+1, rec.Code)
		}
	}

	// 6th attempt from the same RemoteAddr with yet another spoofed IP MUST be blocked (429)
	req6 := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"victim@example.com","password":"WrongPassword"}`))
	req6.RemoteAddr = "198.51.100.1:54321"
	req6.Header.Set("X-Forwarded-For", "203.0.113.99")
	rec6 := httptest.NewRecorder()

	router.ServeHTTP(rec6, req6)
	if rec6.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests after 5 failures despite spoofed XFF, got %d", rec6.Code)
	}
}

func TestAuthHandler_UserManagementAndProfile(t *testing.T) {
	router, svc, tokenSvc, _ := setupAuthRouter()
	ctx := context.Background()

	// Seed admin and member
	adminUser, err := svc.CreateUser(ctx, "tenant-test", "admin@tenant.com", "AdminPass123!", auth.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	memberUser, err := svc.CreateUser(ctx, "tenant-test", "member@tenant.com", "MemberPass123!", auth.RoleMember)
	if err != nil {
		t.Fatalf("failed to create member: %v", err)
	}

	adminToken, _, _ := tokenSvc.GenerateToken(auth.User{
		ID:       adminUser.ID,
		TenantID: adminUser.TenantID,
		Email:    adminUser.Email,
		Role:     auth.RoleAdmin,
	})
	memberToken, _, _ := tokenSvc.GenerateToken(auth.User{
		ID:       memberUser.ID,
		TenantID: memberUser.TenantID,
		Email:    memberUser.Email,
		Role:     auth.RoleMember,
	})

	// 1. Admin creates new user via POST /auth/users
	newUserPayload := `{"email":"newhire@tenant.com","password":"NewHirePass123!","role":"member"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/auth/users", bytes.NewBufferString(newUserPayload))
	reqCreate.Header.Set("Authorization", "Bearer "+adminToken)
	recCreate := httptest.NewRecorder()
	router.ServeHTTP(recCreate, reqCreate)

	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for admin user creation, got %d, body: %s", recCreate.Code, recCreate.Body.String())
	}

	var createdResp auth.UserResponse
	if err := json.Unmarshal(recCreate.Body.Bytes(), &createdResp); err != nil {
		t.Fatalf("failed to decode created user: %v", err)
	}
	if createdResp.Email != "newhire@tenant.com" || createdResp.TenantID != "tenant-test" {
		t.Errorf("mismatched user attributes: %+v", createdResp)
	}

	// 2. Member calling POST /auth/users is rejected (403 Forbidden)
	reqForbidden := httptest.NewRequest(http.MethodPost, "/auth/users", bytes.NewBufferString(newUserPayload))
	reqForbidden.Header.Set("Authorization", "Bearer "+memberToken)
	recForbidden := httptest.NewRecorder()
	router.ServeHTTP(recForbidden, reqForbidden)

	if recForbidden.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for member user creation, got %d", recForbidden.Code)
	}

	// 3. User calls GET /auth/me
	reqMe := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+memberToken)
	recMe := httptest.NewRecorder()
	router.ServeHTTP(recMe, reqMe)

	if recMe.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /auth/me, got %d", recMe.Code)
	}
	var meResp auth.UserResponse
	if err := json.Unmarshal(recMe.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("failed to decode profile: %v", err)
	}
	if meResp.ID != memberUser.ID || meResp.Email != "member@tenant.com" {
		t.Errorf("unexpected profile data: %+v", meResp)
	}
}

package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
)

func TestAuthMiddleware(t *testing.T) {
	tokenSvc, _ := auth.NewTokenService("test-secret-at-least-32-bytes-long", "opspilot", "opspilot-api", 1*time.Hour)
	revStore := auth.NewMemoryRevocationStore()

	var recordedClaims *auth.Claims
	sampleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.UserFromContext(r.Context())
		if ok {
			recordedClaims = claims
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := auth.Authenticate(tokenSvc, revStore, nil)
	wrapped := middleware(sampleHandler)

	user := auth.User{
		ID:       "user-1",
		TenantID: "tenant-alpha",
		Email:    "member@example.com",
		Role:     auth.RoleMember,
	}
	validToken, _, _ := tokenSvc.GenerateToken(user)

	// 1. Success
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}
	if recordedClaims == nil || recordedClaims.TenantID != "tenant-alpha" {
		t.Errorf("context claims not properly propagated: %+v", recordedClaims)
	}

	// 2. Missing authorization header
	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing header, got %d", rec2.Code)
	}

	// 3. Bad authorization header format
	req3 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req3.Header.Set("Authorization", "Basic 12345")
	rec3 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for bad format, got %d", rec3.Code)
	}

	// 4. Revoked token
	_ = revStore.Revoke(recordedClaims.JTI, time.Now().UTC().Add(1*time.Hour))
	req4 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req4.Header.Set("Authorization", "Bearer "+validToken)
	rec4 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for revoked token, got %d", rec4.Code)
	}
}

func TestRequireRoleMiddleware(t *testing.T) {
	adminOnly := auth.RequireRole(auth.RoleAdmin)
	target := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := adminOnly(target)

	// 1. User with Admin role -> 200 OK
	adminUser := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, TenantID: "t1"}
	req1 := httptest.NewRequest(http.MethodDelete, "/resource", nil)
	req1 = req1.WithContext(auth.WithUser(req1.Context(), adminUser))
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("expected 200 OK for admin, got %d", rec1.Code)
	}

	// 2. User with Member role -> 403 Forbidden
	memberUser := auth.Claims{Subject: "member-1", Role: auth.RoleMember, TenantID: "t1"}
	req2 := httptest.NewRequest(http.MethodDelete, "/resource", nil)
	req2 = req2.WithContext(auth.WithUser(req2.Context(), memberUser))
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for member, got %d", rec2.Code)
	}

	// 3. Unauthenticated request -> 401 Unauthorized
	req3 := httptest.NewRequest(http.MethodDelete, "/resource", nil)
	rec3 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for unauthenticated, got %d", rec3.Code)
	}
}

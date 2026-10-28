package auth_test

import (
	"strings"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
)

func TestTokenService_GenerateAndValidate(t *testing.T) {
	secret := "super-secure-secret-key-at-least-32-bytes"
	svc, err := auth.NewTokenService(secret, auth.DefaultIssuer, auth.DefaultAudience, 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error creating TokenService: %v", err)
	}

	user := auth.User{
		ID:       "user-123",
		TenantID: "tenant-abc",
		Email:    "alice@example.com",
		Role:     auth.RoleAdmin,
	}

	tokenStr, claims, err := svc.GenerateToken(user)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	if tokenStr == "" {
		t.Fatal("token should not be empty")
	}

	if claims.Subject != user.ID || claims.TenantID != user.TenantID || claims.Role != user.Role {
		t.Fatalf("claims mismatch: %+v", claims)
	}

	// Validate valid token
	validated, err := svc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("unexpected error validating valid token: %v", err)
	}

	if validated.Subject != user.ID || validated.TenantID != user.TenantID || validated.Role != user.Role {
		t.Errorf("validated claims mismatch: %+v", validated)
	}
}

func TestTokenService_TamperedToken(t *testing.T) {
	secret := "super-secure-secret-key-at-least-32-bytes"
	svc, err := auth.NewTokenService(secret, auth.DefaultIssuer, auth.DefaultAudience, 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error creating TokenService: %v", err)
	}

	user := auth.User{ID: "user-1", TenantID: "tenant-1", Role: auth.RoleMember}
	tokenStr, _, err := svc.GenerateToken(user)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	parts := strings.Split(tokenStr, ".")
	// Tamper payload
	tampered := parts[0] + ".e30." + parts[2]
	_, err = svc.ValidateToken(tampered)
	if err == nil {
		t.Error("expected error for tampered token")
	}

	// Tampered signature
	tamperedSig := parts[0] + "." + parts[1] + ".invalidSig"
	_, err = svc.ValidateToken(tamperedSig)
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestTokenService_ExpiredToken(t *testing.T) {
	secret := "super-secure-secret-key-at-least-32-bytes"
	// Create service with negative/immediate expiration
	svc, err := auth.NewTokenService(secret, auth.DefaultIssuer, auth.DefaultAudience, -1*time.Second)
	if err != nil {
		t.Fatalf("unexpected error creating TokenService: %v", err)
	}

	user := auth.User{ID: "user-exp", TenantID: "tenant-exp", Role: auth.RoleViewer}
	tokenStr, _, err := svc.GenerateToken(user)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	// Wait a tiny bit to ensure now > exp
	time.Sleep(10 * time.Millisecond)

	_, err = svc.ValidateToken(tokenStr)
	if err != auth.ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestTokenService_IssuerAndAudienceValidation(t *testing.T) {
	secret := "super-secure-secret-key-at-least-32-bytes"
	svc1, _ := auth.NewTokenService(secret, "issuer-1", "audience-1", 1*time.Hour)
	svc2, _ := auth.NewTokenService(secret, "issuer-2", "audience-1", 1*time.Hour)
	svc3, _ := auth.NewTokenService(secret, "issuer-1", "audience-2", 1*time.Hour)

	user := auth.User{ID: "user-1", TenantID: "tenant-1", Role: auth.RoleAdmin}
	tokenStr, _, _ := svc1.GenerateToken(user)

	// Validate with different issuer
	if _, err := svc2.ValidateToken(tokenStr); err == nil {
		t.Error("expected error validating token with mismatched issuer")
	}

	// Validate with different audience
	if _, err := svc3.ValidateToken(tokenStr); err == nil {
		t.Error("expected error validating token with mismatched audience")
	}
}

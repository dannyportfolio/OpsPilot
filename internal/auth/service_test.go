package auth_test

import (
	"context"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
)

func TestAuthService_LoginSuccessAndLogout(t *testing.T) {
	repo := auth.NewMemoryUserRepository()
	tokenSvc, _ := auth.NewTokenService("test-secret-at-least-32-bytes-long", "opspilot", "opspilot-api", 1*time.Hour)
	revStore := auth.NewMemoryRevocationStore()
	limiter := auth.NewMemoryRateLimiter(5, 1*time.Minute, 1*time.Minute)

	svc := auth.NewService(repo, tokenSvc, revStore, limiter, nil)
	ctx := context.Background()

	// Seed user
	userResp, err := svc.CreateUser(ctx, "tenant-1", "admin@example.com", "Password123!", auth.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if userResp.Role != auth.RoleAdmin {
		t.Errorf("expected RoleAdmin, got %s", userResp.Role)
	}

	// Successful login
	loginResp, err := svc.Login(ctx, auth.LoginRequest{
		Email:    "admin@example.com",
		Password: "Password123!",
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("expected login to succeed, got: %v", err)
	}

	if loginResp.Token == "" {
		t.Error("expected non-empty token")
	}
	if loginResp.User.Email != "admin@example.com" {
		t.Errorf("expected user email admin@example.com, got %s", loginResp.User.Email)
	}

	// Validate token
	claims, err := tokenSvc.ValidateToken(loginResp.Token)
	if err != nil {
		t.Fatalf("expected valid token, got: %v", err)
	}

	// Logout
	err = svc.Logout(ctx, claims)
	if err != nil {
		t.Fatalf("expected logout to succeed, got: %v", err)
	}

	// Verify token is now revoked
	if !revStore.IsRevoked(claims.JTI) {
		t.Error("expected token JTI to be revoked")
	}
}

func TestAuthService_InvalidCredentials(t *testing.T) {
	repo := auth.NewMemoryUserRepository()
	tokenSvc, _ := auth.NewTokenService("test-secret-at-least-32-bytes-long", "opspilot", "opspilot-api", 1*time.Hour)
	revStore := auth.NewMemoryRevocationStore()
	limiter := auth.NewMemoryRateLimiter(5, 1*time.Minute, 1*time.Minute)

	svc := auth.NewService(repo, tokenSvc, revStore, limiter, nil)
	ctx := context.Background()

	_, _ = svc.CreateUser(ctx, "tenant-1", "user@example.com", "CorrectPassword123", auth.RoleMember)

	// Wrong password
	_, err := svc.Login(ctx, auth.LoginRequest{
		Email:    "user@example.com",
		Password: "WrongPassword!",
	}, "127.0.0.1")
	if err != auth.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for wrong password, got: %v", err)
	}

	// Non-existent user
	_, err = svc.Login(ctx, auth.LoginRequest{
		Email:    "unknown@example.com",
		Password: "AnyPassword123",
	}, "127.0.0.1")
	if err != auth.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for unknown user, got: %v", err)
	}
}

func TestAuthService_SuspendedUser(t *testing.T) {
	repo := auth.NewMemoryUserRepository()
	tokenSvc, _ := auth.NewTokenService("test-secret-at-least-32-bytes-long", "opspilot", "opspilot-api", 1*time.Hour)
	svc := auth.NewService(repo, tokenSvc, nil, nil, nil)
	ctx := context.Background()

	created, _ := svc.CreateUser(ctx, "tenant-1", "suspended@example.com", "Password123!", auth.RoleViewer)

	// Update to suspended
	rawUser, _ := repo.GetByID(ctx, created.ID)
	rawUser.Status = auth.StatusSuspended
	_, _ = repo.Update(ctx, rawUser)

	_, err := svc.Login(ctx, auth.LoginRequest{
		Email:    "suspended@example.com",
		Password: "Password123!",
	}, "127.0.0.1")

	if err != auth.ErrUserSuspended {
		t.Errorf("expected ErrUserSuspended, got: %v", err)
	}
}

func TestAuthService_RateLimiting(t *testing.T) {
	repo := auth.NewMemoryUserRepository()
	tokenSvc, _ := auth.NewTokenService("test-secret-at-least-32-bytes-long", "opspilot", "opspilot-api", 1*time.Hour)
	limiter := auth.NewMemoryRateLimiter(2, 1*time.Minute, 1*time.Minute)
	svc := auth.NewService(repo, tokenSvc, nil, limiter, nil)
	ctx := context.Background()

	// 2 failed attempts
	_, _ = svc.Login(ctx, auth.LoginRequest{Email: "user@example.com", Password: "wrong"}, "1.2.3.4")
	_, _ = svc.Login(ctx, auth.LoginRequest{Email: "user@example.com", Password: "wrong"}, "1.2.3.4")

	// 3rd attempt should be blocked by rate limiter
	_, err := svc.Login(ctx, auth.LoginRequest{Email: "user@example.com", Password: "wrong"}, "1.2.3.4")
	if err != auth.ErrRateLimitExceeded {
		t.Errorf("expected ErrRateLimitExceeded, got: %v", err)
	}
}

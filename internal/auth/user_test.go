package auth_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
)

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		role  auth.Role
		valid bool
	}{
		{auth.RoleAdmin, true},
		{auth.RoleMember, true},
		{auth.RoleViewer, true},
		{auth.Role("superadmin"), false},
		{auth.Role(""), false},
	}

	for _, tt := range tests {
		if got := tt.role.IsValid(); got != tt.valid {
			t.Errorf("Role(%q).IsValid() = %v, expected %v", tt.role, got, tt.valid)
		}
	}
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		status auth.Status
		valid  bool
	}{
		{auth.StatusActive, true},
		{auth.StatusSuspended, true},
		{auth.Status("deleted"), false},
		{auth.Status(""), false},
	}

	for _, tt := range tests {
		if got := tt.status.IsValid(); got != tt.valid {
			t.Errorf("Status(%q).IsValid() = %v, expected %v", tt.status, got, tt.valid)
		}
	}
}

func TestPasswordHashing(t *testing.T) {
	password := "SecretP@ssw0rd!"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	if hash == password {
		t.Fatal("hash should not equal plaintext password")
	}

	if !auth.CheckPassword(hash, password) {
		t.Error("expected valid password check to return true")
	}

	if auth.CheckPassword(hash, "WrongPassword123") {
		t.Error("expected invalid password check to return false")
	}

	// Short password
	_, err = auth.HashPassword("short")
	if err == nil {
		t.Error("expected error for password shorter than 8 chars")
	}
}

func TestUserResponse_RedactsPasswordHash(t *testing.T) {
	user := auth.User{
		ID:           "test-id",
		TenantID:     "tenant-1",
		Email:        "user@example.com",
		PasswordHash: "$2a$12$abcdefghijklmnopqrstuv",
		Role:         auth.RoleAdmin,
		Status:       auth.StatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("failed to marshal user: %v", err)
	}

	if strings.Contains(string(data), "password_hash") {
		t.Errorf("User JSON should omit password_hash, got: %s", string(data))
	}

	resp := user.ToResponse()
	respData, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal user response: %v", err)
	}

	if strings.Contains(string(respData), "password") {
		t.Errorf("UserResponse JSON should omit password, got: %s", string(respData))
	}
}

func TestValidateEmail(t *testing.T) {
	valid, err := auth.ValidateEmail("User@Example.COM ")
	if err != nil || valid != "user@example.com" {
		t.Errorf("expected normalized email user@example.com, got %q, err: %v", valid, err)
	}

	_, err = auth.ValidateEmail("invalid-email")
	if err == nil {
		t.Error("expected error for invalid email")
	}

	_, err = auth.ValidateEmail("")
	if err == nil {
		t.Error("expected error for empty email")
	}
}

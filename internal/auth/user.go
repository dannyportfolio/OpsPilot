package auth

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Domain errors.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserSuspended      = errors.New("account suspended")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("insufficient permissions")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrTokenExpired       = errors.New("invalid or expired token")
	ErrRateLimitExceeded  = errors.New("too many requests")
	ErrInvalidInput       = errors.New("invalid input")
)

// Role represents a user's RBAC role.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// IsValid checks if role matches supported enums.
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleMember, RoleViewer:
		return true
	default:
		return false
	}
}

// Status represents the operational state of a user account.
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
)

// IsValid checks if status matches supported enums.
func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusSuspended:
		return true
	default:
		return false
	}
}

// User represents an authenticated identity bound to a tenant.
type User struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	Status       Status    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserResponse is the safe representation of User excluding any credential data.
type UserResponse struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse returns safe DTO for API transport.
func (u User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		TenantID:  u.TenantID,
		Email:     u.Email,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// HashPassword computes a secure bcrypt hash of the given plaintext password with cost factor 12.
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters long")
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword verifies that a plaintext password matches a stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidateEmail checks email address format.
func ValidateEmail(email string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(email))
	if trimmed == "" {
		return "", errors.New("email is required")
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed {
		return "", errors.New("invalid email address format")
	}
	return trimmed, nil
}

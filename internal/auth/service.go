package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// LoginRequest contains credentials submitted during authentication.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse contains the access token and user metadata.
type LoginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	TokenType string       `json:"token_type"`
	User      UserResponse `json:"user"`
}

// Service provides high-level authentication and user lifecycle operations.
type Service interface {
	Login(ctx context.Context, req LoginRequest, clientIP string) (LoginResponse, error)
	Logout(ctx context.Context, claims *Claims) error
	CreateUser(ctx context.Context, tenantID, email, rawPassword string, role Role) (UserResponse, error)
	GetProfile(ctx context.Context, userID string) (UserResponse, error)
}

type authService struct {
	repo            UserRepository
	tokenService    TokenService
	revocationStore RevocationStore
	rateLimiter     RateLimiter
	logger          *slog.Logger
}

// NewService constructs a new AuthService instance.
func NewService(
	repo UserRepository,
	tokenSvc TokenService,
	revStore RevocationStore,
	limiter RateLimiter,
	logger *slog.Logger,
) Service {
	return &authService{
		repo:            repo,
		tokenService:    tokenSvc,
		revocationStore: revStore,
		rateLimiter:     limiter,
		logger:          logger,
	}
}

// Login authenticates a user against credentials with rate-limiting and timing attack defense.
func (s *authService) Login(ctx context.Context, req LoginRequest, clientIP string) (LoginResponse, error) {
	normEmail, err := ValidateEmail(req.Email)
	if err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	rateKey := clientIP + ":" + normEmail
	if s.rateLimiter != nil && !s.rateLimiter.Allow(rateKey) {
		if s.logger != nil {
			s.logger.Warn("login rate limit exceeded",
				"action", "auth.rate_limit_exceeded",
				"email", normEmail,
				"client_ip", clientIP,
			)
		}
		return LoginResponse{}, ErrRateLimitExceeded
	}

	user, err := s.repo.GetByEmail(ctx, normEmail)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Perform dummy hash comparison to mitigate timing attacks
			_ = CheckPassword("$2a$12$e80yq7KjB8aG05hTqYpTRe1gXN3u3.2j3f2Ld5mHnUv3W6qWbE3aK", req.Password)
			if s.rateLimiter != nil {
				s.rateLimiter.RecordFailure(rateKey)
			}
			if s.logger != nil {
				s.logger.Warn("login failed: unknown user",
					"action", "auth.login_failed",
					"email", normEmail,
					"client_ip", clientIP,
				)
			}
			return LoginResponse{}, ErrInvalidCredentials
		}
		return LoginResponse{}, fmt.Errorf("failed to retrieve user: %w", err)
	}

	if user.Status == StatusSuspended {
		if s.logger != nil {
			s.logger.Warn("login rejected: account suspended",
				"action", "auth.account_suspended",
				"user_id", user.ID,
				"tenant_id", user.TenantID,
				"email", normEmail,
			)
		}
		return LoginResponse{}, ErrUserSuspended
	}

	if !CheckPassword(user.PasswordHash, req.Password) {
		if s.rateLimiter != nil {
			s.rateLimiter.RecordFailure(rateKey)
		}
		if s.logger != nil {
			s.logger.Warn("login failed: invalid password",
				"action", "auth.login_failed",
				"user_id", user.ID,
				"tenant_id", user.TenantID,
				"email", normEmail,
				"client_ip", clientIP,
			)
		}
		return LoginResponse{}, ErrInvalidCredentials
	}

	if s.rateLimiter != nil {
		s.rateLimiter.Reset(rateKey)
	}

	tokenStr, claims, err := s.tokenService.GenerateToken(user)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("failed to issue access token: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("login successful",
			"action", "auth.login_success",
			"user_id", user.ID,
			"tenant_id", user.TenantID,
			"role", string(user.Role),
		)
	}

	return LoginResponse{
		Token:     tokenStr,
		ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC(),
		TokenType: "Bearer",
		User:      user.ToResponse(),
	}, nil
}

// Logout invalidates the token by writing its JTI to the revocation store.
func (s *authService) Logout(ctx context.Context, claims *Claims) error {
	if claims == nil || claims.JTI == "" {
		return ErrUnauthorized
	}

	exp := time.Unix(claims.ExpiresAt, 0).UTC()
	if s.revocationStore != nil {
		if err := s.revocationStore.Revoke(claims.JTI, exp); err != nil {
			return fmt.Errorf("failed to revoke token: %w", err)
		}
	}

	if s.logger != nil {
		s.logger.Info("token revoked on logout",
			"action", "auth.logout",
			"user_id", claims.Subject,
			"tenant_id", claims.TenantID,
		)
	}
	return nil
}

// CreateUser registers a new user with bcrypt-hashed credentials.
func (s *authService) CreateUser(ctx context.Context, tenantID, email, rawPassword string, role Role) (UserResponse, error) {
	normEmail, err := ValidateEmail(email)
	if err != nil {
		return UserResponse{}, err
	}

	if !role.IsValid() {
		return UserResponse{}, fmt.Errorf("%w: invalid role %s", ErrInvalidInput, role)
	}

	hash, err := HashPassword(rawPassword)
	if err != nil {
		return UserResponse{}, err
	}

	// Generate UUID v4
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return UserResponse{}, err
	}
	idBytes[6] = (idBytes[6] & 0x0f) | 0x40
	idBytes[8] = (idBytes[8] & 0x3f) | 0x80
	userID := fmt.Sprintf("%x-%x-%x-%x-%x",
		idBytes[0:4], idBytes[4:6], idBytes[6:8], idBytes[8:10], idBytes[10:16])

	u := User{
		ID:           userID,
		TenantID:     tenantID,
		Email:        normEmail,
		PasswordHash: hash,
		Role:         role,
		Status:       StatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	created, err := s.repo.Create(ctx, u)
	if err != nil {
		return UserResponse{}, err
	}

	return created.ToResponse(), nil
}

// GetProfile retrieves user profile info.
func (s *authService) GetProfile(ctx context.Context, userID string) (UserResponse, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return UserResponse{}, err
	}
	return u.ToResponse(), nil
}

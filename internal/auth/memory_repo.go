package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrUserAlreadyExists = errors.New("user already exists")

// MemoryUserRepository is a thread-safe in-memory adapter implementing UserRepository.
type MemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]User // key: user.ID
}

// NewMemoryUserRepository creates a new in-memory user repository.
func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users: make(map[string]User),
	}
}

// Create stores a new user. Rejects duplicate emails.
func (r *MemoryUserRepository) Create(ctx context.Context, u User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	normEmail := strings.ToLower(strings.TrimSpace(u.Email))
	for _, existing := range r.users {
		if strings.ToLower(existing.Email) == normEmail && existing.TenantID == u.TenantID {
			return User{}, ErrUserAlreadyExists
		}
	}

	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}
	u.UpdatedAt = u.CreatedAt

	r.users[u.ID] = u
	return u, nil
}

// GetByID retrieves a user by ID.
func (r *MemoryUserRepository) GetByID(ctx context.Context, id string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, found := r.users[id]
	if !found {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

// GetByEmail retrieves a user by their unique email.
func (r *MemoryUserRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normEmail := strings.ToLower(strings.TrimSpace(email))
	for _, u := range r.users {
		if strings.ToLower(u.Email) == normEmail {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

// ListByTenant returns all users belonging to a tenant.
func (r *MemoryUserRepository) ListByTenant(ctx context.Context, tenantID string) ([]User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []User
	for _, u := range r.users {
		if u.TenantID == tenantID {
			list = append(list, u)
		}
	}
	return list, nil
}

// Update modifies an existing user.
func (r *MemoryUserRepository) Update(ctx context.Context, u User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, found := r.users[u.ID]
	if !found {
		return User{}, ErrUserNotFound
	}

	u.CreatedAt = existing.CreatedAt
	u.UpdatedAt = time.Now().UTC()
	r.users[u.ID] = u
	return u, nil
}

// Delete removes a user by ID.
func (r *MemoryUserRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, found := r.users[id]; !found {
		return ErrUserNotFound
	}
	delete(r.users, id)
	return nil
}

package auth

import "context"

// UserRepository defines the persistence port for user records.
type UserRepository interface {
	Create(ctx context.Context, u User) (User, error)
	GetByID(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	ListByTenant(ctx context.Context, tenantID string) ([]User, error)
	Update(ctx context.Context, u User) (User, error)
	Delete(ctx context.Context, id string) error
}

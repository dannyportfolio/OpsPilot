package auth_test

import (
	"context"
	"testing"

	"opspilot/task-api/internal/auth"
)

func TestMemoryUserRepository_CRUD(t *testing.T) {
	repo := auth.NewMemoryUserRepository()
	ctx := context.Background()

	u := auth.User{
		ID:           "user-1",
		TenantID:     "tenant-1",
		Email:        "user1@example.com",
		PasswordHash: "hashed-pwd",
		Role:         auth.RoleAdmin,
		Status:       auth.StatusActive,
	}

	created, err := repo.Create(ctx, u)
	if err != nil {
		t.Fatalf("unexpected error creating user: %v", err)
	}
	if created.ID != u.ID || created.Email != u.Email {
		t.Fatalf("created user mismatch: %+v", created)
	}

	// Duplicate email in same tenant should fail
	_, err = repo.Create(ctx, u)
	if err != auth.ErrUserAlreadyExists {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}

	// Get by ID
	fetched, err := repo.GetByID(ctx, u.ID)
	if err != nil || fetched.ID != u.ID {
		t.Errorf("unexpected error fetching by ID: %v", err)
	}

	// Get by email (case-insensitive)
	fetchedEmail, err := repo.GetByEmail(ctx, "USER1@example.com")
	if err != nil || fetchedEmail.ID != u.ID {
		t.Errorf("unexpected error fetching by Email: %v", err)
	}

	// List by tenant
	list, err := repo.ListByTenant(ctx, "tenant-1")
	if err != nil || len(list) != 1 {
		t.Errorf("unexpected error listing by tenant: %v, len=%d", err, len(list))
	}

	// List for non-existent tenant
	emptyList, err := repo.ListByTenant(ctx, "tenant-other")
	if err != nil || len(emptyList) != 0 {
		t.Errorf("expected 0 users for other tenant, got %d", len(emptyList))
	}

	// Update
	u.Role = auth.RoleMember
	updated, err := repo.Update(ctx, u)
	if err != nil || updated.Role != auth.RoleMember {
		t.Errorf("unexpected error updating user: %v", err)
	}

	// Delete
	err = repo.Delete(ctx, u.ID)
	if err != nil {
		t.Errorf("unexpected error deleting user: %v", err)
	}

	// Get after delete
	_, err = repo.GetByID(ctx, u.ID)
	if err != auth.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PostgresUserRepository provides a PostgreSQL persistence adapter implementing UserRepository.
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository initializes a new PostgreSQL user repository.
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Create inserts a new user record into PostgreSQL.
func (r *PostgresUserRepository) Create(ctx context.Context, u User) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	query := `
	INSERT INTO users (id, tenant_id, email, password_hash, role, status, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id, tenant_id, email, password_hash, role, status, created_at, updated_at;
	`

	var created User
	var createdAt, updatedAt time.Time
	var roleStr, statusStr string

	err := r.db.QueryRowContext(
		ctx,
		query,
		u.ID,
		u.TenantID,
		strings.ToLower(strings.TrimSpace(u.Email)),
		u.PasswordHash,
		string(u.Role),
		string(u.Status),
		u.CreatedAt,
		u.UpdatedAt,
	).Scan(
		&created.ID,
		&created.TenantID,
		&created.Email,
		&created.PasswordHash,
		&roleStr,
		&statusStr,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return User{}, ErrUserAlreadyExists
		}
		return User{}, fmt.Errorf("failed to insert user: %w", err)
	}

	created.Role = Role(roleStr)
	created.Status = Status(statusStr)
	created.CreatedAt = createdAt.UTC()
	created.UpdatedAt = updatedAt.UTC()

	return created, nil
}

// GetByID fetches a user by their unique UUID.
func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	query := `
	SELECT id, tenant_id, email, password_hash, role, status, created_at, updated_at
	FROM users
	WHERE id = $1;
	`

	var u User
	var createdAt, updatedAt time.Time
	var roleStr, statusStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.TenantID,
		&u.Email,
		&u.PasswordHash,
		&roleStr,
		&statusStr,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("failed to query user by id: %w", err)
	}

	u.Role = Role(roleStr)
	u.Status = Status(statusStr)
	u.CreatedAt = createdAt.UTC()
	u.UpdatedAt = updatedAt.UTC()

	return u, nil
}

// GetByEmail retrieves a user by email across all tenants. If multiple tenants have a user with this email, ErrAmbiguousUser is returned.
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	normEmail := strings.ToLower(strings.TrimSpace(email))
	query := `
	SELECT id, tenant_id, email, password_hash, role, status, created_at, updated_at
	FROM users
	WHERE email = $1;
	`

	rows, err := r.db.QueryContext(ctx, query, normEmail)
	if err != nil {
		return User{}, fmt.Errorf("failed to query users by email: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var createdAt, updatedAt time.Time
		var roleStr, statusStr string

		if err := rows.Scan(
			&u.ID,
			&u.TenantID,
			&u.Email,
			&u.PasswordHash,
			&roleStr,
			&statusStr,
			&createdAt,
			&updatedAt,
		); err != nil {
			return User{}, fmt.Errorf("failed to scan user row: %w", err)
		}

		u.Role = Role(roleStr)
		u.Status = Status(statusStr)
		u.CreatedAt = createdAt.UTC()
		u.UpdatedAt = updatedAt.UTC()
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return User{}, err
	}

	if len(users) == 0 {
		return User{}, ErrUserNotFound
	}
	if len(users) > 1 {
		return User{}, ErrAmbiguousUser
	}

	return users[0], nil
}

// GetByTenantAndEmail retrieves a user by tenant identifier and email.
func (r *PostgresUserRepository) GetByTenantAndEmail(ctx context.Context, tenantID, email string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	normEmail := strings.ToLower(strings.TrimSpace(email))
	query := `
	SELECT id, tenant_id, email, password_hash, role, status, created_at, updated_at
	FROM users
	WHERE tenant_id = $1 AND email = $2;
	`

	var u User
	var createdAt, updatedAt time.Time
	var roleStr, statusStr string

	err := r.db.QueryRowContext(ctx, query, tenantID, normEmail).Scan(
		&u.ID,
		&u.TenantID,
		&u.Email,
		&u.PasswordHash,
		&roleStr,
		&statusStr,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("failed to query user by tenant and email: %w", err)
	}

	u.Role = Role(roleStr)
	u.Status = Status(statusStr)
	u.CreatedAt = createdAt.UTC()
	u.UpdatedAt = updatedAt.UTC()

	return u, nil
}

// ListByTenant returns all users belonging to a tenant.
func (r *PostgresUserRepository) ListByTenant(ctx context.Context, tenantID string) ([]User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	query := `
	SELECT id, tenant_id, email, password_hash, role, status, created_at, updated_at
	FROM users
	WHERE tenant_id = $1
	ORDER BY created_at ASC;
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by tenant: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		var createdAt, updatedAt time.Time
		var roleStr, statusStr string

		if err := rows.Scan(
			&u.ID,
			&u.TenantID,
			&u.Email,
			&u.PasswordHash,
			&roleStr,
			&statusStr,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}

		u.Role = Role(roleStr)
		u.Status = Status(statusStr)
		u.CreatedAt = createdAt.UTC()
		u.UpdatedAt = updatedAt.UTC()
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

// Update modifies an existing user record.
func (r *PostgresUserRepository) Update(ctx context.Context, u User) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}

	query := `
	UPDATE users
	SET email = $1, password_hash = $2, role = $3, status = $4, updated_at = $5
	WHERE id = $6 AND tenant_id = $7
	RETURNING id, tenant_id, email, password_hash, role, status, created_at, updated_at;
	`

	var updated User
	var createdAt, updatedAt time.Time
	var roleStr, statusStr string

	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.ToLower(strings.TrimSpace(u.Email)),
		u.PasswordHash,
		string(u.Role),
		string(u.Status),
		time.Now().UTC(),
		u.ID,
		u.TenantID,
	).Scan(
		&updated.ID,
		&updated.TenantID,
		&updated.Email,
		&updated.PasswordHash,
		&roleStr,
		&statusStr,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("failed to update user: %w", err)
	}

	updated.Role = Role(roleStr)
	updated.Status = Status(statusStr)
	updated.CreatedAt = createdAt.UTC()
	updated.UpdatedAt = updatedAt.UTC()

	return updated, nil
}

// Delete removes a user record by ID.
func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	query := `DELETE FROM users WHERE id = $1;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

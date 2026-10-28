package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// EnsureSchemaMigrationsTable creates the migration tracking table if it does not already exist.
func EnsureSchemaMigrationsTable(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is required")
	}
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(64) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`
	_, err := db.ExecContext(ctx, query)
	return err
}

// MigrateUp applies all pending up migrations found in the given migrations directory in lexical order.
func MigrateUp(ctx context.Context, db *sql.DB, migrationsDir string) error {
	if err := EnsureSchemaMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %q: %w", migrationsDir, err)
	}

	var upFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			upFiles = append(upFiles, entry.Name())
		}
	}
	sort.Strings(upFiles)

	for _, file := range upFiles {
		version := strings.TrimSuffix(file, ".up.sql")

		// Check if already applied
		var exists bool
		err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", version, err)
		}
		if exists {
			continue
		}

		// Read and execute inside a transaction
		content, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin migration transaction for %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)", version, time.Now().UTC()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration transaction for %s: %w", version, err)
		}
	}

	return nil
}

// MigrateDown rolls back the most recent migration found in the migrations directory.
func MigrateDown(ctx context.Context, db *sql.DB, migrationsDir string) error {
	if err := EnsureSchemaMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	var latestVersion string
	err := db.QueryRowContext(ctx, "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&latestVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // No migrations to roll back
		}
		return fmt.Errorf("failed to query latest migration: %w", err)
	}

	downFile := latestVersion + ".down.sql"
	downPath := filepath.Join(migrationsDir, downFile)

	content, err := os.ReadFile(downPath)
	if err != nil {
		return fmt.Errorf("failed to read down migration %s: %w", downFile, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin rollback transaction: %w", err)
	}

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to execute down migration %s: %w", downFile, err)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM schema_migrations WHERE version = $1", latestVersion); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to remove migration record %s: %w", latestVersion, err)
	}

	return tx.Commit()
}

package database_test

import (
	"context"
	"os"
	"testing"
	"time"

	"opspilot/task-api/internal/platform/database"
)

func getTestDatabaseURL() string {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/opspilot?sslmode=disable"
	}
	return url
}

func TestConnect_Validation(t *testing.T) {
	ctx := context.Background()
	_, err := database.Connect(ctx, "")
	if err == nil {
		t.Fatal("expected error when databaseURL is empty")
	}
}

func TestMigrate_Validation(t *testing.T) {
	ctx := context.Background()

	// 1. Nil DB on EnsureSchemaMigrationsTable
	err := database.EnsureSchemaMigrationsTable(ctx, nil)
	if err == nil {
		t.Fatal("expected error when db is nil on EnsureSchemaMigrationsTable")
	}

	// 2. Nil DB on MigrateUp
	err = database.MigrateUp(ctx, nil, "migrations")
	if err == nil {
		t.Fatal("expected error when db is nil on MigrateUp")
	}

	// 3. Nil DB on MigrateDown
	err = database.MigrateDown(ctx, nil, "migrations")
	if err == nil {
		t.Fatal("expected error when db is nil on MigrateDown")
	}
}

func TestPostgres_ConnectAndMigrate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbURL := getTestDatabaseURL()
	db, err := database.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("skipping PostgreSQL integration test (database unreachable: %v)", err)
		return
	}
	defer db.Close()

	// Locate migrations directory (relative to internal/platform/database)
	migrationsDir := "../../../migrations"
	if _, err := os.Stat(migrationsDir); err != nil {
		migrationsDir = "migrations"
	}

	// 1. Run migrations up
	if err := database.MigrateUp(ctx, db, migrationsDir); err != nil {
		t.Fatalf("MigrateUp failed: %v", err)
	}

	// 2. Verify tasks table exists
	var tasksTableExists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'tasks')").Scan(&tasksTableExists)
	if err != nil {
		t.Fatalf("failed to query information_schema for tasks: %v", err)
	}
	if !tasksTableExists {
		t.Fatal("expected 'tasks' table to exist after MigrateUp")
	}

	// 3. Verify users table exists
	var usersTableExists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&usersTableExists)
	if err != nil {
		t.Fatalf("failed to query information_schema for users: %v", err)
	}
	if !usersTableExists {
		t.Fatal("expected 'users' table to exist after MigrateUp")
	}

	// 4. Verify schema_migrations records exist
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version IN ('000001_create_tasks_table', '000002_create_users_and_tenant_support')").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query schema_migrations: %v", err)
	}
	if count < 2 {
		t.Errorf("expected at least 2 migration records, got %d", count)
	}
}

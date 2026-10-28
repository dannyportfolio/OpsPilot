package config_test

import (
	"os"
	"testing"
	"time"

	"opspilot/task-api/internal/platform/config"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear env variables
	os.Unsetenv("PORT")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("STORAGE_DRIVER")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("AUTH_JWT_SECRET")
	os.Unsetenv("AUTH_TOKEN_TTL_SECONDS")

	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default Port '8080', got %q", cfg.Port)
	}
	if cfg.DatabaseURL == "" {
		t.Error("expected non-empty default DatabaseURL")
	}
	if cfg.StorageDriver != "memory" {
		t.Errorf("expected default StorageDriver 'memory', got %q", cfg.StorageDriver)
	}
	if cfg.LogLevel != "INFO" {
		t.Errorf("expected default LogLevel 'INFO', got %q", cfg.LogLevel)
	}
	if cfg.JWTSecret == "" {
		t.Error("expected default non-empty JWTSecret")
	}
	if cfg.TokenTTL != 1*time.Hour {
		t.Errorf("expected default TokenTTL 1h, got %v", cfg.TokenTTL)
	}
}

func TestLoad_CustomEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://user:pass@remote:5432/db")
	t.Setenv("STORAGE_DRIVER", "postgres")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("AUTH_JWT_SECRET", "custom-super-secure-production-secret-key")
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "7200")

	cfg := config.Load()

	if cfg.Port != "9090" {
		t.Errorf("expected Port '9090', got %q", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://user:pass@remote:5432/db" {
		t.Errorf("expected custom DatabaseURL, got %q", cfg.DatabaseURL)
	}
	if cfg.StorageDriver != "postgres" {
		t.Errorf("expected StorageDriver 'postgres', got %q", cfg.StorageDriver)
	}
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("expected LogLevel 'DEBUG', got %q", cfg.LogLevel)
	}
	if cfg.JWTSecret != "custom-super-secure-production-secret-key" {
		t.Errorf("expected custom JWTSecret, got %q", cfg.JWTSecret)
	}
	if cfg.TokenTTL != 2*time.Hour {
		t.Errorf("expected custom TokenTTL 2h, got %v", cfg.TokenTTL)
	}
}

func TestLoad_InvalidAndEdgeEnv(t *testing.T) {
	// Non-numeric TTL string should fall back to default 1 hour
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "not-a-number")
	cfg := config.Load()
	if cfg.TokenTTL != 1*time.Hour {
		t.Errorf("expected default 1h on non-numeric TTL, got %v", cfg.TokenTTL)
	}

	// Negative TTL should fall back to default 1 hour
	t.Setenv("AUTH_TOKEN_TTL_SECONDS", "-50")
	cfg = config.Load()
	if cfg.TokenTTL != 1*time.Hour {
		t.Errorf("expected default 1h on negative TTL, got %v", cfg.TokenTTL)
	}

	// Whitespace-only values should be trimmed to defaults
	t.Setenv("PORT", "   ")
	t.Setenv("STORAGE_DRIVER", "   ")
	t.Setenv("LOG_LEVEL", "   ")
	t.Setenv("AUTH_JWT_SECRET", "   ")
	cfg = config.Load()
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080 for whitespace port, got %q", cfg.Port)
	}
	if cfg.StorageDriver != "memory" {
		t.Errorf("expected default storage driver memory, got %q", cfg.StorageDriver)
	}
	if cfg.LogLevel != "INFO" {
		t.Errorf("expected default log level INFO, got %q", cfg.LogLevel)
	}
	if cfg.JWTSecret == "" {
		t.Error("expected fallback default secret for whitespace JWTSecret")
	}
}

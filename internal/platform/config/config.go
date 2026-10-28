package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds environment-driven operational settings with safe defaults.
type Config struct {
	Port          string
	DatabaseURL   string
	StorageDriver string
	LogLevel      string
	JWTSecret     string
	TokenTTL      time.Duration
}

// Load reads configuration from environment variables, applying safe defaults for local development.
func Load() *Config {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/opspilot?sslmode=disable"
	}

	driver := strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_DRIVER")))
	if driver == "" {
		driver = "memory"
	}

	level := strings.ToUpper(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if level == "" {
		level = "INFO"
	}

	jwtSecret := strings.TrimSpace(os.Getenv("AUTH_JWT_SECRET"))
	if jwtSecret == "" {
		jwtSecret = "dev-secret-do-not-use-in-production-32-bytes-minimum"
	}

	ttlSecondsStr := strings.TrimSpace(os.Getenv("AUTH_TOKEN_TTL_SECONDS"))
	ttlSeconds, err := strconv.Atoi(ttlSecondsStr)
	if err != nil || ttlSeconds <= 0 {
		ttlSeconds = 3600 // Default 1 hour
	}

	return &Config{
		Port:          port,
		DatabaseURL:   dbURL,
		StorageDriver: driver,
		LogLevel:      level,
		JWTSecret:     jwtSecret,
		TokenTTL:      time.Duration(ttlSeconds) * time.Second,
	}
}

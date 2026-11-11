package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"opspilot/task-api/internal/auth"
	"opspilot/task-api/internal/platform/config"
	"opspilot/task-api/internal/platform/database"
	"opspilot/task-api/internal/platform/logger"
	"opspilot/task-api/internal/task"
)

// App encapsulates all initialized domain services, persistence layers, and root HTTP handler.
type App struct {
	Handler http.Handler
	DB      *sql.DB
	Config  *config.Config
	Logger  *slog.Logger
}

// BuildOptions customizes application composition for different runtimes.
type BuildOptions struct {
	IsServerless bool
	MaxOpenConns int
	MaxIdleConns int
}

// Build initializes and wires all application dependencies, returning the unified App container.
func Build(ctx context.Context, cfg *config.Config, opts BuildOptions) (*App, error) {
	if cfg == nil {
		cfg = config.Load()
	}

	log := logger.New(cfg.LogLevel)

	var db *sql.DB
	var taskRepo task.Repository
	var userRepo auth.UserRepository

	if cfg.StorageDriver == "postgres" {
		connCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var err error
		db, err = database.Connect(connCtx, cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		if opts.IsServerless {
			maxOpen := opts.MaxOpenConns
			if maxOpen <= 0 {
				maxOpen = 2
			}
			maxIdle := opts.MaxIdleConns
			if maxIdle <= 0 {
				maxIdle = 2
			}
			db.SetMaxOpenConns(maxOpen)
			db.SetMaxIdleConns(maxIdle)
			db.SetConnMaxLifetime(5 * time.Minute)
			db.SetConnMaxIdleTime(2 * time.Minute)
		} else {
			migrationsDir := "migrations"
			if err := database.MigrateUp(connCtx, db, migrationsDir); err != nil {
				return nil, fmt.Errorf("failed to run database migrations: %w", err)
			}
		}

		taskRepo = task.NewPostgresRepository(db)
		userRepo = auth.NewPostgresUserRepository(db)
	} else {
		taskRepo = task.NewMemoryRepository()
		userRepo = auth.NewMemoryUserRepository()
	}

	tokenSvc, err := auth.NewTokenService(cfg.JWTSecret, "opspilot", "opspilot-api", cfg.TokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token service: %w", err)
	}

	revStore := auth.NewMemoryRevocationStore()
	rateLimiter := auth.NewMemoryRateLimiter(5, 1*time.Minute, 1*time.Minute)
	authSvc := auth.NewService(userRepo, tokenSvc, revStore, rateLimiter, log)

	// Seed dev user if explicitly configured or local non-production memory
	if os.Getenv("SEED_DEV_DATA") == "true" || (cfg.StorageDriver == "memory" && cfg.AppEnv != "production") {
		_, _ = authSvc.CreateUser(ctx, "default-tenant", "admin@opspilot.internal", "AdminSecure123!", auth.RoleAdmin)
	}

	taskSvc := task.NewService(taskRepo)

	authRouter := auth.NewRouter(authSvc, tokenSvc, revStore, log)
	taskRouter := task.NewSecuredRouter(taskSvc, tokenSvc, revStore, log)

	rootMux := http.NewServeMux()
	rootMux.Handle("/auth/", authRouter)
	rootMux.Handle("/", taskRouter)

	return &App{
		Handler: rootMux,
		DB:      db,
		Config:  cfg,
		Logger:  log,
	}, nil
}

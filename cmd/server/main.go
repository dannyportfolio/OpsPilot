package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opspilot/task-api/internal/auth"
	"opspilot/task-api/internal/platform/config"
	"opspilot/task-api/internal/platform/database"
	"opspilot/task-api/internal/platform/logger"
	"opspilot/task-api/internal/task"
)

func main() {
	// 1. Load environment configuration with safe defaults
	cfg := config.Load()

	// 2. Initialize structured JSON logger
	log := logger.New(cfg.LogLevel)
	log.Info("initializing Task REST API with Authentication & RBAC",
		"port", cfg.Port,
		"storage_driver", cfg.StorageDriver,
		"log_level", cfg.LogLevel,
		"token_ttl", cfg.TokenTTL.String(),
	)

	// 3. Initialize repository adapter based on configuration
	var repo task.Repository
	var db *sql.DB

	if cfg.StorageDriver == "postgres" {
		initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var err error
		db, err = database.Connect(initCtx, cfg.DatabaseURL)
		if err != nil {
			log.Error("failed to connect to PostgreSQL", "error", err)
			os.Exit(1)
		}

		// Apply database migrations
		migrationsDir := "migrations"
		if err := database.MigrateUp(initCtx, db, migrationsDir); err != nil {
			log.Error("failed to run database migrations", "error", err)
			os.Exit(1)
		}

		repo = task.NewPostgresRepository(db)
		log.Info("using PostgreSQL persistence adapter", "migrations_applied", true)
	} else {
		repo = task.NewMemoryRepository()
		log.Info("using in-memory storage adapter (volatile; resets on restart)")
	}

	// 4. Initialize Auth Domain components
	tokenSvc, err := auth.NewTokenService(cfg.JWTSecret, "opspilot", "opspilot-api", cfg.TokenTTL)
	if err != nil {
		log.Error("failed to initialize token service", "error", err)
		os.Exit(1)
	}

	revStore := auth.NewMemoryRevocationStore()
	rateLimiter := auth.NewMemoryRateLimiter(5, 1*time.Minute, 1*time.Minute)
	userRepo := auth.NewMemoryUserRepository()
	authSvc := auth.NewService(userRepo, tokenSvc, revStore, rateLimiter, log)

	// Seed default development administrator
	seedCtx := context.Background()
	_, err = authSvc.CreateUser(seedCtx, "default-tenant", "admin@opspilot.internal", "AdminSecure123!", auth.RoleAdmin)
	if err == nil {
		log.Info("seeded initial admin credentials for development",
			"tenant_id", "default-tenant",
			"email", "admin@opspilot.internal",
		)
	}

	// 5. Initialize application service
	svc := task.NewService(repo)

	// 6. Build decoupled routers
	authRouter := auth.NewRouter(authSvc, tokenSvc, revStore, log)
	taskRouter := task.NewSecuredRouter(svc, tokenSvc, revStore, log)

	rootMux := http.NewServeMux()
	rootMux.Handle("/auth/", authRouter)
	rootMux.Handle("/", taskRouter)

	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      rootMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 7. Graceful shutdown handler
	shutdownErr := make(chan error, 1)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		sig := <-sigChan
		log.Info("shutdown signal received", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if db != nil {
			log.Info("closing database connection pool...")
			_ = db.Close()
		}

		shutdownErr <- srv.Shutdown(ctx)
	}()

	log.Info("server listening", "address", fmt.Sprintf("http://localhost%s", addr))

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server startup failed", "error", err)
		os.Exit(1)
	}

	if err := <-shutdownErr; err != nil {
		log.Error("graceful shutdown failed", "error", err)
	} else {
		log.Info("server stopped cleanly")
	}
}

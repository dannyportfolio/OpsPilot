package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	awslambda "github.com/aws/aws-lambda-go/lambda"
	"opspilot/task-api/internal/auth"
	"opspilot/task-api/internal/platform/config"
	"opspilot/task-api/internal/platform/database"
	"opspilot/task-api/internal/platform/lambda"
	"opspilot/task-api/internal/platform/logger"
	"opspilot/task-api/internal/task"
)

// buildApp initializes all domain components, repositories, and routes with serverless connection tuning.
func buildApp() (http.Handler, error) {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	log.Info("initializing OpsPilot AWS Lambda execution environment",
		"storage_driver", cfg.StorageDriver,
		"log_level", cfg.LogLevel,
		"token_ttl", cfg.TokenTTL.String(),
	)

	var repo task.Repository
	var db *sql.DB

	if cfg.StorageDriver == "postgres" {
		initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var err error
		db, err = database.Connect(initCtx, cfg.DatabaseURL)
		if err != nil {
			log.Error("failed to connect to PostgreSQL in serverless environment", "error", err)
			return nil, fmt.Errorf("failed to connect to postgres: %w", err)
		}

		// Serverless-tuned connection pool limits to prevent connection exhaustion across concurrent Lambdas
		db.SetMaxOpenConns(2)
		db.SetMaxIdleConns(2)
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetConnMaxIdleTime(2 * time.Minute)

		repo = task.NewPostgresRepository(db)
	} else {
		repo = task.NewMemoryRepository()
	}

	tokenSvc, err := auth.NewTokenService(cfg.JWTSecret, "opspilot", "opspilot-api", cfg.TokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token service: %w", err)
	}

	revStore := auth.NewMemoryRevocationStore()
	rateLimiter := auth.NewMemoryRateLimiter(5, 1*time.Minute, 1*time.Minute)
	userRepo := auth.NewMemoryUserRepository()
	authSvc := auth.NewService(userRepo, tokenSvc, revStore, rateLimiter, log)

	_, _ = authSvc.CreateUser(context.Background(), "default-tenant", "admin@opspilot.internal", "AdminSecure123!", auth.RoleAdmin)

	svc := task.NewService(repo)

	authRouter := auth.NewRouter(authSvc, tokenSvc, revStore, log)
	taskRouter := task.NewSecuredRouter(svc, tokenSvc, revStore, log)

	rootMux := http.NewServeMux()
	rootMux.Handle("/auth/", authRouter)
	rootMux.Handle("/", taskRouter)

	return rootMux, nil
}

func main() {
	handler, err := buildApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lambda cold-start initialization failed: %v\n", err)
		os.Exit(1)
	}

	adapter := lambda.New(handler)
	awslambda.Start(adapter.ProxyWithContext)
}

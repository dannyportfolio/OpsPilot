package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opspilot/task-api/internal/platform/app"
	"opspilot/task-api/internal/platform/config"
)

func main() {
	cfg := config.Load()

	initCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	application, err := app.Build(initCtx, cfg, app.BuildOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize application: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if application.DB != nil {
			_ = application.DB.Close()
		}
	}()

	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      application.Handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	application.Logger.Info("server starting", "addr", addr, "env", cfg.AppEnv)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			application.Logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	application.Logger.Info("initiating graceful shutdown")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		application.Logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	application.Logger.Info("server stopped safely")
}

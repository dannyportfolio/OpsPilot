package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	awslambda "github.com/aws/aws-lambda-go/lambda"
	"opspilot/task-api/internal/platform/app"
	"opspilot/task-api/internal/platform/config"
	"opspilot/task-api/internal/platform/lambda"
)

// buildApp initializes all domain components and routes via the unified composition root with serverless tuning.
func buildApp() (http.Handler, error) {
	cfg := config.Load()
	application, err := app.Build(context.Background(), cfg, app.BuildOptions{
		IsServerless: true,
		MaxOpenConns: 2,
		MaxIdleConns: 2,
	})
	if err != nil {
		return nil, err
	}
	return application.Handler, nil
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

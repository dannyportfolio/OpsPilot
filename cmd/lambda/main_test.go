package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"opspilot/task-api/internal/platform/lambda"
)

func TestBuildApp_ServerlessExecution(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "memory")
	t.Setenv("AUTH_JWT_SECRET", "test-secret-at-least-32-bytes-long")

	handler, err := buildApp()
	if err != nil {
		t.Fatalf("buildApp() failed: %v", err)
	}

	adapter := lambda.New(handler)

	// 1. Health check via Lambda event
	reqHealth := events.APIGatewayV2HTTPRequest{
		RawPath: "/health",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
				Path:   "/health",
			},
		},
	}

	respHealth, err := adapter.ProxyWithContext(context.Background(), reqHealth)
	if err != nil {
		t.Fatalf("health check proxy failed: %v", err)
	}

	if respHealth.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /health, got %d", respHealth.StatusCode)
	}

	// 2. Unauthenticated tasks request returns 401
	reqTasks := events.APIGatewayV2HTTPRequest{
		RawPath: "/tasks",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
				Path:   "/tasks",
			},
		},
	}

	respTasks, err := adapter.ProxyWithContext(context.Background(), reqTasks)
	if err != nil {
		t.Fatalf("tasks proxy failed: %v", err)
	}

	if respTasks.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for /tasks, got %d", respTasks.StatusCode)
	}
}

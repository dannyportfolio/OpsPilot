package lambda_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"opspilot/task-api/internal/auth"
	"opspilot/task-api/internal/platform/lambda"
	"opspilot/task-api/internal/task"
)

func setupParityApp() (http.Handler, *lambda.Adapter, string) {
	tokenSvc, _ := auth.NewTokenService("parity-secret-at-least-32-bytes-long", "opspilot", "opspilot-api", 1*time.Hour)
	revStore := auth.NewMemoryRevocationStore()
	limiter := auth.NewMemoryRateLimiter(5, 1*time.Minute, 1*time.Minute)
	userRepo := auth.NewMemoryUserRepository()
	authSvc := auth.NewService(userRepo, tokenSvc, revStore, limiter, nil)

	_, _ = authSvc.CreateUser(context.Background(), "tenant-parity", "admin@example.com", "Password123!", auth.RoleAdmin)
	loginResp, _ := authSvc.Login(context.Background(), auth.LoginRequest{
		Email:    "admin@example.com",
		Password: "Password123!",
	}, "127.0.0.1")

	taskRepo := task.NewMemoryRepository()
	taskSvc := task.NewService(taskRepo)

	authRouter := auth.NewRouter(authSvc, tokenSvc, revStore, nil)
	taskRouter := task.NewSecuredRouter(taskSvc, tokenSvc, revStore, nil)

	rootMux := http.NewServeMux()
	rootMux.Handle("/auth/", authRouter)
	rootMux.Handle("/", taskRouter)

	adapter := lambda.New(rootMux)
	return rootMux, adapter, loginResp.Token
}

func TestParity_HealthCheck(t *testing.T) {
	httpHandler, adapter, _ := setupParityApp()

	// 1. HTTP Server
	httpReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	httpRec := httptest.NewRecorder()
	httpHandler.ServeHTTP(httpRec, httpReq)

	// 2. Lambda Adapter
	lambdaReq := events.APIGatewayV2HTTPRequest{
		RawPath: "/health",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
				Path:   "/health",
			},
		},
	}
	lambdaResp, err := adapter.ProxyWithContext(context.Background(), lambdaReq)
	if err != nil {
		t.Fatalf("Lambda proxy failed: %v", err)
	}

	// Compare Parity
	if httpRec.Code != lambdaResp.StatusCode {
		t.Errorf("status parity mismatch: HTTP=%d, Lambda=%d", httpRec.Code, lambdaResp.StatusCode)
	}
	if httpRec.Body.String() != lambdaResp.Body {
		t.Errorf("body parity mismatch: HTTP=%q, Lambda=%q", httpRec.Body.String(), lambdaResp.Body)
	}
}

func TestParity_AuthenticationError(t *testing.T) {
	httpHandler, adapter, _ := setupParityApp()

	// 1. HTTP Server invalid credentials
	badBody := `{"email":"admin@example.com","password":"WrongPassword!"}`
	httpReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(badBody))
	httpRec := httptest.NewRecorder()
	httpHandler.ServeHTTP(httpRec, httpReq)

	// 2. Lambda Adapter invalid credentials
	lambdaReq := events.APIGatewayV2HTTPRequest{
		RawPath: "/auth/login",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "POST",
				Path:   "/auth/login",
			},
		},
		Body: badBody,
	}
	lambdaResp, err := adapter.ProxyWithContext(context.Background(), lambdaReq)
	if err != nil {
		t.Fatalf("Lambda proxy failed: %v", err)
	}

	// Compare Parity
	if httpRec.Code != lambdaResp.StatusCode {
		t.Errorf("status code mismatch: HTTP=%d, Lambda=%d", httpRec.Code, lambdaResp.StatusCode)
	}
	if httpRec.Body.String() != lambdaResp.Body {
		t.Errorf("error envelope mismatch: HTTP=%s, Lambda=%s", httpRec.Body.String(), lambdaResp.Body)
	}
}

func TestParity_TaskCreationAndRetrieval(t *testing.T) {
	httpHandler, adapter, token := setupParityApp()

	createBody := `{"title":"Serverless Parity Task"}`

	// 1. Create task via Lambda Adapter
	lambdaReq := events.APIGatewayV2HTTPRequest{
		RawPath: "/tasks",
		Headers: map[string]string{
			"authorization": "Bearer " + token,
			"content-type":  "application/json",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "POST",
				Path:   "/tasks",
			},
		},
		Body: createBody,
	}
	lambdaResp, err := adapter.ProxyWithContext(context.Background(), lambdaReq)
	if err != nil {
		t.Fatalf("Lambda create task failed: %v", err)
	}

	if lambdaResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body: %s", lambdaResp.StatusCode, lambdaResp.Body)
	}

	var createdTask task.Task
	if err := json.Unmarshal([]byte(lambdaResp.Body), &createdTask); err != nil {
		t.Fatalf("failed to decode created task: %v", err)
	}

	// 2. Retrieve the task via HTTP server
	httpReq := httptest.NewRequest(http.MethodGet, "/tasks/"+createdTask.ID, nil)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpRec := httptest.NewRecorder()
	httpHandler.ServeHTTP(httpRec, httpReq)

	if httpRec.Code != http.StatusOK {
		t.Fatalf("HTTP get task failed: %d", httpRec.Code)
	}

	var fetchedTask task.Task
	_ = json.Unmarshal(httpRec.Body.Bytes(), &fetchedTask)

	if fetchedTask.ID != createdTask.ID || fetchedTask.Title != createdTask.Title {
		t.Errorf("task entity parity mismatch: HTTP=%+v, Lambda=%+v", fetchedTask, createdTask)
	}
}

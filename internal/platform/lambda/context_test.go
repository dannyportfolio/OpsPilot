package lambda_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"opspilot/task-api/internal/platform/lambda"
)

func TestAdapter_ContextDeadlineAndCancellation(t *testing.T) {
	deadlineReached := make(chan struct{})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /timeout", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			close(deadlineReached)
			w.WriteHeader(http.StatusGatewayTimeout)
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	})

	adapter := lambda.New(mux)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	req := events.APIGatewayV2HTTPRequest{
		RawPath: "/timeout",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
				Path:   "/timeout",
			},
		},
	}

	resp, err := adapter.ProxyWithContext(ctx, req)
	if err != nil {
		t.Fatalf("unexpected proxy error: %v", err)
	}

	select {
	case <-deadlineReached:
		// Context cancellation was successfully propagated to the handler
	case <-time.After(1 * time.Second):
		t.Fatal("expected handler context to be cancelled by Lambda context timeout")
	}

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("expected 504 Gateway Timeout, got %d", resp.StatusCode)
	}
}

func TestAdapter_TraceIDAndClientIPPropagation(t *testing.T) {
	var capturedTraceID string
	var capturedRemoteAddr string

	mux := http.NewServeMux()
	mux.HandleFunc("GET /trace", func(w http.ResponseWriter, r *http.Request) {
		capturedTraceID = r.Header.Get("X-Amzn-Trace-Id")
		capturedRemoteAddr = r.RemoteAddr
		w.WriteHeader(http.StatusOK)
	})

	adapter := lambda.New(mux)

	req := events.APIGatewayV2HTTPRequest{
		RawPath: "/trace",
		Headers: map[string]string{
			"x-amzn-trace-id": "Root=1-5759e988-bd862e3fe1be46a994272793",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method:   "GET",
				Path:     "/trace",
				SourceIP: "203.0.113.195",
			},
		},
	}

	resp, err := adapter.ProxyWithContext(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	if capturedTraceID != "Root=1-5759e988-bd862e3fe1be46a994272793" {
		t.Errorf("expected trace ID propagated, got %q", capturedTraceID)
	}

	if capturedRemoteAddr != "203.0.113.195" {
		t.Errorf("expected RemoteAddr 203.0.113.195, got %q", capturedRemoteAddr)
	}
}

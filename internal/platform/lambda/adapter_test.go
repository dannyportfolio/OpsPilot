package lambda_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"opspilot/task-api/internal/platform/lambda"
)

func TestAdapter_GET(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	adapter := lambda.New(mux)

	req := events.APIGatewayV2HTTPRequest{
		RawPath: "/health",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
				Path:   "/health",
			},
		},
	}

	resp, err := adapter.ProxyWithContext(context.Background(), req)
	if err != nil {
		t.Fatalf("ProxyWithContext failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if resp.Body != `{"status":"ok"}` {
		t.Errorf("expected body %q, got %q", `{"status":"ok"}`, resp.Body)
	}

	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", resp.Headers["Content-Type"])
	}
}

func TestAdapter_POST_WithJSONAndHeaders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /items", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		bodyBytes, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(bodyBytes, &payload)

		w.Header().Set("Location", "/items/123")
		w.Header().Set("Set-Cookie", "session=xyz; Path=/")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"123","name":"` + payload["name"] + `"}`))
	})

	adapter := lambda.New(mux)

	// 1. Authorized POST
	req := events.APIGatewayV2HTTPRequest{
		RawPath: "/items",
		Headers: map[string]string{
			"authorization": "Bearer test-token",
			"content-type":  "application/json",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "POST",
				Path:   "/items",
			},
		},
		Body: `{"name":"Widget"}`,
	}

	resp, err := adapter.ProxyWithContext(context.Background(), req)
	if err != nil {
		t.Fatalf("ProxyWithContext failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body: %s", resp.StatusCode, resp.Body)
	}

	if resp.Headers["Location"] != "/items/123" {
		t.Errorf("expected Location /items/123, got %q", resp.Headers["Location"])
	}

	if len(resp.Cookies) != 1 || resp.Cookies[0] != "session=xyz; Path=/" {
		t.Errorf("expected Set-Cookie in Cookies, got %+v", resp.Cookies)
	}

	// 2. Base64 Encoded Body
	b64Body := base64.StdEncoding.EncodeToString([]byte(`{"name":"Base64Widget"}`))
	reqB64 := events.APIGatewayV2HTTPRequest{
		RawPath:         "/items",
		IsBase64Encoded: true,
		Body:            b64Body,
		Headers: map[string]string{
			"authorization": "Bearer test-token",
			"content-type":  "application/json",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "POST",
				Path:   "/items",
			},
		},
	}

	respB64, err := adapter.ProxyWithContext(context.Background(), reqB64)
	if err != nil {
		t.Fatalf("ProxyWithContext with base64 failed: %v", err)
	}
	if respB64.StatusCode != http.StatusCreated {
		t.Errorf("expected 201 for base64 body, got %d", respB64.StatusCode)
	}
}

func TestAdapter_QueryParameters(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(q))
	})

	adapter := lambda.New(mux)

	req := events.APIGatewayV2HTTPRequest{
		RawPath: "/search",
		QueryStringParameters: map[string]string{
			"q": "golang serverless",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
				Path:   "/search",
			},
		},
	}

	resp, err := adapter.ProxyWithContext(context.Background(), req)
	if err != nil {
		t.Fatalf("ProxyWithContext failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Body != "golang serverless" {
		t.Errorf("expected query param 'golang serverless', got %q", resp.Body)
	}
}

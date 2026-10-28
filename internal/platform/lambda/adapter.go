package lambda

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/aws/aws-lambda-go/events"
)

// Adapter bridges AWS API Gateway HTTP API v2 events to standard http.Handler.
type Adapter struct {
	handler http.Handler
}

// New constructs a new serverless HTTP adapter wrapping an http.Handler.
func New(handler http.Handler) *Adapter {
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	return &Adapter{handler: handler}
}

// ProxyWithContext translates the incoming API Gateway v2 HTTP event, executes the wrapped http.Handler,
// and returns an API Gateway v2 response structure.
func (a *Adapter) ProxyWithContext(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	httpReq, err := a.toHTTPRequest(ctx, req)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"error":"failed to parse request"}`,
		}, nil
	}

	rec := newResponseRecorder()
	a.handler.ServeHTTP(rec, httpReq)

	return a.toGatewayResponse(rec)
}

func (a *Adapter) toHTTPRequest(ctx context.Context, req events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	// 1. Decode body
	var bodyReader io.Reader
	if req.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 body: %w", err)
		}
		bodyReader = bytes.NewReader(decoded)
	} else {
		bodyReader = strings.NewReader(req.Body)
	}

	// 2. Determine method and path
	method := req.RequestContext.HTTP.Method
	if method == "" {
		method = http.MethodGet
	}

	path := req.RawPath
	if path == "" {
		path = req.RequestContext.HTTP.Path
	}
	if path == "" {
		path = "/"
	}

	// 3. Assemble query string
	rawQuery := req.RawQueryString
	if rawQuery == "" && len(req.QueryStringParameters) > 0 {
		values := url.Values{}
		for k, v := range req.QueryStringParameters {
			values.Set(k, v)
		}
		rawQuery = values.Encode()
	}

	targetURL := path
	if rawQuery != "" {
		targetURL = path + "?" + rawQuery
	}

	u, err := url.ParseRequestURI(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid request uri: %w", err)
	}

	// 4. Construct request with Lambda context
	httpReq, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	// 5. Populate headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// 6. Populate cookies
	if len(req.Cookies) > 0 {
		httpReq.Header.Set("Cookie", strings.Join(req.Cookies, "; "))
	}

	// 7. Client IP and host
	if req.RequestContext.HTTP.SourceIP != "" {
		httpReq.RemoteAddr = req.RequestContext.HTTP.SourceIP
	}

	return httpReq, nil
}

func (a *Adapter) toGatewayResponse(rec *responseRecorder) (events.APIGatewayV2HTTPResponse, error) {
	headers := make(map[string]string)
	var cookies []string

	for k, v := range rec.headers {
		if strings.EqualFold(k, "Set-Cookie") {
			cookies = append(cookies, v...)
			continue
		}
		if len(v) > 0 {
			headers[k] = strings.Join(v, ", ")
		}
	}

	bodyBytes := rec.body.Bytes()
	isBase64 := !utf8.Valid(bodyBytes)
	var bodyStr string

	if isBase64 {
		bodyStr = base64.StdEncoding.EncodeToString(bodyBytes)
	} else {
		bodyStr = string(bodyBytes)
	}

	statusCode := rec.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode:      statusCode,
		Headers:         headers,
		Cookies:         cookies,
		Body:            bodyStr,
		IsBase64Encoded: isBase64,
	}, nil
}

type responseRecorder struct {
	headers    http.Header
	body       *bytes.Buffer
	statusCode int
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{
		headers: make(http.Header),
		body:    new(bytes.Buffer),
	}
}

func (r *responseRecorder) Header() http.Header {
	return r.headers
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

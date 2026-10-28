package task

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"opspilot/task-api/internal/auth"
	"opspilot/task-api/internal/platform/response"
)

// NewRouter registers all task management routes without authentication (for local dev/unsecured test suites).
func NewRouter(svc Service) http.Handler {
	return NewRouterWithLogger(svc, nil)
}

// NewRouterWithRepo initializes a router directly from a repository.
func NewRouterWithRepo(repo Repository) http.Handler {
	return NewRouter(NewService(repo))
}

// NewRouterWithLogger registers task routes with custom structured logger middleware.
func NewRouterWithLogger(svc Service, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	handler := NewHandler(svc)

	// Health check endpoint
	mux.HandleFunc("GET /health", handler.Health)

	// REST endpoints for tasks
	mux.HandleFunc("POST /tasks", handler.Create)
	mux.HandleFunc("GET /tasks", handler.List)
	mux.HandleFunc("GET /tasks/{id}", handler.GetByID)
	mux.HandleFunc("PUT /tasks/{id}", handler.Update)
	mux.HandleFunc("DELETE /tasks/{id}", handler.Delete)

	// Middleware chain
	var h http.Handler = mux
	h = recoveryMiddleware(log, h)
	h = loggingMiddleware(log, h)

	return h
}

// NewSecuredRouter registers task routes protected by cryptographic Bearer token authentication and RBAC.
func NewSecuredRouter(svc Service, tokenSvc auth.TokenService, revStore auth.RevocationStore, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	handler := NewHandler(svc)

	// Health check endpoint remains unauthenticated
	mux.HandleFunc("GET /health", handler.Health)

	// Auth and RBAC middleware
	authenticate := auth.Authenticate(tokenSvc, revStore, log)
	requireRead := auth.RequireRole(auth.RoleAdmin, auth.RoleMember, auth.RoleViewer)
	requireWrite := auth.RequireRole(auth.RoleAdmin, auth.RoleMember)
	requireAdmin := auth.RequireRole(auth.RoleAdmin)

	// Secured endpoints with default-deny authentication and granular RBAC
	mux.Handle("GET /tasks", authenticate(requireRead(http.HandlerFunc(handler.List))))
	mux.Handle("GET /tasks/{id}", authenticate(requireRead(http.HandlerFunc(handler.GetByID))))
	mux.Handle("POST /tasks", authenticate(requireWrite(http.HandlerFunc(handler.Create))))
	mux.Handle("PUT /tasks/{id}", authenticate(requireWrite(http.HandlerFunc(handler.Update))))
	mux.Handle("DELETE /tasks/{id}", authenticate(requireAdmin(http.HandlerFunc(handler.Delete))))

	var h http.Handler = mux
	h = recoveryMiddleware(log, h)
	h = loggingMiddleware(log, h)

	return h
}

// recoveryMiddleware intercepts unhandled panics, logs stack traces via slog, and sends a safe 500 response.
func recoveryMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if log != nil {
					log.Error("panic recovered in HTTP handler",
						"panic", rec,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"path", r.URL.Path,
					)
				}
				response.Error(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs incoming HTTP requests with method, path, and duration (sanitizing authorization header).
func loggingMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if log != nil {
			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", time.Since(start).String(),
			)
		}
	})
}
